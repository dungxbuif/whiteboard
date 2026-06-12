import { create } from "zustand";

import api from "@/lib/api";

export interface Folder {
  id: string;
  workspaceId: string;
  parentId: string | null;
  name: string;
  path: string;
  depth: number;
  isStarred: boolean;
  createdAt: string;
  updatedAt: string;
}

export interface FolderNode extends Folder {
  children: FolderNode[];
}

interface FolderState {
  folders: Folder[];
  currentFolderId: string | null;
  isLoading: boolean;
  fetchFolders: (workspaceId: string) => Promise<void>;
  createFolder: (workspaceId: string, name: string, parentId: string | null) => Promise<Folder>;
  moveFolder: (workspaceId: string, folderId: string, newParentId: string | null) => Promise<void>;
  setCurrentFolderId: (folderId: string | null) => void;
  reset: () => void;
}

const workspaceHeaders = (workspaceId: string) => ({ "X-Workspace-ID": workspaceId });

export function buildFolderTree(folders: Folder[]): FolderNode[] {
  const nodes = new Map<string, FolderNode>();
  const roots: FolderNode[] = [];

  for (const folder of folders) {
    nodes.set(folder.id, { ...folder, children: [] });
  }

  for (const folder of folders) {
    const node = nodes.get(folder.id);
    if (!node) continue;
    const parent = folder.parentId ? nodes.get(folder.parentId) : undefined;
    if (parent) parent.children.push(node);
    else roots.push(node);
  }

  const sortNodes = (items: FolderNode[]) => {
    items.sort((left, right) => left.name.localeCompare(right.name));
    items.forEach((item) => sortNodes(item.children));
  };
  sortNodes(roots);
  return roots;
}

export function getFolderPath(folders: Folder[], folderId: string | null): Folder[] {
  if (!folderId) return [];
  const byId = new Map(folders.map((folder) => [folder.id, folder]));
  const path: Folder[] = [];
  let current = byId.get(folderId);
  while (current) {
    path.unshift(current);
    current = current.parentId ? byId.get(current.parentId) : undefined;
  }
  return path;
}

const useFolderStore = create<FolderState>((set) => ({
  folders: [],
  currentFolderId: null,
  isLoading: false,

  fetchFolders: async (workspaceId) => {
    set({ isLoading: true });
    try {
      const response = await api.get<Folder[]>("/folders?all=true", {
        headers: workspaceHeaders(workspaceId),
      });
      set({ folders: response.data, isLoading: false });
    } catch (error) {
      set({ folders: [], isLoading: false });
      throw error;
    }
  },

  createFolder: async (workspaceId, name, parentId) => {
    const response = await api.post<Folder>(
      "/folders/",
      { name, parentId },
      { headers: workspaceHeaders(workspaceId) },
    );
    set((state) => ({ folders: [...state.folders, response.data] }));
    return response.data;
  },

  moveFolder: async (workspaceId, folderId, newParentId) => {
    await api.patch(
      `/folders/${folderId}/move`,
      { newParentId },
      { headers: workspaceHeaders(workspaceId) },
    );
    await useFolderStore.getState().fetchFolders(workspaceId);
  },

  setCurrentFolderId: (currentFolderId) => set({ currentFolderId }),
  reset: () => set({ folders: [], currentFolderId: null, isLoading: false }),
}));

export default useFolderStore;
