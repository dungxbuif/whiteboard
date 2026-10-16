import {fileURLToPath} from 'node:url';
import {createRenderContext,RENDER_ORIGIN} from './render-sandbox.js';
import {chromium,type Browser} from 'playwright';
import {compileMermaid,compileDiagram} from '@collab/diagram-kit';
import {validateSceneContent} from '@collab/contracts';
import type {RenderJob} from './render-worker.js';

// The trusted parent owns Chromium from spawn and supplies only its generated
// loopback endpoint. This child cannot launch an unreported browser process.
process.once('message',async({job,browserEndpoint}:{job:RenderJob;browserEndpoint:string})=>{
 let browser:Browser|undefined;
 let response:unknown;
 try{
  if(!/^ws:\/\/127\.0\.0\.1:\d+\/devtools\/browser\/[a-zA-Z0-9-]+$/.test(browserEndpoint))throw new Error('Invalid owned browser endpoint.');
  const compiled=job.kind==='mermaid'?await compileMermaid(job.source):job.kind==='diagram'?await compileDiagram(job.input):undefined;
  if(compiled?.elements.some(e=>e.type==='image'))throw new Error('Native conversion returned an image.');
  browser=await chromium.connectOverCDP(browserEndpoint);
  const context=await createRenderContext(browser,fileURLToPath(new URL('../render-dist',import.meta.url)));
  const page=await context.newPage();await page.goto(RENDER_ORIGIN);await page.waitForFunction(()=>typeof (globalThis as any).renderScene==='function');
  const result=compiled?await page.evaluate(async input=>(globalThis as any).normalizeNative(input),compiled):await page.evaluate(async input=>(globalThis as any).renderScene(input),job);
  if(compiled)validateSceneContent({elements:result.elements,appState:{},files:{}});
  await context.close();
  response={result};
 }catch(error:any){response={error:{code:error.code??'RENDER_FAILED',message:error.message??'Rendering failed.'}};}
 finally{
  if(browser){
   try{const session=await browser.newBrowserCDPSession();await session.send('Browser.close');}catch{/* Parent still owns and reaps the process. */}
   await browser.close().catch(()=>{});
  }
 }
 process.send?.(response as any,()=>process.exit(0));
});
