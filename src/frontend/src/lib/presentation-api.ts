import api from './api';
import { publicApi } from './public-api';
import type { LivePresentationAPI, PresentationGrant, PresentationInvite } from '@/collab/live-presentation';

export type PendingPresentationGrant={sessionId:string;workspaceId:string;sceneId:string;principalId:string;displayName:string;token:string;expiresAt:string};

export function presentationAPI(workspaceId:string,sceneId:string):LivePresentationAPI{
 const headers={'X-Workspace-ID':workspaceId};
 return {
  async create(requestId){return (await api.post<PresentationGrant>(`/scenes/${sceneId}/presentations`,{requestId},{headers})).data;},
  async createLink(sessionId,requestId){return (await api.post<PresentationInvite>(`/scenes/${sceneId}/presentations/${sessionId}/links`,{requestId},{headers})).data;},
  async admit(sessionId,connectionId,expectedRevision){return (await api.post<PresentationGrant>(`/scenes/${sceneId}/presentations/${sessionId}/admit`,{connectionId,expectedRevision},{headers})).data;},
 };
}

export async function joinPresentation(token:string,displayName:string):Promise<PendingPresentationGrant>{
 return (await publicApi.post<PendingPresentationGrant>(`/presentations/${encodeURIComponent(token)}/join`,{displayName})).data;
}
