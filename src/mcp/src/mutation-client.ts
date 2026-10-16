import type {SceneMutation,SceneRef} from '@collab/contracts';
import {assertSerializable,MAX_FRAME_BYTES} from '@collab/contracts';
import {AccessError} from './auth.js';
import type {GoClient} from './go-client.js';
export const agentScenePath=(ref:SceneRef)=>`/api/v1/agent/workspaces/${ref.workspaceId}/scenes/${ref.sceneId}`;
export async function submitMutation(api:GoClient,mutation:SceneMutation,tool:string){
 assertSerializable(mutation);
 // Serialize exactly once. No implicit retry or revision rebasing at this boundary.
 const bytes=Buffer.from(JSON.stringify(mutation));
 if(bytes.length>MAX_FRAME_BYTES)throw new AccessError(413,'PROJECTION_TOO_LARGE');
 return api.request(agentScenePath(mutation)+'/commands','POST',{tool,mutationBase64:bytes.toString('base64')});
}
export function mutationResult(api:GoClient,ref:SceneRef,receipt:Record<string,unknown>,createdIds:string[]=[],warnings:unknown[]=[]){
 return {receipt,sceneUrl:api.sceneURL(ref),createdIds,warnings};
}
