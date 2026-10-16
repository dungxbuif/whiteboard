import {fileURLToPath} from 'node:url';
import {createRenderContext,RENDER_ORIGIN} from './render-sandbox.js';
import {chromium} from 'playwright';
import {compileMermaid,compileDiagram} from '@collab/diagram-kit';
import {validateSceneContent} from '@collab/contracts';
import type {RenderJob} from './render-worker.js';
process.once('message',async(job:RenderJob)=>{
 try{
  const compiled=job.kind==='mermaid'?await compileMermaid(job.source):job.kind==='diagram'?await compileDiagram(job.input):undefined;
  if(compiled?.elements.some(e=>e.type==='image'))throw new Error('Native conversion returned an image.');
  const server=await chromium.launchServer({headless:true});
  process.send?.({kind:'browser',pid:server.process().pid});
  const browser=await chromium.connect(server.wsEndpoint());
  const context=await createRenderContext(browser,fileURLToPath(new URL('../render-dist',import.meta.url)));
  const page=await context.newPage();await page.goto(RENDER_ORIGIN);await page.waitForFunction(()=>typeof (globalThis as any).renderScene==='function');
  const result=compiled?await page.evaluate(async input=>(globalThis as any).normalizeNative(input),compiled):await page.evaluate(async input=>(globalThis as any).renderScene(input),job);
  if(compiled)validateSceneContent({elements:result.elements,appState:{},files:{}});
  process.send?.({result});
 }catch(error:any){process.send?.({error:{code:error.code??'RENDER_FAILED',message:error.message??'Rendering failed.'}});}
});
