import { randomUUID } from 'node:crypto';
import * as Y from 'yjs';
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
    await draw(owner, 520);
    await expect(owner.getByRole('status')).toHaveText('Saved');
    const before = (await stack.pool.query('SELECT document_id,sequence::text FROM collab_documents WHERE scene_id=$1', [scene.id])).rows[0];
    const version = await stack.json(`/scenes/${scene.id}/versions`, stack.owner.token, stack.owner.workspaceId, 'POST', {
      requestId: randomUUID(),
      expectedSequence: before.sequence,
      label: 'Before image',
      pinned: true,
    });
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

    const editor = await editorContext.newPage();
    await openScene(editor, stack, scene, stack.editor);
    await expect.poll(async () => (await active(editor)).length).toBe(2);
    await editor.getByRole('button', { name: 'Open history' }).click();
    const editorVersion = editor.locator('article').filter({ hasText: 'Before image' });
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
    const offlineImage=(await active(editor)).find((element:any)=>element.type==='image')!;
    const editorBox=await editor.locator('canvas').first().boundingBox();
    await editor.mouse.click(editorBox!.x+offlineImage.x+offlineImage.width/2,editorBox!.y+offlineImage.y+offlineImage.height/2);
    for(let step=0;step<8;step++)await editor.keyboard.press('ArrowRight');
    await expect.poll(async()=>((await active(editor)).find((element:any)=>element.type==='image') as any)?.x).toBeGreaterThan(offlineImage.x);
    await expect.poll(async () => (await pending(editor)).length).toBeGreaterThan(0);
    const queued=await pending(editor);expect(queued.every((row:any)=>!row.sent&&row.intent?.desired)).toBe(true);expect(queued.some((row:any)=>row.intent.desired.elements.find((element:any)=>element.id===offlineImage.id)?.isDeleted===false)).toBe(true);

    await owner.getByRole('button', { name: 'Open history' }).click();
    const ownerVersion = owner.locator('article').filter({ hasText: 'Before image' });
    await ownerVersion.getByRole('button', { name: 'Restore version' }).click();
    await owner.getByRole('button', { name: 'Confirm restore' }).click();
    await expect.poll(async () => (await active(owner)).length).toBe(1);
    expect((await active(owner))[0].type).toBe('rectangle');
    await owner.keyboard.press('Escape');
    await owner.getByRole('button', { name: 'More scene actions' }).click();
    await expect(owner.getByRole('button', { name: 'Undo my edit' })).toBeDisabled();

    await editorContext.setOffline(false);
    await editor.evaluate(() => window.dispatchEvent(new Event('online')));
    await expect(editor.getByRole('status')).toHaveText('Saved', { timeout: 20_000 });
    await expect.poll(async () => (await active(editor)).length, {timeout:20_000}).toBe(2);
    const replayRequest=(await stack.pool.query("SELECT request_id FROM collab_mutation_audit WHERE scene_id=$1 AND action='scene:update' ORDER BY sequence DESC LIMIT 1",[scene.id])).rows[0].request_id;expect(queued.map((row:any)=>row.requestId)).not.toContain(replayRequest);
    const durable=await stack.pool.query('SELECT snapshot,snapshot_sequence::text FROM collab_documents WHERE scene_id=$1',[scene.id]);const durableDoc=new Y.Doc({gc:false});Y.applyUpdate(durableDoc,durable.rows[0].snapshot);for(const row of (await stack.pool.query('SELECT update_bytes FROM collab_updates WHERE scene_id=$1 AND sequence>$2 ORDER BY sequence',[scene.id,durable.rows[0].snapshot_sequence])).rows)Y.applyUpdate(durableDoc,row.update_bytes);
    expect(durableDoc.getMap<Y.Map<unknown>>('elements').get(offlineImage.id)?.get('isDeleted')).toBe(false);expect(durableDoc.getMap<Y.Map<unknown>>('elements').get(offlineImage.id)?.get('x')).toBeGreaterThan(offlineImage.x);durableDoc.destroy();
    await owner.reload();await expect(owner.getByRole('status')).toHaveText('Saved',{timeout:20_000});await expect.poll(async () => (await active(owner)).length,{timeout:20_000}).toBe(2);
    const after = (await stack.pool.query('SELECT document_id FROM collab_documents WHERE scene_id=$1', [scene.id])).rows[0];
    expect(after.document_id).toBe(before.document_id);
    const recovery = await stack.pool.query("SELECT count(*) FROM scene_versions WHERE scene_id=$1 AND label='Before restore' AND pinned", [scene.id]);
    expect(recovery.rows[0].count).toBe('1');
  } finally {
    await ownerContext.close();
    await editorContext.close();
  }
});
