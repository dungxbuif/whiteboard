import { Folder, Plus } from "lucide-react";
import { useMemo, useState } from "react";

import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import useFolderStore from "@/store/folder-store";
import useWorkspaceStore from "@/store/workspace-store";
import { FolderActions } from "@/components/folder-actions";

export function FolderList() {
  const folders = useFolderStore((state) => state.folders);
  const currentFolderId = useFolderStore((state) => state.currentFolderId);
  const createFolder = useFolderStore((state) => state.createFolder);
  const setCurrentFolderId = useFolderStore((state) => state.setCurrentFolderId);
  const activeWorkspaceId = useWorkspaceStore((state) => state.activeWorkspaceId);
  const role = useWorkspaceStore(state => state.workspaces.find(workspace => workspace.id === state.activeWorkspaceId)?.role);
  const canEdit = role === "owner" || role === "editor";
  const [open, setOpen] = useState(false);
  const [name, setName] = useState("");
  const [isSubmitting, setIsSubmitting] = useState(false);

  const visibleFolders = useMemo(
    () => folders.filter((folder) => folder.parentId === currentFolderId).sort((a, b) => a.name.localeCompare(b.name)),
    [folders, currentFolderId],
  );

  async function handleCreate(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!activeWorkspaceId || !name.trim()) return;
    setIsSubmitting(true);
    try {
      await createFolder(activeWorkspaceId, name.trim(), currentFolderId);
      setName("");
      setOpen(false);
    } finally {
      setIsSubmitting(false);
    }
  }

  return (
    <section className="grid gap-4" aria-label="Folder contents">
      <div className="flex items-center justify-between gap-4">
        <h2 className="text-lg font-semibold">Folders</h2>
        {canEdit && <Dialog open={open} onOpenChange={setOpen}>
          <DialogTrigger asChild>
            <Button disabled={!activeWorkspaceId} size="sm" type="button" variant="outline">
              <Plus /> New folder
            </Button>
          </DialogTrigger>
          <DialogContent>
            <form className="grid gap-4" onSubmit={handleCreate}>
              <DialogHeader>
                <DialogTitle>Create folder</DialogTitle>
                <DialogDescription>The folder will be created in the current location.</DialogDescription>
              </DialogHeader>
              <Input
                autoFocus
                aria-label="Folder name"
                onChange={(event) => setName(event.target.value)}
                placeholder="Folder name"
                value={name}
              />
              <DialogFooter>
                <Button disabled={isSubmitting || !name.trim()} type="submit">
                  {isSubmitting ? "Creating..." : "Create"}
                </Button>
              </DialogFooter>
            </form>
          </DialogContent>
        </Dialog>}
      </div>

      {visibleFolders.length ? (
        <div className="grid grid-cols-[repeat(auto-fill,minmax(210px,1fr))] gap-3">
          {visibleFolders.map((folder) => (
            <div className="flex items-center gap-2 rounded-md border bg-card p-2 shadow-sm" key={folder.id}>
              <Button
                className="h-auto min-w-0 flex-1 justify-start p-2 text-left"
                onClick={() => setCurrentFolderId(folder.id)}
                type="button"
                variant="ghost"
              >
                <Folder className="text-primary" />
                <span className="truncate">{folder.name}</span>
              </Button>
              {canEdit && <FolderActions folder={folder} />}
            </div>
          ))}
        </div>
      ) : (
        <div className="rounded-lg border border-dashed p-8 text-center text-sm text-muted-foreground">
          No folders in this location.
        </div>
      )}
    </section>
  );
}
