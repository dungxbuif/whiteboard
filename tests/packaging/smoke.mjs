import assert from 'node:assert/strict';
import {execFileSync} from 'node:child_process';
import {resolve} from 'node:path';
import WebSocket from '../../src/node_modules/ws/wrapper.mjs';
import {SignJWT} from '../../src/node_modules/jose/dist/webapi/index.js';
import {Client} from '../../src/node_modules/@modelcontextprotocol/sdk/dist/esm/client/index.js';
import {StreamableHTTPClientTransport} from '../../src/node_modules/@modelcontextprotocol/sdk/dist/esm/client/streamableHttp.js';
const root=resolve(import.meta.dirname,'../..'),name='whiteboard-e2-'+process.pid,containers=[];
const docker=(...args)=>execFileSync('docker',args,{encoding:'utf8',stdio:['ignore','pipe','pipe']}).trim();
const run=(suffix,args)=>{const n=name+'-'+suffix;containers.push(n);docker('run','-d','--name',n,'--network',name,...args);return n;};
const pause=ms=>new Promise(r=>setTimeout(r,ms));
let client,transport,ws;
try{
 docker('network','create',name);docker('volume','create',name+'-audit');
 run('db',['--network-alias','db','-e','POSTGRES_PASSWORD=fixture','-e','POSTGRES_USER=fixture','-e','POSTGRES_DB=packaging_test','postgres:16-alpine']);
 const collab=run('collab',['--network-alias','collab','--init','--read-only','--tmpfs','/tmp','--cap-drop','ALL','--security-opt','no-new-privileges:true','-v',root+'/tests/packaging:/app/packaging:ro','-v',root+'/src/backend/migrations:/migrations:ro','whiteboard-e2-collab:local','node','/app/packaging/collab-fixture.mjs']);
 const mcp=run('mcp',['--network-alias','mcp','--network-alias','backend','--init','--read-only','--tmpfs','/tmp:size=512m','--memory','2g','--pids-limit','256','--shm-size','256m','--cap-drop','ALL','--security-opt','no-new-privileges:true','-v',root+'/tests/packaging:/app/packaging:ro','-v',name+'-audit:/var/lib/whiteboard-mcp','-e','WHITEBOARD_API_URL=http://127.0.0.1:8080','-e','WHITEBOARD_PUBLIC_URL=https://whiteboard.example','-e','MCP_ALLOWED_ORIGINS=https://whiteboard.example','-e','MCP_AUDIT_PATH=/var/lib/whiteboard-mcp/audit.jsonl','whiteboard-e2-mcp:local','node','/app/packaging/mcp-fixture.mjs']);
 for(const [container,url,status] of [[collab,'http://127.0.0.1:8082/readyz',200],[mcp,'http://127.0.0.1:3002/mcp',401]]){
  let ready=false;for(let i=0;i<60;i++){try{docker('exec',container,'node','-e',`fetch('${url}').then(r=>process.exit(r.status===${status}?0:1)).catch(()=>process.exit(1))`);ready=true;break;}catch{await pause(500);}}
  if(!ready)throw new Error('Not ready: '+container+' '+docker('logs',container));
 }
 const frontend=run('frontend',['--read-only','--tmpfs','/tmp:size=64m','--cap-drop','ALL','--security-opt','no-new-privileges:true','-p','127.0.0.1::80','whiteboard-e2-frontend:local']);
 const port=docker('port',frontend,'80/tcp').split(':').at(-1),base='http://127.0.0.1:'+port;
 await pause(700);assert.equal((await fetch(base+'/healthz')).status,200);
 for(const path of ['/internal','/internal/authority/decisions','/internal/scenes/x/commands','/api/internal/authority/decisions'])assert.equal((await fetch(base+path)).status,404,path);
 for(const path of ['/mcp','/mcp/exports','/mcp/artifacts/aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa'])assert.equal((await fetch(base+path)).status,401,path);
 ws=new WebSocket(base.replace('http:','ws:')+'/collab',{origin:'https://whiteboard.example'});
 const messages=[];ws.on('message',(data,binary)=>{if(!binary)messages.push(JSON.parse(data.toString()));});
 await new Promise((r,j)=>{ws.once('open',r);ws.once('error',j);});
 const now=Math.floor(Date.now()/1000),user='11111111-1111-4111-8111-111111111111',workspaceId='22222222-2222-4222-8222-222222222222';
 const token=await new SignJWT({iss:'whiteboard-api',aud:'whiteboard-collab',purpose:'room',iat:now,exp:now+600,jti:'packaging',sub:user,workspaceId,sceneId:'33333333-3333-4333-8333-333333333333',principalId:user,principalType:'user',grantId:'membership:'+workspaceId+':'+user,access:'edit',aclVersion:'2'}).setProtectedHeader({alg:'HS256'}).sign(new TextEncoder().encode('r'.repeat(32)));
 ws.send(JSON.stringify({type:'auth',requestId:'packaging',payload:{token}}));
 for(let i=0;i<50&&!messages.some(m=>m.type==='ready');i++)await pause(100);
 assert(messages.some(m=>m.type==='ready'),JSON.stringify(messages));
 client=new Client({name:'packaging',version:'1'});transport=new StreamableHTTPClientTransport(new URL(base+'/mcp'),{requestInit:{headers:{Authorization:'Bearer packaging-fixture',Origin:'https://whiteboard.example'}}});
 await client.connect(transport);assert((await client.listTools()).tools.some(t=>t.name==='export_scene'));
 assert.equal((await client.callTool({name:'list_workspaces',arguments:{}})).isError,undefined);
 const exportResponse=await fetch(base+'/mcp/exports',{method:'POST',headers:{Authorization:'Bearer packaging-fixture','Content-Type':'application/json'},body:JSON.stringify({sceneRef:{workspaceId:'22222222-2222-4222-8222-222222222222',sceneId:'33333333-3333-4333-8333-333333333333'},format:'json'})});
 assert.equal(exportResponse.status,200);const exported=await exportResponse.json();assert.equal(new URL(exported.url).origin,'https://whiteboard.example');
 const artifact=await fetch(base+new URL(exported.url).pathname,{headers:{Authorization:'Bearer packaging-fixture'}});assert.equal(artifact.status,200);assert.equal((await artifact.json()).type,'excalidraw');
 docker('exec',mcp,'node','--input-type=module','-e',"import {RenderWorker} from '/app/mcp/dist/render-worker.js';const w=new RenderWorker();const d=await w.compileMermaid('flowchart LR\\nA[Packaging] --> B[Native]');const s=await w.render({elements:d.elements,appState:{},files:{}},'png',{});if(s.bytes.subarray(0,8).toString('hex')!=='89504e470d0a1a0a')process.exit(1);");
 await transport.close();await pause(200);
 const stream=await fetch(base+'/mcp',{headers:{Authorization:'Bearer packaging-fixture','Mcp-Session-Id':transport.sessionId,Accept:'text/event-stream'},signal:AbortSignal.timeout(3000)});
 assert.equal(stream.status,200);assert.match(stream.headers.get('content-type'),/event-stream/);assert.equal(stream.headers.get('x-accel-buffering'),'no');await stream.body.cancel();
 assert.equal((await fetch(base+'/mcp',{headers:{Authorization:'Bearer packaging-fixture',Origin:'https://evil.example'}})).status,403);
 for(const container of [collab,mcp,frontend])assert.notEqual(docker('exec',container,'id','-u'),'0');
 assert.equal(docker('exec',mcp,'stat','-c','%a','/var/lib/whiteboard-mcp'),'700');assert.equal(docker('exec',mcp,'stat','-c','%a','/var/lib/whiteboard-mcp/audit.jsonl'),'600');
 docker('exec',mcp,'node','-e',"require('fs').accessSync('/app/mcp/dist/render-child.js');require('fs').accessSync('/app/mcp/render-dist/index.html')");
 assert(!docker('logs',frontend).includes('packaging-fixture'));
 console.log(JSON.stringify({pass:true,checks:['nonroot images','collab Upgrade + authenticated ready with PostgreSQL','MCP initialize/discovery/call/SSE/origin enforcement','MCP export/artifact route forwarding','public internal route denial','persistent audit directory0700/journal0600','renderer artifacts and native Chromium PNG'],authority:'signed fixture replaces upstream Go HTTP only; E3 owns full real Go topology proof'}));
}finally{
 ws?.close();await client?.close().catch(()=>{});
 for(const container of containers.reverse()){try{docker('rm','-f',container);}catch{}}
 try{docker('volume','rm',name+'-audit');}catch{}try{docker('network','rm',name);}catch{}
}
