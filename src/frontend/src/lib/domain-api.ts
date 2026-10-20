import api from '@/lib/api';

export type WorkspaceDomainStatus = 'pending' | 'verified' | 'captured' | 'revoked';
export type WorkspaceDomain = { id:string; workspaceId:string; domain:string; status:WorkspaceDomainStatus; challengeExpiresAt:string; verifiedAt?:string; capturedAt?:string; revision:number; createdAt:string; updatedAt:string };
export type DomainChallenge = WorkspaceDomain & { dnsName:string; txtValue:string };
export type MembershipResult = { domain:WorkspaceDomain; dataMoved:false };
export interface DomainClient { list(workspaceId:string):Promise<WorkspaceDomain[]>; begin(workspaceId:string,value:string,requestId?:string):Promise<DomainChallenge>; verify(workspaceId:string,id:string,requestId?:string):Promise<WorkspaceDomain>; capture(workspaceId:string,id:string,requestId?:string):Promise<MembershipResult>; revoke(workspaceId:string,id:string,requestId?:string):Promise<WorkspaceDomain> }
const root=(workspaceId:string)=>`/workspaces/${encodeURIComponent(workspaceId)}/domains`;
const requestId=()=>crypto.randomUUID();
export const domainClient:DomainClient={
  async list(workspaceId){return (await api.get<{domains:WorkspaceDomain[]}>(root(workspaceId))).data.domains},
  async begin(workspaceId,value,id=requestId()){return (await api.post<DomainChallenge>(root(workspaceId),{domain:value,requestId:id})).data},
  async verify(workspaceId,domainId,id=requestId()){return (await api.post<WorkspaceDomain>(`${root(workspaceId)}/${encodeURIComponent(domainId)}/verify`,{requestId:id})).data},
  async capture(workspaceId,domainId,id=requestId()){return (await api.post<MembershipResult>(`${root(workspaceId)}/${encodeURIComponent(domainId)}/capture`,{requestId:id})).data},
  async revoke(workspaceId,domainId,id=requestId()){return (await api.post<WorkspaceDomain>(`${root(workspaceId)}/${encodeURIComponent(domainId)}/revoke`,{requestId:id})).data},
};
