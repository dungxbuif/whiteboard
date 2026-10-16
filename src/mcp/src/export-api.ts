import {z} from 'zod';
import type {Tool} from '@modelcontextprotocol/sdk/types.js';
import type {GoClient} from './go-client.js';
import {AccessError} from './auth.js';
import {sceneRefSchema} from './scene-tools.js';
import {ExportService,type ExportAuthority,type ExportProjection} from './export-tools.js';
import {RenderError} from './render-worker.js';
import {agentScenePath} from './mutation-client.js';
export const exportSchema=z.object({sceneRef:sceneRefSchema,format:z.enum(['json','svg','png'])}).strict();
export const exportTools:Tool[]=[{name:'export_scene',description:'Export the current durable scene as editable JSON, SVG or PNG. Requires independent scene:export. Download URL requires the same bearer key and expires within five minutes; every download rechecks current access.',inputSchema:z.toJSONSchema(exportSchema) as Tool['inputSchema'],annotations:{readOnlyHint:true,destructiveHint:false,idempotentHint:false,openWorldHint:false}}];
export function exportAuthority(api:GoClient):ExportAuthority{
 if(!api.identity)throw new AccessError(401,'UNAUTHENTICATED');
 return {principalId:api.identity.principalId,projection:async ref=>await api.exportRequest(agentScenePath(ref)+'/export-projection') as ExportProjection,asset:async(ref,fileId,assetId)=>{
  const a=await api.exportRequest(agentScenePath(ref)+`/export-assets/${encodeURIComponent(fileId)}?assetId=${encodeURIComponent(assetId)}`);
  if(typeof a.mimeType!=='string'||typeof a.data!=='string')throw new AccessError(502,'INVALID_API_RESPONSE');return {mimeType:a.mimeType,data:a.data};
 }};
}
export function renderAccessError(error:unknown):never{if(error instanceof RenderError)throw new AccessError(error.code==='ARTIFACT_NOT_FOUND'?404:error.code==='RENDER_BUSY'||error.code==='EXPORT_CAPACITY'?429:error.code==='RENDER_TIMEOUT'?503:400,error.code);throw error;}
export async function callExportTool(name:string,args:Record<string,unknown>,api:GoClient,service?:ExportService,publicURL?:string):Promise<Record<string,unknown>|undefined>{
 if(name!=='export_scene')return;
 try{const a=exportSchema.parse(args);if(!service)return api.remoteExport(a);const result=await service.exportScene(exportAuthority(api),a.sceneRef,a.format);return {...result,url:publicURL?new URL(`/mcp/artifacts/${result.id}`,publicURL).href:api.artifactURL(result.id)};}catch(e){renderAccessError(e);}
}
