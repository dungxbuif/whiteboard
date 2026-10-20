import api from '@/lib/api';
import { publicApi } from '@/lib/public-api';

export type SceneRef = { workspaceId: string; sceneId: string };
export type CommentPrincipal = { kind: 'user' | 'public'; id: string; owner?: boolean; canResolve?: boolean; token?: string };
export type CommentAnchor = { elementId?: string; x: number; y: number };
export type CommentMessage = { id: string; threadId: string; authorKind: string; authorName: string; body: string; revision: string; mine: boolean; editedAt?: string; deletedAt?: string; createdAt: string; updatedAt: string };
export type CommentThread = { id: string; sceneId: string; creatorKind: string; creatorName: string; anchor: CommentAnchor; status: 'open' | 'resolved'; revision: string; subscribed: boolean; messages: CommentMessage[]; nextMessageCursor?: string; createdAt: string; updatedAt: string };
export type CommentPage = { items: CommentThread[]; nextCursor?: string };
export type Notification = { id: string; workspaceId: string; sceneId: string; kind: string; actorName: string; threadId: string; deepLink: string; readAt?: string; createdAt: string };
export type NotificationPage = { items: Notification[]; nextCursor?: string };

export interface CommentClient {
  list(ref: SceneRef): Promise<CommentPage>;
  create(ref: SceneRef, anchor: CommentAnchor, body: string): Promise<CommentThread>;
  get(ref: SceneRef, threadId: string, messageCursor?: string): Promise<CommentThread>;
  reply(ref: SceneRef, threadId: string, body: string): Promise<CommentMessage>;
  updateThread(ref: SceneRef, thread: CommentThread, input: { resolved?: boolean; subscribed?: boolean }): Promise<CommentThread>;
  deleteMessage(ref: SceneRef, threadId: string, messageId: string, revision: string): Promise<void>;
}

const requestId = () => crypto.randomUUID();

export function createCommentClient(principal: CommentPrincipal): CommentClient {
  const transport = principal.kind === 'public' ? publicApi : api;
  const config = (ref: SceneRef) => ({ headers: { 'X-Workspace-ID': ref.workspaceId, ...(principal.kind === 'public' && principal.token ? { Authorization: `Bearer ${principal.token}` } : {}) } });
  return {
    async list(ref) { return (await transport.get<CommentPage>(`/scenes/${ref.sceneId}/comments?limit=50`, config(ref))).data; },
    async create(ref, anchor, body) { return (await transport.post<CommentThread>(`/scenes/${ref.sceneId}/comments`, { requestId: requestId(), anchor, body }, config(ref))).data; },
    async get(ref, threadId, cursor = '') { return (await transport.get<CommentThread>(`/scenes/${ref.sceneId}/comments/${threadId}${cursor ? `?messageCursor=${encodeURIComponent(cursor)}&messageLimit=50` : ''}`, config(ref))).data; },
    async reply(ref, threadId, body) { return (await transport.post<CommentMessage>(`/scenes/${ref.sceneId}/comments/${threadId}/replies`, { requestId: requestId(), body }, config(ref))).data; },
    async updateThread(ref, thread, input) { return (await transport.patch<CommentThread>(`/scenes/${ref.sceneId}/comments/${thread.id}`, { requestId: requestId(), expectedRevision: thread.revision, ...input }, config(ref))).data; },
    async deleteMessage(ref, threadId, messageId, revision) { await transport.delete(`/scenes/${ref.sceneId}/comments/${threadId}/messages/${messageId}`, { ...config(ref), data: { requestId: requestId(), expectedRevision: revision } }); },
  };
}

export async function listNotifications(cursor = '', limit = 50): Promise<NotificationPage> {
  return (await api.get<NotificationPage>(`/notifications?limit=${limit}${cursor ? `&cursor=${encodeURIComponent(cursor)}` : ''}`)).data;
}

export async function markNotificationRead(id: string): Promise<void> {
  await api.post(`/notifications/${id}/read`, { requestId: requestId() });
}
