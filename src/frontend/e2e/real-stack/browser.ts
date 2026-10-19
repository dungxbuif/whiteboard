import { expect, type Page } from '@playwright/test';
import type { Account, RealStack } from './harness';
export async function login(page: Page, stack: RealStack, account = stack.owner) {
  await page.goto(stack.baseURL + '/');
  await page.getByLabel('Email', { exact: true }).fill(account.email);
  await page.getByLabel('Password', { exact: true }).fill(account.password);
  await page.getByRole('button', { name: 'Sign in', exact: true }).click();
  await expect.poll(() => page.evaluate(() => !!localStorage.getItem('auth_token'))).toBe(true);
}
export async function openScene(page: Page, stack: RealStack, scene: any, account: Account = stack.owner) {
  await login(page, stack, account);
  await page.goto(stack.baseURL + '/w/' + stack.owner.workspaceId + '/scenes/' + scene.id);
  await expect(page.getByRole('status')).toHaveText(account === stack.viewer ? 'Read-only' : 'Saved', { timeout: 30000 });
  await expect(page.locator('canvas').first()).toBeVisible();
  // Prime observation modules while online; offline assertions must not fetch Vite modules.
  await page.evaluate(async()=>{await Promise.all([import('/src/collab/offline-store.ts' as string),import('/src/store/scene-store.ts' as string),import('/src/store/auth-store.ts' as string)]);});
}
export const elements = (page: Page) => page.evaluate(async () => { const s = await import('/src/store/scene-store.ts' as string); return s.default.getState().activeScene.content.elements; });
export async function draw(page: Page, x = 300, type = 'Rectangle — R or 2') {
  await page.bringToFront();
  await page.getByTitle(type, { exact: true }).click();
  const box = await page.locator('canvas').first().boundingBox(); if (!box) throw new Error('Native canvas absent');
  await page.mouse.move(box.x + x, box.y + 150); await page.mouse.down(); await page.mouse.move(box.x + x + 100, box.y + 240, { steps: 4 }); await page.mouse.up();
}
export async function pending(page: Page) {
  return page.evaluate(async () => { const m = await import('/src/collab/offline-store.ts' as string); const s = await import('/src/store/scene-store.ts' as string); const a = await import('/src/store/auth-store.ts' as string); const scene = s.default.getState().activeScene; const scopes = await m.offlineStore.scopes(a.default.getState().user.id, scene.workspaceId, scene.id); return (await Promise.all(scopes.map((scope: any) => m.offlineStore.pending(scope)))).flat(); });
}
