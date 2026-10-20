import api from '@/lib/api';

export type SCIMTokenStatus={configured:boolean;prefix?:string;createdAt?:string};
export type SCIMTokenResult={token:string;prefix:string;createdAt:string};
export interface SCIMClient{status(workspaceId:string):Promise<SCIMTokenStatus>;rotate(workspaceId:string):Promise<SCIMTokenResult>;revoke(workspaceId:string):Promise<void>}
const root=(workspaceId:string)=>`/workspaces/${encodeURIComponent(workspaceId)}/scim/credential`;
export const scimClient:SCIMClient={
 async status(workspaceId){return (await api.get<SCIMTokenStatus>(root(workspaceId))).data},
 async rotate(workspaceId){return (await api.post<SCIMTokenResult>(root(workspaceId))).data},
 async revoke(workspaceId){await api.delete(root(workspaceId))},
};
