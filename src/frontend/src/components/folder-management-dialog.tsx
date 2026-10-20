import { useEffect, useMemo, useState, type FormEvent } from "react";

import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import type { Folder } from "@/store/folder-store";

interface Props {
  folder: Folder;
  folders: Folder[];
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSave: (name: string, parentId: string | null) => Promise<void>;
}

export function FolderManagementDialog({ folder, folders, open, onOpenChange, onSave }: Props) {
  const [name, setName] = useState(folder.name);
  const [parentId, setParentId] = useState(folder.parentId ?? "");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  useEffect(() => {
    if (open) { setName(folder.name); setParentId(folder.parentId ?? ""); setError(""); }
  }, [open, folder.id, folder.name, folder.parentId]);

  const destinations = useMemo(() => {
    const byId = new Map(folders.map((item) => [item.id, item]));
    const subtreeDepth = Math.max(0, ...folders.filter((item) => item.path.startsWith(folder.path)).map((item) => item.depth - folder.depth));
    return folders.filter((candidate) => {
      if (candidate.id === folder.id || candidate.workspaceId !== folder.workspaceId) return false;
      let current: Folder | undefined = candidate;
      while (current) {
        if (current.id === folder.id) return false;
        current = current.parentId ? byId.get(current.parentId) : undefined;
      }
      return candidate.depth + 1 + subtreeDepth < 30;
    }).sort((left, right) => left.name.localeCompare(right.name));
  }, [folder, folders]);

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const trimmed = name.trim();
    if (!trimmed) { setError("Folder name is required"); return; }
    setBusy(true); setError("");
    try { await onSave(trimmed, parentId || null); onOpenChange(false); }
    catch { setError("Could not update folder"); }
    finally { setBusy(false); }
  }

  return <Dialog open={open} onOpenChange={onOpenChange}>
    <DialogContent>
      <DialogHeader><DialogTitle>Manage folder</DialogTitle><DialogDescription>Rename this folder or choose a new location.</DialogDescription></DialogHeader>
      <form className="grid gap-4" onSubmit={(event) => void submit(event)}>
        <label className="grid gap-2 text-sm font-medium">Folder name<Input autoFocus aria-label="Folder name" maxLength={100} value={name} onChange={(event) => setName(event.target.value)} /></label>
        <label className="grid gap-2 text-sm font-medium">Move to
          <select aria-label="Move to" className="h-10 rounded-md border border-input bg-background px-3 text-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring" value={parentId} onChange={(event) => setParentId(event.target.value)}>
            <option value="">Workspace root</option>
            {destinations.map((item) => <option key={item.id} value={item.id}>{item.name}</option>)}
          </select>
        </label>
        {error ? <p role="alert" className="text-sm text-destructive">{error}</p> : null}
        <DialogFooter><Button type="button" variant="outline" onClick={() => onOpenChange(false)}>Cancel</Button><Button type="submit" disabled={busy || !name.trim()}>{busy ? "Saving..." : "Save changes"}</Button></DialogFooter>
      </form>
    </DialogContent>
  </Dialog>;
}
