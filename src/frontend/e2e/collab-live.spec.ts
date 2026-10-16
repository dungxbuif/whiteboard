import { test, expect, type Page } from '@playwright/test';
import { spawn } from 'node:child_process';
import { mkdtempSync, existsSync, readFileSync, writeFileSync, rmSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { fileURLToPath } from 'node:url';
import { Pool } from 'pg';
import { Authority } from '../../collab/src/auth';
import { Persistence } from '../../collab/src/persistence';
import { startServer } from '../../collab/src/server';

async function until(check: () => boolean | Promise<boolean>, timeout = 15000) { const end = Date.now() + timeout; while (!await check()) { if (Date.now() > end) throw new Error('Live fixture timeout'); await new Promise(r => setTimeout(r, 25)); } }
// HTTP shell metadata is fixture data; the credential was minted by the real Go
// fixture. Admission/authority, WebSocket bytes, acknowledgements and storage are real.
test('native canvas edits converge through real Go authority, Node and PostgreSQL; offline reload replays durable work', async ({ browser }, testInfo) => {
  test.setTimeout(85000);
  const dir = mkdtempSync(join(tmpdir(), 'b3-browser-')), ready = join(dir, 'ready.json');
  const db = process.env.ACCESS_TEST_DATABASE_URL ?? 'postgres://test:test@127.0.0.1:5432/whiteboard_test?sslmode=disable';
  const child = spawn('go', ['test', './internal/repository/postgres', '-run', '^TestAgentLiveFixture$', '-count=1', '-timeout=100s'], { cwd: fileURLToPath(new URL('../../backend/', import.meta.url)), env: { ...process.env, PATH: (process.env.PATH ?? '') + ':/opt/homebrew/bin:/usr/local/go/bin', ACCESS_TEST_DATABASE_URL: db, D3_FIXTURE_READY: ready, D3_COLLAB_URL: 'http://127.0.0.1:1' } });
  let logs = ''; child.stdout.on('data', b => logs += b); child.stderr.on('data', b => logs += b);
  let runtime: Awaited<ReturnType<typeof startServer>> | undefined; let pool: Pool | undefined;
  const a = await browser.newContext(), b = await browser.newContext();
  try {
    await until(() => { if (child.exitCode !== null) throw new Error(logs); return existsSync(ready); }, 30000);
    const fixture = JSON.parse(readFileSync(ready, 'utf8'));
    const claims = JSON.parse(Buffer.from(fixture.roomToken.split('.')[1], 'base64url').toString());
    pool = new Pool({ connectionString: db, options: '-c search_path=' + fixture.schema + ',public', application_name: fixture.schema, max: 8 });
    runtime = await startServer(new Persistence(pool, new Authority({ roomSecret: 'r'.repeat(32), grantSecret: 'g'.repeat(32), serviceSecret: 's'.repeat(32), url: fixture.api }), { promoteV2: true }), { host: '127.0.0.1', port: 0, internalHost: '127.0.0.1', internalPort: 0, internalSecret: 'i'.repeat(32), allowedOrigins: ['http://localhost:5173'], updatesPerMinute: 120, maxBufferedBytes: 4194304 });
    const workspace = { id: fixture.ref.workspaceId, name: 'Live workspace', ownerId: claims.principalId, role: 'owner' };
    const scene = { id: fixture.ref.sceneId, workspaceId: workspace.id, title: 'Live collaboration', access: 'edit', owner: true, writerMode: 'v2', content: { elements: [], appState: {}, files: {} }, contentVersion: 1 };
    const setup = async (page: Page) => {
      await page.addInitScript(() => localStorage.setItem('auth_token', 'shell-fixture-login'));
      await page.route('**/api/v1/**', route => {
        const path = new URL(route.request().url()).pathname.replace(/\/$/, '');
        if (path.endsWith('/room-token')) return route.fulfill({ json: { token: fixture.roomToken, expiresAt: new Date(claims.exp * 1000).toISOString() } });
        if (path.endsWith('/users/me')) return route.fulfill({ json: { id: claims.principalId, email: 'live@example.com', name: 'Live user' } });
        if (path.endsWith('/workspaces')) return route.fulfill({ json: [workspace] });
        if (path.includes('/folders')) return route.fulfill({ json: [] });
        if (path.endsWith('/scenes')) return route.fulfill({ json: [scene] });
        if (path.endsWith('/scenes/' + scene.id)) return route.fulfill({ json: scene });
        return route.fulfill({ status: 404, json: { code: 'NOT_FOUND' } });
      });
      await page.addInitScript(({ target }) => {
        const NativeSocket = window.WebSocket;
        window.WebSocket = class extends NativeSocket { constructor(url: string | URL, protocols?: string | string[]) { super(String(url).endsWith('/collab') ? target : url, protocols); } };
      }, { target: runtime!.url + '/collab' });
      await page.goto('/w/' + workspace.id + '/scenes/' + scene.id);
      await expect(page.getByRole('status')).toHaveText('Saved', { timeout: 15000 });
      await expect(page.locator('canvas').first()).toBeVisible();
    };
    const left = await a.newPage(), right = await b.newPage();
    const errors: string[] = []; left.on('pageerror', e => { errors.push(e.message); console.log('browser error:', e.message); }); right.on('pageerror', e => { errors.push(e.message); console.log('browser error:', e.message); });
    await setup(left); await setup(right);
    const elements = (page: Page) => page.evaluate(async () => { const store = await import('/src/store/scene-store.ts' as string); return store.default.getState().activeScene.content.elements; });
    const draw = async (page: Page, x: number) => {
      await page.getByTitle('Rectangle — R or 2', { exact: true }).click(); await expect(page.getByTitle('Rectangle — R or 2', { exact: true }).locator('input')).toBeChecked(); const box = await page.locator('canvas').first().boundingBox(); if (!box) throw new Error('Missing native canvas');
      await page.mouse.move(box.x + x, box.y + 160); await page.mouse.down(); await page.mouse.move(box.x + x + 120, box.y + 250, { steps: 5 }); await page.mouse.up();
    };
    await left.bringToFront();
    await draw(left, 340);
    await expect.poll(async () => (await elements(left)).length).toBe(1);
    await expect(left.getByRole('status')).toHaveText('Saved');
    await expect.poll(async () => (await elements(right)).length).toBe(1);
    expect(await elements(right)).toEqual(await elements(left));
    await expect(right.getByRole('button', { name: 'Undo my edit' })).toBeDisabled();
    const committed = (await pool.query('SELECT sequence FROM collab_documents')).rows[0].sequence;
    expect(BigInt(committed)).toBeGreaterThan(0n);
    // Browser networking is actually cut; no mocked protocol frames are injected.
    await a.setOffline(true);
    await left.evaluate(() => window.dispatchEvent(new Event('offline')));
    await expect(left.getByRole('status')).toContainText('Offline');
    await draw(left, 450);
    await expect.poll(async () => (await elements(left)).length).toBe(2);
    await expect.poll(async () => left.evaluate(async () => { const module = await import('/src/collab/offline-store.ts' as string); const s = await import('/src/store/scene-store.ts' as string); const u = await import('/src/store/auth-store.ts' as string); const scene = s.default.getState().activeScene; const scopes = await module.offlineStore.snapshots(u.default.getState().user.id, scene.workspaceId, scene.id); return scopes.length ? (await module.offlineStore.pending(scopes[0].scope)).length : 0; })).toBeGreaterThan(0);
    expect((await pool.query('SELECT sequence FROM collab_documents')).rows[0].sequence).toBe(committed);
    await a.setOffline(false); await left.reload();
    await expect(left.getByRole('status')).toHaveText('Saved');
    await expect.poll(async () => (await elements(right)).length).toBe(2);
    expect(await elements(right)).toEqual(await elements(left));
    await expect(left.getByRole('button', { name: 'Undo my edit' })).toBeDisabled();
    for (const element of await elements(left)) expect(element.boundElements).toBeNull();
    const afterReplay = (await pool.query('SELECT sequence FROM collab_documents')).rows[0].sequence;
    // Beyond the 600ms send interval: renderer restoration must not write back.
    await left.waitForTimeout(1000);
    expect((await pool.query('SELECT sequence FROM collab_documents')).rows[0].sequence).toBe(afterReplay);
    await left.screenshot({ path: testInfo.outputPath('native-canvas-two-elements.png') });
    expect(errors).toEqual([]);
  } finally {
    await Promise.allSettled([a.close(), b.close()]); await Promise.allSettled([runtime?.close(), pool?.end()]); writeFileSync(join(dir, 'stop'), 'stop'); await until(() => child.exitCode !== null); rmSync(dir, { recursive: true, force: true });
  }
  expect(child.exitCode, logs).toBe(0);
});
