import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { HistoryPanel } from './history-panel';

const version = { id: 'version-1', sceneId: 'scene-1', documentId: 'document-1', sequence: '7', hash: 'a'.repeat(64), pinned: true, kind: 'manual', creator: { kind: 'user', id: 'user-1' }, metadataRevision: '1', createdAt: '2026-10-20T00:00:00Z', updatedAt: '2026-10-20T00:00:00Z' };

describe('HistoryPanel', () => {
  it('loads semantic diff and restores only after an explicit warning', async () => {
    const api = {
      list: vi.fn().mockResolvedValue({ items: [version] }),
      diff: vi.fn().mockResolvedValue({ addedIds: ['new'], removedIds: [], changedIds: ['old'], changedAppStateFields: ['theme'] }),
      restore: vi.fn().mockResolvedValue({ requestId: 'request-1', sequence: '8', stateVector: 'AA==', preRestoreVersionId: 'recovery-1' }),
    };
    const restored = vi.fn();
    render(<HistoryPanel api={api} canRestore currentSequence="7" onRestored={restored} sceneRef={{ workspaceId: 'workspace-1', sceneId: 'scene-1' }} />);
    expect(await screen.findByText('Pinned checkpoint')).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: 'Compare version' }));
    expect(await screen.findByText('1 added')).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: 'Restore version' }));
    expect(screen.getByText(/offline edits can replay after the restore/i)).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: 'Confirm restore' }));
    await waitFor(() => expect(api.restore).toHaveBeenCalledWith({ workspaceId: 'workspace-1', sceneId: 'scene-1' }, 'version-1', expect.objectContaining({ expectedSequence: '7' })));
    expect(restored).toHaveBeenCalledWith(expect.objectContaining({ sequence: '8' }));
  });

  it('keeps the panel actionable after a stale sequence conflict', async () => {
    const api = { list: vi.fn().mockResolvedValue({ items: [version] }), diff: vi.fn(), restore: vi.fn().mockRejectedValue({ response: { status: 409 } }) };
    render(<HistoryPanel api={api} canRestore currentSequence="7" onRestored={vi.fn()} sceneRef={{ workspaceId: 'workspace-1', sceneId: 'scene-1' }} />);
    await screen.findByText('Pinned checkpoint');
    fireEvent.click(screen.getByRole('button', { name: 'Restore version' }));
    fireEvent.click(screen.getByRole('button', { name: 'Confirm restore' }));
    expect(await screen.findByRole('alert')).toHaveTextContent(/scene changed/i);
    expect(screen.getByRole('button', { name: 'Confirm restore' })).toBeEnabled();
  });
});
