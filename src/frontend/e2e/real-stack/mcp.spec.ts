import { test, expect } from './fixtures';
import { openScene, draw, elements } from './browser';
import { Client } from '@modelcontextprotocol/sdk/client/index.js';
import { StreamableHTTPClientTransport } from '@modelcontextprotocol/sdk/client/streamableHttp.js';
import { StdioClientTransport } from '@modelcontextprotocol/sdk/client/stdio.js';
import { join } from 'node:path';
import { readFileSync } from 'node:fs';
import { root } from './harness';
async function call(client:Client,name:string,args:any){const r=await client.callTool({name,arguments:args});if(r.isError)throw new Error(JSON.stringify(r.content));return JSON.parse((r.content as {text:string}[])[0].text);}
test('HTTP and stdio MCP authenticate real keys; native IDs render and stale writes preserve human edits',async({page,stack},info)=>{
  const scene=await stack.scene();await openScene(page,stack,scene);const sceneRef={workspaceId:stack.owner.workspaceId,sceneId:scene.id};
  const key=await stack.json('/workspaces/'+stack.owner.workspaceId+'/agent-keys',stack.owner.token,stack.owner.workspaceId,'POST',{scopes:['scene:read','scene:write','scene:export']});
  const write=await stack.json('/workspaces/'+stack.owner.workspaceId+'/agent-keys',stack.owner.token,stack.owner.workspaceId,'POST',{scopes:['scene:write']});
  const c=new Client({name:'e1-http',version:'1'}),stdio=new Client({name:'e1-stdio',version:'1'});
  try {
    await c.connect(new StreamableHTTPClientTransport(new URL(stack.baseURL+'/mcp'),{requestInit:{headers:{authorization:'Bearer '+key.key}}}));expect((await c.listTools()).tools.map(t=>t.name)).toContain('generate_diagram');
    // Remote adapter receives only public/API addresses and its user-issued key.
    await stdio.connect(new StdioClientTransport({command:'rtk',args:['proxy',process.execPath,join(root,'mcp/dist/stdio.js')],env:{PATH:stack.env.PATH!,HOME:stack.env.HOME!,WHITEBOARD_API_URL:stack.api,WHITEBOARD_PUBLIC_URL:stack.baseURL,WHITEBOARD_AGENT_KEY:write.key,MCP_AUDIT_PATH:join(stack.dir,'stdio-audit.jsonl')}}));expect((await stdio.listTools()).tools.map(t=>t.name)).toContain('patch_elements');await expect(call(stdio,'get_scene',sceneRef)).rejects.toThrow();
    const args={sceneRef,preset:'flowchart',data:{nodes:[{id:'a',label:'Native start'},{id:'b',label:'Native end'}],edges:[{id:'ab',from:'a',to:'b'}]},instanceId:'e1-native',requestId:'e1-generate',expectedSequence:'0'};
    const created=await call(stdio,'generate_diagram',args);await expect.poll(async()=>(await elements(page)).map((e:any)=>e.id).sort()).toEqual(created.createdIds.slice().sort());expect(await call(stdio,'generate_diagram',args)).toEqual(created);
    await draw(page,800);await expect(page.getByRole('status')).toHaveText('Saved');const human=(await elements(page)).find((e:any)=>!created.createdIds.includes(e.id));expect(human).toBeTruthy();
    await expect(call(stdio,'patch_elements',{sceneRef,requestId:'stale-e1',expectedSequence:created.receipt.sequence,patches:[{id:created.createdIds[0],fields:{x:900}}]})).rejects.toThrow('CONFLICT');
    const live=await call(c,'get_scene',sceneRef);expect(live.content.elements.find((e:any)=>e.id===human.id)).toEqual(human);
    await call(stdio,'patch_elements',{sceneRef,requestId:'fresh-e1',expectedSequence:live.sequence,patches:[{id:created.createdIds[0],fields:{x:900}}]});await expect.poll(async()=>(await elements(page)).find((e:any)=>e.id===created.createdIds[0]).x).toBe(900);expect((await elements(page)).find((e:any)=>e.id===human.id)).toEqual(human);
    const exported=await call(c,'export_scene',{sceneRef,format:'png'});const download=await fetch(exported.url,{headers:{authorization:'Bearer '+key.key}});expect(download.status).toBe(200);expect(download.headers.get('content-type')).toContain('image/png');expect(Buffer.from(await download.arrayBuffer()).subarray(0,8).toString('hex')).toBe('89504e470d0a1a0a');
    const audit=(await stack.pool.query("SELECT correlation_id,provenance,outcome FROM agent_operation_audit WHERE scene_id=$1 AND tool='generate_diagram'",[scene.id])).rows;expect(audit.some(a=>a.provenance==='go_operation'&&a.outcome==='success')).toBe(true);
    expect((await stack.pool.query('SELECT count(*) FROM collab_mutation_audit WHERE scene_id=$1 AND request_id=$2',[scene.id,'e1-generate'])).rows[0].count).toBe('1');
    const journal=readFileSync(join(stack.dir,'stdio-audit.jsonl'),'utf8');expect(journal).toContain('adapter_observed');expect(journal).not.toContain(write.key);expect(journal).not.toContain('Native start');
    await page.screenshot({path:info.outputPath('native-mcp-human-edit.png')});
    await stack.json('/workspaces/'+stack.owner.workspaceId+'/agent-keys/'+key.id,stack.owner.token,stack.owner.workspaceId,'DELETE');expect((await fetch(exported.url,{headers:{authorization:'Bearer '+key.key}})).status).toBe(401);
  }finally{await Promise.allSettled([c.close(),stdio.close()]);}
});
