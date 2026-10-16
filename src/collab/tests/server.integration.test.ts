import {it,expect,afterEach} from 'vitest';
import WebSocket from 'ws';
import * as Y from 'yjs';
import {encodeFrame} from '@collab/contracts';
import {createEncoder,writeVarUint,writeVarUint8Array,toUint8Array} from 'lib0/encoding';
import {Persistence} from '../src/persistence.js';
import {startServer} from '../src/server.js';
import {fixture,signed} from './fixture.js';
let cleanup:(()=>Promise<void>)[]=[];afterEach(async()=>{const closing=cleanup;cleanup=[];for(const close of closing.reverse())await close();});
async function setup(){const f=await fixture();cleanup.push(()=>f.close());const store=new Persistence(f.pool,f.authority,{promoteV2:true});const server=await startServer(store,{host:'127.0.0.1',port:0,internalHost:'127.0.0.1',internalPort:0,internalSecret:'i'.repeat(32),allowedOrigins:['http://localhost:3000'],updatesPerMinute:120,maxBufferedBytes:4*1024*1024});cleanup.push(()=>server.close());const now=Math.floor(Date.now()/1000),token=await signed({iss:'whiteboard-api',aud:'whiteboard-collab',purpose:'room',iat:now,exp:now+600,jti:'id',sub:f.principal.principalId,...f.ref,principalId:f.principal.principalId,principalType:'user',grantId:'membership:'+f.ref.workspaceId+':'+f.principal.principalId,access:'edit',aclVersion:'2'});return {...f,store,server,token};}
async function socket(f:Awaited<ReturnType<typeof setup>>){const ws=new WebSocket(f.server.url+'/collab',{origin:'http://localhost:3000'});const messages:any[]=[];ws.on('message',(data,binary)=>messages.push(binary?{binary:data}:JSON.parse(data.toString())));await new Promise<void>((resolve,reject)=>{ws.once('open',resolve);ws.once('error',reject);});ws.send(JSON.stringify({type:'auth',requestId:'auth',payload:{token:f.token}}));await until(()=>messages.some(m=>m.type==='ready'));return {ws,messages};}
async function until(fn:()=>boolean){for(let i=0;i<200&&!fn();i++)await new Promise(r=>setTimeout(r,10));expect(fn()).toBe(true);}
function change(doc:Y.Doc){const client=new Y.Doc({gc:false});Y.applyUpdate(client,Y.encodeStateAsUpdate(doc));const vector=Y.encodeStateVector(client);client.getMap('appState').set('theme','dark');return Y.encodeStateAsUpdate(client,vector);}
it('real WebSocket acknowledges only committed updates and retries same sequence',async()=>{const f=await setup(),a=await socket(f),state=await f.store.read(f.ref,f.principal),bytes=change(state.doc);a.ws.send(encodeFrame({channel:3,updateId:'socket-write',payload:bytes}));await until(()=>a.messages.some(m=>m.type==='ack'));const first=a.messages.find(m=>m.type==='ack').payload;expect((await f.pool.query('SELECT count(*) FROM collab_receipts')).rows[0].count).toBe('1');a.ws.send(encodeFrame({channel:3,updateId:'socket-write',payload:bytes}));await until(()=>a.messages.filter(m=>m.type==='ack').length===2);expect(a.messages.filter(m=>m.type==='ack')[1].payload).toEqual(first);a.ws.close();});
it('database loss emits no ack or broadcast; readonly sync-step2 cannot persist',async()=>{const f=await setup(),a=await socket(f),state=await f.store.read(f.ref,f.principal),bytes=change(state.doc);await f.pool.query("CREATE FUNCTION kill_append() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN PERFORM pg_terminate_backend(pg_backend_pid()); RETURN NEW; END $$; CREATE TRIGGER kill_append BEFORE INSERT ON collab_updates FOR EACH ROW EXECUTE FUNCTION kill_append()");a.ws.send(encodeFrame({channel:3,updateId:'fail',payload:bytes}));await until(()=>a.messages.some(m=>m.type==='error'));expect(a.messages.some(m=>m.type==='ack'||m.binary)).toBe(false);expect((await f.pool.query('SELECT count(*) FROM collab_receipts')).rows[0].count).toBe('0');a.ws.close();});
it('viewer sync-step2 is forbidden even with a valid writable Yjs payload',async()=>{const f=await setup();const decide=f.authority.decide.bind(f.authority);f.authority.decide=async(...args)=>({...await decide(...args),access:'view'});const a=await socket(f),state=await f.store.read(f.ref,f.principal);const enc=createEncoder();writeVarUint(enc,1);writeVarUint8Array(enc,change(state.doc));a.ws.send(encodeFrame({channel:1,updateId:'viewer-step2',payload:toUint8Array(enc)}));await until(()=>a.messages.some(m=>m.type==='error'));expect(a.messages.find(m=>m.type==='error').payload.code).toBe('FORBIDDEN');expect((await f.pool.query('SELECT count(*) FROM collab_receipts')).rows[0].count).toBe('0');a.ws.close();});
it('ACL NOTIFY invalidates existing sockets and forces admission again',async()=>{const f=await setup(),a=await socket(f);await f.pool.query("SELECT pg_notify('acl_changed',$1)",[JSON.stringify({workspaceId:f.ref.workspaceId})]);await until(()=>a.messages.some(m=>m.type==='resync_required'));expect(a.messages.find(m=>m.type==='resync_required').payload.reason).toBe('ACL_CHANGED');});

it('lost LISTEN connection forces ACL resync and recovers readiness',async()=>{const f=await setup(),a=await socket(f);const listener=(await f.pool.query("SELECT pid FROM pg_stat_activity WHERE application_name=$1 AND query='LISTEN acl_changed'",[f.schema])).rows[0];expect(listener).toBeTruthy();await f.pool.query('SELECT pg_terminate_backend($1)',[listener.pid]);await until(()=>a.messages.some(m=>m.type==='resync_required'));expect(a.messages.find(m=>m.type==='resync_required').payload.reason).toBe('ACL_RESYNC_REQUIRED');await until(()=>a.ws.readyState===WebSocket.CLOSED);let ready=false;for(let i=0;i<150&&!ready;i++){ready=(await fetch(f.server.httpURL+'/readyz')).status===200;if(!ready)await new Promise(r=>setTimeout(r,10));}expect(ready).toBe(true);});
it('single active runtime owner is enforced per schema',async()=>{const f=await setup();await expect(startServer(f.store,{host:'127.0.0.1',port:0,internalHost:'127.0.0.1',internalPort:0,internalSecret:'i'.repeat(32),allowedOrigins:['http://localhost:3000'],updatesPerMinute:120,maxBufferedBytes:4194304})).rejects.toThrow(/one collab process/);expect((await fetch(f.server.httpURL+'/readyz')).status).toBe(200);});

it('private command listener requires its own credential and verifies exact original bytes',async()=>{
 const f=await setup();await f.store.read(f.ref,f.principal);const {rectangle,grantSecret}=await import('./fixture.js');
 const bytes=Buffer.from(JSON.stringify({...f.ref,requestId:'http-command',expectedSequence:'0',commands:[{op:'add',elements:[rectangle()]}]}));
 const decision=await f.authority.decide('fixture',f.ref,'scene:command','http-command',bytes,'0'),grant=await signed(decision,grantSecret),path='/internal/scenes/'+f.ref.sceneId+'/commands';
 const send=(url:string,secret:string)=>fetch(url+path,{method:'POST',headers:{authorization:'Bearer '+secret,'content-type':'application/json'},body:JSON.stringify({grant,mutationBase64:bytes.toString('base64')})});
 expect((await send(f.server.httpURL,'i'.repeat(32))).status).toBe(404);expect((await send(f.server.internalURL,'wrong')).status).toBe(401);
 const response=await send(f.server.internalURL,'i'.repeat(32));expect(response.status).toBe(200);expect(await response.json()).toMatchObject({requestId:'http-command',sequence:'1'});
});

it('backpressure tells the client to resync while durable receipt remains replayable',async()=>{
 const f=await fixture();cleanup.push(()=>f.close());const store=new Persistence(f.pool,f.authority,{promoteV2:true}),state=await store.read(f.ref,f.principal),bytes=change(state.doc),receipt=await store.appendUpdate(f.ref,f.principal,'retained',bytes);
 const server=await startServer(store,{host:'127.0.0.1',port:0,internalHost:'127.0.0.1',internalPort:0,internalSecret:'i'.repeat(32),allowedOrigins:['http://localhost:3000'],updatesPerMinute:120,maxBufferedBytes:32});cleanup.push(()=>server.close());
 const now=Math.floor(Date.now()/1000),token=await signed({iss:'whiteboard-api',aud:'whiteboard-collab',purpose:'room',iat:now,exp:now+600,jti:'id',sub:f.principal.principalId,...f.ref,principalId:f.principal.principalId,principalType:'user',grantId:'membership:'+f.ref.workspaceId+':'+f.principal.principalId,access:'edit',aclVersion:'2'});
 const ws=new WebSocket(server.url+'/collab',{origin:'http://localhost:3000'}),messages:any[]=[];ws.on('message',d=>messages.push(JSON.parse(d.toString())));await new Promise<void>(r=>ws.once('open',r));ws.send(JSON.stringify({type:'auth',requestId:'auth',payload:{token}}));await until(()=>messages.some(m=>m.type==='resync_required'));expect(messages[0].payload.reason).toBe('BACKPRESSURE');expect(await store.appendUpdate(f.ref,f.principal,'retained',bytes)).toEqual(receipt);
});

async function eventually(check:()=>Promise<boolean>){for(let i=0;i<300;i++){if(await check())return;await new Promise(r=>setTimeout(r,10));}throw new Error('Expected database/server state not observed');}
async function signedCommand(f:Awaited<ReturnType<typeof setup>>,id:string){const {grantSecret}=await import('./fixture.js');const bytes=Buffer.from(JSON.stringify({...f.ref,requestId:id,expectedSequence:'0',commands:[{op:'add',elements:[]}]}));const decision=await f.authority.decide('fixture',f.ref,'scene:command',id,bytes,'0');return {grant:await signed(decision,grantSecret),mutationBase64:bytes.toString('base64')};}
async function submitCommand(f:Awaited<ReturnType<typeof setup>>,id:string){const envelope=await signedCommand(f,id);return fetch(f.server.internalURL+'/internal/scenes/'+f.ref.sceneId+'/commands',{method:'POST',headers:{authorization:'Bearer '+'i'.repeat(32),'content-type':'application/json'},body:JSON.stringify(envelope)});}
it('ownership loss fences private commands until the primary owner can be reacquired',async()=>{
 const f=await setup();await f.store.read(f.ref,f.principal);let committed=0;f.store.on('committed',()=>committed++);const competitor=await f.pool.connect();
 try{
  const listener=(await f.pool.query("SELECT pid FROM pg_stat_activity WHERE application_name=$1 AND query='LISTEN acl_changed'",[f.schema])).rows[0];await competitor.query('SELECT pg_terminate_backend($1)',[listener.pid]);
  await eventually(async()=>{return (await competitor.query("SELECT pg_try_advisory_lock(hashtext(current_database()),hashtext(current_schema() || ':collab-writer')) AS owned")).rows[0].owned;});
  await eventually(async()=>(await fetch(f.server.httpURL+'/readyz')).status===503);
  const denied=await submitCommand(f,'after-owner-loss');expect(denied.status).toBe(503);expect((await f.pool.query('SELECT count(*) FROM collab_receipts')).rows[0].count).toBe('0');expect(committed).toBe(0);
  await competitor.query('SELECT pg_advisory_unlock_all()');await eventually(async()=>(await fetch(f.server.httpURL+'/readyz')).status===200);expect((await submitCommand(f,'reacquired')).status).toBe(200);
 }finally{await competitor.query('SELECT pg_advisory_unlock_all()');competitor.release();}
});
it('ownership handoff waits for an in-flight append and lost ownership rolls that append back',async()=>{
 const f=await setup();await f.store.read(f.ref,f.principal);const blocker=await f.pool.connect(),competitor=await f.pool.connect();let committed=0;f.store.on('committed',()=>committed++);const key=Math.floor(Math.random()*1e9);
 try{
  const blockerPid=(await blocker.query('SELECT pg_backend_pid() AS pid')).rows[0].pid;await blocker.query('SELECT pg_advisory_lock($1)',[key]);
  await f.pool.query(`CREATE FUNCTION pause_owner_append() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN PERFORM pg_advisory_xact_lock(${key}); RETURN NEW; END $$; CREATE TRIGGER pause_owner_append BEFORE INSERT ON collab_updates FOR EACH ROW EXECUTE FUNCTION pause_owner_append()`);
  const submitted=submitCommand(f,'in-flight-owner-loss');void submitted.catch(()=>{});await eventually(async()=>(await f.pool.query('SELECT EXISTS(SELECT 1 FROM pg_stat_activity WHERE $1=ANY(pg_blocking_pids(pid))) AS blocked',[blockerPid])).rows[0].blocked);
  const listener=(await f.pool.query("SELECT pid FROM pg_stat_activity WHERE application_name=$1 AND query='LISTEN acl_changed'",[f.schema])).rows[0];await competitor.query('SELECT pg_terminate_backend($1)',[listener.pid]);
  await eventually(async()=>(await competitor.query("SELECT pg_try_advisory_lock(hashtext(current_database()),hashtext(current_schema() || ':collab-writer')) AS owned")).rows[0].owned);
  expect((await competitor.query("SELECT pg_try_advisory_lock(hashtext(current_database()),hashtext(current_schema() || ':collab-handoff')) AS owned")).rows[0].owned).toBe(false);
  await blocker.query('SELECT pg_advisory_unlock_all()');expect((await submitted).status).toBe(503);expect((await f.pool.query('SELECT count(*) FROM collab_receipts')).rows[0].count).toBe('0');expect(committed).toBe(0);
 }finally{await blocker.query('SELECT pg_advisory_unlock_all()');await competitor.query('SELECT pg_advisory_unlock_all()');blocker.release();competitor.release();}
});

it('loss after final precommit check retains commit receipt but suppresses old acknowledgement and publication',async()=>{
 const f=await setup();await f.store.read(f.ref,f.principal);const competitor=await f.pool.connect();let committed=0;f.store.on('committed',()=>committed++);
 let reached!:()=>void,release!:()=>void;const atCommit=new Promise<void>(r=>reached=r),gate=new Promise<void>(r=>release=r);let armed=true;
 const originalPool=f.store.pool;f.store.pool=new Proxy(originalPool,{get(target,key){if(key==='connect')return async()=>{const client=await target.connect(),query=client.query.bind(client);client.query=(async(text:any,...args:any[])=>{if(text==='COMMIT'&&armed){armed=false;reached();await gate;}return (query as any)(text,...args);}) as any;return client;};const value=Reflect.get(target,key);return typeof value==='function'?value.bind(target):value;}});
 try{
  const submitted=submitCommand(f,'committed-during-loss');void submitted.catch(()=>{});await atCommit;
  const listener=(await f.pool.query("SELECT pid FROM pg_stat_activity WHERE application_name=$1 AND query='LISTEN acl_changed'",[f.schema])).rows[0];await competitor.query('SELECT pg_terminate_backend($1)',[listener.pid]);
  await eventually(async()=>(await competitor.query("SELECT pg_try_advisory_lock(hashtext(current_database()),hashtext(current_schema() || ':collab-writer')) AS owned")).rows[0].owned);
  expect((await competitor.query("SELECT pg_try_advisory_lock(hashtext(current_database()),hashtext(current_schema() || ':collab-handoff')) AS owned")).rows[0].owned).toBe(false);
  release();const response=await submitted;expect(response.status).toBe(503);expect(committed).toBe(0);
  const saved=(await f.pool.query('SELECT sequence FROM collab_receipts WHERE request_id=$1',['committed-during-loss'])).rows[0];expect(saved.sequence).toBe('1');
  await competitor.query('SELECT pg_advisory_unlock_all()');await eventually(async()=>(await fetch(f.server.httpURL+'/readyz')).status===200);
  const replay=await submitCommand(f,'committed-during-loss');expect(replay.status).toBe(200);expect(await replay.json()).toMatchObject({requestId:'committed-during-loss',sequence:saved.sequence});expect(committed).toBe(0);expect((await f.pool.query('SELECT count(*) FROM collab_mutation_audit')).rows[0].count).toBe('1');
 }finally{release();f.store.pool=originalPool;await competitor.query('SELECT pg_advisory_unlock_all()');competitor.release();}
});
it('restart skips dirty trash, compacts another room without disconnects and restores acknowledged content',async()=>{
 const f=await setup(),state=await f.store.read(f.ref,f.principal),bytes=change(state.doc);await f.store.appendUpdate(f.ref,f.principal,'trash-before-restart',bytes);
 const {randomUUID}=await import('node:crypto');const activeRef={...f.ref,sceneId:randomUUID()};await f.pool.query("INSERT INTO scenes(id,workspace_id,title,content) VALUES($1,$2,'Active after restart',$3)",[activeRef.sceneId,activeRef.workspaceId,{elements:[],appState:{},files:{}}]);
 const activeState=await f.store.read(activeRef,f.principal);await f.store.appendUpdate(activeRef,f.principal,'active-before-restart',change(activeState.doc));
 await f.pool.query('UPDATE scenes SET deleted_at=clock_timestamp() WHERE id=$1',[f.ref.sceneId]);await f.pool.query("UPDATE collab_documents SET snapshot_at=clock_timestamp()-interval '10 seconds'");await f.server.close();
 const restartedStore=new Persistence(f.pool,f.authority,{promoteV2:true}),server=await startServer(restartedStore,{host:'127.0.0.1',port:0,internalHost:'127.0.0.1',internalPort:0,internalSecret:'i'.repeat(32),allowedOrigins:['http://localhost:3000'],updatesPerMinute:120,maxBufferedBytes:4194304});cleanup.push(()=>server.close());
 const {decodeJwt}=await import('jose');const token=await signed({...decodeJwt(f.token),...activeRef});const active=await socket({...f,ref:activeRef,server,token});
 await eventually(async()=>(await f.pool.query('SELECT snapshot_sequence FROM collab_documents WHERE scene_id=$1',[activeRef.sceneId])).rows[0].snapshot_sequence==='1');
 await new Promise(r=>setTimeout(r,1100));expect(active.ws.readyState).toBe(WebSocket.OPEN);expect(active.messages.some(m=>m.type==='resync_required')).toBe(false);expect((await f.pool.query('SELECT count(*) FROM collab_updates WHERE scene_id=$1',[f.ref.sceneId])).rows[0].count).toBe('1');
 await f.pool.query('UPDATE scenes SET deleted_at=NULL WHERE id=$1',[f.ref.sceneId]);const restored=await restartedStore.read(f.ref,f.principal);expect(restored.documentId).toBe(state.documentId);expect(restored.content.appState.theme).toBe('dark');expect((await f.pool.query('SELECT count(*) FROM collab_receipts WHERE scene_id=$1',[f.ref.sceneId])).rows[0].count).toBe('1');active.ws.close();
});

it('a replacement runtime cannot become ready until the old transaction crosses the handoff barrier',async()=>{
 const f=await setup();const state=await f.store.read(f.ref,f.principal);const blocker=await f.pool.connect();const key=Math.floor(Math.random()*1e9);let replacement:Awaited<ReturnType<typeof startServer>>|undefined;let starting:ReturnType<typeof startServer>|undefined;
 try{
  const pid=(await blocker.query('SELECT pg_backend_pid() AS pid')).rows[0].pid;await blocker.query('SELECT pg_advisory_lock($1)',[key]);await f.pool.query(`CREATE FUNCTION pause_takeover_append() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN PERFORM pg_advisory_xact_lock(${key}); RETURN NEW; END $$; CREATE TRIGGER pause_takeover_append BEFORE INSERT ON collab_updates FOR EACH ROW EXECUTE FUNCTION pause_takeover_append()`);
  const pending=f.store.appendUpdate(f.ref,f.principal,'replacement-owner',change(state.doc));void pending.catch(()=>{});await eventually(async()=>(await f.pool.query('SELECT EXISTS(SELECT 1 FROM pg_stat_activity WHERE $1=ANY(pg_blocking_pids(pid))) AS blocked',[pid])).rows[0].blocked);
  const listener=(await f.pool.query("SELECT pid FROM pg_stat_activity WHERE application_name=$1 AND query='LISTEN acl_changed'",[f.schema])).rows[0];await blocker.query('SELECT pg_terminate_backend($1)',[listener.pid]);
  // Stop automatic reacquisition so this test chooses the replacement deterministically.
  await f.server.close();await eventually(async()=>(await blocker.query("SELECT NOT EXISTS(SELECT 1 FROM pg_locks WHERE pid=$1 AND locktype='advisory') AS gone",[listener.pid])).rows[0].gone);
  let becameReady=false;starting=startServer(new Persistence(f.pool,f.authority,{promoteV2:true}),{host:'127.0.0.1',port:0,internalHost:'127.0.0.1',internalPort:0,internalSecret:'i'.repeat(32),allowedOrigins:['http://localhost:3000'],updatesPerMinute:120,maxBufferedBytes:4194304});void starting.then(server=>{replacement=server;becameReady=true;}).catch(()=>{});
  await eventually(async()=>(await f.pool.query("SELECT EXISTS(SELECT 1 FROM pg_stat_activity WHERE application_name=$1 AND query LIKE 'SELECT pg_advisory_lock(%collab-handoff%' AND cardinality(pg_blocking_pids(pid))>0) AS blocked",[f.schema])).rows[0].blocked);expect(becameReady).toBe(false);
  await blocker.query('SELECT pg_advisory_unlock_all()');await expect(pending).rejects.toMatchObject({code:'UNAVAILABLE'});replacement=await starting;expect((await fetch(replacement.httpURL+'/readyz')).status).toBe(200);expect((await f.pool.query('SELECT count(*) FROM collab_receipts')).rows[0].count).toBe('0');
 }finally{await blocker.query('SELECT pg_advisory_unlock_all()');blocker.release();if(starting){try{replacement=await starting;}catch{}}await replacement?.close();}
});
it('shutdown drains a queued reconnect without reacquiring ownership afterward',async()=>{
 const f=await setup();const held:import('pg').PoolClient[]=[];
 try{
  const listener=(await f.pool.query("SELECT pid FROM pg_stat_activity WHERE application_name=$1 AND query='LISTEN acl_changed'",[f.schema])).rows[0];await f.pool.query('SELECT pg_terminate_backend($1)',[listener.pid]);
  held.push(...await Promise.all(Array.from({length:8},()=>f.pool.connect())));await eventually(async()=>f.pool.waitingCount>0);
  const closing=f.server.close();held.pop()!.release();await closing;
  expect((await held[0].query("SELECT pg_try_advisory_lock(hashtext(current_database()),hashtext(current_schema() || ':collab-writer')) AS owned")).rows[0].owned).toBe(true);await held[0].query('SELECT pg_advisory_unlock_all()');expect(f.pool.waitingCount).toBe(0);
 }finally{for(const client of held)client.release();}
});
