import {it,expect} from 'vitest';
import {Client} from '@modelcontextprotocol/sdk/client/index.js';
import {InMemoryTransport} from '@modelcontextprotocol/sdk/inMemory.js';
import {createServer} from 'node:http';
import {mkdtempSync,rmSync} from 'node:fs';
import {tmpdir} from 'node:os';
import {join} from 'node:path';
import {createMcpServer} from '../src/server.js';
import {GoClient} from '../src/go-client.js';
import {AuditJournal} from '../src/audit.js';
it('discovers native mutation tools with truthful destructive and idempotent hints',async()=>{
 const api=createServer((_req,res)=>res.end(JSON.stringify({principalId:'agent:aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa',keyId:'aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa'})));
 await new Promise<void>(resolve=>api.listen(0,'127.0.0.1',resolve));
 const dir=mkdtempSync(join(tmpdir(),'d3-tools-'));
 const server=createMcpServer(new GoClient('http://127.0.0.1:'+(api.address() as {port:number}).port,'test'),new AuditJournal(join(dir,'audit.jsonl')));
 const client=new Client({name:'proof',version:'1'});const [a,b]=InMemoryTransport.createLinkedPair();
 try{await server.connect(a);await client.connect(b);const {tools}=await client.listTools();
  expect(tools.find(t=>t.name==='generate_diagram')).toMatchObject({annotations:{readOnlyHint:false,idempotentHint:true,destructiveHint:false}});
  expect(tools.find(t=>t.name==='delete_elements')).toMatchObject({annotations:{readOnlyHint:false,idempotentHint:true,destructiveHint:true}});
  expect(tools.find(t=>t.name==='get_scene_activity')).toMatchObject({annotations:{readOnlyHint:true}});
  const resources=await client.listResources();expect(resources.resources).toHaveLength(20);
  const catalog=await client.readResource({uri:'whiteboard://diagrams/catalog'});expect(JSON.parse((catalog.contents[0] as {text:string}).text)).toHaveLength(19);
  const templates=await client.callTool({name:'get_diagram_template',arguments:{preset:'flowchart'}});expect(templates.isError).not.toBe(true);
  const invalid=await client.callTool({name:'validate_diagram',arguments:{preset:'flowchart',instanceId:'check',data:{nodes:[],edges:[]}}});expect(JSON.parse((invalid.content as {text:string}[])[0].text).valid).toBe(false);
 }finally{await client.close();await server.close();await new Promise<void>(resolve=>api.close(()=>resolve()));rmSync(dir,{recursive:true,force:true});}
});

import {spawn} from 'node:child_process';
import {existsSync,readFileSync,writeFileSync} from 'node:fs';
import {fileURLToPath} from 'node:url';
import {Pool} from 'pg';
import {WebSocket} from 'ws';
import * as Y from 'yjs';
import {decodeFrame,encodeFrame} from '@collab/contracts';
import {createMcpHTTP} from '../src/server.js';
import {StreamableHTTPClientTransport} from '@modelcontextprotocol/sdk/client/streamableHttp.js';
import {Authority} from '../../collab/src/auth.js';
import {Persistence} from '../../collab/src/persistence.js';
import {startServer} from '../../collab/src/server.js';
async function until(check:()=>boolean|Promise<boolean>,timeout=10000){const end=Date.now()+timeout;while(!await check()){if(Date.now()>end)throw new Error('Timed out waiting for live state');await new Promise(r=>setTimeout(r,15));}}
it('applies real MCP → Go → Node mutations with durable receipts, scopes, activity and socket convergence',async()=>{
 const dir=mkdtempSync(join(tmpdir(),'d3-live-'));const ready=join(dir,'ready.json');
 let runtime:Awaited<ReturnType<typeof startServer>>|undefined;let pool:Pool|undefined;let mcp:ReturnType<typeof createServer>|undefined;
 const clients:Client[]=[];let socket:WebSocket|undefined;const browserDoc=new Y.Doc({gc:false});
 const proxy=createServer(async(req,res)=>{if(!runtime){res.writeHead(503).end();return;}const body=[];for await(const b of req)body.push(b);try{const result=await fetch(runtime.internalURL+req.url,{method:req.method,headers:{authorization:String(req.headers.authorization),'content-type':'application/json'},body:Buffer.concat(body)});res.writeHead(result.status,{'content-type':'application/json'}).end(await result.text());}catch{res.writeHead(503).end();}});
 await new Promise<void>(r=>proxy.listen(0,'127.0.0.1',r));
 const url=process.env.ACCESS_TEST_DATABASE_URL??'postgres://test:test@127.0.0.1:5432/whiteboard_test?sslmode=disable';
 const child=spawn('go',['test','./internal/repository/postgres','-run','^TestAgentLiveFixture$','-count=1','-timeout=100s'],{cwd:fileURLToPath(new URL('../../backend/',import.meta.url)),env:{...process.env,PATH:(process.env.PATH??'')+':/opt/homebrew/bin:/usr/local/go/bin',ACCESS_TEST_DATABASE_URL:url,D3_FIXTURE_READY:ready,D3_COLLAB_URL:'http://127.0.0.1:'+(proxy.address() as {port:number}).port}});
 let logs='';child.stdout.on('data',b=>logs+=b);child.stderr.on('data',b=>logs+=b);
 try{
  await until(()=>{if(child.exitCode!==null)throw new Error('Go fixture exited: '+logs);return existsSync(ready)},30000);
  const f=JSON.parse(readFileSync(ready,'utf8'));
  pool=new Pool({connectionString:url,options:'-c search_path='+f.schema+',public',application_name:f.schema,max:8});
  const authority=new Authority({roomSecret:'r'.repeat(32),grantSecret:'g'.repeat(32),serviceSecret:'s'.repeat(32),url:f.api});
  const config={host:'127.0.0.1',port:0,internalHost:'127.0.0.1',internalPort:0,internalSecret:'i'.repeat(32),allowedOrigins:['http://localhost:3000'],updatesPerMinute:120,maxBufferedBytes:4194304};
  runtime=await startServer(new Persistence(pool,authority,{promoteV2:true}),config);
  mcp=createMcpHTTP({apiURL:f.api,allowedOrigins:[],audit:new AuditJournal(join(dir,'audit.jsonl'))});await new Promise<void>(r=>mcp!.listen(0,'127.0.0.1',r));
  const connect=async(key:string)=>{const c=new Client({name:'live',version:'1'});clients.push(c);await c.connect(new StreamableHTTPClientTransport(new URL('http://127.0.0.1:'+(mcp!.address() as {port:number}).port+'/mcp'),{requestInit:{headers:{Authorization:'Bearer '+key}}}));return c};
  const full=await connect(f.keys.full),write=await connect(f.keys.write),read=await connect(f.keys.read);
  const call=async(client:Client,name:string,args:Record<string,unknown>)=>{const r=await client.callTool({name,arguments:args});if(r.isError)throw new Error((r.content as {text:string}[])[0].text);return JSON.parse((r.content as {text:string}[])[0].text)};
  const args={sceneRef:f.ref,preset:'flowchart',data:{nodes:[{id:'a',label:'Start'},{id:'b',label:'End'}],edges:[{id:'ab',from:'a',to:'b'}]},instanceId:'live-diagram',requestId:'generate-one',expectedSequence:'0'};
  const preview=await call(write,'generate_diagram',{...args,dryRun:true});expect(preview.dryRun).toBe(true);expect((await pool.query('SELECT count(*) FROM collab_documents')).rows[0].count).toBe('0');
  // A real browser-protocol socket consumes the same native IDs. Visual browser proof is B3/E1-owned.
  socket=new WebSocket(runtime.url+'/collab',{origin:'http://localhost:3000'});const controls:any[]=[];
  socket.on('message',(raw,binary)=>{if(binary)Y.applyUpdate(browserDoc,decodeFrame(new Uint8Array(raw as Buffer)).payload);else{const frame=JSON.parse(raw.toString());controls.push(frame);if(frame.type==='sync_chunk')Y.applyUpdate(browserDoc,Buffer.from(frame.payload.data,'base64'));}});
  await new Promise<void>(r=>socket!.once('open',r));socket.send(JSON.stringify({type:'auth',requestId:'auth',payload:{token:f.roomToken}}));await until(()=>controls.some(x=>x.type==='ready'));
  const generated=await call(write,'generate_diagram',args);expect(generated.receipt.sequence).toBe('1');expect(generated.createdIds).toEqual(preview.createdIds);
  await until(()=>generated.createdIds.every((id:string)=>browserDoc.getMap('elements').has(id)));
  expect(await call(write,'generate_diagram',args)).toEqual(generated);expect((await pool.query('SELECT count(*) FROM collab_mutation_audit')).rows[0].count).toBe('1');
  await expect(call(write,'generate_diagram',{...args,data:{...args.data,nodes:[{id:'a',label:'Changed'},{id:'b',label:'End'}]}})).rejects.toThrow('REQUEST_ID_REUSED');
  const id=generated.createdIds[0];
  await expect(call(write,'get_scene',f.ref)).rejects.toThrow('ACCESS_DENIED');
  await expect(call(read,'patch_elements',{sceneRef:f.ref,patches:[{id,fields:{x:30}}],requestId:'denied',expectedSequence:'1'})).rejects.toThrow('ACCESS_DENIED');
  // A human edits a different field through the actual socket update writer.
  const before=Y.encodeStateVector(browserDoc);(browserDoc.getMap('elements').get(id) as Y.Map<unknown>).set('y',321);const human=Y.encodeStateAsUpdate(browserDoc,before);
  socket.send(encodeFrame({channel:3,updateId:'human-edit',payload:human}));await until(()=>controls.some(x=>x.type==='ack'&&x.requestId==='human-edit'));
  await expect(call(write,'patch_elements',{sceneRef:f.ref,patches:[{id,fields:{x:77}}],requestId:'stale',expectedSequence:'1'})).rejects.toThrow('CONFLICT');
  const current=await call(full,'get_scene',f.ref);expect(current.sequence).toBe('2');expect(current.content.elements.find((e:any)=>e.id===id).y).toBe(321);
  const patched=await call(write,'patch_elements',{sceneRef:f.ref,patches:[{id,fields:{x:77}}],requestId:'fresh',expectedSequence:current.sequence});expect(patched.receipt.sequence).toBe('3');
  const after=await call(full,'get_scene',f.ref);expect(after.content.elements.find((e:any)=>e.id===id)).toMatchObject({x:77,y:321});
  await call(write,'delete_elements',{sceneRef:f.ref,ids:[id],requestId:'delete-one',expectedSequence:'3'});expect((await call(full,'get_scene',f.ref)).content.elements.find((e:any)=>e.id===id).isDeleted).toBe(true);
  const activity=await call(read,'get_scene_activity',{sceneRef:f.ref,limit:2});expect(activity.items).toHaveLength(2);expect(activity.nextCursor).toBe('2');expect(JSON.stringify(activity)).not.toMatch(/payloadHash|principalId|grant|elements|wb_agent/);expect((await call(read,'get_scene_activity',{sceneRef:f.ref,cursor:activity.nextCursor,limit:2})).items).toHaveLength(2);
  const route=f.api+`/api/v1/agent/workspaces/${f.ref.workspaceId}/scenes/${f.ref.sceneId}`;
  expect((await fetch(route+'/export-projection',{headers:{authorization:'Bearer '+f.keys.export}})).status).toBe(200);expect((await fetch(route,{headers:{authorization:'Bearer '+f.keys.export}})).status).toBe(403);expect((await fetch(route+'/export-projection',{headers:{authorization:'Bearer '+f.keys.read}})).status).toBe(403);
  const exporter=await connect(f.keys.export);
  await expect(call(read,'export_scene',{sceneRef:f.ref,format:'json'})).rejects.toThrow('ACCESS_DENIED');
  const base='http://127.0.0.1:'+(mcp!.address() as {port:number}).port;
  const grants=[];
  for(const format of ['json','svg','png']){
    const artifact=await call(exporter,'export_scene',{sceneRef:f.ref,format});grants.push(artifact);
    expect(artifact.revision).toBe('4');expect(artifact.url).toBe(base+'/mcp/artifacts/'+artifact.id);
    const response=await fetch(artifact.url,{headers:{authorization:'Bearer '+f.keys.export}});expect(response.status).toBe(200);expect(response.headers.get('content-type')).toBe(artifact.mimeType);
    const bytes=Buffer.from(await response.arrayBuffer());if(format==='json')expect(JSON.parse(bytes.toString()).type).toBe('excalidraw');else if(format==='svg')expect(bytes.toString()).toContain('<svg');else expect(bytes.subarray(0,8).toString('hex')).toBe('89504e470d0a1a0a');
    expect((await fetch(artifact.url)).status).toBe(401);
    expect((await fetch(artifact.url,{headers:{authorization:'Bearer '+f.keys.read}})).status).toBe(404);
  }
  // The stdio adapter's remote export method uses only its bearer key and public endpoint.
  const remote=new GoClient(f.api,f.keys.export,undefined,base);const forwarded=await remote.remoteExport({sceneRef:f.ref,format:'json'});
  expect(forwarded.url).toBe(base+'/mcp/artifacts/'+forwarded.id);
  await pool.query('UPDATE agent_keys SET scopes=$1 WHERE id=$2',[JSON.stringify(["scene:read"]),f.keyIds.export]);
  expect((await fetch(grants[0].url,{headers:{authorization:'Bearer '+f.keys.export}})).status).toBe(403);
  await expect(remote.remoteExport({sceneRef:f.ref,format:'json'})).rejects.toMatchObject({code:'ACCESS_DENIED'});
  await pool.query('UPDATE agent_keys SET scopes=$1,revoked_at=now() WHERE id=$2',[JSON.stringify(["scene:export"]),f.keyIds.export]);
  expect((await fetch(grants[0].url,{headers:{authorization:'Bearer '+f.keys.export}})).status).toBe(401);
  await expect(call(exporter,'export_scene',{sceneRef:f.ref,format:'json'})).rejects.toThrow();
  const counts=(await pool.query('SELECT count(*) FROM collab_mutation_audit')).rows[0].count;
  await call(write,'generate_diagram',{...args,requestId:'dry-after',expectedSequence:'4',dryRun:true});expect((await pool.query('SELECT count(*) FROM collab_mutation_audit')).rows[0].count).toBe(counts);
  socket.close();await runtime.close();runtime=await startServer(new Persistence(pool,authority,{promoteV2:true}),config);
  const restored=await call(full,'get_scene',f.ref);expect(restored.documentId).toBe(current.documentId);expect(restored.sequence).toBe('4');expect(restored.content.elements.map((e:any)=>e.id)).toEqual(generated.createdIds);expect(await call(write,'generate_diagram',args)).toEqual(generated);
  const created=await call(full,'create_scene',{workspaceId:f.ref.workspaceId,title:'Created through Go'});
  const createdRef={workspaceId:f.ref.workspaceId,sceneId:created.id};
  const createdDiagram=await call(full,'generate_diagram',{...args,sceneRef:createdRef,instanceId:'created-via-go',requestId:'created-diagram'});
  expect(createdDiagram.receipt.sequence).toBe('1');expect((await call(full,'get_scene',createdRef)).content.elements.map((e:any)=>e.id)).toEqual(createdDiagram.createdIds);
  expect(createdDiagram.sceneUrl).toBe(base+`/w/${f.ref.workspaceId}/scenes/${created.id}`);
  await expect(call(full,'patch_elements',{sceneRef:f.ref,requestId:'immutable-id',expectedSequence:'4',patches:[{id,fields:{id:'changed-id'}}]})).rejects.toThrow('INVALID_UPDATE');
  // A second key cannot claim the first key's receipt; revoked keys cannot replay.
  await expect(call(full,'generate_diagram',args)).rejects.toThrow('REQUEST_ID_REUSED');
  await pool.query('UPDATE agent_keys SET revoked_at=now() WHERE id=$1',[f.keyIds.write]);await expect(call(write,'generate_diagram',args)).rejects.toThrow();
  // All new private reads require ingress authentication and fresh bound authority.
  for(const operation of ['projection','activity','export-projection']){
   const response=await fetch(runtime.internalURL+`/internal/scenes/${f.ref.sceneId}/${operation}`,{method:'POST',headers:{authorization:'Bearer wrong','content-type':'application/json'},body:'{}'});expect(response.status).toBe(401);
   expect((await fetch(runtime.httpURL+`/internal/scenes/${f.ref.sceneId}/${operation}`,{method:'POST'})).status).toBe(404);
  }
  const competitor=await pool.connect();
  try{
   const listener=(await pool.query("SELECT pid FROM pg_stat_activity WHERE application_name=$1 AND query='LISTEN acl_changed'",[f.schema])).rows[0];
   await competitor.query('SELECT pg_terminate_backend($1)',[listener.pid]);
   await until(async()=>(await competitor.query("SELECT pg_try_advisory_lock(hashtext(current_database()),hashtext(current_schema() || ':collab-writer')) AS owned")).rows[0].owned);
   for(const operation of ['projection','activity','export-projection']){
    const response=await fetch(runtime.internalURL+`/internal/scenes/${f.ref.sceneId}/${operation}`,{method:'POST',headers:{authorization:'Bearer '+'i'.repeat(32),'content-type':'application/json'},body:JSON.stringify({...f.ref,grant:'invalid'})});expect(response.status).toBe(503);
   }
  }finally{await competitor.query('SELECT pg_advisory_unlock_all()');competitor.release();}
  const journal=readFileSync(join(dir,'audit.jsonl'),'utf8');expect(journal).not.toContain(f.keys.full);expect(journal).not.toContain('Start');expect(journal).toContain(f.ref.sceneId);
 }finally{
  socket?.terminate();browserDoc.destroy();await Promise.all(clients.map(c=>c.close()));if(mcp)await new Promise<void>(r=>{mcp!.closeAllConnections();mcp!.close(()=>r())});await runtime?.close();await pool?.end();await new Promise<void>(r=>{proxy.closeAllConnections();proxy.close(()=>r())});writeFileSync(join(dir,'stop'),'stop');await until(()=>child.exitCode!==null,15000);rmSync(dir,{recursive:true,force:true});
 }
 expect(child.exitCode,logs).toBe(0);
},60000);

import {fixture,signed,grantSecret} from '../../collab/tests/fixture.js';
it('paginates durable activity numerically across 9 and 99 without omissions',async()=>{
 const f=await fixture();const store=new Persistence(f.pool,f.authority,{promoteV2:true});
 try{
  for(let sequence=0;sequence<102;sequence++){
   const requestId=`page-${sequence+1}`;const bytes=Buffer.from(JSON.stringify({...f.ref,requestId,expectedSequence:String(sequence),commands:[{op:'add',elements:[]}]}));
   const decision=await f.authority.decide(f.principal.roomToken,f.ref,'scene:command',requestId,bytes,String(sequence));
   await store.applyCommands(await signed(decision,grantSecret),bytes);
  }
  let cursor='0';const sequences:string[]=[];
  do{
   const decision=await f.authority.decide(f.principal.roomToken,f.ref,'scene:read','activity-page',new Uint8Array());
   const page=await store.readActivity(await signed(decision,grantSecret),f.ref,cursor,7) as {items:{sequence:string}[];nextCursor?:string};
   sequences.push(...page.items.map(item=>item.sequence));cursor=page.nextCursor??'';
  }while(cursor);
  expect(sequences).toEqual(Array.from({length:102},(_,i)=>String(i+1)));
  expect(new Set(sequences).size).toBe(102);
  const recovery=await store.exportRecovery(f.ref) as {updates:{sequence:string}[]};
  expect(recovery.updates.map(update=>update.sequence)).toEqual(sequences);
 }finally{await f.close();}
},30000);
it('uses the trusted frontend origin through audited client copies and routes remote exports there',async()=>{
 const seen:string[]=[];const api=createServer((req,res)=>{seen.push('api:'+req.url);res.end(JSON.stringify({principalId:'agent:aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa',keyId:'aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa'}))});
 const frontend=createServer((req,res)=>{seen.push('public:'+req.url);res.end(JSON.stringify({id:'artifact'}))});
 await new Promise<void>(r=>api.listen(0,'127.0.0.1',r));await new Promise<void>(r=>frontend.listen(0,'127.0.0.1',r));
 const apiURL='http://127.0.0.1:'+(api.address() as {port:number}).port,publicURL='http://127.0.0.1:'+(frontend.address() as {port:number}).port;
 const ref={workspaceId:'aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa',sceneId:'bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb'};
 try{
  const missing=new GoClient(apiURL,'test').forCall('11111111-1111-4111-8111-111111111111');
  expect(()=>missing.sceneURL(ref)).toThrow('WHITEBOARD_PUBLIC_URL required');
  for(const origin of ['https://user:password@frontend.example','https://frontend.example/path','http://frontend.example','https://frontend.example?private=1'])expect(()=>new GoClient(apiURL,'test',undefined,origin)).toThrow();
  const client=new GoClient(apiURL,'test',undefined,publicURL).forCall('11111111-1111-4111-8111-111111111111');
  expect(client.sceneURL(ref)).toBe(publicURL+`/w/${ref.workspaceId}/scenes/${ref.sceneId}`);
  expect(client.artifactURL('artifact')).toBe(publicURL+'/mcp/artifacts/artifact');
  await client.authenticate();await client.remoteExport({sceneRef:ref,format:'json'});
  expect(seen).toEqual(['api:/api/v1/agent/auth','public:/mcp/exports']);
 }finally{for(const server of [api,frontend])await new Promise<void>(r=>{server.closeAllConnections();server.close(()=>r())});}
});
