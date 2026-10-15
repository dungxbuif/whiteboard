import { useState } from "react";

import api from "@/lib/api";
import type { Scene } from "@/store/scene-store";
import useWorkspaceStore from "@/store/workspace-store";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";

interface ShareDialogProps {
  scene: Scene;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function ShareDialog({ scene, open, onOpenChange }: ShareDialogProps) {
  const workspaceId = useWorkspaceStore((state) => state.activeWorkspaceId);
  const [active, setActive] = useState(true);
  const [password, setPassword] = useState("");
  const [shareURL, setShareURL] = useState("");
  const [error, setError] = useState("");
  const [isSaving, setIsSaving] = useState(false);

  async function saveShare() {
    if (!workspaceId) return;
    setIsSaving(true);
    setError("");
    try {
      const response = await api.put<{ token: string }>(`/scenes/${scene.id}/share`, { active, password }, { headers: { "X-Workspace-ID": workspaceId } });
      setShareURL(`${window.location.origin}/p/${response.data.token}`);
    } catch {
      setError("Sharing settings could not be saved.");
    } finally {
      setIsSaving(false);
    }
  }

  async function copyShareURL() {
    if (shareURL) await navigator.clipboard?.writeText(shareURL);
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Share {scene.title}</DialogTitle>
          <DialogDescription>Create a read-only link for this scene.</DialogDescription>
        </DialogHeader>
        <div className="grid gap-4">
          <label className="flex items-center gap-2 text-sm font-medium">
            <input checked={active} onChange={(event) => setActive(event.target.checked)} type="checkbox" /> Link active
          </label>
          <label className="grid gap-2 text-sm font-medium">
            Password (optional)
            <Input minLength={6} onChange={(event) => setPassword(event.target.value)} placeholder="At least 6 characters" value={password} />
          </label>
          {error ? <p className="text-sm text-destructive" role="alert">{error}</p> : null}
          {shareURL ? (
            <div className="flex gap-2">
              <Input aria-label="Share URL" readOnly value={shareURL} />
              <Button onClick={() => void copyShareURL()} type="button">Copy</Button>
            </div>
          ) : null}
        </div>
        <DialogFooter>
          <Button disabled={isSaving} onClick={() => void saveShare()} type="button">{isSaving ? "Saving..." : "Save link"}</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
