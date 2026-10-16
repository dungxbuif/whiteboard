import {AccessError,apiBase} from './auth.js';
export type AgentIdentity={principalId:string;keyId:string};
export class GoClient {
 private readonly base:URL;
 constructor(url:string,private readonly key:string){this.base=apiBase(url)}
 async request(path:string,method='GET',body?:unknown):Promise<Record<string,unknown>> {
  if(!path.startsWith('/api/v1/agent/'))throw new Error('Invalid agent API path');
  const response=await fetch(new URL(path,this.base),{method,headers:{Authorization:`Bearer ${this.key}`,...(body===undefined?{}:{'Content-Type':'application/json'})},body:body===undefined?undefined:JSON.stringify(body),redirect:'error',signal:AbortSignal.timeout(10000)});
  if(!response.ok){await response.body?.cancel();throw new AccessError(response.status,response.status===401?'UNAUTHENTICATED':response.status===403?'ACCESS_DENIED':response.status===404?'NOT_FOUND':'AGENT_API_ERROR')}
  if(!response.body)throw new AccessError(502,'INVALID_API_RESPONSE');const reader=response.body.getReader();const chunks:Uint8Array[]=[];let size=0;
  try{for(;;){const part=await reader.read();if(part.done)break;size+=part.value.byteLength;if(size>256*1024)throw new AccessError(413,'PROJECTION_TOO_LARGE');chunks.push(part.value)}}finally{await reader.cancel()}
  let result:unknown;try{result=JSON.parse(Buffer.concat(chunks).toString('utf8'))}catch{throw new AccessError(502,'INVALID_API_RESPONSE')}
  if(!result||typeof result!=='object'||Array.isArray(result))throw new AccessError(502,'INVALID_API_RESPONSE');return result as Record<string,unknown>;
 }
 async authenticate():Promise<AgentIdentity>{const result=await this.request('/api/v1/agent/auth');if(typeof result.principalId!=='string'||typeof result.keyId!=='string')throw new AccessError(502,'INVALID_API_RESPONSE');return result as AgentIdentity}
 async roomCredential(workspaceId:string,sceneId:string):Promise<Record<string,unknown>>{return this.request(`/api/v1/agent/workspaces/${encodeURIComponent(workspaceId)}/scenes/${encodeURIComponent(sceneId)}/room-token`,'POST')}
}
