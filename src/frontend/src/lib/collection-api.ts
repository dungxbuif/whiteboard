import api from '@/lib/api';

export type CollectionVisibility = 'workspace' | 'private';

export type CollectionScene = {
  id: string;
  publicId?: string;
  title: string;
  position: number;
  updatedAt: string;
};

export type Collection = {
  id: string;
  name: string;
  visibility: CollectionVisibility;
  revision: string;
  sceneCount: number;
  mine: boolean;
  canSetPrivate: boolean;
  scenes: CollectionScene[];
  createdAt: string;
  updatedAt: string;
};

export type CollectionPage = { items: Collection[]; nextCursor?: string };
export type CollectionDraft = { name: string; visibility: CollectionVisibility };

export interface CollectionClient {
  list(workspaceId: string, cursor?: string): Promise<CollectionPage>;
  get(workspaceId: string, collectionId: string): Promise<Collection>;
  create(workspaceId: string, draft: CollectionDraft): Promise<Collection>;
  update(workspaceId: string, collectionId: string, revision: string, draft: CollectionDraft): Promise<Collection>;
  delete(workspaceId: string, collectionId: string, revision: string): Promise<void>;
  addScene(workspaceId: string, collectionId: string, revision: string, sceneId: string): Promise<Collection>;
  removeScene(workspaceId: string, collectionId: string, revision: string, sceneId: string): Promise<Collection>;
  reorder(workspaceId: string, collectionId: string, revision: string, sceneIds: string[]): Promise<Collection>;
}

const headers = (workspaceId: string) => ({ 'X-Workspace-ID': workspaceId });
const requestId = () => crypto.randomUUID();
const rootPath = (workspaceId: string) => `/workspaces/${encodeURIComponent(workspaceId)}/collections`;

export function collectionPath(workspaceSlug: string, collectionId: string): string {
  return `/w/${encodeURIComponent(workspaceSlug)}/collections/${encodeURIComponent(collectionId)}`;
}

export const collectionClient: CollectionClient = {
  async list(workspaceId, cursor = '') {
    const response = await api.get<CollectionPage>(rootPath(workspaceId), {
      headers: headers(workspaceId), params: { limit: 50, ...(cursor ? { cursor } : {}) },
    });
    return response.data;
  },
  async get(workspaceId, collectionId) {
    return (await api.get<Collection>(`${rootPath(workspaceId)}/${encodeURIComponent(collectionId)}`, { headers: headers(workspaceId) })).data;
  },
  async create(workspaceId, draft) {
    return (await api.post<Collection>(rootPath(workspaceId), { requestId: requestId(), ...draft }, { headers: headers(workspaceId) })).data;
  },
  async update(workspaceId, collectionId, expectedRevision, draft) {
    return (await api.patch<Collection>(`${rootPath(workspaceId)}/${encodeURIComponent(collectionId)}`, { requestId: requestId(), expectedRevision, ...draft }, { headers: headers(workspaceId) })).data;
  },
  async delete(workspaceId, collectionId, expectedRevision) {
    await api.delete(`${rootPath(workspaceId)}/${encodeURIComponent(collectionId)}`, { headers: headers(workspaceId), data: { requestId: requestId(), expectedRevision } });
  },
  async addScene(workspaceId, collectionId, expectedRevision, sceneId) {
    return (await api.post<Collection>(`${rootPath(workspaceId)}/${encodeURIComponent(collectionId)}/scenes`, { requestId: requestId(), expectedRevision, sceneId }, { headers: headers(workspaceId) })).data;
  },
  async removeScene(workspaceId, collectionId, expectedRevision, sceneId) {
    return (await api.delete<Collection>(`${rootPath(workspaceId)}/${encodeURIComponent(collectionId)}/scenes/${encodeURIComponent(sceneId)}`, { headers: headers(workspaceId), data: { requestId: requestId(), expectedRevision } })).data;
  },
  async reorder(workspaceId, collectionId, expectedRevision, sceneIds) {
    return (await api.put<Collection>(`${rootPath(workspaceId)}/${encodeURIComponent(collectionId)}/order`, { requestId: requestId(), expectedRevision, sceneIds }, { headers: headers(workspaceId) })).data;
  },
};
