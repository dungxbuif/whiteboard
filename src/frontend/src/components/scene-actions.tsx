import { Copy, FolderInput, MoreHorizontal, Pencil, Share2, Star, Trash2 } from "lucide-react";
import { useState } from "react";

import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import type { Scene } from "@/store/scene-store";
import useSceneStore from "@/store/scene-store";
import useWorkspaceStore from "@/store/workspace-store";
import { ShareDialog } from "@/components/share-dialog";
import { SceneDuplicateDialog, SceneMoveDialog } from "@/components/scene-duplicate-dialog";
import useFolderStore from "@/store/folder-store";
import { navigate } from "@/lib/router";
import { workspaceDashboardPath } from "@/lib/dashboard-navigation";

export function SceneActions({ scene }: { scene: Scene }) {
  const [shareOpen, setShareOpen] = useState(false);
  const [duplicateOpen, setDuplicateOpen] = useState(false);
  const [moveOpen, setMoveOpen] = useState(false);
  const activeWorkspaceId = useWorkspaceStore((state) => state.activeWorkspaceId);
  const activeWorkspace = useWorkspaceStore((state) => state.workspaces.find((workspace) => workspace.id === state.activeWorkspaceId));
  const folders = useFolderStore((state) => state.folders);
  const currentFolderId = useFolderStore((state) => state.currentFolderId);
  const renameScene = useSceneStore((state) => state.renameScene);
  const moveScene = useSceneStore((state) => state.moveScene);
  const duplicateScene = useSceneStore((state) => state.duplicateScene);
  const setSceneStar = useSceneStore((state) => state.setSceneStar);
  const trashScene = useSceneStore((state) => state.trashScene);

  async function handleRename() {
    if (!activeWorkspaceId) return;
    const title = window.prompt("Scene name", scene.title)?.trim();
    if (!title || title === scene.title) return;
    await renameScene(activeWorkspaceId, scene.id, title);
  }

  if ((activeWorkspace?.role !== "owner" && activeWorkspace?.role !== "editor") || (scene.access !== "edit" && !scene.owner)) return null;

  return <>
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button aria-label={`Actions for ${scene.title}`} size="icon" type="button" variant="ghost">
          <MoreHorizontal size={16} aria-hidden="true" />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end">
        <DropdownMenuItem onSelect={() => void handleRename()}>
          <Pencil size={16} aria-hidden="true" /> Rename
        </DropdownMenuItem>
        <DropdownMenuItem onSelect={() => setMoveOpen(true)}><FolderInput size={16} aria-hidden="true" /> Move</DropdownMenuItem>
        <DropdownMenuItem onSelect={() => setDuplicateOpen(true)}><Copy size={16} aria-hidden="true" /> Duplicate</DropdownMenuItem>
        <DropdownMenuItem
          onSelect={() => {
            if (activeWorkspaceId) void setSceneStar(activeWorkspaceId, scene.id, !scene.isStarred);
          }}
        >
          <Star size={16} aria-hidden="true" /> {scene.isStarred ? "Unstar" : "Star"}
        </DropdownMenuItem>
        {scene.owner && <DropdownMenuItem onSelect={() => setShareOpen(true)}>
          <Share2 size={16} aria-hidden="true" /> Share
        </DropdownMenuItem>}
        <DropdownMenuItem
          className="text-destructive focus:text-destructive"
          onSelect={() => {
            if (activeWorkspaceId) void trashScene(activeWorkspaceId, scene.id);
          }}
        >
          <Trash2 size={16} aria-hidden="true" /> Move to trash
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
    <SceneMoveDialog folderId={scene.folderId} folders={folders} open={moveOpen} onOpenChange={setMoveOpen} onMove={async (folderId) => {
      if (!activeWorkspaceId) return;
      await moveScene(activeWorkspaceId, scene.id, folderId, currentFolderId);
    }} />
    <SceneDuplicateDialog title={scene.title} folderId={scene.folderId} folders={folders} open={duplicateOpen} onOpenChange={setDuplicateOpen} onDuplicate={async (title, folderId) => {
      if (!activeWorkspaceId) return;
      const copy = await duplicateScene(activeWorkspaceId, scene.id, title, folderId);
      navigate(`${workspaceDashboardPath(activeWorkspace)}/scenes/${encodeURIComponent(copy.publicId || copy.id)}`);
    }} />
    <ShareDialog onOpenChange={setShareOpen} open={shareOpen} scene={scene} />
  </>;
}
