import { create } from "zustand";

import api from "@/lib/api";
import { writeCollaborativeContent, legacyConflict } from "@/collab/writers";
import useFolderStore from "@/store/folder-store";

export const SCENE_SAVE_DEBOUNCE_MS = 750;
export const EMPTY_SCENE_CONTENT = {
  elements: [],
  appState: {},
  files: {},
  presentationOrder: [],
};

export interface SceneContent {
  elements: unknown[];
  appState: Record<string, unknown>;
  files: Record<string, unknown>;
  presentationOrder?: string[];
}

export function syncPresentationOrder(elements: unknown[], previousOrder: string[] = []): string[] {
  const frameIds = elements.flatMap((element) => {
    if (!element || typeof element !== "object") return [];
    const candidate = element as { id?: unknown; type?: unknown };
    return candidate.type === "frame" && typeof candidate.id === "string" ? [candidate.id] : [];
  });
  const frameIdSet = new Set(frameIds);
  const preserved = previousOrder.filter((id, index) => frameIdSet.has(id) && previousOrder.indexOf(id) === index);
  const seen = new Set(preserved);
  return [...preserved, ...frameIds.filter((id) => !seen.has(id) && seen.add(id))];
}

export interface Scene {
  id: string;
  publicId?: string;
  access?: "view" | "edit";
  owner?: boolean;
  writerMode?: "v1" | "v2";
  workspaceId: string;
  folderId: string | null;
  title: string;
  content: SceneContent;
  contentVersion: number;
  isStarred: boolean;
  createdAt: string;
  updatedAt: string;
}

type SaveStatus = "idle" | "pending" | "saving" | "saved" | "error";

interface SceneState {
  scenes: Scene[];
  activeScene: Scene | null;
  isLoading: boolean;
  saveStatus: SaveStatus;
  error: string | null;
  fetchScenes: (workspaceId: string, folderId: string | null) => Promise<void>;
  createScene: (workspaceId: string, folderId: string | null, title?: string, content?: SceneContent) => Promise<Scene>;
  openScene: (scene: Scene) => void;
  openSceneById: (workspaceId: string, sceneId: string) => Promise<void>;
  closeScene: () => Promise<void>;
  cancelSceneOpen: () => void;
  scheduleSave: (workspaceId: string, sceneId: string, content: SceneContent) => void;
  flushSave: () => Promise<void>;
  renameScene: (workspaceId: string, sceneId: string, title: string) => Promise<void>;
  moveScene: (workspaceId: string, sceneId: string, folderId: string | null, currentFolderId: string | null) => Promise<void>;
  duplicateScene: (workspaceId: string, sceneId: string, title: string, folderId: string | null) => Promise<Scene>;
  setSceneStar: (workspaceId: string, sceneId: string, isStarred: boolean) => Promise<void>;
  trashScene: (workspaceId: string, sceneId: string) => Promise<void>;
  updatePresentationOrder: (workspaceId: string, sceneId: string, presentationOrder: string[]) => void;
  reset: () => void;
}

interface PendingSave {
  workspaceId: string;
  sceneId: string;
  content: SceneContent;
  generation: number;
}

let saveTimer: ReturnType<typeof setTimeout> | null = null;
let pendingSave: PendingSave | null = null;
let saveGeneration = 0;
let openGeneration = 0;
let listGeneration = 0;

const workspaceHeaders = (workspaceId: string) => ({ "X-Workspace-ID": workspaceId });

function updateScene(items: Scene[], updated: Scene) {
  return items.map((scene) => (scene.id === updated.id ? updated : scene));
}

const useSceneStore = create<SceneState>((set, get) => {
  const performSave = async () => {
    const save = pendingSave;
    if (!save) return;

    pendingSave = null;
    if (saveTimer) {
      clearTimeout(saveTimer);
      saveTimer = null;
    }
    set({ saveStatus: "saving", error: null });

    try {
      const response = await api.patch<Scene>(
        `/scenes/${save.sceneId}/`,
        { content: save.content },
        { headers: workspaceHeaders(save.workspaceId) },
      );
      if (save.generation !== saveGeneration) return;
      set((state) => ({
        scenes: updateScene(state.scenes, response.data),
        activeScene: state.activeScene?.id === response.data.id ? response.data : state.activeScene,
        saveStatus: "saved",
      }));
    } catch (error) {
      if (save.generation !== saveGeneration) return;
      if ((error as { response?: { data?: { code?: string } } }).response?.data?.code === 'COLLAB_REQUIRED') {
        set(state => ({ activeScene: state.activeScene?.id === save.sceneId ? { ...state.activeScene, writerMode: 'v2' } : state.activeScene, saveStatus: 'error', error: 'Collaboration is required. Previous edits are retained for recovery.' }));
        legacyConflict(save.workspaceId, save.sceneId, save.content);
        throw error;
      }
      pendingSave = save;
      set({ saveStatus: "error", error: "Scene could not be saved." });
      throw error;
    }
  };

  return {
    scenes: [],
    activeScene: null,
    isLoading: false,
    saveStatus: "idle",
    error: null,

    fetchScenes: async (workspaceId, folderId) => {
      const generation = ++listGeneration;
      set({ isLoading: true, error: null });
      try {
        const response = await api.get<Scene[]>("/scenes", {
          headers: workspaceHeaders(workspaceId),
          params: folderId ? { folderId } : undefined,
        });
        if (generation !== listGeneration) return;
        set({ scenes: response.data, isLoading: false });
      } catch (error) {
        if (generation !== listGeneration) return;
        set({ scenes: [], isLoading: false, error: "Scenes could not be loaded." });
        throw error;
      }
    },

    createScene: async (workspaceId, folderId, title = "Untitled scene", content = EMPTY_SCENE_CONTENT) => {
      const response = await api.post<Scene>(
        "/scenes/",
        { title, folderId, content },
        { headers: workspaceHeaders(workspaceId) },
      );
      set((state) => ({ scenes: [response.data, ...state.scenes], activeScene: response.data }));
      return response.data;
    },

    openScene: (activeScene) => { openGeneration += 1; set({ activeScene, saveStatus: "idle", error: null }); },

    openSceneById: async (workspaceId, sceneId) => {
      const generation = ++openGeneration;
      const response = await api.get<Scene>(`/scenes/${sceneId}/`, {
        headers: workspaceHeaders(workspaceId),
      });
      if (generation !== openGeneration) return;
      set((state) => ({
        scenes: state.scenes.some((scene) => scene.id === response.data.id)
          ? updateScene(state.scenes, response.data)
          : [response.data, ...state.scenes],
        activeScene: response.data,
        saveStatus: "idle",
        error: null,
      }));
    },

    cancelSceneOpen: () => { openGeneration += 1; set({ activeScene: null }); },

    closeScene: async () => {
      openGeneration += 1;
      await performSave();
      set({ activeScene: null, saveStatus: "idle" });
    },

    scheduleSave: (workspaceId, sceneId, content) => {
      if (writeCollaborativeContent(workspaceId, sceneId, content)) return;
      if (get().activeScene?.id === sceneId && get().activeScene?.writerMode === 'v2') {
        set({ saveStatus: 'error', error: 'Collaboration connection required; HTTP content writes are disabled.' }); return;
      }
      saveGeneration += 1;
      pendingSave = { workspaceId, sceneId, content, generation: saveGeneration };
      if (saveTimer) clearTimeout(saveTimer);
      set({ saveStatus: "pending", error: null });
      saveTimer = setTimeout(() => {
        void performSave().catch(() => undefined);
      }, SCENE_SAVE_DEBOUNCE_MS);
    },

    flushSave: performSave,

    renameScene: async (workspaceId, sceneId, title) => {
      const response = await api.patch<Scene>(
        `/scenes/${sceneId}/`,
        { title },
        { headers: workspaceHeaders(workspaceId) },
      );
      set((state) => ({
        scenes: updateScene(state.scenes, response.data),
        activeScene: state.activeScene?.id === sceneId ? response.data : state.activeScene,
      }));
    },

    moveScene: async (workspaceId, sceneId, folderId, currentFolderId) => {
      const response = await api.patch<Scene>(`/scenes/${sceneId}/`, { folderId }, { headers: workspaceHeaders(workspaceId) });
      set((state) => ({
        scenes: updateScene(state.scenes, response.data),
        activeScene: state.activeScene?.id === sceneId ? response.data : state.activeScene,
      }));
      await get().fetchScenes(workspaceId, currentFolderId);
    },

    duplicateScene: async (workspaceId, sceneId, title, folderId) => {
      const source = await api.get<Scene>(`/scenes/${sceneId}/`, { headers: workspaceHeaders(workspaceId) });
      const duplicate = await get().createScene(workspaceId, folderId, title, source.data.content);
      await get().fetchScenes(workspaceId, useFolderStore.getState().currentFolderId).catch(() => undefined);
      return duplicate;
    },

    setSceneStar: async (workspaceId, sceneId, isStarred) => {
      const response = await api.post<Scene>(
        `/scenes/${sceneId}/star`,
        { isStarred },
        { headers: workspaceHeaders(workspaceId) },
      );
      set((state) => ({
        scenes: updateScene(state.scenes, response.data),
        activeScene: state.activeScene?.id === sceneId ? response.data : state.activeScene,
      }));
    },

    trashScene: async (workspaceId, sceneId) => {
      await api.delete(`/scenes/${sceneId}/`, { headers: workspaceHeaders(workspaceId) });
      set((state) => ({
        scenes: state.scenes.filter((scene) => scene.id !== sceneId),
        activeScene: state.activeScene?.id === sceneId ? null : state.activeScene,
      }));
    },

    updatePresentationOrder: (workspaceId, sceneId, presentationOrder) => {
      const activeScene = get().activeScene;
      if (!activeScene || activeScene.id !== sceneId) return;
      const content = { ...activeScene.content, presentationOrder: [...presentationOrder] };
      set({ activeScene: { ...activeScene, content } });
      get().scheduleSave(workspaceId, sceneId, content);
    },

    reset: () => {
      openGeneration += 1;
      listGeneration += 1;
      if (saveTimer) clearTimeout(saveTimer);
      saveTimer = null;
      pendingSave = null;
      saveGeneration += 1;
      set({ scenes: [], activeScene: null, isLoading: false, saveStatus: "idle", error: null });
    },
  };
});

export default useSceneStore;
