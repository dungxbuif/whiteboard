import { MoreHorizontal, Pencil, Share2, Star, Trash2 } from "lucide-react";
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

export function SceneActions({ scene }: { scene: Scene }) {
  const [shareOpen, setShareOpen] = useState(false);
  const activeWorkspaceId = useWorkspaceStore((state) => state.activeWorkspaceId);
  const renameScene = useSceneStore((state) => state.renameScene);
  const setSceneStar = useSceneStore((state) => state.setSceneStar);
  const trashScene = useSceneStore((state) => state.trashScene);

  async function handleRename() {
    if (!activeWorkspaceId) return;
    const title = window.prompt("Scene name", scene.title)?.trim();
    if (!title || title === scene.title) return;
    await renameScene(activeWorkspaceId, scene.id, title);
  }

  return (
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
        <DropdownMenuItem
          onSelect={() => {
            if (activeWorkspaceId) void setSceneStar(activeWorkspaceId, scene.id, !scene.isStarred);
          }}
        >
          <Star size={16} aria-hidden="true" /> {scene.isStarred ? "Unstar" : "Star"}
        </DropdownMenuItem>
        <DropdownMenuItem onSelect={() => setShareOpen(true)}>
          <Share2 size={16} aria-hidden="true" /> Share
        </DropdownMenuItem>
        <DropdownMenuItem
          className="text-destructive focus:text-destructive"
          onSelect={() => {
            if (activeWorkspaceId) void trashScene(activeWorkspaceId, scene.id);
          }}
        >
          <Trash2 size={16} aria-hidden="true" /> Move to trash
        </DropdownMenuItem>
      </DropdownMenuContent>
      <ShareDialog onOpenChange={setShareOpen} open={shareOpen} scene={scene} />
    </DropdownMenu>
  );
}
