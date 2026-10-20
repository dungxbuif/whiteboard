import api from '@/lib/api';

export type RetentionPolicyInput={activityDays:number;notificationDays:number;automaticVersionLimit:number;manualVersionDays?:number;commentsDays?:number;trashHardPurge:false};
export type RetentionCounts={activity:number;notifications:number;versions:number;manualVersions:number;comments:number};
export type RetentionPreview={id:string;workspaceId:string;policyRevision:string;policy:RetentionPolicyInput;policyHash:string;counts:RetentionCounts;earliestDestructiveAt:string;expiresAt:string;createdAt:string};
export type RetentionPolicy={workspaceId:string;revision:string;policy:RetentionPolicyInput;policyHash:string;effectiveAt?:string;scheduledAt?:string};
export type RetentionHold={id:string;reference:{workspaceId:string;kind:string;id:string};versionId?:string;held:boolean;updatedAt:string};
export interface RetentionClient{policy(workspaceId:string):Promise<RetentionPolicy>;preview(workspaceId:string,input:RetentionPolicyInput):Promise<RetentionPreview>;schedule(workspaceId:string,previewId:string,revision:string,requestId:string):Promise<RetentionPolicy>;hold(workspaceId:string,kind:string,id:string,held:boolean,requestId:string,versionId?:string):Promise<RetentionHold>}
const root=(workspaceId:string)=>`/workspaces/${encodeURIComponent(workspaceId)}/retention`;
export const retentionClient:RetentionClient={
 async policy(workspaceId){return (await api.get<RetentionPolicy>(`${root(workspaceId)}/policy`)).data},
 async preview(workspaceId,input){return (await api.post<RetentionPreview>(`${root(workspaceId)}/previews`,input)).data},
 async schedule(workspaceId,previewId,revision,requestId){return (await api.post<RetentionPolicy>(`${root(workspaceId)}/schedule`,{previewId,expectedRevision:revision,requestId})).data},
 async hold(workspaceId,kind,id,held,requestId,versionId){return (await api.put<RetentionHold>(`${root(workspaceId)}/holds`,{kind,id,held,requestId,...(versionId?{versionId}:{})})).data},
};
