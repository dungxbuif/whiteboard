import api from '@/lib/api';

export type ActivityItem = {
  id: string;
  workspaceId: string;
  sceneId?: string;
  actorKind: 'user' | 'public' | 'agent' | 'system';
  actorName: string;
  kind: string;
  subjectId?: string;
  occurredAt: string;
  safeMetadata: Record<string, string | number | boolean>;
};

export type ActivityPage = { items: ActivityItem[]; nextCursor?: string; retainedSince?: string };

export interface ActivityClient {
  list(workspaceId: string, sceneId?: string, cursor?: string): Promise<ActivityPage>;
}

export function activityPath(workspaceId: string, sceneId?: string): string {
  return sceneId ? `/scenes/${encodeURIComponent(sceneId)}/activity` : `/workspaces/${encodeURIComponent(workspaceId)}/activity`;
}

export const activityClient: ActivityClient = {
  async list(workspaceId, sceneId, cursor = '') {
    const query = `?limit=50${cursor ? `&cursor=${encodeURIComponent(cursor)}` : ''}`;
    return (await api.get<ActivityPage>(activityPath(workspaceId, sceneId) + query, { headers: { 'X-Workspace-ID': workspaceId } })).data;
  },
};

