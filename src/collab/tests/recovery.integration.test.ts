import {afterEach,expect,it} from 'vitest';
import {mkdtemp,rm,readFile,mkdir,writeFile} from 'node:fs/promises';
import {tmpdir} from 'node:os';
import {join} from 'node:path';
import * as Y from 'yjs';
import WebSocket from 'ws';
import {encodeFrame} from '@collab/contracts';
import {fixture,rectangle,signed} from './fixture.js';
import {Persistence} from '../src/persistence.js';
import {startServer} from '../src/server.js';
import {readArchive,sealCheckpoint,verifyArchive,writeArchive} from '../src/recovery.js';
import {checkpointFixture,recoverFixture,restoreFixture,guardDatabase} from './recovery-fixture.js';
const database=postgres://test:test@127.0.0.1:5432/whiteboard_test?sslmode=disable'postgres://test:test@127.0.0.1:5432/whiteboard_test?sslmode=disable';
const cleanup:(()=>Promise<unknown>)[]=[];
afterEach(async()=>{for(const close of cleanup.splice(0).reverse())await close();});
async function setup(){guardDatabase(database);const f=await fixture();cleanup.push(()=>f.close());const dir=await mkdtemp(join(tmpdir(),'recovery-drill-'));cleanup.push(()=>rm(dir,{recursive:true,force:true}));return {...f,path:join(dir,'checkpoint.json'),store:new Persistence(f.pool,f.authority,{promoteV2:true})};}
function update(doc:Y.Doc,color:string){const copy=new Y.Doc({gc:false});try{Y.applyUpdate(copy,Y.encodeStateAsUpdate(doc));const before=Y.encodeStateVector(copy);copy.getMap('appState').set('viewBackgroundColor',color);return Y.encodeStateAsUpdate(copy,before);}finally{copy.destroy();}}
async function until(check:()=>boolean){for(let i=0;i<300;i++){if(check())return;await new Promise(resolve=>setTimeout(resolve,10));}throw new Error('Timed out waiting for socket evidence');}
async function connect(server:Awaited<ReturnType<typeof startServer>>,f:Awaited<ReturnType<typeof setup>>){
 const ws=new WebSocket(server.url+'/collab',{origin:'http://localhost:3000'});ws.on('error',()=>{});cleanup.push(async()=>{ws.terminate();});const messages:any[]=[];ws.on('message',(data,binary)=>{if(!binary)messages.push(JSON.parse(data.toString()));});await new Promise<void>(resolve=>ws.once('open',resolve));
 const now=Math.floor(Date.now()/1000),token=await signed({iss:'whiteboard-api',aud:'whiteboard-collab',purpose:'room',iat:now,exp:now+600,jti:'recovery',sub:f.principal.principalId,...f.ref,principalId:f.principal.principalId,principalType:'user',grantId:'membership:'+f.ref.workspaceId+':'+f.principal.principalId,access:'edit',aclVersion:'2'});
 ws.send(JSON.stringify({type:'auth',requestId:'auth',payload:{token}}));return {ws,messages};
}
const config={host:'127.0.0.1',port:0,internalHost:'127.0.0.1',internalPort:0,internalSecret:'i'.repeat(32),allowedOrigins:['http://localhost:3000'],updatesPerMinute:120,maxBufferedBytes:4194304};
it.each(['postgres://u@10.0.0.1:55439/devtool_a1_test','postgres://u@127.0.0.1:5432/devtool_a1_test','postgres://u@127.0.0.1:55439/production','postgres://u@127.0.0.1:55439/devtool_a1_test?host=remote'])('refuses unsafe target %s before database access',url=>{expect(()=>guardDatabase(url)).toThrow(/Disposable/);});
it('drains real writer, archives all history, fences old queues and restores in another schema',async()=>{
 const f=await setup();const original={elements:[rectangle('retained-element')],appState:{theme:'light'},files:{}};await f.pool.query('UPDATE scenes SET content=$2,content_version=7 WHERE id=$1',[f.ref.sceneId,JSON.stringify(original)]);
 const runtime=await startServer(f.store,config);let closed=false;cleanup.push(async()=>{if(!closed)await runtime.close();});
 const client=await connect(runtime,f);await until(()=>client.messages.some(m=>m.type==='ready'));
 const initial=await f.store.read(f.ref,f.principal);const first=await f.store.appendUpdate(f.ref,f.principal,'before-compaction',update(initial.doc,'#123456'));initial.doc.destroy();await f.store.compactScene(f.ref);
 const state=await f.store.read(f.ref,f.principal);const secondBytes=update(state.doc,'#abcdef');const queued=update(state.doc,'#fedcba');state.doc.destroy();
 client.ws.send(encodeFrame({channel:3,updateId:'after-compaction',payload:secondBytes}));await until(()=>client.messages.some(m=>m.type==='ack'&&m.requestId==='after-compaction'));
 expect(first.sequence).toBe('1');expect(client.messages.find(m=>m.type==='ack'&&m.requestId==='after-compaction').payload.sequence).toBe('2');
 await expect(checkpointFixture(f.pool,f.ref,f.path)).rejects.toThrow(/active writer/);
 await runtime.close();closed=true;await until(()=>client.ws.readyState===WebSocket.CLOSED);
 await expect(f.store.appendUpdate(f.ref,f.principal,'after-stop',queued)).rejects.toMatchObject({code:'UNAVAILABLE'});
 const archive=await checkpointFixture(f.pool,f.ref,f.path);const bytesBefore=await readFile(f.path,'utf8');
 expect(archive.checkpoint.snapshotSequence).toBe('1');expect(archive.checkpoint.updates.map((u:any)=>u.sequence)).toEqual(['2']);expect(archive.checkpoint.receipts.map((r:any)=>r.request_id)).toEqual(['before-compaction','after-compaction']);expect(archive.checkpoint.audit).toHaveLength(2);expect(archive.checkpoint.promotionCheckpoint).toEqual({content:original,contentVersion:'7'});expect(verifyArchive(archive).appState.viewBackgroundColor).toBe('#abcdef');
 const replacement=await recoverFixture(f.pool,f.path);expect(replacement.sceneId).not.toBe(f.ref.sceneId);
 expect((await f.pool.query('SELECT writer_mode,content FROM scenes WHERE id=$1',[replacement.sceneId])).rows[0]).toEqual({writer_mode:'v1',content:archive.checkpoint.content});
 expect((await f.pool.query('SELECT count(*) FROM collab_receipts WHERE scene_id=$1',[f.ref.sceneId])).rows[0].count).toBe('2');
 const restarted=new Persistence(f.pool,f.authority,{promoteV2:true}),nextRuntime=await startServer(restarted,config);cleanup.push(()=>nextRuntime.close());
 const pending={sceneId:f.ref.sceneId,documentId:archive.checkpoint.documentId,requestId:'offline-old-epoch',bytes:Buffer.from(queued).toString('base64')};const pendingBefore=JSON.stringify(pending);
 await expect(restarted.appendUpdate(f.ref,f.principal,pending.requestId,queued)).rejects.toMatchObject({code:'FORBIDDEN'});
 const old=await connect(nextRuntime,f);await until(()=>old.messages.some(m=>m.type==='error'));expect(old.messages.find(m=>m.type==='error').payload.code).toBe('FORBIDDEN');await until(()=>old.messages.some(m=>m.type==='resync_required'));expect(old.messages.some(m=>m.type==='ready'||m.type==='ack')).toBe(false);
 const promoted=await restarted.read(replacement,f.principal);expect(promoted.documentId).not.toBe(archive.checkpoint.documentId);expect(promoted.sequence).toBe('0');expect(promoted.content).toEqual(archive.checkpoint.content);promoted.doc.destroy();expect(JSON.stringify(pending)).toBe(pendingBefore);
 expect((await f.pool.query('SELECT count(*) FROM collab_receipts WHERE request_id=$1',[pending.requestId])).rows[0].count).toBe('0');expect(await readFile(f.path,'utf8')).toBe(bytesBefore);
 const target=await setup();await restoreFixture(target.pool,f.path,target.principal.principalId);const restored=await target.store.exportRecovery(f.ref);expect(sealCheckpoint(restored).sha256).toBe(archive.sha256);expect((await readArchive(f.path)).sha256).toBe(archive.sha256);
 const evidence={proof:'E3 isolated replacement recovery',sequence:'2',snapshotSequence:'1',remainingUpdates:1,receipts:2,audit:2,archiveSHA256:archive.sha256,oldScene:f.ref.sceneId,newScene:replacement.sceneId,oldDocument:archive.checkpoint.documentId,newDocument:promoted.documentId,restoredOriginalHistory:true};
 if(process.env.RECOVERY_EVIDENCE_DIR){const dir=process.env.RECOVERY_EVIDENCE_DIR;await mkdir(dir,{recursive:true,mode:0o700});await writeArchive(join(dir,'checkpoint.json'),archive.checkpoint);await writeFile(join(dir,'result.json'),JSON.stringify(evidence,null,2)+'\n',{flag:'wx',mode:0o600});}
 console.log(JSON.stringify(evidence));
},20000);
it.each(['asset metadata','scene override'])('refuses %s without trashing source or changing history',async kind=>{
 const f=await setup();(await f.store.read(f.ref,f.principal)).doc.destroy();
 if(kind==='asset metadata')await f.pool.query("INSERT INTO scene_assets(workspace_id,scene_id,asset_kind,storage_bucket,storage_key,mime_type,byte_size) VALUES($1,$2,'excalidraw_file','fixture','original.png','image/png',1)",[f.ref.workspaceId,f.ref.sceneId]);
 else await f.pool.query("INSERT INTO scene_member_overrides(workspace_id,scene_id,user_id,role) VALUES($1,$2,$3,'viewer')",[f.ref.workspaceId,f.ref.sceneId,f.principal.principalId]);
 await checkpointFixture(f.pool,f.ref,f.path);await expect(recoverFixture(f.pool,f.path)).rejects.toThrow(/Unsupported/);expect((await f.pool.query('SELECT deleted_at FROM scenes WHERE id=$1',[f.ref.sceneId])).rows[0].deleted_at).toBeNull();expect((await f.pool.query('SELECT count(*) FROM scenes')).rows[0].count).toBe('1');expect((await f.pool.query('SELECT count(*) FROM collab_documents')).rows[0].count).toBe('1');
});
it('refuses a checkpoint made before a later durable write',async()=>{const f=await setup(),state=await f.store.read(f.ref,f.principal);await checkpointFixture(f.pool,f.ref,f.path);await f.store.appendUpdate(f.ref,f.principal,'later-write',update(state.doc,'#111111'));state.doc.destroy();await expect(recoverFixture(f.pool,f.path)).rejects.toThrow(/changed/);expect((await f.pool.query('SELECT deleted_at FROM scenes WHERE id=$1',[f.ref.sceneId])).rows[0].deleted_at).toBeNull();expect((await f.pool.query('SELECT count(*) FROM collab_receipts')).rows[0].count).toBe('1');});
