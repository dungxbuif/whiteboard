import {AuditJournal} from './audit.js';
import {mkdtempSync,readFileSync,rmSync} from 'node:fs';
import {tmpdir} from 'node:os';
import {join} from 'node:path';
import {execFileSync,spawn} from 'node:child_process';
import {fileURLToPath} from 'node:url';
import {StdioClientTransport} from '@modelcontextprotocol/sdk/client/stdio.js';
import {describe,it,expect,afterEach,beforeAll} from 'vitest';
import {createServer,type Server} from 'node:http';
import {Client} from '@modelcontextprotocol/sdk/client/index.js';
import {StreamableHTTPClientTransport} from '@modelcontextprotocol/sdk/client/streamableHttp.js';
import {createMcpHTTP} from './server.js';
const keyA='aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa';
const keyB='bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb';
const auditDirs:string[]=[];
function audit(){const dir=mkdtempSync(join(tmpdir(),'mcp-http-audit-'));auditDirs.push(dir);return new AuditJournal(join(dir,'events.jsonl'))}
const servers:Server[]=[];const clients:Client[]=[];
async function listen(server:Server){servers.push(server);await new Promise<void>(r=>server.listen(0,'127.0.0.1',r));const a=server.address();return `http://127.0.0.1:${typeof a==='object'&&a?a.port:0}`}
afterEach(async()=>{await Promise.all(clients.splice(0).map(c=>c.close()));await Promise.all(servers.splice(0).map(s=>new Promise<void>(r=>{s.closeAllConnections();s.close(()=>r())})));for(const dir of auditDirs.splice(0))rmSync(dir,{recursive:true,force:true}) });
async function fixture(){let revoked=false;const seen:string[]=[];const correlations:string[]=[];const api=await listen(createServer((req,res)=>{seen.push(req.url||'');if(req.headers['x-agent-call-id'])correlations.push(String(req.headers['x-agent-call-id']));res.setHeader('Content-Type','application/json');if(revoked||!['Bearer test-key-a','Bearer test-key-b'].includes(req.headers.authorization||'')){res.writeHead(401);res.end('{}');return}if(req.url==='/api/v1/agent/auth'){res.end(JSON.stringify({principalId:'agent:'+(req.headers.authorization==='Bearer test-key-a'?keyA:keyB),keyId:req.headers.authorization==='Bearer test-key-a'?keyA:keyB}));return}if(req.url?.startsWith('/api/v1/agent/workspaces')){res.end(JSON.stringify({items:[{id:'11111111-1111-4111-8111-111111111111',name:'Allowed'}]}));return}res.writeHead(404);res.end('{}')}));const journal=audit();const base=await listen(createMcpHTTP({apiURL:api,allowedOrigins:['https://whiteboard.example'],audit:journal}));return {base,api,seen,correlations,journal,revoke(){revoked=true}}}
async function connect(base:string,key='test-key-a'){const client=new Client({name:'test',version:'1.0'});clients.push(client);const transport=new StreamableHTTPClientTransport(new URL(base+'/mcp'),{requestInit:{headers:{Authorization:`Bearer ${key}`}}});await client.connect(transport);return {client,transport}}
describe('MCP HTTP lifecycle',()=>{
 it('authenticates every call and rejects bad origins',async()=>{const f=await fixture();expect((await fetch(f.base+'/mcp',{method:'POST'})).status).toBe(401);expect((await fetch(f.base+'/mcp',{headers:{Authorization:'Bearer test-key-a',Origin:'https://evil.example'}})).status).toBe(403)});
 it('initializes, discovers tools, returns tenant page without totals, and rejects revoked keys',async()=>{const f=await fixture();const {client}=await connect(f.base);const tools=await client.listTools();expect(tools.tools.map(t=>t.name).sort()).toEqual(['create_scene','get_scene','list_scenes','list_workspaces']);const result=await client.callTool({name:'list_workspaces',arguments:{limit:50}});expect(result.isError).not.toBe(true);expect(JSON.stringify(result)).toContain('Allowed');expect(JSON.stringify(result)).not.toContain('total');expect(f.seen.some(p=>p.includes('limit=50'))).toBe(true);f.revoke();await expect(client.listTools()).rejects.toThrow()});
 it('binds session to authenticated principal',async()=>{const f=await fixture();const {transport}=await connect(f.base);const result=await fetch(f.base+'/mcp',{method:'POST',headers:{Authorization:'Bearer test-key-b','Mcp-Session-Id':transport.sessionId!,'Content-Type':'application/json',Accept:'application/json, text/event-stream'},body:JSON.stringify({jsonrpc:'2.0',id:3,method:'tools/list'})});expect(result.status).toBe(403)});
});

describe('MCP stdio remote adapter',()=>{
 beforeAll(()=>{execFileSync(process.execPath,[fileURLToPath(new URL('../../node_modules/typescript/bin/tsc',import.meta.url)),'-p',fileURLToPath(new URL('../tsconfig.json',import.meta.url))])});
 it('initializes and discovers tools in a subprocess using remote API only',async()=>{const f=await fixture();const transport=new StdioClientTransport({command:process.execPath,args:[fileURLToPath(new URL('../dist/stdio.js',import.meta.url))],env:{WHITEBOARD_API_URL:f.api,WHITEBOARD_AGENT_KEY:'test-key-a',MCP_AUDIT_PATH:audit().path},stderr:'pipe'});let diagnostics='';transport.stderr?.on('data',chunk=>{diagnostics+=chunk.toString()});const client=new Client({name:'stdio-test',version:'1'});clients.push(client);await client.connect(transport);expect((await client.listTools()).tools).toHaveLength(4);expect((await client.callTool({name:'list_workspaces',arguments:{}})).isError).not.toBe(true);await client.close();expect(diagnostics).not.toContain('test-key-a')});
 it('writes only JSON-RPC MCP frames to stdout',async()=>{const f=await fixture();const child=spawn(process.execPath,[fileURLToPath(new URL('../dist/stdio.js',import.meta.url))],{env:{WHITEBOARD_API_URL:f.api,WHITEBOARD_AGENT_KEY:'test-key-a',MCP_AUDIT_PATH:audit().path},stdio:['pipe','pipe','pipe']});let output='';let stderr='';child.stdout.on('data',chunk=>{output+=chunk.toString()});child.stderr.on('data',chunk=>{stderr+=chunk.toString()});try{child.stdin.write(JSON.stringify({jsonrpc:'2.0',id:1,method:'initialize',params:{protocolVersion:'2025-11-25',capabilities:{},clientInfo:{name:'frames',version:'1'}}})+'\n');await new Promise<void>((resolve,reject)=>{const timer=setTimeout(()=>reject(new Error('No initialization frame')),3000);child.stdout.on('data',()=>{if(output.includes('\n')){clearTimeout(timer);resolve()}})});const frames=output.trim().split('\n').map(line=>JSON.parse(line));expect(frames[0]).toMatchObject({jsonrpc:'2.0',id:1,result:{serverInfo:{name:'whiteboard'}}});expect(output+stderr).not.toContain('test-key-a')}finally{child.kill();await new Promise<void>(r=>child.once('exit',()=>r()))}});
});

it('caps the entire MCP projection response at 256 KiB',async()=>{
 const upstream=await listen(createServer((req,res)=>{res.setHeader('Content-Type','application/json');res.end(JSON.stringify(req.url==='/api/v1/agent/auth'?{principalId:'agent:a',keyId:'key'}:{id:'11111111-1111-4111-8111-111111111111',content:{elements:[],appState:{label:'x'.repeat(150*1024)},files:{}}}))}));
 const base=await listen(createMcpHTTP({apiURL:upstream,allowedOrigins:[],audit:audit()}));const {client}=await connect(base);const result=await client.callTool({name:'get_scene',arguments:{workspaceId:'11111111-1111-4111-8111-111111111111',sceneId:'11111111-1111-4111-8111-111111111111'}});expect(result.isError).not.toBe(true);expect(Buffer.byteLength(JSON.stringify(result))).toBeLessThanOrEqual(256*1024);
});

it('preserves the Go projection-limit code over actual HTTP',async()=>{
 const upstream=await listen(createServer((req,res)=>{res.setHeader('Content-Type','application/json');if(req.url==='/api/v1/agent/auth'){res.end(JSON.stringify({principalId:'agent:a',keyId:'key'}));return}res.writeHead(413);res.end(JSON.stringify({code:'PROJECTION_TOO_LARGE',message:'Scene exceeds projection limit',requestId:'fixture'}))}));
 const base=await listen(createMcpHTTP({apiURL:upstream,allowedOrigins:[],audit:audit()}));const {client}=await connect(base);const result=await client.callTool({name:'get_scene',arguments:{workspaceId:'11111111-1111-4111-8111-111111111111',sceneId:'11111111-1111-4111-8111-111111111111'}});expect(result.isError).toBe(true);expect(result.content).toEqual([{type:'text',text:'PROJECTION_TOO_LARGE'}]);
});

it('journals actual successful and invalid MCP calls',async()=>{const f=await fixture();const {client}=await connect(f.base);const success=await client.callTool({name:'list_workspaces',arguments:{}});const invalid=await client.callTool({name:'list_scenes',arguments:{workspaceId:'private-invalid-target',limit:999}});expect(success.isError).not.toBe(true);expect(invalid.isError).toBe(true);const rows=readFileSync(f.journal.path,'utf8').trim().split('\n').filter(Boolean).map(r=>JSON.parse(r));expect(rows).toHaveLength(4);expect(rows.map(r=>r.outcome)).toEqual(['started','success','started','invalid']);expect(readFileSync(f.journal.path,'utf8')).not.toContain('private-invalid-target');expect(success._meta).toHaveProperty('agentCallId',rows[0].correlationId);expect(rows[1]).toMatchObject({keyId:keyA,principalId:'agent:'+keyA,scope:'workspace:read',provenance:'adapter_observed'});expect(f.correlations).toContain(rows[0].correlationId)});


it('journals revoked credentials and unreachable upstream over actual HTTP',async()=>{
 const f=await fixture();const {client,transport}=await connect(f.base);
 f.revoke();
 const denied=await fetch(f.base+'/mcp',{method:'POST',headers:{Authorization:'Bearer test-key-a','Mcp-Session-Id':transport.sessionId!,'Content-Type':'application/json',Accept:'application/json, text/event-stream'},body:JSON.stringify({jsonrpc:'2.0',id:10,method:'tools/call',params:{name:'list_workspaces',arguments:{}}})});
 expect(denied.status).toBe(401);expect((await denied.json()).error).toBe('UNAUTHENTICATED');
 const upstream=servers.find(s=>{const a=s.address();return typeof a==='object'&&a&&f.api.endsWith(':'+a.port)})!;
 await new Promise<void>(resolve=>{upstream.closeAllConnections();upstream.close(()=>resolve())});
 await expect(client.callTool({name:'list_workspaces',arguments:{}})).rejects.toThrow();
 const rows=readFileSync(f.journal.path,'utf8').trim().split('\n').map(row=>JSON.parse(row));
 expect(rows.map(r=>r.outcome)).toEqual(['started','denied','started','failure']);
 expect(rows[1]).toMatchObject({keyId:keyA,failureClass:'UNAUTHENTICATED'});
 expect(rows[3]).toMatchObject({keyId:keyA,failureClass:'UPSTREAM_UNAVAILABLE'});
});


describe('SDK envelope audit boundary', () => {
 beforeAll(() => {
  execFileSync(process.execPath, [fileURLToPath(new URL('../../node_modules/typescript/bin/tsc', import.meta.url)), '-p', fileURLToPath(new URL('../tsconfig.json', import.meta.url))]);
 });
 function assertEnvelopeAudit(journal: AuditJournal, responses: any[]) {
  const raw = readFileSync(journal.path, 'utf8');
  const rows = raw.trim().split('\n').filter(Boolean).map(row => JSON.parse(row));
  expect(rows).toHaveLength(6);
  expect(rows.map(row => row.outcome)).toEqual(['started', 'invalid', 'started', 'invalid', 'started', 'success']);
  for (let i = 0; i < 3; i++) {
   expect(rows[2*i].correlationId).toBe(rows[2*i+1].correlationId);
   expect(rows[2*i].keyId).toBe(keyA);
   const correlation = responses[i].error?.data?.agentCallId ?? responses[i].result?._meta?.agentCallId;
   expect(correlation).toBe(rows[2*i].correlationId);
  }
  expect(new Set(rows.map(row => row.correlationId)).size).toBe(3);
  expect(raw).not.toContain('private-envelope-content');
  expect(raw).not.toContain('test-key-a');
  expect(raw).not.toContain('expected record');
 }
 it('audits SDK-rejected HTTP arrays and valid calls exactly once', async () => {
  const f = await fixture();
  const { transport } = await connect(f.base);
  const replies = [];
  for (const [index, args] of [[], ['private-envelope-content'], {}].entries()) {
   const response = await fetch(f.base+'/mcp', {
    method:'POST', headers:{Authorization:'Bearer test-key-a','Mcp-Session-Id':transport.sessionId!, 'Content-Type':'application/json',Accept:'application/json, text/event-stream'},
    body:JSON.stringify({jsonrpc:'2.0',id:70+index,method:'tools/call',params:{name:'list_workspaces',arguments:args}}),
   });
   expect(response.status).toBe(200);
   replies.push(await response.json());
  }
  assertEnvelopeAudit(f.journal, replies);
  expect(replies[0].error).toMatchObject({code:-32602,message:'INVALID_INPUT'});
  expect(replies[2].result.isError).not.toBe(true);
  expect(f.seen.filter(url => url.startsWith('/api/v1/agent/workspaces'))).toHaveLength(1);
 });
 it('audits SDK-rejected raw stdio arrays and valid calls exactly once', async () => {
  const f = await fixture();
  const journal = audit();
  const child = spawn(process.execPath, [fileURLToPath(new URL('../dist/stdio.js',import.meta.url))], {
   env:{WHITEBOARD_API_URL:f.api,WHITEBOARD_AGENT_KEY:'test-key-a',MCP_AUDIT_PATH:journal.path},stdio:['pipe','pipe','pipe'],
  });
  let buffer = '';
  let diagnostic = '';
  const frames = new Map<number, any>();
  child.stdout.on('data', chunk => {
   buffer += chunk.toString();
   let end;
   while ((end=buffer.indexOf('\n'))>=0) {
    const frame = JSON.parse(buffer.slice(0,end));buffer=buffer.slice(end+1);frames.set(frame.id,frame);
   }
  });
  child.stderr.on('data', chunk => { diagnostic += chunk.toString(); });
  const request = async (id:number,method:string,params:unknown) => {
   child.stdin.write(JSON.stringify({jsonrpc:'2.0',id,method,params})+'\n');
   const until = Date.now()+3000;
   while (!frames.has(id)) {
    if(Date.now()>until)throw new Error('Missing stdio response');
    await new Promise(resolve=>setTimeout(resolve,10));
   }
   return frames.get(id);
  };
  try {
   await request(1,'initialize',{protocolVersion:'2025-11-25',capabilities:{},clientInfo:{name:'raw-audit',version:'1'}});
   child.stdin.write(JSON.stringify({jsonrpc:'2.0',method:'notifications/initialized'})+'\n');
   const replies=[];
   for(const [index,args] of [[],['private-envelope-content'],{}].entries()) {
    replies.push(await request(70+index,'tools/call',{name:'list_workspaces',arguments:args}));
   }
   assertEnvelopeAudit(journal,replies);
   expect(replies[0].error).toMatchObject({code:-32602,message:'INVALID_INPUT'});
   expect(replies[2].result.isError).not.toBe(true);
   expect(f.seen.filter(url=>url.startsWith('/api/v1/agent/workspaces'))).toHaveLength(1);
   expect(diagnostic).not.toContain('test-key-a');
  } finally {
   child.kill();await new Promise<void>(resolve=>child.once('exit',()=>resolve()));
  }
 });
});
