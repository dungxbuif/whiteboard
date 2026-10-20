import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';

import { CollectionList } from './collection-list';
import { CollectionView } from './collection-view';
import { collectionPath, type Collection, type CollectionClient } from '@/lib/collection-api';

const workspaceCollection: Collection = { id: 'collection-1', name: 'Launch boards', visibility: 'workspace', revision: '2', sceneCount: 1, mine: true, canSetPrivate: true, scenes: [{ id: 'scene-1', title: 'Launch', position: 0, updatedAt: '2026-10-20T00:00:00Z' }], createdAt: '2026-10-20T00:00:00Z', updatedAt: '2026-10-20T00:00:00Z' };

function client(overrides: Partial<CollectionClient> = {}): CollectionClient {
  return {
    list: vi.fn().mockResolvedValue({ items: [workspaceCollection] }),
    get: vi.fn().mockResolvedValue(workspaceCollection),
    create: vi.fn().mockResolvedValue(workspaceCollection),
    update: vi.fn().mockResolvedValue(workspaceCollection),
    delete: vi.fn().mockResolvedValue(undefined),
    addScene: vi.fn().mockResolvedValue(workspaceCollection),
    removeScene: vi.fn().mockResolvedValue({ ...workspaceCollection, revision: '3', sceneCount: 0, scenes: [] }),
    reorder: vi.fn().mockResolvedValue(workspaceCollection),
    ...overrides,
  };
}

describe('legacy collections', () => {
  it('renders a reloadable legacy-style sidebar link and creates with keyboard submit', async () => {
    const api = client();
    render(<CollectionList api={api} canManage workspaceId="workspace-1" workspaceSlug="workspace-alias" />);
    const link = await screen.findByRole('link', { name: /Launch boards/ });
    expect(link).toHaveAttribute('href', '/w/workspace-alias/collections/collection-1');
    expect(collectionPath('workspace-alias', 'collection-1')).toBe('/w/workspace-alias/collections/collection-1');
    fireEvent.click(screen.getByRole('button', { name: 'New collection' }));
    fireEvent.change(screen.getByLabelText('Collection name'), { target: { value: 'Private plans' } });
    fireEvent.change(screen.getByLabelText('Collection visibility'), { target: { value: 'private' } });
    fireEvent.submit(screen.getByRole('form', { name: 'Create collection' }));
    await waitFor(() => expect(api.create).toHaveBeenCalledWith('workspace-1', { name: 'Private plans', visibility: 'private' }));
  });

  it('implements remove and stable reorder without moving scenes', async () => {
    const second = { id: 'scene-2', title: 'Review', position: 1, updatedAt: '2026-10-20T00:00:00Z' };
    const value = { ...workspaceCollection, scenes: [...workspaceCollection.scenes, second], sceneCount: 2 };
    const api = client({ get: vi.fn().mockResolvedValue(value), removeScene: vi.fn().mockResolvedValue(value), reorder: vi.fn().mockResolvedValue(value) });
    render(<CollectionView api={api} canManage collectionId="collection-1" workspaceId="workspace-1" workspaceSlug="workspace-alias" availableScenes={[]} />);
    expect(await screen.findByText('Launch boards')).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: 'Move Review up' }));
    await waitFor(() => expect(api.reorder).toHaveBeenCalledWith('workspace-1', 'collection-1', '2', ['scene-2', 'scene-1']));
    fireEvent.click(screen.getByRole('button', { name: 'Remove Launch' }));
    await waitFor(() => expect(api.removeScene).toHaveBeenCalledWith('workspace-1', 'collection-1', '2', 'scene-1'));
    expect(screen.getByRole('link', { name: 'Open Launch' })).toHaveAttribute('href', '/w/workspace-alias/scenes/scene-1');
  });

  it('hides management actions for viewers', async () => {
    render(<CollectionView api={client()} canManage={false} collectionId="collection-1" workspaceId="workspace-1" workspaceSlug="workspace-alias" availableScenes={[]} />);
    expect(await screen.findByText('Launch boards')).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: /Remove/ })).not.toBeInTheDocument();
    expect(screen.queryByRole('button', { name: /Delete collection/ })).not.toBeInTheDocument();
  });

  it('filters collections by name without losing canonical links', async () => {
    const other = { ...workspaceCollection, id: 'collection-2', name: 'Archive', sceneCount: 0, scenes: [] };
    render(<CollectionList api={client({ list: vi.fn().mockResolvedValue({ items: [workspaceCollection, other] }) })} canManage={false} workspaceId="workspace-1" workspaceSlug="workspace-alias" />);
    expect(await screen.findByRole('link', { name: /Launch boards/ })).toBeInTheDocument();
    fireEvent.change(screen.getByLabelText('Filter collections'), { target: { value: 'arch' } });
    expect(screen.queryByRole('link', { name: /Launch boards/ })).not.toBeInTheDocument();
    expect(screen.getByRole('link', { name: /Archive/ })).toHaveAttribute('href', '/w/workspace-alias/collections/collection-2');
  });

  it('does not offer private conversion to a noncreator editor', async () => {
    const shared = { ...workspaceCollection, mine: false, canSetPrivate: false };
    render(<CollectionView api={client({ get: vi.fn().mockResolvedValue(shared) })} canManage collectionId="collection-1" workspaceId="workspace-1" workspaceSlug="workspace-alias" availableScenes={[]} />);
    expect(await screen.findByText('Launch boards')).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: 'Edit collection' }));
    expect(screen.queryByRole('option', { name: 'Private' })).not.toBeInTheDocument();
  });
});
