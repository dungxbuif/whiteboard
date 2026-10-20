import api from '@/lib/api';

export type LibraryItem={id:string;name:string;currentRevision:string;createdAt:string;updatedAt:string};
export type LibraryRevision={libraryItemId:string;revision:string;content:unknown;contentHash:string;byteSize:number;createdAt:string};
export type LibraryInsertResult={requestId:string;sequence:string;stateVector:string;elementIds:string[]};
export type LibraryPage={items:LibraryItem[]};
export interface LibraryClient{
 list(workspaceId:string,query?:string):Promise<LibraryPage>;
 import(workspaceId:string,name:string,content:unknown):Promise<{item:LibraryItem;revision:LibraryRevision}>;
 publish(workspaceId:string,itemId:string,expectedRevision:string,name:string,content:unknown):Promise<{item:LibraryItem;revision:LibraryRevision}>;
 revision(workspaceId:string,itemId:string,revision:number):Promise<LibraryRevision>;
 export(workspaceId:string,itemId:string,revision:number):Promise<unknown>;
 insert(workspaceId:string,sceneId:string,itemId:string,revision:number,expectedSequence:string,nativeItemId?:string):Promise<LibraryInsertResult>;
 cancelPending():void;
}
type Http={get<T>(url:string,config?:unknown):Promise<{data:T}>;post<T>(url:string,body?:unknown,config?:unknown):Promise<{data:T}>;put<T>(url:string,body?:unknown,config?:unknown):Promise<{data:T}>};
const root=(workspaceId:string)=>`/workspaces/${encodeURIComponent(workspaceId)}/libraries`;
const headers=(workspaceId:string)=>({'X-Workspace-ID':workspaceId});
const definitive=(error:unknown)=>typeof error==='object'&&error!==null&&'response'in error&&Boolean((error as {response?:unknown}).response);
export const libraryPath=(workspaceSlug:string)=>`/w/${encodeURIComponent(workspaceSlug)}/libraries`;
export function createLibraryClient(http:Http=api):LibraryClient{
 const pending=new Map<string,string>();
 const mutation=async<T>(key:string,send:(requestId:string)=>Promise<{data:T}>)=>{const requestId=pending.get(key)??crypto.randomUUID();pending.set(key,requestId);try{const response=await send(requestId);pending.delete(key);return response.data}catch(error){if(definitive(error))pending.delete(key);throw error}};
 return {
  async list(workspaceId,query=''){return (await http.get<LibraryPage>(root(workspaceId),{headers:headers(workspaceId),params:{limit:50,...(query?{q:query}:{})}})).data},
  async import(workspaceId,name,content){return mutation(JSON.stringify(['import',workspaceId,name,content]),requestId=>http.post(`${root(workspaceId)}/import`,{requestId,name,content},{headers:headers(workspaceId)}))},
  async publish(workspaceId,itemId,expectedRevision,name,content){return mutation(JSON.stringify(['publish',workspaceId,itemId,expectedRevision,name,content]),requestId=>http.put(`${root(workspaceId)}/${encodeURIComponent(itemId)}`,{requestId,expectedRevision,name,content},{headers:headers(workspaceId)}))},
  async revision(workspaceId,itemId,revision){return (await http.get<LibraryRevision>(`${root(workspaceId)}/${encodeURIComponent(itemId)}/revisions/${revision}`,{headers:headers(workspaceId)})).data},
  async export(workspaceId,itemId,revision){return (await http.get<unknown>(`${root(workspaceId)}/${encodeURIComponent(itemId)}/revisions/${revision}/export`,{headers:headers(workspaceId)})).data},
  async insert(workspaceId,sceneId,itemId,revision,expectedSequence,nativeItemId){const key=JSON.stringify(['insert',workspaceId,sceneId,itemId,revision,expectedSequence,nativeItemId]);return mutation(key,requestId=>http.post(`/workspaces/${encodeURIComponent(workspaceId)}/scenes/${encodeURIComponent(sceneId)}/library-insertions`,{requestId,libraryItemId:itemId,revision,expectedSequence,...(nativeItemId?{nativeItemId}:{})},{headers:headers(workspaceId)}))},
  cancelPending(){pending.clear()},
 };
}
export const libraryClient=createLibraryClient();

