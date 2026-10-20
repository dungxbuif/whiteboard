import { randomUUID } from 'node:crypto';
import { expect, test } from './fixtures';
import { draw, elements, openScene } from './browser';

test('version restore keeps the document generation and creates one recovery checkpoint', async ({ page, stack }) => {
  const scene = await stack.scene('Enterprise versions');
  await stack.pool.query('UPDATE workspace_feature_policies SET versions_enabled=true WHERE workspace_id=$1', [stack.owner.workspaceId]);
  await openScene(page, stack, scene);
  await draw(page, 250);
  await expect(page.getByRole('status')).toHaveText('Saved');
  const before = (await stack.pool.query('SELECT document_id,sequence::text FROM collab_documents WHERE scene_id=$1', [scene.id])).rows[0];
  const version = await stack.json(`/scenes/${scene.id}/versions`, stack.owner.token, stack.owner.workspaceId, 'POST', { requestId: randomUUID(), expectedSequence: before.sequence, label: 'One shape', pinned: true });

  await draw(page, 500);
  await expect(page.getByRole('status')).toHaveText('Saved');
  expect(await elements(page)).toHaveLength(2);
  const changed = (await stack.pool.query('SELECT sequence::text FROM collab_documents WHERE scene_id=$1', [scene.id])).rows[0];
  const diff = await stack.json(`/scenes/${scene.id}/versions/${version.id}/diff?againstSequence=${changed.sequence}`, stack.owner.token, stack.owner.workspaceId);
  expect(diff.addedIds).toHaveLength(1);
  await page.getByRole('button', { name: 'Open history' }).click();
  await expect(page.getByRole('dialog', { name: 'Version history' })).toBeVisible();
  await expect(page.getByText('One shape')).toBeVisible();
  await page.getByRole('button', { name: 'Restore version' }).click();
  await expect(page.getByText(/offline edits can replay after the restore/i)).toBeVisible();
  await page.getByRole('button', { name: 'Confirm restore' }).click();
  await expect.poll(async () => (await elements(page)).filter((element: any) => !element.isDeleted).length).toBe(1);

  const after = (await stack.pool.query('SELECT document_id,sequence::text FROM collab_documents WHERE scene_id=$1', [scene.id])).rows[0];
  expect(after.document_id).toBe(before.document_id);
  expect(BigInt(after.sequence)).toBeGreaterThan(BigInt(before.sequence));
  const recovery = await stack.pool.query("SELECT count(*) FROM scene_versions WHERE scene_id=$1 AND label='Before restore' AND pinned", [scene.id]);
  expect(recovery.rows[0].count).toBe('1');
  expect(version.documentId).toBe(before.document_id);
});
