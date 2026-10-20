import { randomUUID } from 'node:crypto';
import { expect, test } from './fixtures';
import { draw, elements, openScene, pending } from './browser';

const active = async (page: import('@playwright/test').Page) =>
  (await elements(page)).filter((element: any) => !element.isDeleted);

test('image restore, remote sequence, undo reset and offline replay use one document generation', async ({ browser, stack }) => {
  const scene = await stack.scene('Enterprise versions');
  await stack.pool.query('UPDATE workspace_feature_policies SET versions_enabled=true WHERE workspace_id=$1', [stack.owner.workspaceId]);
  const ownerContext = await browser.newContext();
  const editorContext = await browser.newContext();
  try {
    const owner = await ownerContext.newPage();
    await openScene(owner, stack, scene);
    const box = await owner.locator('canvas').first().boundingBox();
    await owner.mouse.click(box!.x + 300, box!.y + 200);
    await owner.evaluate(async () => {
      const canvas = document.createElement('canvas');
      canvas.width = 32;
      canvas.height = 24;
      canvas.getContext('2d')!.fillRect(0, 0, 32, 24);
      const blob = await new Promise<Blob>((resolve) => canvas.toBlob((value) => resolve(value!), 'image/png'));
      const data = new DataTransfer();
      data.items.add(new File([blob], 'restore.png', { type: 'image/png' }));
      document.dispatchEvent(new ClipboardEvent('paste', { clipboardData: data, bubbles: true }));
    });
    await expect.poll(async () => (await active(owner)).filter((element: any) => element.type === 'image').length, { timeout: 20_000 }).toBe(1);
    await expect(owner.getByRole('status')).toHaveText('Saved', { timeout: 20_000 });
    const before = (await stack.pool.query('SELECT document_id,sequence::text FROM collab_documents WHERE scene_id=$1', [scene.id])).rows[0];
    const version = await stack.json(`/scenes/${scene.id}/versions`, stack.owner.token, stack.owner.workspaceId, 'POST', {
      requestId: randomUUID(),
      expectedSequence: before.sequence,
      label: 'One image',
      pinned: true,
    });

    const editor = await editorContext.newPage();
    await openScene(editor, stack, scene, stack.editor);
    await draw(owner, 520);
    await expect(owner.getByRole('status')).toHaveText('Saved');
    await expect.poll(async () => (await active(editor)).length).toBe(2);
    await editor.getByRole('button', { name: 'Open history' }).click();
    const editorVersion = editor.locator('article').filter({ hasText: 'One image' });
    await editorVersion.getByRole('button', { name: 'Compare version' }).click();
    await expect(editorVersion.getByText('1 added')).toBeVisible();
    await editor.keyboard.press('Escape');

    const current = (await stack.pool.query('SELECT sequence::text FROM collab_documents WHERE scene_id=$1', [scene.id])).rows[0].sequence;
    expect((await stack.request(`/scenes/${scene.id}/versions/${version.id}/restore`, stack.viewer.token, stack.owner.workspaceId, 'POST', {
      requestId: randomUUID(),
      expectedSequence: current,
    })).status).toBe(403);
    await editorContext.setOffline(true);
    await editor.evaluate(() => window.dispatchEvent(new Event('offline')));
    await expect(editor.getByRole('status')).toContainText('Offline');
    await draw(editor, 720);
    await expect.poll(async () => (await pending(editor)).length).toBeGreaterThan(0);

    await owner.getByRole('button', { name: 'Open history' }).click();
    const ownerVersion = owner.locator('article').filter({ hasText: 'One image' });
    await ownerVersion.getByRole('button', { name: 'Restore version' }).click();
    await owner.getByRole('button', { name: 'Confirm restore' }).click();
    await expect.poll(async () => (await active(owner)).length).toBe(1);
    expect((await active(owner))[0].type).toBe('image');
    await owner.keyboard.press('Escape');
    await owner.getByRole('button', { name: 'More scene actions' }).click();
    await expect(owner.getByRole('button', { name: 'Undo my edit' })).toBeDisabled();

    await editorContext.setOffline(false);
    await editor.evaluate(() => window.dispatchEvent(new Event('online')));
    await expect(editor.getByRole('status')).toHaveText('Saved', { timeout: 20_000 });
    await expect.poll(async () => (await active(owner)).length).toBe(2);
    expect((await active(owner)).some((element: any) => element.type === 'image')).toBe(true);
    const after = (await stack.pool.query('SELECT document_id FROM collab_documents WHERE scene_id=$1', [scene.id])).rows[0];
    expect(after.document_id).toBe(before.document_id);
    const recovery = await stack.pool.query("SELECT count(*) FROM scene_versions WHERE scene_id=$1 AND label='Before restore' AND pinned", [scene.id]);
    expect(recovery.rows[0].count).toBe('1');
  } finally {
    await ownerContext.close();
    await editorContext.close();
  }
});
