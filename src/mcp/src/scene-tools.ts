import {z} from 'zod';
import type {Tool} from '@modelcontextprotocol/sdk/types.js';
import type {GoClient} from './go-client.js';
import {agentScenePath,mutationResult,submitMutation} from './mutation-client.js';
export const sceneRefSchema=z.object({workspaceId:z.uuid(),sceneId:z.uuid()}).strict();
export const sequenceSchema=z.string().regex(/^(0|[1-9][0-9]*)$/).refine(v=>BigInt(v)<=9223372036854775807n);
export const mutationFields={sceneRef:sceneRefSchema,requestId:z.string().regex(/^[\x20-\x7e]{1,128}$/),expectedSequence:sequenceSchema};
const elementId=z.string().regex(/^[A-Za-z0-9_-]{1,256}$/);
export const sceneSchemas={
 patch_elements:z.object({...mutationFields,patches:z.array(z.object({id:elementId,fields:z.record(z.string(),z.unknown())}).strict()).min(1).max(500)}).strict(),
 delete_elements:z.object({...mutationFields,ids:z.array(elementId).min(1).max(500)}).strict(),
 get_scene_activity:z.object({sceneRef:sceneRefSchema,cursor:sequenceSchema.default('0'),limit:z.number().int().min(1).max(50).default(50)}).strict(),
};
export const sceneTools:Tool[]=Object.entries(sceneSchemas).map(([name,schema])=>({name,description:name==='get_scene_activity'?'Read bounded redacted committed activity. Requires scene:read.':'Apply targeted changes with an exact durable sequence. Requires scene:write; read is independent. Retain requestId and arguments on uncertain retries.',inputSchema:z.toJSONSchema(schema) as Tool['inputSchema'],annotations:{readOnlyHint:name==='get_scene_activity',destructiveHint:name!=='get_scene_activity',idempotentHint:true,openWorldHint:false}}));
export async function callSceneTool(name:string,args:Record<string,unknown>,api:GoClient):Promise<Record<string,unknown>|undefined>{
 if(name==='get_scene_activity'){const a=sceneSchemas.get_scene_activity.parse(args);return api.request(agentScenePath(a.sceneRef)+`/activity?cursor=${a.cursor}&limit=${a.limit}`);}
 if(name==='patch_elements'){const a=sceneSchemas.patch_elements.parse(args);const receipt=await submitMutation(api,{...a.sceneRef,requestId:a.requestId,expectedSequence:a.expectedSequence,commands:[{op:'patch',patches:a.patches}]},name);return mutationResult(api,a.sceneRef,receipt);}
 if(name==='delete_elements'){const a=sceneSchemas.delete_elements.parse(args);const receipt=await submitMutation(api,{...a.sceneRef,requestId:a.requestId,expectedSequence:a.expectedSequence,commands:[{op:'delete',ids:a.ids}]},name);return mutationResult(api,a.sceneRef,receipt);}
}
