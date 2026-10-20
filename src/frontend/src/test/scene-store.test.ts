import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import api from "../lib/api";
import useSceneStore, { SCENE_SAVE_DEBOUNCE_MS, syncPresentationOrder, type Scene, type SceneContent } from "../store/scene-store";

vi.mock("../lib/api", () => ({
  default: {
    get: vi.fn(),
    post: vi.fn(),
    patch: vi.fn(),
    delete: vi.fn(),
  },
}));

const emptyContent: SceneContent = { elements: [], appState: {}, files: {}, presentationOrder: [] };
const scene: Scene = {
  id: "scene-1",
  workspaceId: "workspace-1",
  folderId: null,
  title: "Scene",
  content: emptyContent,
  contentVersion: 1,
  isStarred: false,
  createdAt: "2026-06-12T00:00:00Z",
  updatedAt: "2026-06-12T00:00:00Z",
};

describe("scene store", () => {
  beforeEach(() => {
    useSceneStore.getState().reset();
    vi.clearAllMocks();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("keeps existing frame order and appends newly created frames", () => {
    expect(syncPresentationOrder([
      { id: "frame-new", type: "frame" },
      { id: "frame-old", type: "frame" },
      { id: "shape", type: "rectangle" },
    ], ["frame-old", "deleted-frame"])).toEqual(["frame-old", "frame-new"]);
  });

  it("loads scenes for the active workspace and folder", async () => {
    vi.mocked(api.get).mockResolvedValue({ data: [scene] });

    await useSceneStore.getState().fetchScenes("workspace-1", "folder-1");

    expect(api.get).toHaveBeenCalledWith("/scenes", {
      headers: { "X-Workspace-ID": "workspace-1" },
      params: { folderId: "folder-1" },
    });
    expect(useSceneStore.getState().scenes).toEqual([scene]);
  });

  it("debounces rapid scene changes into one save", async () => {
    vi.useFakeTimers();
    const latestContent: SceneContent = { elements: [{ id: "latest" }], appState: {}, files: {}, presentationOrder: [] };
    vi.mocked(api.patch).mockResolvedValue({ data: { ...scene, content: latestContent, contentVersion: 2 } });

    useSceneStore.getState().openScene(scene);
    useSceneStore.getState().scheduleSave("workspace-1", scene.id, { ...emptyContent, elements: [{ id: "first" }] });
    useSceneStore.getState().scheduleSave("workspace-1", scene.id, latestContent);
    await vi.advanceTimersByTimeAsync(SCENE_SAVE_DEBOUNCE_MS);

    expect(api.patch).toHaveBeenCalledTimes(1);
    expect(api.patch).toHaveBeenCalledWith(
      `/scenes/${scene.id}/`,
      { content: latestContent },
      { headers: { "X-Workspace-ID": "workspace-1" } },
    );
    expect(useSceneStore.getState().saveStatus).toBe("saved");
  });

  it("ignores an older save completion after a newer generation", async () => {
    vi.useFakeTimers();
    let resolveFirst!: (value: { data: Scene }) => void;
    let resolveSecond!: (value: { data: Scene }) => void;
    vi.mocked(api.patch)
      .mockReturnValueOnce(new Promise((resolve) => { resolveFirst = resolve; }))
      .mockReturnValueOnce(new Promise((resolve) => { resolveSecond = resolve; }));

    useSceneStore.getState().openScene(scene);
    useSceneStore.getState().scheduleSave("workspace-1", scene.id, { ...emptyContent, elements: [{ id: "old" }] });
    await vi.advanceTimersByTimeAsync(SCENE_SAVE_DEBOUNCE_MS);
    useSceneStore.getState().scheduleSave("workspace-1", scene.id, { ...emptyContent, elements: [{ id: "new" }] });
    await vi.advanceTimersByTimeAsync(SCENE_SAVE_DEBOUNCE_MS);

    const newest = { ...scene, title: "Newest", contentVersion: 3 };
    resolveSecond({ data: newest });
    await Promise.resolve();
    resolveFirst({ data: { ...scene, title: "Stale", contentVersion: 2 } });
    await Promise.resolve();

    expect(useSceneStore.getState().activeScene?.title).toBe("Newest");
    expect(useSceneStore.getState().saveStatus).toBe("saved");
  });

  it("keeps the active scene open when a save fails", async () => {
    vi.useFakeTimers();
    vi.mocked(api.patch).mockRejectedValue(new Error("network down"));

    useSceneStore.getState().openScene(scene);
    useSceneStore.getState().scheduleSave("workspace-1", scene.id, { ...emptyContent, elements: [{ id: "unsaved" }] });
    await vi.advanceTimersByTimeAsync(SCENE_SAVE_DEBOUNCE_MS);

    await expect(useSceneStore.getState().closeScene()).rejects.toThrow("network down");
    expect(useSceneStore.getState().activeScene?.id).toBe(scene.id);
    expect(useSceneStore.getState().saveStatus).toBe("error");
  });

  it("renames, stars, and trashes a scene through workspace-scoped APIs", async () => {
    vi.mocked(api.get).mockResolvedValue({ data: [scene] });
    vi.mocked(api.patch).mockResolvedValue({ data: { ...scene, title: "Renamed" } });
    vi.mocked(api.post).mockResolvedValue({ data: { ...scene, title: "Renamed", isStarred: true } });
    vi.mocked(api.delete).mockResolvedValue({ data: undefined });

    await useSceneStore.getState().fetchScenes("workspace-1", null);
    await useSceneStore.getState().renameScene("workspace-1", scene.id, "Renamed");
    await useSceneStore.getState().setSceneStar("workspace-1", scene.id, true);
    await useSceneStore.getState().trashScene("workspace-1", scene.id);

    expect(api.delete).toHaveBeenCalledWith(`/scenes/${scene.id}/`, {
      headers: { "X-Workspace-ID": "workspace-1" },
    });
    expect(useSceneStore.getState().scenes).toEqual([]);
  });

  it("duplicates canonical server content and opens the new scene", async () => {
    const canonical = { ...scene, content: { elements: [{ id: "stale" }], appState: {}, files: {}, presentationOrder: [] }, writerMode: "v2" as const };
    const durable = { elements: [{ id: "recent" }], appState: { zoom: 1 }, files: {}, presentationOrder: [] };
    const duplicate = { ...canonical, id: "scene-2", title: "Scene copy", folderId: "folder-2" };
    vi.mocked(api.get).mockResolvedValueOnce({ data: canonical }).mockResolvedValueOnce({ data: { content: durable, sequence: "2" } }).mockResolvedValueOnce({ data: [scene] });
    vi.mocked(api.post).mockResolvedValue({ data: duplicate });

    const result = await useSceneStore.getState().duplicateScene("workspace-1", scene.id, "Scene copy", "folder-2");

    expect(api.get).toHaveBeenCalledWith("/scenes/scene-1/", { headers: { "X-Workspace-ID": "workspace-1" } });
    expect(api.get).toHaveBeenCalledWith("/scenes/scene-1/projection", { headers: { "X-Workspace-ID": "workspace-1" } });
    expect(api.post).toHaveBeenCalledWith("/scenes/", { title: "Scene copy", folderId: "folder-2", content: durable }, { headers: { "X-Workspace-ID": "workspace-1" } });
    expect(api.get).toHaveBeenCalledWith("/scenes", { headers: { "X-Workspace-ID": "workspace-1" }, params: undefined });
    expect(result.id).toBe("scene-2");
    expect(useSceneStore.getState().activeScene?.id).toBe("scene-2");
  });

  it("moves a scene with the existing update route and reloads its list", async () => {
    vi.mocked(api.patch).mockResolvedValue({ data: { ...scene, folderId: "folder-2" } });
    vi.mocked(api.get).mockResolvedValue({ data: [] });
    await useSceneStore.getState().moveScene("workspace-1", scene.id, "folder-2", null);
    expect(api.patch).toHaveBeenCalledWith("/scenes/scene-1/", { folderId: "folder-2" }, { headers: { "X-Workspace-ID": "workspace-1" } });
    expect(api.get).toHaveBeenCalledWith("/scenes", { headers: { "X-Workspace-ID": "workspace-1" }, params: undefined });
  });

  it("returns a created duplicate even if refreshing the list fails", async () => {
    const duplicate = { ...scene, id: "scene-2" };
    vi.mocked(api.get).mockResolvedValueOnce({ data: scene }).mockRejectedValueOnce(new Error("refresh failed"));
    vi.mocked(api.post).mockResolvedValue({ data: duplicate });
    await expect(useSceneStore.getState().duplicateScene("workspace-1", scene.id, "Copy", null)).resolves.toEqual(duplicate);
    expect(useSceneStore.getState().activeScene?.id).toBe("scene-2");
  });

  it("keeps a moved scene visible when the list refresh fails after PATCH", async () => {
    useSceneStore.setState({ scenes: [scene] });
    vi.mocked(api.patch).mockResolvedValue({ data: { ...scene, folderId: "folder-2" } });
    vi.mocked(api.get).mockRejectedValue(new Error("refresh failed"));
    await expect(useSceneStore.getState().moveScene("workspace-1", scene.id, "folder-2", null)).resolves.toBeUndefined();
    expect(useSceneStore.getState().scenes[0].folderId).toBe("folder-2");
  });
});
