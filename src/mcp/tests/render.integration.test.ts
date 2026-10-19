import {describe,test,expect} from 'vitest';
import {mkdtemp,writeFile,readFile,rm} from 'node:fs/promises';
import {tmpdir} from 'node:os';
import {join} from 'node:path';
import {createServer} from 'node:http';
import {chromium} from 'playwright';
import {createRenderContext,RENDER_ORIGIN} from '../src/render-sandbox.js';
import {fileURLToPath} from 'node:url';
import {RenderWorker} from '../src/render-worker.js';
import {compileMermaid} from '@collab/diagram-kit';
describe('bounded native renderer',()=>{
 test('compiles supported Mermaid into editable elements',async()=>{const worker=new RenderWorker();const result=await worker.compileMermaid('flowchart LR\nA[Hello] --> B[World]');expect(result.elements.some(e=>e.type==='text')).toBe(true);expect(result.elements.every(e=>e.type!=='image')).toBe(true);});
 test('rejects unsupported native syntax with DSL alternative',async()=>{await expect(new RenderWorker().compileMermaid('erDiagram\n CUSTOMER ||--o{ ORDER : places')).rejects.toMatchObject({code:'UNSUPPORTED_NATIVE_DIAGRAM'});});
 test('exports native SVG with labels and PNG with correct signature',async()=>{const diagram=await compileMermaid('flowchart LR\nA[Hello] --> B[Unicode 日本語]');const scene={elements:diagram.elements,appState:{},files:{}};const worker=new RenderWorker();const svg=await worker.render(scene,'svg',{});expect(svg.mimeType).toBe('image/svg+xml');expect(svg.bytes.toString()).toContain('Hello');const png=await worker.render(scene,'png',{});expect(png.mimeType).toBe('image/png');expect(png.bytes.subarray(0,8).toString('hex')).toBe('89504e470d0a1a0a');},30000);
 test('rejects oversized source and caller URL data',async()=>{const worker=new RenderWorker();await expect(worker.compileMermaid('x'.repeat(50001))).rejects.toMatchObject({code:'RENDER_LIMIT'});await expect(worker.render({url:'http://127.0.0.1/'} as any,'svg',{})).rejects.toThrow();});
});

test('kills a non-cooperative worker and rejects concurrent jobs',async()=>{
 const dir=await mkdtemp(join(tmpdir(),'renderer-test-'));const path=join(dir,'spin.cjs');await writeFile(path,'process.on("message",()=>{while(true){}})');let pid=0;
 try{const worker=new RenderWorker({deadlineMs:2000,childPath:path,onProcess:p=>pid=p});const running=worker.compileMermaid('flowchart LR\nA');await expect(worker.compileMermaid('flowchart LR\nB')).rejects.toMatchObject({code:'RENDER_BUSY'});await expect(running).rejects.toMatchObject({code:'RENDER_TIMEOUT'});await new Promise(resolve=>setTimeout(resolve,30));expect(()=>process.kill(pid,0)).toThrow();}finally{await rm(dir,{recursive:true,force:true});}
});
test('embeds bundled sketch fonts and Unicode multiline labels',async()=>{
 const diagram=await compileMermaid('flowchart TB\nA[Tiếng Việt 日本語 🚀] --> B[Second line]',{theme:'sketch'});const result=await new RenderWorker().render({elements:diagram.elements,appState:{},files:{}},'svg',{});const svg=result.bytes.toString();expect(svg).toContain('Tiếng Việt');expect(svg).toContain('日本語');expect(svg).toContain('@font-face');expect(svg).toMatch(/data:(application|font)\//);expect(svg).not.toContain('https://esm.sh');
},15000);
test('enforces dimensions, node and edge limits before browser allocation',async()=>{
 const worker=new RenderWorker();const d=await compileMermaid('flowchart LR\nA');const scene={elements:d.elements.map(e=>({...e,width:5000})),appState:{},files:{}};await expect(worker.render(scene,'png',{})).rejects.toMatchObject({code:'RENDER_LIMIT'});
 await expect(worker.compileMermaid('flowchart LR\n'+Array.from({length:501},(_,i)=>'N'+i).join('\n'))).rejects.toThrow();
 await expect(worker.compileMermaid('flowchart LR\n'+Array.from({length:1001},()=> 'A --> B').join('\n'))).rejects.toThrow();
});

test('private browser cannot fetch external, loopback, file or websocket resources',async()=>{
 let hits=0;const server=createServer((_req,res)=>{hits++;res.end('secret');});await new Promise<void>(resolve=>server.listen(0,'127.0.0.1',resolve));const port=(server.address() as any).port;const browser=await chromium.launch();
 try{const context=await createRenderContext(browser,fileURLToPath(new URL('../render-dist',import.meta.url)));const page=await context.newPage();await page.goto(RENDER_ORIGIN);const results=await page.evaluate(async port=>Promise.all([`http://127.0.0.1:${port}/secret`,'http://169.254.169.254/latest/meta-data','https://example.com/','file:///etc/passwd'].map(async url=>{try{await fetch(url);return true;}catch{return false;}})),port);expect(results).toEqual([false,false,false,false]);const connected=await page.evaluate(port=>new Promise<boolean>(resolve=>{const ws=new WebSocket(`ws://127.0.0.1:${port}/socket`);ws.onopen=()=>{ws.close();resolve(true);};ws.onerror=()=>resolve(false);ws.onclose=()=>resolve(false);setTimeout(()=>{ws.close();resolve(false);},1000);}),port);expect(connected).toBe(false);expect(hits).toBe(0);await context.close();}finally{await browser.close();await new Promise<void>(resolve=>server.close(()=>resolve()));}
},15000);

test('typed DSL compilation uses the same disposable worker boundary',async()=>{const worker=new RenderWorker();expect(typeof (worker as any).compileDiagram).toBe('function');const result=await (worker as any).compileDiagram({preset:'flowchart',instanceId:'dsl',schemaVersion:1,theme:'clean',direction:'LR',data:{nodes:[{id:'a',label:'Native'}],edges:[]}});expect(result.elements.some((e:any)=>e.type==='text')).toBe(true);});

test('exports authorized image bytes and rejects oversized encoded image dimensions',async()=>{
 const diagram=await compileMermaid('flowchart LR\nA');const rectangle=diagram.elements[0];const png='iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+jRZkAAAAASUVORK5CYII=';
 const image={...rectangle,type:'image',id:'image1',fileId:'file1',status:'saved',scale:[1,1],crop:null,boundElements:null,width:40,height:40};const scene={elements:[image],appState:{},files:{file1:{id:'file1',assetId:'asset1',mimeType:'image/png',created:0}}} as any;
 const worker=new RenderWorker();const svg=await worker.render(scene,'svg',{file1:{mimeType:'image/png',data:png}});expect(svg.bytes.toString()).toContain('data:image/png;base64,');
 const oversized=Buffer.from(png,'base64');oversized.writeUInt32BE(20000,16);
 await expect(worker.render(scene,'svg',{file1:{mimeType:'image/png',data:oversized.toString('base64')}})).rejects.toMatchObject({code:'RENDER_LIMIT'});
 await expect(worker.render(scene,'svg',{file1:{mimeType:'image/png',data:Buffer.from('<svg onload="alert(1)"/>').toString('base64')}})).rejects.toMatchObject({code:'INVALID_ASSET'});
},15000);

test('official conversion keeps multiline text inside its native container and retries deterministic',async()=>{
 const worker=new RenderWorker();const input={preset:'flowchart',schemaVersion:1,instanceId:'bounds',theme:'sketch',direction:'LR',data:{nodes:[{id:'a',label:'Tiếng Việt\nNative labels'},{id:'b',label:'日本語日本語日本語日本語日本語日本語日本語日本語'}],edges:[]}};const a=await worker.compileDiagram(input),b=await worker.compileDiagram(input);expect(a).toEqual(b);
 for(const text of a.elements.filter(e=>e.type==='text')){const box=a.elements.find(e=>e.id===text.containerId)!;expect(text.x).toBeGreaterThanOrEqual(box.x);expect(text.y).toBeGreaterThanOrEqual(box.y);expect(text.x+text.width).toBeLessThanOrEqual(box.x+box.width);expect(text.y+text.height).toBeLessThanOrEqual(box.y+box.height);}
},15000);

test('deadline kills Chromium even when its renderer JavaScript never yields',async()=>{
 const dir=await mkdtemp(join(tmpdir(),'browser-timeout-')),path=join(dir,'spin.mjs'),marker=join(dir,'loop-started');let browserPid=0;
 await writeFile(path,`import {chromium} from ${JSON.stringify(import.meta.resolve('playwright'))};import{writeFileSync}from'node:fs';process.once('message',async({browserEndpoint})=>{const browser=await chromium.connectOverCDP(browserEndpoint);const page=await browser.newPage();writeFileSync(${JSON.stringify(marker)},'started');await page.evaluate(()=>{while(true){}});});`);
 try{const worker=new RenderWorker({deadlineMs:3000,childPath:path,onBrowserProcess:pid=>browserPid=pid});await expect(worker.compileMermaid('flowchart LR\nA')).rejects.toMatchObject({code:'RENDER_TIMEOUT'});expect(await readFile(marker,'utf8')).toBe('started');expect(()=>process.kill(browserPid,0)).toThrow();}finally{await rm(dir,{recursive:true,force:true});}
},10000);
