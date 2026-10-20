import { describe, it, expect, beforeEach, vi } from 'vitest';
import useWorkspaceStore from '../store/workspace-store';
import api from '../lib/api';

vi.mock('../lib/api', () => ({
  default: {
    get: vi.fn(),
    post: vi.fn(),
    patch: vi.fn(),
  },
}));

describe('WorkspaceStore', () => {
  beforeEach(() => {
    useWorkspaceStore.setState({
      workspaces: [],
      activeWorkspaceId: null,
      isLoading: false,
    });
    vi.clearAllMocks();
  });

  it('should fetch workspaces successfully', async () => {
    const mockWorkspaces = [
      { id: '1', name: 'WS 1', ownerId: 'user1', createdAt: 'date' },
    ];
    (api.get as any).mockResolvedValue({ data: mockWorkspaces });

    await useWorkspaceStore.getState().fetchWorkspaces();

    const state = useWorkspaceStore.getState();
    expect(state.workspaces).toHaveLength(1);
    expect(state.activeWorkspaceId).toBe('1'); // Should auto-set if none active
  });

  it('should create workspace and update state', async () => {
    const newWS = { id: '2', name: 'New WS', ownerId: 'user1', createdAt: 'date' };
    (api.post as any).mockResolvedValue({ data: newWS });

    await useWorkspaceStore.getState().createWorkspace('New WS');

    const state = useWorkspaceStore.getState();
    expect(state.workspaces).toContainEqual(newWS);
    expect(state.activeWorkspaceId).toBe('2');
  });

  it('renames an active workspace without changing its id', async () => {
    const original = { id: 'w1', name: 'Old', ownerId: 'user', role: 'owner' as const, createdAt: 'date' };
    useWorkspaceStore.setState({ workspaces: [original], activeWorkspaceId: 'w1' });
    vi.mocked(api.patch).mockResolvedValue({ data: { ...original, name: 'New' } });

    await useWorkspaceStore.getState().renameWorkspace('w1', 'New');

    expect(api.patch).toHaveBeenCalledWith('/workspaces/w1', { name: 'New' }, { headers: { 'X-Workspace-ID': 'w1' } });
    expect(useWorkspaceStore.getState().workspaces[0].name).toBe('New');
    expect(useWorkspaceStore.getState().activeWorkspaceId).toBe('w1');
  });
});
