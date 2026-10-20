import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';

import { ActivityPanel } from './activity-panel';
import { activityPath, type ActivityClient, type ActivityPage } from '@/lib/activity-api';

const first: ActivityPage = {
  items: [{ id: '2', workspaceId: 'workspace-1', sceneId: 'scene-1', actorKind: 'user', actorName: '<Owner>', kind: 'comment.created', occurredAt: '2026-10-20T02:00:00Z', safeMetadata: { body: 'private comment', token: 'secret' } }],
  nextCursor: 'next',
  retainedSince: '2026-10-01T00:00:00Z',
};

describe('activity', () => {
  it('renders safe activity and loads the stable next page', async () => {
    const api: ActivityClient = { list: vi.fn().mockResolvedValueOnce(first).mockResolvedValueOnce({ items: [{ ...first.items[0], id: '1', actorName: 'Agent', kind: 'agent.generate.success' }], retainedSince: first.retainedSince }) };
    render(<ActivityPanel api={api} workspaceId="workspace-1" />);
    expect(await screen.findByText('<Owner>')).toBeInTheDocument();
    expect(document.querySelector('owner')).toBeNull();
    expect(screen.getByText('comment.created')).toBeInTheDocument();
    expect(screen.queryByText('private comment')).not.toBeInTheDocument();
    expect(screen.queryByText('secret')).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: 'Load older activity' }));
    await waitFor(() => expect(api.list).toHaveBeenLastCalledWith('workspace-1', undefined, 'next'));
    expect(await screen.findByText('agent.generate.success')).toBeInTheDocument();
  });

  it('keeps an authorized empty and error state inside the panel', async () => {
    const empty: ActivityClient = { list: vi.fn().mockResolvedValue({ items: [], retainedSince: '2026-10-01T00:00:00Z' }) };
    const view = render(<ActivityPanel api={empty} sceneId="scene-1" workspaceId="workspace-1" />);
    expect(await screen.findByText('No activity yet.')).toBeInTheDocument();
    const failed: ActivityClient = { list: vi.fn().mockRejectedValue(new Error('Activity unavailable')) };
    view.rerender(<ActivityPanel api={failed} sceneId="scene-1" workspaceId="workspace-1" />);
    expect(await screen.findByRole('alert')).toHaveTextContent('Activity unavailable');
  });

  it('clears cached rows when access is revoked while paging', async () => {
    const api: ActivityClient = { list: vi.fn().mockResolvedValueOnce(first).mockRejectedValueOnce(new Error('Access revoked')) };
    render(<ActivityPanel api={api} workspaceId="workspace-1" />);
    expect(await screen.findByText('comment.created')).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: 'Load older activity' }));
    expect(await screen.findByRole('alert')).toHaveTextContent('Access revoked');
    expect(screen.queryByText('comment.created')).not.toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'Load older activity' })).not.toBeInTheDocument();
  });

  it('builds canonical workspace and scene endpoints', () => {
    expect(activityPath('workspace-1')).toBe('/workspaces/workspace-1/activity');
    expect(activityPath('workspace-1', 'scene-1')).toBe('/scenes/scene-1/activity');
  });
});

