import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';

import { CommentAnchor, resolveCommentAnchor } from './comment-anchor';
import { CommentPanel } from './comment-panel';
import type { CommentClient, CommentThread } from '@/lib/comment-api';

const thread: CommentThread = {
  id: 'thread-1', sceneId: 'scene-1', creatorKind: 'public', creatorName: '<Guest>',
  anchor: { elementId: 'element-1', x: 10, y: 20 }, status: 'open', revision: '1', subscribed: false,
  messages: [{ id: 'message-1', threadId: 'thread-1', authorKind: 'public', authorName: '<Guest>', body: '<script>alert(1)</script>', revision: '1', mine: false, createdAt: '', updatedAt: '' }],
  createdAt: '', updatedAt: '',
};

function client(overrides: Partial<CommentClient> = {}): CommentClient {
  return {
    list: vi.fn().mockResolvedValue({ items: [thread] }),
    create: vi.fn().mockResolvedValue(thread),
    get: vi.fn().mockResolvedValue(thread),
    reply: vi.fn().mockRejectedValue(Object.assign(new Error('changed'), { code: 'REVISION_CONFLICT' })),
    updateThread: vi.fn().mockResolvedValue(thread),
    deleteMessage: vi.fn().mockResolvedValue(undefined),
    ...overrides,
  };
}

describe('comments', () => {
  it('anchorTracksMoveAndDeletedElement', () => {
    const state = { scrollX: 20, scrollY: -10, zoom: { value: 2 }, offsetLeft: 5, offsetTop: 7 };
    const api = { getSceneElements: () => [{ id: 'element-1', x: 100, y: 50, width: 40, height: 20 }], getAppState: () => state };
    expect(resolveCommentAnchor(thread.anchor, api)).toEqual({ x: 285, y: 107, unavailable: false });
    const deleted = { ...api, getSceneElements: () => [] };
    expect(resolveCommentAnchor(thread.anchor, deleted)).toEqual({ x: 65, y: 27, unavailable: true });
    const { rerender } = render(<CommentAnchor anchor={thread.anchor} api={deleted} label="1" />);
    expect(screen.getByLabelText('Comment 1: Element unavailable')).toBeInTheDocument();
    state.scrollX = 30;
    rerender(<CommentAnchor anchor={thread.anchor} api={api} label="1" />);
    expect(screen.getByLabelText('Comment 1')).toHaveStyle({ left: '305px', top: '107px' });
  });

  it('hides moderation from viewers and reports rejected owner moderation', async () => {
    const api = client({ updateThread: vi.fn().mockRejectedValue(new Error('permission changed')) });
    const view = render(<CommentPanel api={api} principal={{ kind: 'user', id: 'viewer' }} sceneRef={{ workspaceId: 'workspace-1', sceneId: 'scene-1' }} />);
    await screen.findByText('<script>alert(1)</script>');
    expect(screen.queryByRole('button', { name: 'Resolve' })).toBeNull();
    view.rerender(<CommentPanel api={api} principal={{ kind: 'user', id: 'editor', canResolve: true }} sceneRef={{ workspaceId: 'workspace-1', sceneId: 'scene-1' }} />);
    fireEvent.click(screen.getByRole('button', { name: 'Resolve' }));
    expect(await screen.findByRole('alert')).toHaveTextContent('permission changed');
    expect(screen.getByRole('button', { name: 'Resolve' })).toBeInTheDocument();
  });

  it('guestReplyAndOwnerRedactionPreservesConflictDraft', async () => {
    const api = client();
    const view = render(<CommentPanel api={api} principal={{ kind: 'public', id: 'guest', token: 'scene-session' }} sceneRef={{ workspaceId: 'workspace-1', sceneId: 'scene-1' }} />);
    expect(await screen.findByText('<script>alert(1)</script>')).toBeInTheDocument();
    expect(document.querySelector('script')).toBeNull();
    fireEvent.change(screen.getByLabelText('Reply'), { target: { value: 'keep my draft' } });
    fireEvent.click(screen.getByRole('button', { name: 'Reply' }));
    expect(await screen.findByRole('alert')).toHaveTextContent('changed');
    expect(screen.getByLabelText('Reply')).toHaveValue('keep my draft');
    expect(api.reply).toHaveBeenCalledWith(expect.anything(), 'thread-1', 'keep my draft');
    view.rerender(<CommentPanel api={api} principal={{ kind: 'user', id: 'owner', owner: true }} sceneRef={{ workspaceId: 'workspace-1', sceneId: 'scene-1' }} />);
    fireEvent.click(screen.getByRole('button', { name: 'Redact message' }));
    await waitFor(() => expect(api.deleteMessage).toHaveBeenCalledWith(expect.anything(), 'thread-1', 'message-1', '1'));
  });

  it('selects the exact deep-linked thread even when it is outside the first page', async () => {
    const linked = {
      ...thread,
      id: 'thread-linked',
      messages: [{ ...thread.messages[0], id: 'message-linked', threadId: 'thread-linked', body: 'Linked discussion' }],
    };
    const api = client({ get: vi.fn().mockResolvedValue(linked) });
    const view = render(<CommentPanel api={api} initialThreadId="thread-1" principal={{ kind: 'user', id: 'viewer' }} sceneRef={{ workspaceId: 'workspace-1', sceneId: 'scene-1' }} />);
    await screen.findByText('<script>alert(1)</script>');
    view.rerender(<CommentPanel api={api} initialThreadId="thread-linked" principal={{ kind: 'user', id: 'viewer' }} sceneRef={{ workspaceId: 'workspace-1', sceneId: 'scene-1' }} />);
    expect(await screen.findByText('Linked discussion')).toBeInTheDocument();
    expect(api.get).toHaveBeenCalledWith({ workspaceId: 'workspace-1', sceneId: 'scene-1' }, 'thread-linked');
  });
});
