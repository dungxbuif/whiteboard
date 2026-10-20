import { expect, test } from './fixtures';
import { login } from './browser';

test('activityMobileAndKeyboard keeps paging authorized and the scene panel inside the canvas', async ({ browser, stack }) => {
  const scene = await stack.scene('Enterprise activity');
  await stack.pool.query('UPDATE workspace_feature_policies SET activity_enabled=true WHERE workspace_id=$1', [stack.owner.workspaceId]);
  await stack.pool.query(`INSERT INTO workspace_events(workspace_id,scene_id,actor_kind,actor_id,kind,source,source_key)
    VALUES ($1,$2,'user',$3,'comment.created','activity_e2e','visible')`, [stack.owner.workspaceId, scene.id, stack.owner.id]);
  await stack.pool.query(`INSERT INTO workspace_events(workspace_id,actor_kind,actor_id,kind,source,source_key)
    VALUES ($1,'system','system','acl.workspace_members.update','activity_e2e','owner-only')`, [stack.owner.workspaceId]);
  for (const kind of ['scene.updated', 'share.created', 'agent.generate.success']) {
    await stack.pool.query(`INSERT INTO workspace_events(workspace_id,scene_id,actor_kind,actor_id,kind,source,source_key)
      VALUES ($1,$2,$3,$4,$5,'activity_e2e',$6)`, [stack.owner.workspaceId, scene.id, kind.startsWith('agent.') ? 'agent' : 'user', kind.startsWith('agent.') ? 'agent:test' : stack.owner.id, kind, kind]);
  }
  for (let index = 0; index < 51; index++) {
    await stack.pool.query(`INSERT INTO workspace_events(workspace_id,scene_id,actor_kind,actor_id,kind,occurred_at,source,source_key)
      VALUES ($1,$2,'user',$3,'scene.updated',clock_timestamp() - ($4 * interval '1 minute'),'activity_e2e',$5)`, [stack.owner.workspaceId, scene.id, stack.owner.id, index + 10, `filler-${index}`]);
  }
  for (let replay = 0; replay < 2; replay++) {
    await stack.pool.query(`INSERT INTO workspace_events(workspace_id,scene_id,actor_kind,actor_id,kind,source,source_key)
      VALUES ($1,$2,'agent','agent:test','agent.generate.success','activity_replay','same') ON CONFLICT(source,source_key) DO NOTHING`, [stack.owner.workspaceId, scene.id]);
  }

  const ownerContext = await browser.newContext({ viewport: { width: 1440, height: 900 } });
  const viewerContext = await browser.newContext({ viewport: { width: 390, height: 844 } });
  const emptyContext = await browser.newContext({ viewport: { width: 390, height: 844 } });
  try {
    await stack.pool.query('UPDATE workspace_feature_policies SET activity_enabled=true WHERE workspace_id=$1', [stack.outsider.workspaceId]);
    const empty = await emptyContext.newPage();
    await login(empty, stack, stack.outsider);
    await empty.goto(`${stack.baseURL}/w/${stack.outsider.workspaceId}/activity`);
    await expect(empty.getByText('No activity yet.')).toBeVisible();
    await expect(empty.getByText(/Available since/)).toHaveCount(0);

    const owner = await ownerContext.newPage();
    await login(owner, stack);
    await owner.goto(`${stack.baseURL}/w/${stack.owner.workspaceId}/activity`);
    await expect(owner.getByText('comment.created').first()).toBeVisible();
    await expect(owner.getByText('acl.workspace_members.update')).toBeVisible();
    await expect(owner.getByText('share.created')).toBeVisible();
    await expect(owner.getByText('agent.generate.success')).toHaveCount(2);
    await expect(owner.getByText(/Available since/)).toBeVisible();

    const viewer = await viewerContext.newPage();
    await login(viewer, stack, stack.viewer);
    await viewer.goto(`${stack.baseURL}/w/${stack.owner.workspaceId}/activity`);
    await expect(viewer.getByText('comment.created').first()).toBeVisible();
    await expect(viewer.getByText('acl.workspace_members.update')).toHaveCount(0);
    await stack.pool.query('DELETE FROM workspace_members WHERE workspace_id=$1 AND user_id=$2', [stack.owner.workspaceId, stack.viewer.id]);
    await viewer.getByRole('button', { name: 'Load older activity' }).click();
    await expect(viewer.getByRole('alert')).toBeVisible();
    await expect(viewer.getByText('comment.created')).toHaveCount(0);

    await owner.goto(`${stack.baseURL}/w/${stack.owner.workspaceId}/scenes/${scene.id}`);
    await expect(owner.getByRole('status')).toHaveText('Saved', { timeout: 30_000 });
    await expect(owner.locator('canvas').first()).toBeVisible();
    const before = await owner.getByTestId('scene-canvas-root').boundingBox();
    await owner.getByRole('button', { name: 'Open activity' }).click();
    await expect(owner.getByRole('dialog', { name: 'Activity' })).toBeVisible();
    expect(await owner.getByTestId('scene-canvas-root').boundingBox()).toEqual(before);
    await owner.keyboard.press('Escape');
    await expect(owner.getByRole('dialog', { name: 'Activity' })).toHaveCount(0);
    await expect(owner.getByRole('button', { name: 'Open activity' })).toBeFocused();
  } finally {
    await ownerContext.close();
    await viewerContext.close();
    await emptyContext.close();
  }
});

