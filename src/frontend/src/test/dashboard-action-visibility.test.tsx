import { fireEvent, render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it } from "vitest";
import { FolderActions } from "../components/folder-actions";
import { SceneActions } from "../components/scene-actions";
import useFolderStore, { type Folder } from "../store/folder-store";
import useSceneStore, { type Scene } from "../store/scene-store";
import useWorkspaceStore from "../store/workspace-store";

const folder: Folder = { id: "f", workspaceId: "w", parentId: null, name: "Folder", path: "/f/", depth: 0, isStarred: false, createdAt: "date", updatedAt: "date" };
const scene: Scene = { id: "s", workspaceId: "w", folderId: null, title: "Scene", content: { elements: [], appState: {}, files: {} }, contentVersion: 1, isStarred: false, createdAt: "date", updatedAt: "date", access: "edit" };

describe("dashboard management visibility", () => {
  beforeEach(() => {
    globalThis.ResizeObserver = class { observe() {} unobserve() {} disconnect() {} };
    useFolderStore.setState({ folders: [folder], currentFolderId: null });
    useSceneStore.getState().reset();
    useWorkspaceStore.setState({ activeWorkspaceId: "w", workspaces: [{ id: "w", name: "Workspace", ownerId: "u", createdAt: "date", role: "viewer" }] });
  });

  it("hides folder and scene actions from workspace viewers", () => {
    render(<><FolderActions folder={folder} /><SceneActions scene={scene} /></>);
    expect(screen.queryByRole("button", { name: /Actions for/ })).not.toBeInTheDocument();
  });

  it("shows folder and scene management to editors", () => {
    useWorkspaceStore.setState({ workspaces: [{ id: "w", name: "Workspace", ownerId: "u", createdAt: "date", role: "editor" }] });
    render(<><FolderActions folder={folder} /><SceneActions scene={scene} /></>);
    expect(screen.getByRole("button", { name: "Actions for Folder" })).toBeInTheDocument();
    fireEvent.pointerDown(screen.getByRole("button", { name: "Actions for Scene" }), { button: 0, ctrlKey: false, pointerType: "mouse" });
    expect(screen.getByRole("menuitem", { name: "Move" })).toBeInTheDocument();
    expect(screen.getByRole("menuitem", { name: "Duplicate" })).toBeInTheDocument();
  });
});
