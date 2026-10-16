import type { Page } from '@playwright/test';
// Dashboard tests exercise the explicitly disabled v1 migration path. B3 live
// tests separately exercise genuine WebSocket and PostgreSQL collaboration.
export async function legacyCollabFixture(page: Page) {
  await page.route('**/api/v1/scenes/*/room-token', route => route.fulfill({ json: { token: 'fixture-room-token', expiresAt: new Date(Date.now() + 600000).toISOString() } }));
  await page.routeWebSocket('**/collab', socket => { socket.onMessage(() => socket.send(JSON.stringify({ type: 'error', requestId: 'auth', payload: { code: 'COLLAB_DISABLED', retryable: false } }))); });
}
