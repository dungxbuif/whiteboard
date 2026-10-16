import {validateRasterAsset} from './render-assets.js';
import {fork} from 'node:child_process';
import {fileURLToPath} from 'node:url';
import {validateSceneContent,type SceneContent} from '@collab/contracts';
import {validateDiagram,type CompiledDiagram} from '@collab/diagram-kit';

export const RENDER_LIMITS=Object.freeze({deadlineMs:10_000,dimension:4096,artifactBytes:20*1024*1024,sceneBytes:25*1024*1024,sourceChars:50_000,assets:100,nodes:500,edges:1000});
export type ResolvedAsset={mimeType:string;data:string};
export type RenderedArtifact={mimeType:string;bytes:Buffer};
export class RenderError extends Error {constructor(public code:string,message:string){super(message);this.name='RenderError';}}
export type RenderJob={kind:'mermaid';source:string}|{kind:'diagram';input:unknown}|{kind:'render';scene:SceneContent;format:'svg'|'png';assets:Record<string,ResolvedAsset>};
const childPath=fileURLToPath(new URL('../dist/render-child.js',import.meta.url));
function killGroup(pid:number|undefined){if(!pid)return;try{process.kill(-pid,'SIGKILL');}catch{try{process.kill(pid,'SIGKILL');}catch{}}}

/** One job per instance; there is deliberately no unbounded in-memory queue. */
export class RenderWorker {
 private busy=false;
 constructor(private readonly options:{deadlineMs?:number;childPath?:string;onProcess?:(pid:number)=>void}={}){}
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
  return new Promise((resolve,reject)=>{
   const child=fork(this.options.childPath??childPath,[],{detached:true,stdio:['ignore','ignore','ignore','ipc'],execArgv:['--max-old-space-size=256'],env:{PATH:process.env.PATH,HOME:process.env.HOME,NODE_ENV:'production',PLAYWRIGHT_BROWSERS_PATH:process.env.PLAYWRIGHT_BROWSERS_PATH}});
   let browserPid:number|undefined,settled=false;
   const finish=(error?:Error,result?:unknown)=>{if(settled)return;settled=true;clearTimeout(timer);killGroup(browserPid);killGroup(child.pid);this.busy=false;error?reject(error):resolve(result);};
   const timer=setTimeout(()=>finish(new RenderError('RENDER_TIMEOUT','Rendering exceeded its wall deadline; worker terminated.')),Math.min(this.options.deadlineMs??RENDER_LIMITS.deadlineMs,RENDER_LIMITS.deadlineMs));
   this.options.onProcess?.(child.pid!);
   child.on('message',(message:any)=>{if(message.kind==='browser'){browserPid=message.pid;return;}if(message.error)finish(new RenderError(message.error.code??'RENDER_FAILED',message.error.message));else finish(undefined,message.result);});
   child.on('error',error=>finish(error));child.on('exit',()=>finish(new RenderError('RENDER_FAILED','Worker exited before completing.')));
   child.send(job,error=>{if(error)finish(error);});
  });
 }
}
