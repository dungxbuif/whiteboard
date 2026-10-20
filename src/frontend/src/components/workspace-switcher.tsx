import { Check, ChevronsUpDown, Plus, Folder, Pencil } from "lucide-react";
import * as React from "react";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { selectWorkspace } from "@/lib/dashboard-navigation";
import useWorkspaceStore from "@/store/workspace-store";

export function WorkspaceSwitcher() {
  const { workspaces, activeWorkspaceId } = useWorkspaceStore();
  const createWorkspace = useWorkspaceStore((state) => state.createWorkspace);
  const renameWorkspace = useWorkspaceStore((state) => state.renameWorkspace);
  const [createDialogOpen, setCreateDialogOpen] = React.useState(false);
  const [workspaceName, setWorkspaceName] = React.useState("");
  const [createError, setCreateError] = React.useState("");
  const [isCreating, setIsCreating] = React.useState(false);
  const [renameDialogOpen, setRenameDialogOpen] = React.useState(false);
  const [renameName, setRenameName] = React.useState("");
  const [renameError, setRenameError] = React.useState("");
  const [isRenaming, setIsRenaming] = React.useState(false);

  const activeWorkspace = Array.isArray(workspaces) ? workspaces.find((w) => w.id === activeWorkspaceId) : null;

  return (
    <>
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button
          variant="ghost"
          size="sm"
          role="combobox"
          className="h-14 w-full justify-between px-2 font-medium hover:bg-accent"
        >
          <div className="flex items-center gap-2 overflow-hidden">
            <div className="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg border bg-background">
              <span aria-hidden="true" className="text-xs font-semibold">{(activeWorkspace?.name || "W").slice(0, 2).toUpperCase()}</span>
            </div>
            <span className="truncate">{activeWorkspace?.name || "Select Workspace"}</span>
          </div>
          <ChevronsUpDown className="ml-2 h-4 w-4 shrink-0 opacity-50" />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="start" className="w-[240px]">
        {Array.isArray(workspaces) && workspaces.map((workspace) => (
          <DropdownMenuItem
            key={workspace.id}
            onClick={() => selectWorkspace(workspace)}
            className="flex items-center justify-between"
          >
            <div className="flex min-w-0 items-center gap-2">
              <Folder className="h-4 w-4" />
              <span className="truncate">{workspace.name}</span>
            </div>
            {activeWorkspaceId === workspace.id && <Check className="h-4 w-4" />}
          </DropdownMenuItem>
        ))}
        {activeWorkspace?.role === "owner" && <>
          <DropdownMenuSeparator />
          <DropdownMenuItem className="gap-2" onSelect={() => { setRenameName(activeWorkspace.name); setRenameError(""); setRenameDialogOpen(true); }}>
            <Pencil className="h-4 w-4" /> Rename workspace
          </DropdownMenuItem>
        </>}
        <DropdownMenuSeparator />
        <DropdownMenuItem
          className="gap-2 font-medium"
          onSelect={() => {
            setWorkspaceName("");
            setCreateError("");
            setCreateDialogOpen(true);
          }}
        >
          <div className="flex h-6 w-6 items-center justify-center rounded-md border bg-background">
            <Plus className="h-4 w-4" />
          </div>
          <div className="text-muted-foreground">New Workspace</div>
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
    <Dialog open={createDialogOpen} onOpenChange={setCreateDialogOpen}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Create workspace</DialogTitle>
          <DialogDescription>Give this workspace a name so you can organize scenes and folders.</DialogDescription>
        </DialogHeader>
        <form
          className="grid gap-4"
          onSubmit={async (event) => {
            event.preventDefault();
            const name = workspaceName.trim();
            if (!name) {
              setCreateError("Workspace name is required");
              return;
            }
            setIsCreating(true);
            setCreateError("");
            try {
              const workspace = await createWorkspace(name);
              selectWorkspace(workspace);
              setCreateDialogOpen(false);
            } catch {
              setCreateError("Could not create workspace");
            } finally {
              setIsCreating(false);
            }
          }}
        >
          <label className="grid gap-2 text-sm font-medium" htmlFor="workspace-name">
            Workspace name
            <Input autoFocus id="workspace-name" onChange={(event) => setWorkspaceName(event.target.value)} value={workspaceName} />
          </label>
          {createError ? <p className="text-sm text-destructive" role="alert">{createError}</p> : null}
          <DialogFooter>
            <Button disabled={isCreating} onClick={() => setCreateDialogOpen(false)} type="button" variant="outline">Cancel</Button>
            <Button disabled={isCreating} type="submit">{isCreating ? "Creating..." : "Create"}</Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
    <Dialog open={renameDialogOpen} onOpenChange={setRenameDialogOpen}>
      <DialogContent>
        <DialogHeader><DialogTitle>Rename workspace</DialogTitle><DialogDescription>Choose a name for this workspace.</DialogDescription></DialogHeader>
        <form className="grid gap-4" onSubmit={async (event) => {
          event.preventDefault();
          if (!activeWorkspace) return;
          const name = renameName.trim();
          if (name.length < 3) { setRenameError("Workspace name must be at least 3 characters"); return; }
          setIsRenaming(true); setRenameError("");
          try { await renameWorkspace(activeWorkspace.id, name); setRenameDialogOpen(false); }
          catch { setRenameError("Could not rename workspace"); }
          finally { setIsRenaming(false); }
        }}>
          <label className="grid gap-2 text-sm font-medium" htmlFor="rename-workspace-name">Workspace name<Input autoFocus id="rename-workspace-name" maxLength={100} value={renameName} onChange={(event) => setRenameName(event.target.value)} /></label>
          {renameError ? <p className="text-sm text-destructive" role="alert">{renameError}</p> : null}
          <DialogFooter><Button type="button" variant="outline" onClick={() => setRenameDialogOpen(false)}>Cancel</Button><Button type="submit" disabled={isRenaming || renameName.trim().length < 3}>{isRenaming ? "Saving..." : "Save"}</Button></DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
    </>
  );
}
