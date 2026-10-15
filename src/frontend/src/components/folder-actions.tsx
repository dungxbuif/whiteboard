import { MoreHorizontal, Star, Trash2 } from "lucide-react";

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

export function FolderActions({ folder }: { folder: Folder }) {
  const activeWorkspaceId = useWorkspaceStore((state) => state.activeWorkspaceId);
  const setFolderStar = useFolderStore((state) => state.setFolderStar);
  const trashFolder = useFolderStore((state) => state.trashFolder);

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button aria-label={`Actions for ${folder.name}`} size="icon" type="button" variant="ghost">
          <MoreHorizontal size={16} aria-hidden="true" />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end">
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
  );
}
