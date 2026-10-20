import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { beforeEach, expect, it, vi } from 'vitest';

import { NotificationInbox } from './notification-inbox';
import { listNotifications } from '@/lib/comment-api';

vi.mock('@/lib/comment-api', () => ({
  listNotifications: vi.fn(),
  markNotificationRead: vi.fn().mockResolvedValue(undefined),
}));
vi.mock('@/lib/router', () => ({ navigate: vi.fn() }));

const notification = (id: string) => ({
  id,
  workspaceId: 'workspace-1',
  sceneId: 'scene-1',
  kind: 'comment.replied',
  actorName: `Actor ${id}`,
  threadId: `thread-${id}`,
  deepLink: `/w/workspace-1/scenes/scene-1?thread=thread-${id}`,
  createdAt: '2026-10-20T00:00:00Z',
});

beforeEach(() => vi.mocked(listNotifications).mockReset());

it('continues from the server cursor and appends each notification once', async () => {
  vi.mocked(listNotifications)
    .mockResolvedValueOnce({ items: [notification('one')], nextCursor: 'raw-row-50' })
    .mockResolvedValueOnce({ items: [notification('one'), notification('two')] });
  render(<NotificationInbox />);
  fireEvent.click(screen.getByRole('button', { name: 'Notifications' }));
  expect(await screen.findByText('Actor one')).toBeInTheDocument();
  fireEvent.click(screen.getByRole('button', { name: 'Load more' }));
  expect(await screen.findByText('Actor two')).toBeInTheDocument();
  await waitFor(() => expect(listNotifications).toHaveBeenNthCalledWith(2, 'raw-row-50'));
  expect(screen.getAllByText('Actor one')).toHaveLength(1);
  expect(screen.queryByRole('button', { name: 'Load more' })).toBeNull();
});
