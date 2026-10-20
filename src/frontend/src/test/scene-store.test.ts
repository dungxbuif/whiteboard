import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import api from "../lib/api";
import useFolderStore from "../store/folder-store";
import useSceneStore, { SCENE_SAVE_DEBOUNCE_MS, syncPresentationOrder, type Scene, type SceneContent } from "../store/scene-store";
import useWorkspaceStore from "../store/workspace-store";

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
    useWorkspaceStore.setState({ activeWorkspaceId: null });
    useFolderStore.getState().reset();
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
    const duplicate = { ...scene, id: "scene-2", title: "Scene copy", folderId: "folder-2" };
    vi.mocked(api.get).mockResolvedValue({ data: [scene] });
    vi.mocked(api.post).mockResolvedValue({ data: duplicate });

    const result = await useSceneStore.getState().duplicateScene("workspace-1", scene.id, "Scene copy", "folder-2");

    expect(api.post).toHaveBeenCalledWith("/scenes/scene-1/duplicate", { title: "Scene copy", folderId: "folder-2" }, { headers: { "X-Workspace-ID": "workspace-1" } });
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
    vi.mocked(api.get).mockRejectedValue(new Error("refresh failed"));
    vi.mocked(api.post).mockResolvedValue({ data: duplicate });
    await expect(useSceneStore.getState().duplicateScene("workspace-1", scene.id, "Copy", null)).resolves.toEqual(duplicate);
    expect(useSceneStore.getState().activeScene?.id).toBe("scene-2");
  });

  it("removes a moved scene from its old folder when refresh fails after PATCH", async () => {
    useSceneStore.setState({ scenes: [scene] });
    vi.mocked(api.patch).mockResolvedValue({ data: { ...scene, folderId: "folder-2" } });
    vi.mocked(api.get).mockRejectedValue(new Error("refresh failed"));
    await expect(useSceneStore.getState().moveScene("workspace-1", scene.id, "folder-2", "folder-1")).resolves.toBeUndefined();
    expect(useSceneStore.getState().scenes).toEqual([]);
  });

  it("removes a scene moved out of the root when refresh fails after PATCH", async () => {
    useSceneStore.setState({ scenes: [scene] });
    vi.mocked(api.patch).mockResolvedValue({ data: { ...scene, folderId: "folder-2" } });
    vi.mocked(api.get).mockRejectedValue(new Error("refresh failed"));
    await useSceneStore.getState().moveScene("workspace-1", scene.id, "folder-2", null);
    expect(useSceneStore.getState().scenes).toEqual([]);
  });

  it("does not retain a duplicate in the root when its destination is a folder and refresh fails", async () => {
    const duplicate = { ...scene, id: "scene-2", folderId: "folder-2" };
    useFolderStore.getState().setCurrentFolderId(null);
    useSceneStore.setState({ scenes: [scene] });
    vi.mocked(api.post).mockResolvedValue({ data: duplicate });
    vi.mocked(api.get).mockRejectedValue(new Error("refresh failed"));
    await expect(useSceneStore.getState().duplicateScene("workspace-1", scene.id, "Copy", "folder-2")).resolves.toEqual(duplicate);
    expect(useSceneStore.getState().scenes).toEqual([scene]);
    expect(useSceneStore.getState().activeScene).toEqual(duplicate);
  });

  it("ignores a move response after switching workspaces", async () => {
    let resolvePatch!: (value: { data: Scene }) => void;
    useWorkspaceStore.setState({ activeWorkspaceId: "workspace-1" });
    useSceneStore.setState({ scenes: [scene] });
    vi.mocked(api.patch).mockReturnValue(new Promise((resolve) => { resolvePatch = resolve; }));
    const pending = useSceneStore.getState().moveScene("workspace-1", scene.id, "folder-2", null);
    useWorkspaceStore.setState({ activeWorkspaceId: "workspace-2" });
    useSceneStore.getState().reset();
    const other = { ...scene, id: "scene-other", workspaceId: "workspace-2" };
    useSceneStore.setState({ scenes: [other] });
    resolvePatch({ data: { ...scene, folderId: "folder-2" } });
    await pending;
    expect(useSceneStore.getState().scenes).toEqual([other]);
    expect(api.get).not.toHaveBeenCalled();
  });

  it("ignores a duplicate and list response after switching workspaces", async () => {
    let resolvePost!: (value: { data: Scene }) => void;
    useWorkspaceStore.setState({ activeWorkspaceId: "workspace-1" });
    vi.mocked(api.post).mockReturnValue(new Promise((resolve) => { resolvePost = resolve; }));
    const pending = useSceneStore.getState().duplicateScene("workspace-1", scene.id, "Copy", null);
    useWorkspaceStore.setState({ activeWorkspaceId: "workspace-2" });
    useSceneStore.getState().reset();
    const other = { ...scene, id: "scene-other", workspaceId: "workspace-2" };
    useSceneStore.setState({ scenes: [other] });
    resolvePost({ data: { ...scene, id: "scene-copy" } });
    await pending;
    expect(useSceneStore.getState().scenes).toEqual([other]);
    expect(useSceneStore.getState().activeScene).toBeNull();
    expect(api.get).not.toHaveBeenCalled();
  });

  it("ignores an old scene refresh after switching workspaces", async () => {
    let resolveGet!: (value: { data: Scene[] }) => void;
    useWorkspaceStore.setState({ activeWorkspaceId: "workspace-1" });
    vi.mocked(api.get).mockReturnValue(new Promise((resolve) => { resolveGet = resolve; }));
    const pending = useSceneStore.getState().fetchScenes("workspace-1", null);
    useWorkspaceStore.setState({ activeWorkspaceId: "workspace-2" });
    useSceneStore.getState().reset();
    const other = { ...scene, id: "scene-other", workspaceId: "workspace-2" };
    useSceneStore.setState({ scenes: [other] });
    resolveGet({ data: [scene] });
    await pending;
    expect(useSceneStore.getState().scenes).toEqual([other]);
  });

  it("does not append or activate a scene created before a workspace switch", async () => {
    let resolvePost!: (value: { data: Scene }) => void;
    useWorkspaceStore.setState({ activeWorkspaceId: "workspace-1" });
    vi.mocked(api.post).mockReturnValue(new Promise((resolve) => { resolvePost = resolve; }));
    const pending = useSceneStore.getState().createScene("workspace-1", null);
    useWorkspaceStore.setState({ activeWorkspaceId: "workspace-2" });
    useSceneStore.getState().reset();
    const other = { ...scene, id: "other", workspaceId: "workspace-2" };
    useSceneStore.setState({ scenes: [other] });
    resolvePost({ data: scene });
    await pending;
    expect(useSceneStore.getState().scenes).toEqual([other]);
    expect(useSceneStore.getState().activeScene).toBeNull();
  });

  it("does not open a scene fetched before a workspace switch", async () => {
    let resolveGet!: (value: { data: Scene }) => void;
    useWorkspaceStore.setState({ activeWorkspaceId: "workspace-1" });
    vi.mocked(api.get).mockReturnValue(new Promise((resolve) => { resolveGet = resolve; }));
    const pending = useSceneStore.getState().openSceneById("workspace-1", scene.id);
    useWorkspaceStore.setState({ activeWorkspaceId: "workspace-2" });
    useSceneStore.getState().reset();
    const other = { ...scene, id: "other", workspaceId: "workspace-2" };
    useSceneStore.setState({ scenes: [other] });
    resolveGet({ data: scene });
    await pending;
    expect(useSceneStore.getState().scenes).toEqual([other]);
    expect(useSceneStore.getState().activeScene).toBeNull();
  });
});
