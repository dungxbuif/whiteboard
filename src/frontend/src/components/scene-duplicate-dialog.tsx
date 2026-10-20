import { useEffect, useState, type FormEvent } from "react";

import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import type { Folder } from "@/store/folder-store";

function FolderSelect({ label, folders, value, onChange }: { label: string; folders: Folder[]; value: string; onChange: (value: string) => void }) {
  return <label className="grid gap-2 text-sm font-medium">{label}<select aria-label={label} className="h-10 rounded-md border border-input bg-background px-3 text-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring" value={value} onChange={(event) => onChange(event.target.value)}><option value="">Workspace root</option>{folders.map((folder) => <option key={folder.id} value={folder.id}>{folder.name}</option>)}</select></label>;
}

interface CommonProps { folderId: string | null; folders: Folder[]; open: boolean; onOpenChange: (open: boolean) => void }

export function SceneDuplicateDialog({ title, folderId, folders, open, onOpenChange, onDuplicate }: CommonProps & { title: string; onDuplicate: (title: string, folderId: string | null) => Promise<void> }) {
  const [name, setName] = useState(`${title} copy`);
  const [target, setTarget] = useState(folderId ?? "");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  useEffect(() => { if (open) { setName(`${title} copy`); setTarget(folderId ?? ""); setError(""); } }, [open, title, folderId]);
  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!name.trim()) { setError("Copy name is required"); return; }
    setBusy(true); setError("");
    try { await onDuplicate(name.trim(), target || null); onOpenChange(false); }
    catch { setError("Could not duplicate scene"); }
    finally { setBusy(false); }
  }
  return <Dialog open={open} onOpenChange={onOpenChange}><DialogContent><DialogHeader><DialogTitle>Duplicate scene</DialogTitle><DialogDescription>Create a copy of the saved scene in this workspace.</DialogDescription></DialogHeader><form className="grid gap-4" onSubmit={(event) => void submit(event)}><label className="grid gap-2 text-sm font-medium">Copy name<Input autoFocus aria-label="Copy name" maxLength={200} value={name} onChange={(event) => setName(event.target.value)} /></label><FolderSelect label="Copy to" folders={folders} value={target} onChange={setTarget} />{error ? <p role="alert" className="text-sm text-destructive">{error}</p> : null}<DialogFooter><Button type="button" variant="outline" onClick={() => onOpenChange(false)}>Cancel</Button><Button type="submit" disabled={busy || !name.trim()}>{busy ? "Duplicating..." : "Duplicate"}</Button></DialogFooter></form></DialogContent></Dialog>;
}

export function SceneMoveDialog({ folderId, folders, open, onOpenChange, onMove }: CommonProps & { onMove: (folderId: string | null) => Promise<void> }) {
  const [target, setTarget] = useState(folderId ?? "");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  useEffect(() => { if (open) { setTarget(folderId ?? ""); setError(""); } }, [open, folderId]);
  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault(); setBusy(true); setError("");
    try { await onMove(target || null); onOpenChange(false); }
    catch { setError("Could not move scene"); }
    finally { setBusy(false); }
  }
  return <Dialog open={open} onOpenChange={onOpenChange}><DialogContent><DialogHeader><DialogTitle>Move scene</DialogTitle><DialogDescription>Choose a folder in this workspace.</DialogDescription></DialogHeader><form className="grid gap-4" onSubmit={(event) => void submit(event)}><FolderSelect label="Move to" folders={folders} value={target} onChange={setTarget} />{error ? <p role="alert" className="text-sm text-destructive">{error}</p> : null}<DialogFooter><Button type="button" variant="outline" onClick={() => onOpenChange(false)}>Cancel</Button><Button type="submit" disabled={busy || target === (folderId ?? "")}>{busy ? "Moving..." : "Move"}</Button></DialogFooter></form></DialogContent></Dialog>;
}
