import { fireEvent, render, screen } from "@testing-library/react";
import { beforeEach, expect, it, vi } from "vitest";
import { DashboardHeader } from "../components/dashboard/dashboard-header";
import useSceneStore, { type Scene } from "../store/scene-store";
import useWorkspaceStore from "../store/workspace-store";

const { navigate } = vi.hoisted(() => ({ navigate: vi.fn() }));
vi.mock("../lib/router", () => ({ navigate }));

beforeEach(() => {
  navigate.mockClear();
  useSceneStore.getState().reset();
  useWorkspaceStore.setState({ activeWorkspaceId: "w1", workspaces: [
    { id: "w1", name: "First", ownerId: "u", createdAt: "date", role: "owner" },
    { id: "w2", name: "Second", ownerId: "u", createdAt: "date", role: "owner" },
  ] });
});

it("does not navigate to a scene created in a previous workspace", async () => {
  let resolveCreate!: (value: Scene) => void;
  const createScene = vi.fn(() => new Promise<Scene>((resolve) => { resolveCreate = resolve; }));
  useSceneStore.setState({ createScene });
  render(<DashboardHeader />);
  fireEvent.click(screen.getByRole("button", { name: "New scene" }));
  expect(createScene).toHaveBeenCalled();
  useWorkspaceStore.setState({ activeWorkspaceId: "w2" });
  resolveCreate({ id: "s1", workspaceId: "w1", folderId: null, title: "Untitled", content: { elements: [], appState: {}, files: {} }, contentVersion: 1, isStarred: false, createdAt: "date", updatedAt: "date" });
  await Promise.resolve();
  await Promise.resolve();
  expect(navigate).not.toHaveBeenCalled();
});
