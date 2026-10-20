import { randomUUID } from 'node:crypto';
import { expect, test } from './fixtures';
import { login } from './browser';

test('collections keep ACL, deep-link, mobile and scene-location behavior', async ({ browser, stack }) => {
  await stack.pool.query('UPDATE workspace_feature_policies SET collections_enabled=true WHERE workspace_id=$1', [stack.owner.workspaceId]);
  const launch = await stack.scene('Launch board');
  const review = await stack.scene('Review board');
  const before = await stack.pool.query('SELECT folder_id FROM scenes WHERE workspace_id=$1 AND id=$2', [stack.owner.workspaceId, launch.id]);

  const context = await browser.newContext({ viewport: { width: 390, height: 844 } });
  const viewerContext = await browser.newContext({ viewport: { width: 390, height: 844 } });
  try {
    const page = await context.newPage();
    await login(page, stack);
    await page.goto(`${stack.baseURL}/w/${stack.owner.workspaceId}`);
    await page.getByRole('button', { name: 'Open navigation' }).click();
    await page.getByRole('button', { name: 'New collection' }).click();
    await page.getByLabel('Collection name').fill('Launch plans');
    await page.getByLabel('Collection visibility').selectOption('workspace');
    await page.getByRole('form', { name: 'Create collection' }).press('Enter');
    const link = page.getByRole('link', { name: /Launch plans/ });
    await expect(link).toBeVisible();
    await page.getByLabel('Filter collections').fill('launch');
    await link.click();
    await expect(page).toHaveURL(/\/collections\/[0-9a-f-]+$/);
    await expect(page.getByRole('heading', { name: 'Launch plans' })).toBeVisible();
    const collectionId = page.url().split('/').pop()!;

    const created = await stack.json(`/workspaces/${stack.owner.workspaceId}/collections/${collectionId}`, stack.owner.token, stack.owner.workspaceId);
    let value = await stack.json(`/workspaces/${stack.owner.workspaceId}/collections/${collectionId}/scenes`, stack.owner.token, stack.owner.workspaceId, 'POST', { requestId: randomUUID(), expectedRevision: created.revision, sceneId: launch.id });
    value = await stack.json(`/workspaces/${stack.owner.workspaceId}/collections/${collectionId}/scenes`, stack.owner.token, stack.owner.workspaceId, 'POST', { requestId: randomUUID(), expectedRevision: value.revision, sceneId: review.id });
    await page.reload();
    await expect(page.getByRole('link', { name: 'Open Launch board' })).toBeVisible();
    await page.getByRole('button', { name: 'Remove Launch board' }).click();
    await expect(page.getByRole('link', { name: 'Open Launch board' })).toHaveCount(0);
    const after = await stack.pool.query('SELECT folder_id FROM scenes WHERE workspace_id=$1 AND id=$2', [stack.owner.workspaceId, launch.id]);
    expect(after.rows[0].folder_id).toBe(before.rows[0].folder_id);

    const privateCollection = await stack.json(`/workspaces/${stack.owner.workspaceId}/collections`, stack.editor.token, stack.owner.workspaceId, 'POST', { requestId: randomUUID(), name: 'Editor private', visibility: 'private' });
    const denied = await stack.request(`/workspaces/${stack.owner.workspaceId}/collections/${privateCollection.id}`, stack.owner.token, stack.owner.workspaceId);
    expect(denied.status).toBe(404);

    const viewer = await viewerContext.newPage();
    await login(viewer, stack, stack.viewer);
    await viewer.goto(`${stack.baseURL}/w/${stack.owner.workspaceId}/collections/${collectionId}`);
    await expect(viewer.getByRole('heading', { name: 'Launch plans' })).toBeVisible();
    await expect(viewer.getByRole('button', { name: 'Delete collection' })).toHaveCount(0);
    await expect(viewer.getByRole('button', { name: /Remove/ })).toHaveCount(0);
  } finally {
    await context.close();
    await viewerContext.close();
  }
});
