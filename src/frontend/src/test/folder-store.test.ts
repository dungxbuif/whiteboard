import { beforeEach, describe, expect, it, vi } from "vitest";

import api from "../lib/api";
import useFolderStore, { buildFolderTree, getFolderPath, type Folder } from "../store/folder-store";

vi.mock("../lib/api", () => ({
  default: {
    get: vi.fn(),
    post: vi.fn(),
    patch: vi.fn(),
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
});
