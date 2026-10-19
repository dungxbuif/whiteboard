import {it,expect} from 'vitest';
import {createServer} from 'node:http';
import {spawn} from 'node:child_process';
import {once} from 'node:events';
import * as Y from 'yjs';
import WebSocket from 'ws';
import {encodeFrame} from '@collab/contracts';
import {fixture,signed,secret,grantSecret} from './fixture.js';
it('built dist server and real dist validation worker persist an actual WebSocket update',async()=>{
 const f=await fixture();let child:ReturnType<typeof spawn>|undefined,ws:WebSocket|undefined;let output='';
 const authority=createServer(async(req,res)=>{let body='';for await(const chunk of req)body+=chunk;const input=JSON.parse(body),now=Math.floor(Date.now()/1000);const version=(await f.pool.query('SELECT acl_version::text FROM workspaces WHERE id=$1',[f.ref.workspaceId])).rows[0].acl_version;
 const token=await signed({iss:'whiteboard-api',aud:'whiteboard-mutation',purpose:'mutation-decision',iat:now,exp:now+5,jti:'decision',sub:f.principal.principalId,...f.ref,principalId:f.principal.principalId,principalType:'user',grantId:'membership:'+f.ref.workspaceId+':'+f.principal.principalId,access:'edit',aclVersion:version,action:input.action,requestId:input.requestId,payloadHash:input.payloadHash},grantSecret);res.setHeader('content-type','application/json');res.end(JSON.stringify({token}));});
 try{
  await new Promise<void>(r=>authority.listen(0,'127.0.0.1',r));const port=(authority.address() as any).port;
  const reserve=createServer();await new Promise<void>(r=>reserve.listen(0,'127.0.0.1',r));const publicPort=(reserve.address() as any).port;await new Promise<void>(r=>reserve.close(()=>r()));
  const reserve2=createServer();await new Promise<void>(r=>reserve2.listen(0,'127.0.0.1',r));const privatePort=(reserve2.address() as any).port;await new Promise<void>(r=>reserve2.close(()=>r()));
  const url=new URL(process.env.COLLAB_TEST_DATABASE_URL??'postgres://test:test@127.0.0.1:5432/whiteboard_test?sslmode=disable');url.searchParams.set('options','-c search_path='+f.schema+',public');
  child=spawn(process.execPath,['dist/server.js'],{cwd:new URL('..',import.meta.url),env:{...process.env,DATABASE_URL:url.toString(),COLLAB_JWT_SECRET:secret,MUTATION_GRANT_SECRET:grantSecret,AUTHORITY_SERVICE_SECRET:'s'.repeat(32),AUTHORITY_URL:'http://127.0.0.1:'+port,COLLAB_INTERNAL_SECRET:'i'.repeat(32),COLLAB_PORT:String(publicPort),COLLAB_INTERNAL_PORT:String(privatePort),COLLAB_ALLOWED_ORIGINS:'http://localhost:3000',COLLAB_V2_ENABLED:'true'},stdio:['ignore','pipe','pipe']});child.stdout?.on('data',b=>output+=b);child.stderr?.on('data',b=>output+=b);
  let ready=false;for(let i=0;i<200&&!ready;i++){try{ready=(await fetch('http://127.0.0.1:'+publicPort+'/readyz')).status===200;}catch{}if(!ready)await new Promise(r=>setTimeout(r,10));}expect(ready,output).toBe(true);
  const now=Math.floor(Date.now()/1000),token=await signed({iss:'whiteboard-api',aud:'whiteboard-collab',purpose:'room',iat:now,exp:now+600,jti:'room',sub:f.principal.principalId,...f.ref,principalId:f.principal.principalId,principalType:'user',grantId:'membership:'+f.ref.workspaceId+':'+f.principal.principalId,access:'edit',aclVersion:'2'});
  ws=new WebSocket('ws://127.0.0.1:'+publicPort+'/collab',{origin:'http://localhost:3000'});const messages:any[]=[];ws.on('message',(b,binary)=>{if(!binary)messages.push(JSON.parse(b.toString()));});await once(ws,'open');ws.send(JSON.stringify({type:'auth',requestId:'auth',payload:{token}}));for(let i=0;i<200&&!messages.some(m=>m.type==='sync_chunk');i++)await new Promise(r=>setTimeout(r,10));
  const chunk=messages.find(m=>m.type==='sync_chunk');expect(chunk,output).toBeTruthy();const doc=new Y.Doc({gc:false});Y.applyUpdate(doc,Buffer.from(chunk.payload.data,'base64'));const vector=Y.encodeStateVector(doc);doc.getMap('appState').set('theme','dark');ws.send(encodeFrame({channel:3,updateId:'built-worker',payload:Y.encodeStateAsUpdate(doc,vector)}));
  for(let i=0;i<300&&!messages.some(m=>m.type==='ack');i++)await new Promise(r=>setTimeout(r,10));expect(messages.find(m=>m.type==='ack'),output).toMatchObject({payload:{requestId:'built-worker',sequence:'1'}});expect((await f.pool.query('SELECT count(*) FROM collab_mutation_audit')).rows[0].count).toBe('1');
 }finally{ws?.terminate();if(child){const exited=once(child,'exit');child.kill('SIGTERM');await exited;}await new Promise<void>(r=>authority.close(()=>r()));authority.closeAllConnections();await f.close();}
},15000);
