import { create } from "zustand";

import api from "@/lib/api";
import useFolderStore from "@/store/folder-store";
import useSceneStore from "@/store/scene-store";

export const SEARCH_DEBOUNCE_MS = 250;
export const SEARCH_VIEW_ALL = "all" as const;
export const SEARCH_VIEW_STARRED = "starred" as const;
export const SEARCH_VIEW_TRASH = "trash" as const;

export type SearchView = typeof SEARCH_VIEW_ALL | typeof SEARCH_VIEW_STARRED | typeof SEARCH_VIEW_TRASH;
export type SearchItemType = "folder" | "scene";

export interface SearchItem {
  id: string;
  workspaceId: string;
  type: SearchItemType;
  name: string;
  folderId?: string | null;
  isStarred: boolean;
  deletedAt?: string | null;
  updatedAt: string;
}

interface SearchState {
  query: string;
  view: SearchView;
  results: SearchItem[];
  isLoading: boolean;
  error: string | null;
  setQuery: (query: string) => void;
  setView: (view: SearchView) => void;
  search: (workspaceId: string, query?: string, view?: SearchView) => Promise<void>;
  restore: (workspaceId: string, item: SearchItem) => Promise<void>;
  reset: () => void;
}

const workspaceHeaders = (workspaceId: string) => ({ "X-Workspace-ID": workspaceId });

const useSearchStore = create<SearchState>((set, get) => ({
  query: "",
  view: SEARCH_VIEW_ALL,
  results: [],
  isLoading: false,
  error: null,

  setQuery: (query) => set({ query }),
  setView: (view) => set({ view, error: null }),

  search: async (workspaceId, query = get().query, view = get().view) => {
    set({ isLoading: true, error: null, query, view });
    try {
      const response = await api.get<SearchItem[]>("/search", {
        headers: workspaceHeaders(workspaceId),
        params: { q: query.trim() || undefined, view },
      });
      set({ results: response.data, isLoading: false });
    } catch (error) {
      set({ results: [], isLoading: false, error: "Search results could not be loaded." });
      throw error;
    }
  },

  restore: async (workspaceId, item) => {
    await api.post(`/${item.type === "scene" ? "scenes" : "folders"}/${item.id}/restore`, undefined, {
      headers: workspaceHeaders(workspaceId),
    });
    await get().search(workspaceId);
    if (item.type === "scene") {
      await useSceneStore.getState().fetchScenes(workspaceId, item.folderId ?? null);
    } else {
      await useFolderStore.getState().fetchFolders(workspaceId);
    }
  },

  reset: () => set({ query: "", view: SEARCH_VIEW_ALL, results: [], isLoading: false, error: null }),
}));

export default useSearchStore;
