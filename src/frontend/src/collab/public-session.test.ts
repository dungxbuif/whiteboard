import { it, expect } from 'vitest';
import { readGuestSession, saveGuestSession, clearGuestSession } from './public-session';
it('collab guest identity survives same-tab reload and stays separate from account credentials', () => {
  localStorage.setItem('auth_token', 'login');
  const session = { token: 'guest', expiresAt: new Date(Date.now() + 600000).toISOString(), principalId: 'guest:stable', workspaceId: 'w', sceneId: 's', access: 'edit' as const, displayName: 'Guest' };
  saveGuestSession('link', session, 'Scene'); expect(readGuestSession('link')?.session).toEqual(session);
  expect(readGuestSession('another-link')).toBeNull(); expect(localStorage.getItem('auth_token')).toBe('login');
  saveGuestSession('link', { ...session, expiresAt: '2000-01-01' }, 'Scene');
  expect(readGuestSession('link')?.session).toMatchObject({ token: '', principalId: 'guest:stable' });
  clearGuestSession('link'); expect(readGuestSession('link')).toBeNull();
});
