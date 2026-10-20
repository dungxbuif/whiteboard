import { create } from "zustand";

import api from "@/lib/api";
import useWorkspaceStore from "@/store/workspace-store";

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
  renameFolder: (workspaceId: string, folderId: string, name: string) => Promise<void>;
  updateFolder: (workspaceId: string, folderId: string, changes: { name?: string; newParentId?: string | null }) => Promise<void>;
  setFolderStar: (workspaceId: string, folderId: string) => Promise<void>;
  trashFolder: (workspaceId: string, folderId: string) => Promise<void>;
  setCurrentFolderId: (folderId: string | null) => void;
  reset: () => void;
}

const workspaceHeaders = (workspaceId: string) => ({ "X-Workspace-ID": workspaceId });
let folderGeneration = 0;
const workspaceIsCurrent = (workspaceId: string) => {
  const active = useWorkspaceStore.getState().activeWorkspaceId;
  return active === null || active === workspaceId;
};

function applyCommittedFolderChange(folders: Folder[], folderId: string, changes: { name?: string; newParentId?: string | null }): Folder[] {
  const target = folders.find((folder) => folder.id === folderId);
  if (!target) return folders;
  const parent = changes.newParentId ? folders.find((folder) => folder.id === changes.newParentId) : null;
  const path = changes.newParentId === undefined ? target.path : `${parent?.path ?? "/"}${target.id}/`;
  const depth = changes.newParentId === undefined ? target.depth : (parent?.depth ?? -1) + 1;
  const delta = depth - target.depth;
  return folders.map((folder) => {
    if (folder.id !== target.id && !folder.path.startsWith(target.path)) return folder;
    return {
      ...folder,
      path: `${path}${folder.path.slice(target.path.length)}`,
      depth: folder.depth + delta,
      ...(folder.id === target.id ? {
        ...(changes.name !== undefined ? { name: changes.name } : {}),
        ...(changes.newParentId !== undefined ? { parentId: changes.newParentId } : {}),
      } : {}),
    };
  });
}

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
    const generation = ++folderGeneration;
    set({ isLoading: true });
    try {
      const response = await api.get<Folder[]>("/folders?all=true", {
        headers: workspaceHeaders(workspaceId),
      });
      if (generation === folderGeneration && workspaceIsCurrent(workspaceId)) set({ folders: response.data, isLoading: false });
    } catch (error) {
      if (generation === folderGeneration && workspaceIsCurrent(workspaceId)) set({ folders: [], isLoading: false });
      throw error;
    }
  },

  createFolder: async (workspaceId, name, parentId) => {
    const generation = folderGeneration;
    const response = await api.post<Folder>(
      "/folders/",
      { name, parentId },
      { headers: workspaceHeaders(workspaceId) },
    );
    if (generation === folderGeneration && workspaceIsCurrent(workspaceId)) set((state) => ({ folders: [...state.folders, response.data] }));
    return response.data;
  },

  moveFolder: async (workspaceId, folderId, newParentId) => {
    const generation = folderGeneration;
    await api.patch(
      `/folders/${folderId}/move`,
      { newParentId },
      { headers: workspaceHeaders(workspaceId) },
    );
    if (generation !== folderGeneration || !workspaceIsCurrent(workspaceId)) return;
    set((state) => ({ folders: applyCommittedFolderChange(state.folders, folderId, { newParentId }) }));
    const committedFolders = useFolderStore.getState().folders;
    try { await useFolderStore.getState().fetchFolders(workspaceId); }
    catch { if (generation + 1 === folderGeneration && workspaceIsCurrent(workspaceId)) set({ folders: committedFolders, isLoading: false }); }
  },

  renameFolder: async (workspaceId, folderId, name) => {
    await useFolderStore.getState().updateFolder(workspaceId, folderId, { name });
  },

  updateFolder: async (workspaceId, folderId, changes) => {
    const generation = folderGeneration;
    await api.patch(`/folders/${folderId}`, changes, { headers: workspaceHeaders(workspaceId) });
    if (generation !== folderGeneration || !workspaceIsCurrent(workspaceId)) return;
    set((state) => ({ folders: applyCommittedFolderChange(state.folders, folderId, changes) }));
    const committedFolders = useFolderStore.getState().folders;
    try { await useFolderStore.getState().fetchFolders(workspaceId); }
    catch { if (generation + 1 === folderGeneration && workspaceIsCurrent(workspaceId)) set({ folders: committedFolders, isLoading: false }); }
  },

  setFolderStar: async (workspaceId, folderId) => {
    await api.post(`/folders/${folderId}/star`, undefined, { headers: workspaceHeaders(workspaceId) });
    await useFolderStore.getState().fetchFolders(workspaceId);
  },

  trashFolder: async (workspaceId, folderId) => {
    await api.delete(`/folders/${folderId}/`, { headers: workspaceHeaders(workspaceId) });
    await useFolderStore.getState().fetchFolders(workspaceId);
  },

  setCurrentFolderId: (currentFolderId) => set({ currentFolderId }),
  reset: () => { folderGeneration += 1; set({ folders: [], currentFolderId: null, isLoading: false }); },
}));

export default useFolderStore;
