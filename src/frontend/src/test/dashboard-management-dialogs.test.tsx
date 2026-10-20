import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { FolderManagementDialog } from "../components/folder-management-dialog";
import { SceneDuplicateDialog, SceneMoveDialog } from "../components/scene-duplicate-dialog";
import type { Folder } from "../store/folder-store";

const root: Folder = { id: "root", workspaceId: "w", parentId: null, name: "Root", path: "/root/", depth: 0, isStarred: false, createdAt: "date", updatedAt: "date" };
const child: Folder = { ...root, id: "child", parentId: "root", name: "Child", path: "/root/child/", depth: 1 };
const other: Folder = { ...root, id: "other", name: "Other", path: "/other/" };

describe("dashboard management dialogs", () => {
  it("submits folder name and target, excluding itself and descendants", async () => {
    const onSave = vi.fn().mockResolvedValue(undefined);
    render(<FolderManagementDialog folder={root} folders={[root, child, other]} open onOpenChange={vi.fn()} onSave={onSave} />);
    expect(screen.queryByRole("option", { name: "Child" })).not.toBeInTheDocument();
    fireEvent.change(screen.getByRole("textbox", { name: "Folder name" }), { target: { value: "Renamed" } });
    fireEvent.change(screen.getByRole("combobox", { name: "Move to" }), { target: { value: "other" } });
    fireEvent.click(screen.getByRole("button", { name: "Save changes" }));
    await waitFor(() => expect(onSave).toHaveBeenCalledWith("Renamed", "other"));
  });

  it("submits scene duplicate title and folder", async () => {
    const onDuplicate = vi.fn().mockResolvedValue(undefined);
    render(<SceneDuplicateDialog title="Scene" folderId={null} folders={[root, other]} open onOpenChange={vi.fn()} onDuplicate={onDuplicate} />);
    fireEvent.change(screen.getByRole("textbox", { name: "Copy name" }), { target: { value: "Copied" } });
    fireEvent.change(screen.getByRole("combobox", { name: "Copy to" }), { target: { value: "other" } });
    fireEvent.click(screen.getByRole("button", { name: "Duplicate" }));
    await waitFor(() => expect(onDuplicate).toHaveBeenCalledWith("Copied", "other"));
  });

  it("moves a scene to the selected folder", async () => {
    const onMove = vi.fn().mockResolvedValue(undefined);
    render(<SceneMoveDialog folderId={null} folders={[root, other]} open onOpenChange={vi.fn()} onMove={onMove} />);
    fireEvent.change(screen.getByRole("combobox", { name: "Move to" }), { target: { value: "other" } });
    fireEvent.click(screen.getByRole("button", { name: "Move" }));
    await waitFor(() => expect(onMove).toHaveBeenCalledWith("other"));
  });
});
