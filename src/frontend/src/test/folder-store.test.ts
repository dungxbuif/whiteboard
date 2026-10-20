import { beforeEach, describe, expect, it, vi } from "vitest";

import api from "../lib/api";
import useFolderStore, { buildFolderTree, getFolderPath, type Folder } from "../store/folder-store";
import useWorkspaceStore from "../store/workspace-store";

vi.mock("../lib/api", () => ({
  default: {
    get: vi.fn(),
    post: vi.fn(),
    patch: vi.fn(),
    delete: vi.fn(),
  },
}));

const root: Folder = {
  id: "root",
  workspaceId: "workspace",
  parentId: null,
  name: "Root",
  path: "/root/",
  depth: 0,
  isStarred: false,
  createdAt: "date",
  updatedAt: "date",
};

const child: Folder = {
  ...root,
  id: "child",
  parentId: "root",
  name: "Child",
  path: "/root/child/",
  depth: 1,
};

describe("folder store", () => {
  beforeEach(() => {
    useWorkspaceStore.setState({ activeWorkspaceId: null });
    useFolderStore.getState().reset();
    vi.clearAllMocks();
  });

  it("builds a nested tree and breadcrumb path", () => {
    const tree = buildFolderTree([child, root]);
    expect(tree).toHaveLength(1);
    expect(tree[0].children[0].id).toBe("child");
    expect(getFolderPath([root, child], "child").map((folder) => folder.id)).toEqual(["root", "child"]);
  });

  it("fetches the flat folder list with workspace isolation", async () => {
    vi.mocked(api.get).mockResolvedValue({ data: [root, child] });

    await useFolderStore.getState().fetchFolders("workspace");

    expect(api.get).toHaveBeenCalledWith("/folders?all=true", {
      headers: { "X-Workspace-ID": "workspace" },
    });
    expect(useFolderStore.getState().folders).toEqual([root, child]);
  });

  it("creates a child in the current location", async () => {
    vi.mocked(api.post).mockResolvedValue({ data: child });

    await useFolderStore.getState().createFolder("workspace", "Child", "root");

    expect(api.post).toHaveBeenCalledWith(
      "/folders/",
      { name: "Child", parentId: "root" },
      { headers: { "X-Workspace-ID": "workspace" } },
    );
    expect(useFolderStore.getState().folders).toContainEqual(child);
  });

  it("renames a folder and reloads paths while keeping the selected folder", async () => {
    vi.mocked(api.patch).mockResolvedValue({ data: undefined });
    vi.mocked(api.get).mockResolvedValue({ data: [{ ...root, name: "Renamed" }, child] });
    useFolderStore.getState().setCurrentFolderId("root");

    await useFolderStore.getState().renameFolder("workspace", "root", "Renamed");

    expect(api.patch).toHaveBeenCalledWith("/folders/root", { name: "Renamed" }, { headers: { "X-Workspace-ID": "workspace" } });
    expect(useFolderStore.getState().folders[0].name).toBe("Renamed");
    expect(useFolderStore.getState().currentFolderId).toBe("root");
  });

  it("updates folder name and parent in one request", async () => {
    vi.mocked(api.patch).mockResolvedValue({ data: undefined });
    vi.mocked(api.get).mockResolvedValue({ data: [{ ...root, name: "Renamed", parentId: "other" }] });
    await useFolderStore.getState().updateFolder("workspace", "root", { name: "Renamed", newParentId: "other" });
    expect(api.patch).toHaveBeenCalledWith("/folders/root", { name: "Renamed", newParentId: "other" }, { headers: { "X-Workspace-ID": "workspace" } });
  });

  it("preserves folder state when refresh fails after a successful PATCH", async () => {
    useFolderStore.setState({ folders: [root] });
    vi.mocked(api.patch).mockResolvedValue({ data: undefined });
    vi.mocked(api.get).mockRejectedValue(new Error("refresh failed"));
    await expect(useFolderStore.getState().renameFolder("workspace", "root", "Renamed")).resolves.toBeUndefined();
    expect(useFolderStore.getState().folders[0].name).toBe("Renamed");
  });

  it("keeps a moved folder when the legacy move route succeeds but refresh fails", async () => {
    const other = { ...root, id: "other", path: "/other/" };
    useFolderStore.setState({ folders: [root, child, other] });
    vi.mocked(api.patch).mockResolvedValue({ data: undefined });
    vi.mocked(api.get).mockRejectedValue(new Error("refresh failed"));
    await expect(useFolderStore.getState().moveFolder("workspace", "root", "other")).resolves.toBeUndefined();
    expect(useFolderStore.getState().folders[0].parentId).toBe("other");
    expect(useFolderStore.getState().folders[0].path).toBe("/other/root/");
    expect(useFolderStore.getState().folders[1].path).toBe("/other/root/child/");
    expect(useFolderStore.getState().folders[1].depth).toBe(2);
  });

  it("ignores an in-flight folder update after switching workspaces", async () => {
    let resolvePatch!: (value: { data: undefined }) => void;
    useWorkspaceStore.setState({ activeWorkspaceId: "workspace" });
    useFolderStore.setState({ folders: [root] });
    vi.mocked(api.patch).mockReturnValue(new Promise((resolve) => { resolvePatch = resolve; }));
    const pending = useFolderStore.getState().renameFolder("workspace", "root", "Renamed");
    useWorkspaceStore.setState({ activeWorkspaceId: "other-workspace" });
    useFolderStore.getState().reset();
    const other = { ...root, id: "other", workspaceId: "other-workspace" };
    useFolderStore.setState({ folders: [other] });
    resolvePatch({ data: undefined });
    await pending;
    expect(useFolderStore.getState().folders).toEqual([other]);
    expect(api.get).not.toHaveBeenCalled();
  });

  it("ignores an old folder refresh after switching workspaces", async () => {
    let resolveGet!: (value: { data: Folder[] }) => void;
    useWorkspaceStore.setState({ activeWorkspaceId: "workspace" });
    vi.mocked(api.get).mockReturnValue(new Promise((resolve) => { resolveGet = resolve; }));
    const pending = useFolderStore.getState().fetchFolders("workspace");
    useWorkspaceStore.setState({ activeWorkspaceId: "other-workspace" });
    useFolderStore.getState().reset();
    const other = { ...root, id: "other", workspaceId: "other-workspace" };
    useFolderStore.setState({ folders: [other] });
    resolveGet({ data: [root] });
    await pending;
    expect(useFolderStore.getState().folders).toEqual([other]);
  });

  it("does not append a folder created before a workspace switch", async () => {
    let resolvePost!: (value: { data: Folder }) => void;
    useWorkspaceStore.setState({ activeWorkspaceId: "workspace" });
    vi.mocked(api.post).mockReturnValue(new Promise((resolve) => { resolvePost = resolve; }));
    const pending = useFolderStore.getState().createFolder("workspace", "Child", null);
    useWorkspaceStore.setState({ activeWorkspaceId: "other-workspace" });
    useFolderStore.getState().reset();
    const other = { ...root, id: "other", workspaceId: "other-workspace" };
    useFolderStore.setState({ folders: [other] });
    resolvePost({ data: child });
    await pending;
    expect(useFolderStore.getState().folders).toEqual([other]);
  });

  it.each(["star", "trash"])("does not let an old %s completion discard the new workspace fetch", async (action) => {
    let resolveMutation!: (value: { data: undefined }) => void;
    let resolveGet!: (value: { data: Folder[] }) => void;
    useWorkspaceStore.setState({ activeWorkspaceId: "workspace" });
    useFolderStore.setState({ folders: [root] });
    const mutation = new Promise<{ data: undefined }>((resolve) => { resolveMutation = resolve; });
    if (action === "star") vi.mocked(api.post).mockReturnValue(mutation);
    else vi.mocked(api.delete).mockReturnValue(mutation);
    vi.mocked(api.get).mockReturnValue(new Promise((resolve) => { resolveGet = resolve; }));

    const pendingMutation = action === "star"
      ? useFolderStore.getState().setFolderStar("workspace", "root")
      : useFolderStore.getState().trashFolder("workspace", "root");
    useWorkspaceStore.setState({ activeWorkspaceId: "other-workspace" });
    useFolderStore.getState().reset();
    const pendingFetch = useFolderStore.getState().fetchFolders("other-workspace");
    resolveMutation({ data: undefined });
    resolveGet({ data: [{ ...root, id: "other", workspaceId: "other-workspace" }] });
    await Promise.all([pendingMutation, pendingFetch]);

    expect(api.get).toHaveBeenCalledTimes(1);
    expect(api.get).toHaveBeenCalledWith("/folders?all=true", { headers: { "X-Workspace-ID": "other-workspace" } });
    expect(useFolderStore.getState().folders.map((folder) => folder.id)).toEqual(["other"]);
    expect(useFolderStore.getState().isLoading).toBe(false);
  });
});
