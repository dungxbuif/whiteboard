import { Import, UserPlus } from "lucide-react";
import { useRef, useState } from "react";
import { Button } from "@/components/ui/button";
import { parseSceneImport } from "@/lib/scene-import";
import { navigate } from "@/lib/router";
import useSceneStore from "@/store/scene-store";
import useFolderStore from "@/store/folder-store";
import useWorkspaceStore from "@/store/workspace-store";

export function DashboardHeader() {
  const { workspaces, activeWorkspaceId } = useWorkspaceStore();
  const canEdit = workspaces.find(workspace => workspace.id === activeWorkspaceId)?.role;
  const folderId = useFolderStore(state => state.currentFolderId);
  const createScene = useSceneStore(state => state.createScene);
  const input = useRef<HTMLInputElement>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  async function create(file?: File) {
    if (!activeWorkspaceId || busy) return;
    setBusy(true); setError("");
    try {
      const content = file ? await parseSceneImport(file) : undefined;
      const title = file ? file.name.replace(/\.(excalidraw|json)$/i, "").trim() || "Imported scene" : undefined;
      if (useWorkspaceStore.getState().activeWorkspaceId !== activeWorkspaceId) return;
      const scene = await createScene(activeWorkspaceId, folderId, title, content);
      if (useWorkspaceStore.getState().activeWorkspaceId !== activeWorkspaceId) return;
      navigate(`/w/${activeWorkspaceId}/scenes/${scene.id}`);
    } catch (cause) {
      setError(cause instanceof Error && !('isAxiosError' in cause) ? cause.message : "Could not create scene. Please try again.");
    } finally { setBusy(false); if (input.current) input.current.value = ""; }
  }
  return <header>
    <p className="mb-1 text-xs text-muted-foreground">Workspace</p>
    <h1 className="break-words text-2xl font-semibold">{workspaces.find(w => w.id === activeWorkspaceId)?.name || "Select Workspace"}</h1>
    <div className="mb-8 mt-6 h-px bg-border" />
    {canEdit === "owner" || canEdit === "editor" ? <div className="flex gap-2 max-sm:flex-col">
      <input ref={input} className="hidden" type="file" aria-label="Scene file" accept=".excalidraw,.json" onChange={event => { const file = event.target.files?.[0]; if (file) void create(file); }} />
      <Button className="h-16 gap-2 [&_svg]:size-6" disabled={!activeWorkspaceId || busy} onClick={() => input.current?.click()}><Import />Import scenes</Button>
      <Button className="h-16 gap-2 [&_svg]:size-6" disabled={!activeWorkspaceId || busy} onClick={() => void create()}><UserPlus />New scene</Button>
    </div>
    : <p className="text-sm text-muted-foreground">Viewer access · Open a scene to read it.</p>}
    {busy ? <p className="mt-3 text-sm text-muted-foreground" role="status">Preparing scene...</p> : null}
    {error ? <p className="mt-3 text-sm text-destructive" role="alert">{error}</p> : null}
  </header>;
}
