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
  cancelPending(): void;
}

type CollectionHttp = {
  get<T>(url: string, config?: unknown): Promise<{ data: T }>;
  post<T>(url: string, body?: unknown, config?: unknown): Promise<{ data: T }>;
  patch<T>(url: string, body?: unknown, config?: unknown): Promise<{ data: T }>;
  put<T>(url: string, body?: unknown, config?: unknown): Promise<{ data: T }>;
  delete<T = unknown>(url: string, config?: unknown): Promise<{ data: T }>;
};

const headers = (workspaceId: string) => ({ 'X-Workspace-ID': workspaceId });
const rootPath = (workspaceId: string) => `/workspaces/${encodeURIComponent(workspaceId)}/collections`;

function hasDefinitiveResponse(error: unknown): boolean {
  return typeof error === 'object' && error !== null && 'response' in error && Boolean((error as { response?: unknown }).response);
}

export function collectionPath(workspaceSlug: string, collectionId: string): string {
  return `/w/${encodeURIComponent(workspaceSlug)}/collections/${encodeURIComponent(collectionId)}`;
}

export function createCollectionClient(http: CollectionHttp = api): CollectionClient {
  const pending = new Map<string, string>();
  const mutationKey = (...parts: unknown[]) => JSON.stringify(parts);
  const mutate = async <T>(key: string, send: (requestId: string) => Promise<{ data: T }>): Promise<T> => {
    const id = pending.get(key) ?? crypto.randomUUID();
    pending.set(key, id);
    try {
      const response = await send(id);
      pending.delete(key);
      return response.data;
    } catch (error) {
      if (hasDefinitiveResponse(error)) pending.delete(key);
      throw error;
    }
  };

  return {
    async list(workspaceId, cursor = '') {
      const response = await http.get<CollectionPage>(rootPath(workspaceId), {
        headers: headers(workspaceId), params: { limit: 50, ...(cursor ? { cursor } : {}) },
      });
      return response.data;
    },
    async get(workspaceId, collectionId) {
      return (await http.get<Collection>(`${rootPath(workspaceId)}/${encodeURIComponent(collectionId)}`, { headers: headers(workspaceId) })).data;
    },
    async create(workspaceId, draft) {
      return mutate(JSON.stringify(['create', workspaceId, draft]), requestId => http.post<Collection>(rootPath(workspaceId), { requestId, ...draft }, { headers: headers(workspaceId) }));
    },
    async update(workspaceId, collectionId, expectedRevision, draft) {
      return mutate(mutationKey('update', workspaceId, collectionId, expectedRevision, draft), requestId => http.patch<Collection>(`${rootPath(workspaceId)}/${encodeURIComponent(collectionId)}`, { requestId, expectedRevision, ...draft }, { headers: headers(workspaceId) }));
    },
    async delete(workspaceId, collectionId, expectedRevision) {
      await mutate(mutationKey('delete', workspaceId, collectionId, expectedRevision), requestId => http.delete(`${rootPath(workspaceId)}/${encodeURIComponent(collectionId)}`, { headers: headers(workspaceId), data: { requestId, expectedRevision } }));
    },
    async addScene(workspaceId, collectionId, expectedRevision, sceneId) {
      return mutate(mutationKey('addScene', workspaceId, collectionId, expectedRevision, sceneId), requestId => http.post<Collection>(`${rootPath(workspaceId)}/${encodeURIComponent(collectionId)}/scenes`, { requestId, expectedRevision, sceneId }, { headers: headers(workspaceId) }));
    },
    async removeScene(workspaceId, collectionId, expectedRevision, sceneId) {
      return mutate(mutationKey('removeScene', workspaceId, collectionId, expectedRevision, sceneId), requestId => http.delete<Collection>(`${rootPath(workspaceId)}/${encodeURIComponent(collectionId)}/scenes/${encodeURIComponent(sceneId)}`, { headers: headers(workspaceId), data: { requestId, expectedRevision } }));
    },
    async reorder(workspaceId, collectionId, expectedRevision, sceneIds) {
      return mutate(mutationKey('reorder', workspaceId, collectionId, expectedRevision, sceneIds), requestId => http.put<Collection>(`${rootPath(workspaceId)}/${encodeURIComponent(collectionId)}/order`, { requestId, expectedRevision, sceneIds }, { headers: headers(workspaceId) }));
    },
    cancelPending() {
      pending.clear();
    },
  };
}

export const collectionClient: CollectionClient = createCollectionClient();
