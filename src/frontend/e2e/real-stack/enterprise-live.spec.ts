import { test, expect } from './fixtures';
import { openScene } from './browser';

test('live presentation admits, denies, removes and transfers on the existing room socket', async ({ browser, stack }) => {
  const scene = await stack.scene();
  const contexts = await Promise.all([
    browser.newContext({ permissions: ['clipboard-read', 'clipboard-write'] }),
    browser.newContext(),
    browser.newContext({ viewport: { width: 390, height: 844 } }),
    browser.newContext(),
  ]);
  try {
    const [host, first, second, editor] = await Promise.all(contexts.map(context => context.newPage()));
    await Promise.all([openScene(host, stack, scene), openScene(editor, stack, scene, stack.editor)]);

    await host.getByRole('button', { name: 'Start live presentation' }).click();
    await expect(host.getByRole('region', { name: 'Live presentation host controls' })).toBeVisible();
    await host.getByRole('button', { name: 'Copy invite' }).click();
    const invite = await host.evaluate(() => navigator.clipboard.readText());
    expect(invite).toMatch(new RegExp('^' + stack.baseURL.replace(/[.*+?^${}()|[\]\\]/g, '\\$&') + '/present/'));

    await Promise.all([first.goto(invite), second.goto(invite)]);
    await first.getByLabel('Display name').fill('First guest');
    await first.getByRole('button', { name: 'Join and wait' }).click();
    await second.getByLabel('Display name').fill('Second guest');
    await second.getByRole('button', { name: 'Join and wait' }).click();
    await expect(first.getByText('Waiting for the host')).toBeVisible();
    await expect(second.getByText('Waiting for the host')).toBeVisible();
    await expect(second.locator('canvas')).toHaveCount(0);

    await host.getByRole('button', { name: 'Admit First guest' }).click();
    await expect(first.locator('canvas').first()).toBeVisible({ timeout: 30_000 });
    await expect(first.getByRole('button', { name: 'Leave' })).toBeVisible();
    await first.getByRole('button', { name: 'React 👍' }).click();
    await expect(host.getByText('👍')).toBeVisible();

    await host.getByRole('button', { name: 'Remove First guest' }).click();
    await expect(first.getByText('You were removed')).toBeVisible();
    await host.getByRole('button', { name: 'Deny Second guest' }).click();
    await expect(second.getByText('You were removed')).toBeVisible();

    await host.getByRole('button', { name: 'editor', exact: true }).click();
    await expect(editor.getByRole('region', { name: 'Live presentation host controls' })).toBeVisible();
    await expect(host.getByRole('region', { name: 'Live presentation host controls' })).toHaveCount(0);
  } finally {
    await Promise.all(contexts.map(context => context.close()));
  }
});
