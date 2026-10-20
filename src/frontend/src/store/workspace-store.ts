import { create } from 'zustand';
import api from '../lib/api';

export interface Workspace {
  id: string;
  publicId?: string;
  role?: 'owner' | 'editor' | 'viewer';
  name: string;
  ownerId: string;
  createdAt: string;
}

interface WorkspaceState {
  workspaces: Workspace[];
  activeWorkspaceId: string | null;
  isLoading: boolean;
  fetchWorkspaces: () => Promise<void>;
  setActiveWorkspaceId: (id: string) => void;
  createWorkspace: (name: string) => Promise<Workspace>;
  renameWorkspace: (id: string, name: string) => Promise<Workspace>;
}

const useWorkspaceStore = create<WorkspaceState>((set, get) => ({
  workspaces: [],
  activeWorkspaceId: null,
  isLoading: false,

  fetchWorkspaces: async () => {
    set({ isLoading: true });
    try {
      const response = await api.get<Workspace[]>('/workspaces');
      const workspaces = response.data;
      set({ workspaces, isLoading: false });
      
      // Default to first workspace if none active
      if (!workspaces.some(workspace => workspace.id === get().activeWorkspaceId)) {
        set({ activeWorkspaceId: workspaces[0]?.id ?? null });
      }
    } catch (error) {
      set({ isLoading: false });
      console.error('Failed to fetch workspaces', error);
    }
  },

  setActiveWorkspaceId: (id: string) => {
    set({ activeWorkspaceId: id });
  },

  createWorkspace: async (name: string) => {
    const response = await api.post<Workspace>('/workspaces', { name });
    const newWorkspace = response.data;
    set((state) => ({
      workspaces: [...state.workspaces, newWorkspace],
      activeWorkspaceId: state.activeWorkspaceId || newWorkspace.id,
    }));
    return newWorkspace;
  },

  renameWorkspace: async (id, name) => {
    const response = await api.patch<Workspace>(`/workspaces/${id}`, { name }, { headers: { "X-Workspace-ID": id } });
    set((state) => ({ workspaces: state.workspaces.map((workspace) => workspace.id === id ? { ...workspace, ...response.data } : workspace) }));
    return response.data;
  },
}));

export default useWorkspaceStore;
