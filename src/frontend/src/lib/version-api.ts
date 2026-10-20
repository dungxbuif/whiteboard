import api from '@/lib/api';

export type VersionSceneRef = { workspaceId: string; sceneId: string };
export type SceneVersion = {
  id: string; sceneId: string; documentId: string; sequence: string; hash: string;
  label?: string; pinned: boolean; kind: string; creator: { kind: string; id: string };
  metadataRevision: string; createdAt: string; updatedAt: string;
};
export type SceneVersionPage = { items: SceneVersion[]; nextCursor?: string };
export type SceneVersionDiff = { addedIds: string[]; removedIds: string[]; changedIds: string[]; changedAppStateFields: string[] };
export type RestoreVersionResult = { requestId: string; sequence: string; stateVector: string; preRestoreVersionId: string };
export type RestoreVersionInput = { requestId: string; expectedSequence: string };

type HTTP = {
  get<T>(path: string, config: object): Promise<{ data: T }>;
  post<T>(path: string, body: unknown, config: object): Promise<{ data: T }>;
  patch?<T>(path: string, body: unknown, config: object): Promise<{ data: T }>;
};

export function createVersionClient(http: HTTP = api) {
  const config = (ref: VersionSceneRef) => ({ headers: { 'X-Workspace-ID': ref.workspaceId } });
  return {
    async list(ref: VersionSceneRef): Promise<SceneVersionPage> {
      return (await http.get<SceneVersionPage>(`/scenes/${encodeURIComponent(ref.sceneId)}/versions?limit=50`, config(ref))).data;
    },
    async diff(ref: VersionSceneRef, versionId: string, againstSequence: string): Promise<SceneVersionDiff> {
      return (await http.get<SceneVersionDiff>(`/scenes/${encodeURIComponent(ref.sceneId)}/versions/${encodeURIComponent(versionId)}/diff?againstSequence=${encodeURIComponent(againstSequence)}`, config(ref))).data;
    },
    async restore(ref: VersionSceneRef, versionId: string, input: RestoreVersionInput): Promise<RestoreVersionResult> {
      return (await http.post<RestoreVersionResult>(`/scenes/${encodeURIComponent(ref.sceneId)}/versions/${encodeURIComponent(versionId)}/restore`, input, config(ref))).data;
    },
    async setMetadata(ref: VersionSceneRef, versionId: string, input: { requestId: string; expectedRevision: string; label?: string; pinned?: boolean }): Promise<SceneVersion> {
      if (!http.patch) throw new Error('Version metadata transport unavailable');
      return (await http.patch<SceneVersion>(`/scenes/${encodeURIComponent(ref.sceneId)}/versions/${encodeURIComponent(versionId)}`, input, config(ref))).data;
    },
  };
}

export type VersionClient = ReturnType<typeof createVersionClient>;
export const versionClient = createVersionClient();
