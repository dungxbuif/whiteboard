import api from '@/lib/api';
import { publicApi } from '@/lib/public-api';
export type WorkspaceRole = 'owner' | 'editor' | 'viewer';
export type Membership = { workspaceId: string; userId: string; role: WorkspaceRole };
export type Invitation = { id: string; email: string; role: 'editor' | 'viewer'; expiresAt: string };
export type ShareLink = { id: string; access: 'view' | 'edit'; active: boolean; expiresAt: string | null };
export type SceneSession = { token: string; expiresAt: string; workspaceId: string; sceneId: string; access: 'view' | 'edit'; principalId: string; displayName: string };
export type PublicSessionInput = { token: string; password?: string; displayName?: string };
export const workspacePath = (id: string) => `/workspaces/${encodeURIComponent(id)}`;
export const sceneHeaders = (id: string) => ({ headers: { 'X-Workspace-ID': id } });
export async function acceptInvite(token: string): Promise<Membership> { return (await api.post<Membership>('/invitations/accept', { token })).data; }
export async function registerInvite(token: string, email: string, name: string, password: string) { await publicApi.post('/invitations/register', { token, email, name, password }); }
export async function createPublicSession(input: PublicSessionInput): Promise<SceneSession> { return (await publicApi.post<SceneSession>('/public/scene-session', input)).data; }
export async function renewPublicRoom(session: SceneSession): Promise<{ token: string; expiresAt: string; sceneSession?: SceneSession }> {
  return (await publicApi.post(`/scenes/${encodeURIComponent(session.sceneId)}/room-token`, {}, { headers: { Authorization: `Bearer ${session.token}`, 'X-Workspace-ID': session.workspaceId } })).data;
}
export function accessError(error: unknown, fallback: string) {
  const code = (error as { response?: { data?: { code?: string } } })?.response?.data?.code;
  const messages: Record<string, string> = { INVITATION_EMAIL_MISMATCH: 'Sign in with the invited email address, then try again.', INVITATION_EXPIRED: 'This invitation expired. Ask the owner for a new link.', INVITATION_INVALID: 'This invitation is unavailable or already used. Ask the owner for a new link.', REGISTRATION_REJECTED: 'Could not register. Sign in if you already have an account, or ask for a new invitation.', SHARE_PASSWORD_REQUIRED: 'Enter the correct share password to continue.' };
  return code && messages[code] || fallback;
}
