import { describe, expect, it, vi } from 'vitest';
import { createCollectionClient, type Collection } from './collection-api';

const collection: Collection = { id: 'collection-1', name: 'Launch', visibility: 'workspace', revision: '1', sceneCount: 0, mine: true, canSetPrivate: true, scenes: [], createdAt: '2026-10-20T00:00:00Z', updatedAt: '2026-10-20T00:00:00Z' };

describe('collection mutation request identity', () => {
  it('reuses requestId after timeout-after-commit until a definitive response', async () => {
    const post = vi.fn()
      .mockRejectedValueOnce(new Error('timeout after commit'))
      .mockResolvedValueOnce({ data: collection })
      .mockResolvedValueOnce({ data: collection });
    const client = createCollectionClient({ post } as never);
    await expect(client.create('workspace-1', { name: 'Launch', visibility: 'workspace' })).rejects.toThrow('timeout');
    await expect(client.create('workspace-1', { name: 'Launch', visibility: 'workspace' })).resolves.toEqual(collection);
    expect(post.mock.calls[0][1].requestId).toBe(post.mock.calls[1][1].requestId);
    await client.create('workspace-1', { name: 'Launch', visibility: 'workspace' });
    expect(post.mock.calls[2][1].requestId).not.toBe(post.mock.calls[1][1].requestId);
  });

  it('drops an uncertain request when explicitly cancelled', async () => {
    const post = vi.fn().mockRejectedValueOnce(new Error('offline')).mockResolvedValueOnce({ data: collection });
    const client = createCollectionClient({ post } as never);
    await expect(client.create('workspace-1', { name: 'Launch', visibility: 'workspace' })).rejects.toThrow();
    const first = post.mock.calls[0][1].requestId;
    client.cancelPending?.();
    await client.create('workspace-1', { name: 'Launch', visibility: 'workspace' });
    expect(post.mock.calls[1][1].requestId).not.toBe(first);
  });

  it('starts a new request after a definitive HTTP error', async () => {
    const conflict = Object.assign(new Error('conflict'), { response: { status: 409 } });
    const post = vi.fn().mockRejectedValueOnce(conflict).mockResolvedValueOnce({ data: collection });
    const client = createCollectionClient({ post } as never);
    await expect(client.create('workspace-1', { name: 'Launch', visibility: 'workspace' })).rejects.toThrow('conflict');
    const first = post.mock.calls[0][1].requestId;
    await client.create('workspace-1', { name: 'Launch', visibility: 'workspace' });
    expect(post.mock.calls[1][1].requestId).not.toBe(first);
  });
});
