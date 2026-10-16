import { test, expect, type Page } from '@playwright/test';

const workspace = { id: '2c7f253e-b06b-4ad4-988c-c4590fbc964a', publicId: 'workspace-alias', name: 'Access workspace', ownerId: 'owner', role: 'owner', createdAt: '2026-10-16T00:00:00Z' };
const scene = { id: '54ea6439-6734-46db-ad60-2d7ff26ce108', publicId: 'scene-alias', workspaceId: workspace.id, folderId: null, title: 'Access scene', access: 'edit', owner: true, content: { elements: [], appState: {}, files: {} }, contentVersion: 1, isStarred: false, createdAt: '', updatedAt: '' };
const token = 'a'.repeat(43);
async function fixture(page: Page, role = 'owner', signedIn = true) {
  if (signedIn) await page.addInitScript(() => localStorage.setItem('auth_token', 'account-token'));
  await page.route('**/api/v1/**', async route => {
    const path = new URL(route.request().url()).pathname.replace(/\/$/, "");
    if (path.endsWith('/users/me')) return route.fulfill({ json: { id: 'owner', email: 'invited@example.com', name: 'Recipient' } });
    if (path.endsWith('/auth/login')) return route.fulfill({ json: { token: 'account-token' } });
    if (path.endsWith('/workspaces')) return route.fulfill({ json: [{ ...workspace, role }] });
    if (path.includes('/folders')) return route.fulfill({ json: [] });
    if (path.endsWith('/members')) return route.fulfill({ json: [{ workspaceId: workspace.id, userId: 'owner', role: 'owner' }, { workspaceId: workspace.id, userId: 'member', role: 'editor' }] });
    if (path.endsWith('/invitations')) return route.fulfill({ json: [] });
    if (path.endsWith('/agent-keys')) return route.fulfill({ json: [] });
    if (path.endsWith('/share-links')) return route.fulfill({ json: [] });
    if (/\/scenes\/[^/]+/.test(path)) return route.fulfill({ json: { ...scene, owner: role === 'owner', access: role === 'viewer' ? 'view' : 'edit' } });
    if (path.endsWith('/scenes')) return route.fulfill({ json: [{ ...scene, owner: role === 'owner', access: role === 'viewer' ? 'view' : 'edit' }] });
    return route.fulfill({ status: 404, json: { code: 'NOT_FOUND' } });
  });
}

test('owner creates a one-time invitation and manages member role and scene override', async ({ page, context }) => {
  await context.grantPermissions(['clipboard-read', 'clipboard-write']);
  await fixture(page);
  const calls: string[] = [];
  await page.route('**/workspaces/*/invitations', async route => route.fulfill({ json: route.request().method() === 'POST' ? { invitation: { id: 'invite', email: 'invited@example.com', role: 'viewer', expiresAt: '2026-10-23T00:00:00Z' }, inviteUrl: '/invite/' + token } : [] }));
  await page.route('**/members/member', async route => { calls.push(route.request().method() + ':' + route.request().url()); await route.fulfill({ status: 204 }); });
  await page.goto('/w/workspace-alias/settings/members');
  await expect(page.getByRole('heading', { name: 'Members', exact: true })).toBeVisible();
  await page.getByRole('button', { name: 'Invite member' }).click();
  await page.getByLabel('Invited email').fill('invited@example.com');
  await page.getByLabel('Invitation role').selectOption('viewer');
  await page.getByRole('button', { name: 'Create invitation' }).click();
  await expect(page.getByLabel('Invitation link')).toHaveValue(new RegExp('/invite/' + token));
  await page.getByRole('button', { name: 'Copy invitation' }).click();
  expect(await page.evaluate(() => navigator.clipboard.readText())).toContain('/invite/' + token);
  await page.getByRole('button', { name: 'Close', exact: true }).click();
  await page.getByLabel('Role for member').selectOption('viewer');
  await expect.poll(() => calls.some(call => call.startsWith('PATCH:'))).toBe(true);
  await page.getByLabel('Scene permission').selectOption(scene.id);
  await page.getByLabel('Override for member').selectOption('viewer');
  await expect.poll(() => calls.some(call => call.startsWith('PUT:'))).toBe(true);
});

test('invite survives sign-in and consumes only after explicit acceptance', async ({ page }) => {
  await fixture(page, 'viewer', false);
  let accepted = false;
  await page.route('**/invitations/accept', async route => { expect(route.request().postDataJSON()).toEqual({ token }); accepted = true; await route.fulfill({ json: { workspaceId: workspace.id, userId: 'owner', role: 'viewer' } }); });
  await page.goto('/invite/' + token);
  await page.getByLabel('Email', { exact: true }).fill('invited@example.com');
  await page.getByLabel('Password', { exact: true }).fill('password123');
  await page.getByRole('button', { name: 'Sign in', exact: true }).click();
  await expect(page.getByRole('button', { name: 'Accept invitation' })).toBeVisible();
  expect(accepted).toBe(false);
  await page.getByRole('button', { name: 'Accept invitation' }).click();
  await expect(page).toHaveURL('/');
  expect(accepted).toBe(true);
  expect(await page.evaluate(() => JSON.stringify(localStorage))).not.toContain(token);
});

for (const [code, message] of [['INVITATION_EMAIL_MISMATCH', 'invited email'], ['INVITATION_EXPIRED', 'expired']]) {
  test('invite rejects ' + code, async ({ page }) => {
    await fixture(page);
    await page.route('**/invitations/accept', route => route.fulfill({ status: code === 'INVITATION_EXPIRED' ? 410 : 403, json: { code } }));
    await page.goto('/invite/' + token);
    await page.getByRole('button', { name: 'Accept invitation' }).click();
    await expect(page.getByRole('alert')).toContainText(message);
    await expect(page).toHaveURL('/invite/' + token);
  });
}

test('viewer sees reading controls and cannot enter owner settings', async ({ page }) => {
  await fixture(page, 'viewer');
  await page.goto('/');
  await expect(page.getByRole('heading', { name: workspace.name })).toBeVisible();
  await expect(page.getByRole('button', { name: 'New scene' })).toHaveCount(0);
  await expect(page.getByRole('button', { name: 'New folder' })).toHaveCount(0);
  await expect(page.getByRole('button', { name: 'Trash', exact: true })).toHaveCount(0);
  await expect(page.getByRole('button', { name: 'Actions for Access scene' })).toHaveCount(0);
  await expect(page.getByRole('link', { name: 'Members', exact: true })).toHaveCount(0);
  await page.goto('/w/workspace-alias/settings/members');
  await expect(page.getByRole('alert')).toContainText('Only workspace owners');
});

test('alias and UUID scene routes survive reload back and forward; home closes scene', async ({ page }) => {
  await fixture(page);
  await page.goto('/w/workspace-alias/scenes/scene-alias');
  await expect(page.getByRole('heading', { name: scene.title })).toBeVisible();
  await page.reload();
  await expect(page.getByRole('heading', { name: scene.title })).toBeVisible();
  await page.getByRole('button', { name: 'Back to dashboard' }).click();
  await expect(page.getByRole('heading', { name: workspace.name })).toBeVisible();
  await page.goBack();
  await expect(page.getByRole('heading', { name: scene.title })).toBeVisible();
  await page.goForward();
  await expect(page.getByRole('heading', { name: workspace.name })).toBeVisible();
  await page.goto('/w/' + workspace.id + '/scenes/' + scene.id);
  await expect(page.getByRole('heading', { name: scene.title })).toBeVisible();
});


test('registration with invitation keeps the account login token separate', async ({ page }) => {
  await fixture(page, 'viewer', false);
  let registration: unknown;
  await page.route('**/invitations/register', async route => { registration = route.request().postDataJSON(); expect(route.request().headers().authorization).toBeUndefined(); await route.fulfill({ status: 201, json: { id: 'new-user' } }); });
  await page.goto('/invite/' + token);
  await page.getByRole('button', { name: 'Create account with invitation' }).click();
  await page.getByLabel('Invited email').fill('invited@example.com');
  await page.getByLabel('Display name').fill('Recipient');
  await page.getByLabel('Password', { exact: true }).fill('password123');
  await page.getByRole('button', { name: 'Create account and join' }).click();
  await expect(page).toHaveURL('/');
  expect(registration).toEqual({ token, email: 'invited@example.com', name: 'Recipient', password: 'password123' });
  expect(await page.evaluate(() => localStorage.getItem('auth_token'))).toBe('account-token');
});

test('owner creates independent view and edit links with password expiry and revoke', async ({ page }) => {
  await fixture(page);
  let links: Array<{ id: string; access: string; active: boolean; expiresAt: string | null }> = [];
  await page.route('**/api/v1/scenes/*/share-links**', async route => {
    const method = route.request().method();
    if (method === 'POST') { const input = route.request().postDataJSON(); expect(input.password).toBe('secret-pass'); expect(input.expiresAt).toMatch(/^2030-/); const link = { id: input.access + '-id', access: input.access, active: true, expiresAt: input.expiresAt }; links.push(link); return route.fulfill({ status: 201, json: { ...link, token: input.access + '-token' } }); }
    if (method === 'DELETE') { const id = new URL(route.request().url()).pathname.split('/').pop(); links = links.map(link => link.id === id ? { ...link, active: false } : link); return route.fulfill({ status: 204 }); }
    return route.fulfill({ json: links });
  });
  await page.goto('/');
  await page.getByRole('button', { name: 'Actions for Access scene' }).click();
  await page.getByRole('menuitem', { name: 'Share', exact: true }).click();
  for (const access of ['view', 'edit']) {
    await page.getByLabel(access + ' link password').fill('secret-pass');
    await page.getByLabel(access + ' link expiry').fill('2030-01-01T12:00');
    await page.getByRole('button', { name: 'Create ' + access + ' link' }).click();
    await expect(page.getByLabel(access + ' link URL')).toHaveValue(new RegExp('/p/' + access + '-token'));
  }
  await page.getByRole('button', { name: 'Revoke view link' }).click();
  await expect(page.getByText('view · Revoked', { exact: false })).toBeVisible();
  await expect(page.getByRole('button', { name: 'Revoke edit link' })).toBeVisible();
});

test('public edit retains same-tab identity across reload and clears only revoked credentials', async ({ page }) => {
  let exchanges = 0, renewals = 0, revoked = false;
  await page.route('**/api/v1/scenes/*/room-token', route => { renewals++; return route.fulfill({ status: revoked ? 403 : 503, json: { code: revoked ? 'FORBIDDEN' : 'UNAVAILABLE' } }); });
  await page.addInitScript(() => localStorage.setItem('auth_token', 'private-login-token'));
  await page.route('**/public/scene-session', async route => {
    expect(route.request().headers().authorization).toBeUndefined();
    exchanges++;
    const input = route.request().postDataJSON();
    if (input.password !== 'secret-pass') return route.fulfill({ status: 401, json: { code: 'SHARE_PASSWORD_REQUIRED' } });
    if (!input.displayName) return route.fulfill({ status: 400, json: { code: 'INVALID_INPUT' } });
    return route.fulfill({ json: { token: 'guest-credential', expiresAt: '2030-01-01T00:00:00Z', workspaceId: workspace.id, sceneId: scene.id, access: 'edit', principalId: 'guest:id', displayName: input.displayName } });
  });
  await page.route('**/public/scenes/edit-token', async route => { expect(route.request().method()).toBe('POST'); const input = route.request().postDataJSON(); if (input.password !== 'secret-pass') return route.fulfill({ status: 401, json: { code: 'SHARE_PASSWORD_REQUIRED' } }); await route.fulfill({ json: scene }); });
  await page.goto('/p/edit-token');
  await page.getByLabel('Share password').fill('secret-pass');
  await page.getByRole('button', { name: 'Open shared scene' }).click();
  await page.getByLabel('Display name').fill('Guest artist');
  await page.getByRole('button', { name: 'Open shared scene' }).click();
  await expect(page.getByRole('heading', { name: scene.title })).toBeVisible();
  await expect(page.getByText('Guest artist', { exact: false })).toBeVisible();
  expect(await page.evaluate(() => localStorage.getItem('auth_token'))).toBe('private-login-token');
  expect(await page.evaluate(() => JSON.stringify(localStorage))).not.toContain('guest-credential');
  await expect.poll(() => renewals).toBeGreaterThan(0);
  const session = () => page.evaluate(() => JSON.parse(sessionStorage.getItem('devtool:guest-session:edit-token')!).session);
  expect((await session()).token).toBe('guest-credential');
  const before = exchanges; await page.reload();
  await expect(page.getByText('Guest artist', { exact: false })).toBeVisible();
  expect(exchanges).toBe(before); expect((await session()).principalId).toBe('guest:id');
  revoked = true; await page.getByRole('button', { name: 'Retry connection / uploads' }).click();
  await expect(page.getByRole('status')).toContainText('Access revoked');
  expect((await session()).token).toBe(''); expect((await session()).principalId).toBe('guest:id');
  expect(await page.evaluate(() => localStorage.getItem('auth_token'))).toBe('private-login-token');
});


const secondWorkspace = { ...workspace, id: '0f00f7e8-4e36-49a4-b1c6-88ff5e413533', publicId: 'second-workspace', name: 'Second workspace' };
const navigationFolder = { id: 'planning-folder', workspaceId: workspace.id, parentId: null, name: 'Planning', path: '/planning/', depth: 0, isStarred: false, createdAt: '', updatedAt: '' };
async function navigationFixture(page: Page) {
  await fixture(page);
  await page.route('**/api/v1/workspaces', route => route.fulfill({ json: [workspace, secondWorkspace] }));
  await page.route('**/api/v1/folders**', route => route.fulfill({ json: route.request().headers()['x-workspace-id'] === workspace.id ? [navigationFolder] : [] }));
  await page.route('**/api/v1/scenes', route => {
    const url = new URL(route.request().url());
    return route.fulfill({ json: [{ ...scene, title: url.searchParams.get('folderId') ? 'Folder drawing' : 'Root drawing' }] });
  });
  await page.route('**/api/v1/scenes?**', route => route.fulfill({ json: [{ ...scene, title: 'Folder drawing' }] }));
  await page.route('**/api/v1/search**', route => {
    const query = new URL(route.request().url()).searchParams;
    const name = query.get('q') ? 'Found ' + query.get('q') : query.get('view') === 'starred' ? 'Starred drawing' : 'Trashed drawing';
    return route.fulfill({ json: [{ id: scene.id, workspaceId: workspace.id, type: 'scene', name, isStarred: query.get('view') === 'starred', updatedAt: '' }] });
  });
}

for (const section of ['members', 'agents']) {
  const heading = section === 'members' ? 'Members' : 'Agent keys';
  test('switching workspace from ' + section + ' updates route and survives history', async ({ page }) => {
    await navigationFixture(page);
    await page.goto('/w/workspace-alias/settings/' + section);
    await expect(page.getByRole('heading', { name: heading, exact: true })).toBeVisible();
    await page.getByRole('combobox').filter({ hasText: workspace.name }).click();
    await page.getByRole('menuitem', { name: secondWorkspace.name, exact: true }).click();
    await expect(page).toHaveURL('/w/' + secondWorkspace.publicId);
    await expect(page.getByRole('heading', { name: secondWorkspace.name, exact: true })).toBeVisible();
    await page.reload();
    await expect(page.getByRole('heading', { name: secondWorkspace.name, exact: true })).toBeVisible();
    await page.goBack();
    await expect(page.getByRole('heading', { name: heading, exact: true })).toBeVisible();
    await expect(page.getByRole('combobox').filter({ hasText: workspace.name })).toBeVisible();
    await page.goForward();
    await expect(page.getByRole('heading', { name: secondWorkspace.name, exact: true })).toBeVisible();
  });

  test('dashboard navigation leaves ' + section + ' preserving folder view and search', async ({ page }) => {
    await navigationFixture(page);
    const settings = '/w/workspace-alias/settings/' + section;
    for (const target of ['All files', 'Planning', 'Starred', 'Trash']) {
      await page.goto(settings);
      await expect(page.getByRole('heading', { name: heading, exact: true })).toBeVisible();
      await page.getByRole('navigation', { name: 'Folders', exact: true }).getByRole('button', { name: target, exact: true }).click();
      await expect(page).toHaveURL('/w/workspace-alias');
      await expect(page.getByRole('heading', { name: heading, exact: true })).toHaveCount(0);
      if (target === 'All files' || target === 'Planning') await expect(page.getByRole('region', { name: 'Scenes' })).toContainText(target === 'Planning' ? 'Folder drawing' : 'Root drawing');
      else await expect(page.getByRole('region', { name: target })).toContainText(target === 'Starred' ? 'Starred drawing' : 'Trashed drawing');
    }
    await page.goto(settings);
    await page.getByRole('textbox', { name: 'Search scenes and folders' }).fill('architecture');
    await expect(page).toHaveURL('/w/workspace-alias');
    await expect(page.getByRole('region', { name: 'Search results' })).toContainText('Found architecture');
    await expect(page.getByRole('textbox', { name: 'Search scenes and folders' })).toHaveValue('architecture');
  });
}

test('a delayed obsolete scene open cannot replace a newer route', async ({ page }) => {
  await fixture(page);
  const oldScene = { ...scene, id: 'old-scene', publicId: 'old-alias', title: 'Old scene' };
  const newScene = { ...scene, id: 'new-scene', publicId: 'new-alias', title: 'New scene' };
  await page.route('**/api/v1/scenes', route => route.fulfill({ json: [oldScene, newScene] }));
  let oldStarted = false;
  let releaseOld!: () => void;
  let oldCompleted!: () => void;
  const pending = new Promise<void>(resolve => { releaseOld = resolve; });
  const completed = new Promise<void>(resolve => { oldCompleted = resolve; });
  await page.route('**/api/v1/scenes/old-scene/', async route => { oldStarted = true; await pending; await route.fulfill({ json: oldScene }); oldCompleted(); });
  await page.route('**/api/v1/scenes/new-scene/', route => route.fulfill({ json: newScene }));
  await page.goto('/');
  await page.locator('article').filter({ hasText: 'Old scene' }).getByRole('button', { name: 'Open', exact: true }).click();
  await expect.poll(() => oldStarted).toBe(true);
  await page.goBack();
  await page.locator('article').filter({ hasText: 'New scene' }).getByRole('button', { name: 'Open', exact: true }).click();
  await expect(page.getByRole('heading', { name: 'New scene', exact: true })).toBeVisible();
  const oldResponse = page.waitForResponse(response => response.url().endsWith('/scenes/old-scene/'));
  releaseOld();
  await completed;
  await (await oldResponse).finished();
  await page.evaluate(async () => {
    await new Promise<void>(resolve => requestAnimationFrame(() => resolve()));
    await new Promise<void>(resolve => requestAnimationFrame(() => resolve()));
  });
  await expect(page.getByRole('heading', { name: 'New scene', exact: true })).toBeVisible();
  await expect(page).toHaveURL(new RegExp('/scenes/new-scene$'));
});
