import { describe, it, expect, beforeEach, vi } from 'vitest';
import useWorkspaceStore from '../store/workspace-store';
import api from '../lib/api';

vi.mock('../lib/api', () => ({
  default: {
    get: vi.fn(),
    post: vi.fn(),
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
});
