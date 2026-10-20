import { expect, test } from './fixtures';
import { openScene } from './browser';

test('member and guest use the native comment panel without shrinking the canvas', async ({ browser, stack }) => {
  const scene = await stack.scene('Enterprise comments');
  await stack.pool.query('UPDATE workspace_feature_policies SET comments_enabled=true WHERE workspace_id=$1', [stack.owner.workspaceId]);
  const ownerContext = await browser.newContext({ viewport: { width: 1440, height: 900 } });
  const guestContext = await browser.newContext({ viewport: { width: 390, height: 844 } });
  try {
    const owner = await ownerContext.newPage();
    await openScene(owner, stack, scene);
    const before = await owner.getByTestId('scene-canvas-root').boundingBox();
    await owner.getByRole('button', { name: 'Open comments' }).click();
    await owner.getByLabel('New discussion').fill('Owner thread');
    await owner.getByRole('button', { name: 'Create discussion' }).click();
    await expect(owner.getByText('Owner thread')).toBeVisible();
    expect(await owner.getByTestId('scene-canvas-root').boundingBox()).toEqual(before);

    const link = await stack.json('/scenes/' + scene.id + '/share-links', stack.owner.token, stack.owner.workspaceId, 'POST', { access: 'view', allowComments: true });
    const guest = await guestContext.newPage();
    await guest.goto(stack.baseURL + '/p/' + link.token);
    await expect(guest.getByRole('status')).toHaveText('Read-only');
    await guest.getByRole('button', { name: 'Open comments' }).click();
    await guest.getByLabel('Reply').fill('<img src=x onerror=alert(1)> guest reply');
    await guest.getByRole('button', { name: 'Reply' }).click();
    await expect(guest.getByText('<img src=x onerror=alert(1)> guest reply')).toBeVisible();
    await expect(guest.locator('img[src="x"]')).toHaveCount(0);
    await guest.keyboard.press('Escape');
    await expect(guest.getByRole('dialog', { name: 'Comments' })).toHaveCount(0);

    await owner.getByRole('button', { name: 'Refresh' }).click();
    await expect(owner.getByText('<img src=x onerror=alert(1)> guest reply')).toBeVisible();
    await owner.getByRole('button', { name: 'Resolve' }).click();
    await expect(owner.getByRole('button', { name: 'Reopen' })).toBeVisible();
  } finally {
    await ownerContext.close();
    await guestContext.close();
  }
});
