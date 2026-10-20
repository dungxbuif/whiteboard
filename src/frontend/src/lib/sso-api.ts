import api from '@/lib/api';

export type SSOMode='optional'|'enforced';
export type SSOConnection={id:string;workspaceId:string;issuer:string;clientId:string;mode:SSOMode;publicLinksEnabled:boolean;revision:number;createdAt:string;updatedAt:string};
export type SSOConnectionInput={issuer:string;clientId:string;clientSecret:string;mode:SSOMode;publicLinksEnabled:boolean;requestId:string;expectedRevision:number};
export interface SSOClient {begin(connectionId:string,returnPath:string):Promise<{url:string;expiresAt:string}>;beginLink(connectionId:string,returnPath:string,requestId:string):Promise<{url:string;expiresAt:string}>;exchange(code:string):Promise<{token:string;user:{id:string;email:string;name:string;avatarUrl?:string}}>;get(workspaceId:string):Promise<SSOConnection>;configure(workspaceId:string,input:SSOConnectionInput):Promise<SSOConnection>}
export const ssoClient:SSOClient={
  async begin(connectionId,returnPath){return (await api.post('/sso/begin',{connectionId,returnPath})).data},
  async beginLink(connectionId,returnPath,requestId){return (await api.post('/sso/link/begin',{connectionId,returnPath,requestId})).data},
  async exchange(code){return (await api.post('/sso/exchange',{code})).data},
  async get(workspaceId){return (await api.get(`/workspaces/${encodeURIComponent(workspaceId)}/sso`)).data},
  async configure(workspaceId,input){return (await api.put(`/workspaces/${encodeURIComponent(workspaceId)}/sso`,input)).data},
};

export function uncertainSSOError(error:unknown){return Boolean((error as {request?:unknown;response?:unknown})?.request)&&!(error as {response?:unknown}).response}
