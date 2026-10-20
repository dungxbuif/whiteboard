import { randomUUID } from 'node:crypto';
import { expect, test } from './fixtures';
import { login } from './browser';

test('domain capture is owner-only, explicit, and exposes a user-free discovery hint', async ({ page, stack }) => {
  const created = await stack.json(`/workspaces/${stack.owner.workspaceId}/domains`, stack.owner.token, stack.owner.workspaceId, 'POST', { domain: 'BÜCHER.example', requestId: randomUUID() });
  expect(created.domain).toBe('xn--bcher-kva.example');
  expect(created.dnsName).toBe('_whiteboard-verification.xn--bcher-kva.example');
  expect(created.txtValue).toMatch(/^whiteboard-verification=/);

  const stored = await stack.pool.query('SELECT challenge_hash,domain,status FROM workspace_domains WHERE id=$1', [created.id]);
  expect(stored.rows[0].domain).toBe('xn--bcher-kva.example');
  expect(stored.rows[0].status).toBe('pending');
  expect(String(stored.rows[0].challenge_hash)).not.toContain(created.txtValue);

  const denied = await stack.request(`/workspaces/${stack.owner.workspaceId}/domains`, stack.editor.token, stack.owner.workspaceId, 'POST', { domain: 'editor.example', requestId: randomUUID() });
  expect(denied.status).toBe(403);

  // Fixture transition only: this is application-flow evidence, not public DNS proof.
  await stack.pool.query("UPDATE workspace_domains SET status='verified',verified_at=clock_timestamp(),revision=revision+1 WHERE id=$1", [created.id]);
  await login(page, stack);
  await page.goto(`${stack.baseURL}/w/${stack.owner.workspaceId}/settings/identity`);
  await expect(page.getByText('xn--bcher-kva.example')).toBeVisible();
  await page.getByRole('button', { name: 'Accept and capture' }).click();
  await expect(page.getByText('captured')).toBeVisible();

  const discovery = await stack.json('/identity/domains/discover?domain=B%C3%9CCHER.example');
  const workspaceHint = await stack.pool.query('SELECT COALESCE(public_id,id::text) AS value FROM workspaces WHERE id=$1', [stack.owner.workspaceId]);
  expect(discovery).toEqual({ domain: 'xn--bcher-kva.example', workspace: workspaceHint.rows[0].value });
  expect(JSON.stringify(discovery)).not.toMatch(/email|user|member/i);
});
