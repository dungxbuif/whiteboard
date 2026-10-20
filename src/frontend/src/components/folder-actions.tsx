import { MoreHorizontal, Pencil, Star, Trash2 } from "lucide-react";
import { useState } from "react";

import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import type { Folder } from "@/store/folder-store";
import useFolderStore from "@/store/folder-store";
import useWorkspaceStore from "@/store/workspace-store";
import { FolderManagementDialog } from "@/components/folder-management-dialog";

export function FolderActions({ folder }: { folder: Folder }) {
  const [manageOpen, setManageOpen] = useState(false);
  const activeWorkspaceId = useWorkspaceStore((state) => state.activeWorkspaceId);
  const role = useWorkspaceStore((state) => state.workspaces.find((workspace) => workspace.id === state.activeWorkspaceId)?.role);
  const folders = useFolderStore((state) => state.folders);
  const updateFolder = useFolderStore((state) => state.updateFolder);
  const setFolderStar = useFolderStore((state) => state.setFolderStar);
  const trashFolder = useFolderStore((state) => state.trashFolder);

  if (role !== "owner" && role !== "editor") return null;

  return <>
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button aria-label={`Actions for ${folder.name}`} size="icon" type="button" variant="ghost">
          <MoreHorizontal size={16} aria-hidden="true" />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end">
        <DropdownMenuItem onSelect={() => setManageOpen(true)}><Pencil size={16} aria-hidden="true" /> Rename or move</DropdownMenuItem>
        <DropdownMenuItem onSelect={() => {
          if (activeWorkspaceId) void setFolderStar(activeWorkspaceId, folder.id);
        }}>
          <Star size={16} aria-hidden="true" /> {folder.isStarred ? "Unstar" : "Star"}
        </DropdownMenuItem>
        <DropdownMenuItem className="text-destructive focus:text-destructive" onSelect={() => {
          if (activeWorkspaceId) void trashFolder(activeWorkspaceId, folder.id);
        }}>
          <Trash2 size={16} aria-hidden="true" /> Move to trash
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
    <FolderManagementDialog folder={folder} folders={folders} open={manageOpen} onOpenChange={setManageOpen} onSave={async (name, parentId) => {
      if (!activeWorkspaceId) return;
      const changes: { name?: string; newParentId?: string | null } = {};
      if (name !== folder.name) changes.name = name;
      if (parentId !== folder.parentId) changes.newParentId = parentId;
      if (Object.keys(changes).length) await updateFolder(activeWorkspaceId, folder.id, changes);
    }} />
  </>;
}
