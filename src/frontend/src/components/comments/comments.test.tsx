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
    expect(resolveCommentAnchor(thread.anchor, [{ id: 'element-1', x: 100, y: 50, width: 40, height: 20 }])).toEqual({ x: 120, y: 60, unavailable: false });
    expect(resolveCommentAnchor(thread.anchor, [])).toEqual({ x: 10, y: 20, unavailable: true });
    const { rerender } = render(<CommentAnchor anchor={thread.anchor} elements={[]} label="1" />);
    expect(screen.getByLabelText('Comment 1: Element unavailable')).toBeInTheDocument();
    rerender(<CommentAnchor anchor={thread.anchor} elements={[{ id: 'element-1', x: 2, y: 4, width: 10, height: 12 }]} label="1" />);
    expect(screen.getByLabelText('Comment 1')).toHaveStyle({ left: '7px', top: '10px' });
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
});
