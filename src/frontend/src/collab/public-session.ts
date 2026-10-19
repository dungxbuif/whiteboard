import type { SceneSession } from '@/lib/collaboration-api';
const key = (token: string) => `devtool:guest-session:${token}`;
export function saveGuestSession(token: string, session: SceneSession, title: string) { sessionStorage.setItem(key(token), JSON.stringify({ session, title })); }
export function clearGuestSession(token: string) { sessionStorage.removeItem(key(token)); }
export function readGuestSession(token: string): { session: SceneSession; title: string } | null {
  try {
    const raw = sessionStorage.getItem(key(token)); if (!raw) return null;
    const value = JSON.parse(raw);
    // Retain expired identity metadata: the canvas can expose recovery while renewal fails closed.
    if (typeof value.session?.token !== 'string' || typeof value.session?.principalId !== 'string' || typeof value.title !== 'string') return null;
    if (Date.parse(value.session.expiresAt) <= Date.now()) { value.session.token = ''; sessionStorage.setItem(key(token), JSON.stringify(value)); }
    return value;
  } catch { return null; }
}
