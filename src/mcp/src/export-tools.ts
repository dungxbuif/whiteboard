import {validateRasterAsset} from './render-assets.js';
import {randomUUID} from 'node:crypto';
import {validateSceneContent,type SceneContent,type SceneRef} from '@collab/contracts';
import {RenderWorker,RenderError,RENDER_LIMITS,type RenderedArtifact,type ResolvedAsset} from './render-worker.js';
export type ExportFormat='json'|'svg'|'png';
export type ScopedArtifact={id:string;mimeType:string;expiresAt:string;url:string;revision:string};
export type ExportProjection=SceneRef&{sequence:string;content:SceneContent};
export interface ExportAuthority {
 principalId:string;
 projection(ref:SceneRef):Promise<ExportProjection>;
 asset(ref:SceneRef,fileId:string,assetId:string):Promise<ResolvedAsset>;
}
const TTL_MS=5*60*1000,MAX_CACHE_BYTES=64*1024*1024,MAX_ARTIFACTS=16;
type Stored={grant:ScopedArtifact;principalId:string;ref:SceneRef;artifact:RenderedArtifact;expires:number};
export class ExportService {
 private readonly artifacts=new Map<string,Stored>();
 private usedBytes=0;
 private active=false;
 constructor(private readonly renderer=new RenderWorker(),private readonly now=Date.now){}
 async exportScene(authority:ExportAuthority,ref:SceneRef,format:ExportFormat):Promise<ScopedArtifact>{
  if(!['json','svg','png'].includes(format))throw new RenderError('INVALID_FORMAT','Use json, svg or png.');
  if(this.active)throw new RenderError('RENDER_BUSY','An export is active; retry later.');
  this.active=true;
  try{
   const projection=await authority.projection(ref);
   if(projection.workspaceId!==ref.workspaceId||projection.sceneId!==ref.sceneId||!/^\d+$/.test(projection.sequence))throw new RenderError('INVALID_PROJECTION','Invalid export authority projection.');
   validateSceneContent(projection.content);
   const assets:Record<string,ResolvedAsset>={};let total=0;
   const used=new Set(projection.content.elements.filter(e=>e.type==='image'&&!e.isDeleted).map(e=>e.fileId));
   if(used.size>RENDER_LIMITS.assets)throw new RenderError('RENDER_LIMIT','Too many assets.');
   for(const id of used){const file=id&&projection.content.files[id];if(!id||!file)throw new RenderError('MISSING_ASSET','Missing image reference.');const asset=await authority.asset(ref,id,file.assetId);validateRasterAsset(asset);if(asset.mimeType!==file.mimeType)throw new RenderError('INVALID_ASSET','Asset MIME differs from scene reference.');total+=Buffer.byteLength(asset.data,'base64');if(total>RENDER_LIMITS.artifactBytes)throw new RenderError('RENDER_LIMIT','Assets exceed 20 MiB.');assets[id]=asset;}
   let artifact:RenderedArtifact;
   if(format==='json'){
    const files=Object.fromEntries(Object.entries(assets).map(([id,a])=>[id,{id,mimeType:a.mimeType,created:projection.content.files[id].created,dataURL:`data:${a.mimeType};base64,${a.data}`}]))
    artifact={mimeType:'application/json',bytes:Buffer.from(JSON.stringify({type:'excalidraw',version:2,source:'whiteboard',...projection.content,files}))};
   }else artifact=await this.renderer.render(projection.content,format,assets);
   if(artifact.bytes.length>RENDER_LIMITS.artifactBytes)throw new RenderError('RENDER_LIMIT','Artifact exceeds 20 MiB.');
   // Recheck after slow rendering so a revocation cannot create a fresh grant.
   await authority.projection(ref);
   this.sweep();
   if(this.artifacts.size>=MAX_ARTIFACTS||this.usedBytes+artifact.bytes.length>MAX_CACHE_BYTES)throw new RenderError('EXPORT_CAPACITY','Artifact capacity reached; retry after expiry.');
   const id=randomUUID(),expires=this.now()+TTL_MS;
   const grant={id,mimeType:artifact.mimeType,expiresAt:new Date(expires).toISOString(),url:`/mcp/artifacts/${id}`,revision:projection.sequence};
   this.artifacts.set(id,{grant,principalId:authority.principalId,ref:{...ref},artifact,expires});this.usedBytes+=artifact.bytes.length;
   return {...grant};
  }finally{this.active=false;}
 }
 async download(authority:ExportAuthority,id:string):Promise<RenderedArtifact>{
  this.sweep();const entry=this.artifacts.get(id);
  if(!entry||entry.principalId!==authority.principalId)throw new RenderError('ARTIFACT_NOT_FOUND','Artifact expired or unavailable.');
  await authority.projection(entry.ref);
  if(entry.expires<=this.now())throw new RenderError('ARTIFACT_NOT_FOUND','Artifact expired or unavailable.');
  return {mimeType:entry.artifact.mimeType,bytes:Buffer.from(entry.artifact.bytes)};
 }
 private sweep(){for(const [id,entry] of this.artifacts)if(entry.expires<=this.now()){this.artifacts.delete(id);this.usedBytes-=entry.artifact.bytes.length;}}
}
