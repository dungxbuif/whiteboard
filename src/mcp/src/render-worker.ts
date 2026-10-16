import {validateRasterAsset} from './render-assets.js';
import {fork,spawn,type ChildProcess} from 'node:child_process';
import {mkdtemp,mkdir,readFile,rm} from 'node:fs/promises';
import {tmpdir} from 'node:os';
import {join} from 'node:path';
import {setTimeout as delay} from 'node:timers/promises';
import {chromium} from 'playwright';
import {fileURLToPath} from 'node:url';
import {validateSceneContent,type SceneContent} from '@collab/contracts';
import {validateDiagram,type CompiledDiagram} from '@collab/diagram-kit';

export const RENDER_LIMITS=Object.freeze({deadlineMs:10_000,dimension:4096,artifactBytes:20*1024*1024,sceneBytes:25*1024*1024,sourceChars:50_000,assets:100,nodes:500,edges:1000});
export type ResolvedAsset={mimeType:string;data:string};
export type RenderedArtifact={mimeType:string;bytes:Buffer};
export class RenderError extends Error {constructor(public code:string,message:string){super(message);this.name='RenderError';}}
export type RenderJob={kind:'mermaid';source:string}|{kind:'diagram';input:unknown}|{kind:'render';scene:SceneContent;format:'svg'|'png';assets:Record<string,ResolvedAsset>};
const childPath=fileURLToPath(new URL('../dist/render-child.js',import.meta.url));
function signalGroup(child:ChildProcess,signal:NodeJS.Signals){
 if(!child.pid)return;
 try{process.kill(-child.pid,signal);}catch{try{child.kill(signal);}catch{}}
}
function groupAlive(child:ChildProcess):boolean{
 if(!child.pid)return false;
 try{process.kill(-child.pid,0);return true;}catch(error:any){return error.code!=='ESRCH';}
}
type OwnedProcess={child:ChildProcess;closed:Promise<void>;error?:Error};
function own(child:ChildProcess):OwnedProcess{
 const owned:OwnedProcess={child,closed:new Promise(resolve=>child.once('close',()=>resolve()))};
 child.on('error',error=>{owned.error=error;});
 return owned;
}
export type RenderWorkerOptions={
 deadlineMs?:number;childPath?:string;browserExecutable?:string;
 onProcess?:(pid:number)=>void;onBrowserProcess?:(pid:number)=>void;
 onTempDirectory?:(directory:string)=>void;
};
// These flags target the pinned Playwright Chromium; E2 must verify the final
// nonroot image. No caller controls flags, profile paths or the CDP endpoint.
function browserArgs(profile:string):string[]{return [
 '--headless','--no-sandbox','--disable-dev-shm-usage','--disable-background-networking',
 '--disable-component-update','--disable-client-side-phishing-detection','--disable-default-apps',
 '--disable-extensions','--disable-sync','--disable-breakpad','--disable-crash-reporter',
 '--disable-features=MediaRouter,OptimizationHints,DialMediaRouteProvider,Translate',
 '--no-first-run','--no-default-browser-check','--metrics-recording-only',
 '--password-store=basic','--use-mock-keychain','--force-color-profile=srgb',
 '--remote-debugging-address=127.0.0.1','--remote-debugging-port=0',
 '--no-startup-window',`--user-data-dir=${profile}`,
 ];}


/** One job per instance; there is deliberately no unbounded in-memory queue. */
export class RenderWorker {
 private busy=false;
 constructor(private readonly options:RenderWorkerOptions={}){}
 async compileDiagram(input:unknown):Promise<CompiledDiagram>{
  if(!validateDiagram(input).valid)throw new RenderError('INVALID_INPUT','Invalid native diagram input.');
  return this.run({kind:'diagram',input}) as Promise<CompiledDiagram>;
 }
 async compileMermaid(source:string):Promise<CompiledDiagram>{
  if(typeof source!=='string'||source.length>RENDER_LIMITS.sourceChars)throw new RenderError('RENDER_LIMIT','Mermaid source exceeds 50,000 characters.');
  return this.run({kind:'mermaid',source}) as Promise<CompiledDiagram>;
 }
 async render(scene:SceneContent,format:'svg'|'png',assets:Record<string,ResolvedAsset>):Promise<RenderedArtifact>{
  validateSceneContent(scene);
  if(!['svg','png'].includes(format))throw new RenderError('INVALID_FORMAT','Use json, svg or png.');
  if(Buffer.byteLength(JSON.stringify(scene))>RENDER_LIMITS.sceneBytes)throw new RenderError('RENDER_LIMIT','Scene exceeds 25 MiB.');
  if(scene.elements.filter(e=>!['text','arrow','line'].includes(e.type)).length>RENDER_LIMITS.nodes||scene.elements.filter(e=>['arrow','line'].includes(e.type)).length>RENDER_LIMITS.edges)throw new RenderError('RENDER_LIMIT','Scene exceeds render element limits.');
  if(scene.elements.length>2*(RENDER_LIMITS.nodes+RENDER_LIMITS.edges))throw new RenderError('RENDER_LIMIT','Scene exceeds 3000 native elements including labels.');
  const live=scene.elements.filter(e=>!e.isDeleted);
  const xs=live.flatMap(e=>[e.x,e.x+e.width]),ys=live.flatMap(e=>[e.y,e.y+e.height]);
  if(live.length&&(Math.max(...xs)-Math.min(...xs)+20>RENDER_LIMITS.dimension||Math.max(...ys)-Math.min(...ys)+20>RENDER_LIMITS.dimension))throw new RenderError('RENDER_LIMIT','Export exceeds 4096 pixels.');
  if(Object.keys(assets).length>RENDER_LIMITS.assets)throw new RenderError('RENDER_LIMIT','Too many assets.');
  let total=0;
  for(const [id,asset] of Object.entries(assets)){
   if(!scene.files[id]||!/^image\/(png|jpeg|webp|gif|avif)$/.test(asset.mimeType)||asset.mimeType!==scene.files[id].mimeType||!asset.data||!/^[A-Za-z0-9+/]*={0,2}$/.test(asset.data))throw new RenderError('INVALID_ASSET','Only authorized raster asset bytes are accepted.');
   validateRasterAsset(asset);
   total+=Buffer.byteLength(asset.data,'base64');if(total>RENDER_LIMITS.artifactBytes)throw new RenderError('RENDER_LIMIT','Assets exceed 20 MiB.');
  }
  for(const e of live)if(e.type==='image'&&(!e.fileId||!assets[e.fileId]))throw new RenderError('MISSING_ASSET','An image asset is unavailable.');
  const result=await this.run({kind:'render',scene,format,assets}) as {mimeType:string;data:string};
  const bytes=Buffer.from(result.data,'base64');
  if(bytes.length>RENDER_LIMITS.artifactBytes)throw new RenderError('RENDER_LIMIT','Artifact exceeds 20 MiB.');
  return {mimeType:result.mimeType,bytes};
 }
 private async run(job:RenderJob):Promise<unknown>{
  if(this.busy)throw new RenderError('RENDER_BUSY','Renderer is busy; retry after the current job.');
  this.busy=true;
  const owned:OwnedProcess[]=[];
  const abort=new AbortController();
  const deadlineMs=Math.max(1,Math.min(this.options.deadlineMs??RENDER_LIMITS.deadlineMs,RENDER_LIMITS.deadlineMs));
  let directory:string|undefined,result:unknown,failure:unknown,timedOut=false;
  const timer=setTimeout(()=>{
   timedOut=true;abort.abort();
   for(const process of owned)signalGroup(process.child,'SIGKILL');
  },deadlineMs);
  try{
   directory=await mkdtemp(join(tmpdir(),'whiteboard-render-'));
   this.options.onTempDirectory?.(directory);
   const profile=join(directory,'profile');await mkdir(profile,{mode:0o700});
   abort.signal.throwIfAborted();
   const env={PATH:process.env.PATH,HOME:directory,TMPDIR:directory,TMP:directory,TEMP:directory,
    NODE_ENV:'production',PLAYWRIGHT_BROWSERS_PATH:process.env.PLAYWRIGHT_BROWSERS_PATH};
   // spawn returns the process handle synchronously, before browser startup.
   const browser=own(spawn(this.options.browserExecutable??chromium.executablePath(),browserArgs(profile),{
    detached:true,stdio:'ignore',env,
   }));
   owned.push(browser);if(browser.child.pid)this.options.onBrowserProcess?.(browser.child.pid);
   let endpoint:string|undefined;
   while(!endpoint){
    abort.signal.throwIfAborted();
    if(browser.error||browser.child.exitCode!==null||browser.child.signalCode!==null)throw new RenderError('RENDER_FAILED','Browser exited during startup.');
    try{
     const [port,path]= (await readFile(join(profile,'DevToolsActivePort'),'utf8')).trim().split('\n');
     if(!/^\d+$/.test(port)||Number(port)<1||Number(port)>65535||!/^\/devtools\/browser\/[a-zA-Z0-9-]+$/.test(path))throw new RenderError('RENDER_FAILED','Invalid browser startup endpoint.');
     endpoint=`ws://127.0.0.1:${port}${path}`;
    }catch(error:any){if(error.code!=='ENOENT')throw error;}
    if(!endpoint)await delay(10,undefined,{signal:abort.signal});
   }
   abort.signal.throwIfAborted();
   const worker=own(fork(this.options.childPath??childPath,[],{
    detached:true,stdio:['ignore','ignore','ignore','ipc'],execArgv:['--max-old-space-size=256'],env,
   }));
   owned.push(worker);if(worker.child.pid)this.options.onProcess?.(worker.child.pid);
   result=await new Promise((resolve,reject)=>{
    worker.child.on('message',(message:any)=>{
     if(message.error)reject(new RenderError(message.error.code??'RENDER_FAILED',message.error.message));
     else if(Object.hasOwn(message,'result'))resolve(message.result);
    });
    worker.child.once('error',reject);
    worker.child.once('close',()=>reject(new RenderError('RENDER_FAILED','Worker exited before completing.')));
    worker.child.send!({job,browserEndpoint:endpoint},error=>{if(error)reject(error);});
   });
  }catch(error){failure=error;}
  finally{
   // Normal completion closes Chrome through CDP in the child. SIGTERM handles
   // failures before connection; grace is bounded by the same external deadline.
   for(const process of owned)signalGroup(process.child,timedOut?'SIGKILL':'SIGTERM');
   const escalation=setTimeout(()=>{for(const process of owned)signalGroup(process.child,'SIGKILL');},250);
   await Promise.all(owned.map(process=>process.closed));
   // A reaped browser leader can leave helpers exiting briefly in its group.
   // Keep the slot and cleanup watchdog until those descendants are gone too.
   const reapDeadline=performance.now()+1000;
   while(owned.some(process=>groupAlive(process.child))&&performance.now()<reapDeadline)await delay(5);
   clearTimeout(escalation);
   try{
    if(owned.some(process=>groupAlive(process.child)))throw new Error('Process group did not exit.');
    if(directory)await rm(directory,{recursive:true,force:true,maxRetries:2,retryDelay:10});
    this.busy=false;
   }catch{
    // Fail closed: never admit another job after incomplete storage cleanup.
    failure=new RenderError('RENDER_FAILED','Renderer process or storage cleanup failed; restart the service.');
   }
   clearTimeout(timer);
  }
  if(timedOut)throw new RenderError('RENDER_TIMEOUT','Rendering exceeded its wall deadline; owned processes terminated.');
  if(failure)throw failure;
  return result;
 }
}
