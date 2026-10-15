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
});
