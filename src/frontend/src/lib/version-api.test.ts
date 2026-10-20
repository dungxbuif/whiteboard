import { describe, expect, it, vi } from 'vitest';
import { createVersionClient } from './version-api';

describe('version API', () => {
  it('binds workspace, scene, version and expected sequence for restore', async () => {
    const transport = { get: vi.fn(), post: vi.fn().mockResolvedValue({ data: { sequence: '9' } }), patch: vi.fn() };
    const client = createVersionClient(transport as never);
    await client.restore({ workspaceId: 'workspace-1', sceneId: 'scene-1' }, 'version-1', { requestId: 'request-1', expectedSequence: '8' });
    expect(transport.post).toHaveBeenCalledWith('/scenes/scene-1/versions/version-1/restore', { requestId: 'request-1', expectedSequence: '8' }, { headers: { 'X-Workspace-ID': 'workspace-1' } });
  });

  it('uses optimistic metadata revision for pin and label changes', async () => {
    const transport = { get: vi.fn(), post: vi.fn(), patch: vi.fn().mockResolvedValue({ data: { id: 'version-1' } }) };
    const client = createVersionClient(transport as never);
    await client.setMetadata({ workspaceId: 'workspace-1', sceneId: 'scene-1' }, 'version-1', { requestId: 'request-1', expectedRevision: '2', pinned: true });
    expect(transport.patch).toHaveBeenCalledWith('/scenes/scene-1/versions/version-1', { requestId: 'request-1', expectedRevision: '2', pinned: true }, { headers: { 'X-Workspace-ID': 'workspace-1' } });
  });
});
