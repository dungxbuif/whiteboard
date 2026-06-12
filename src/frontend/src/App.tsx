import { Plus, Search } from "lucide-react";
import { useEffect } from "react";

import { AdminLogin } from "@/components/admin-login";
import { FolderBreadcrumbs } from "@/components/folder-breadcrumbs";
import { FolderList } from "@/components/folder-list";
import { FolderTree } from "@/components/folder-tree";
import { SceneList } from "@/components/scene-list";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { ExcalidrawWrapper } from "@/components/whiteboard/excalidraw-wrapper";
import { WorkspaceSwitcher } from "@/components/workspace-switcher";
import useAuthStore from "@/store/auth-store";
import useFolderStore from "@/store/folder-store";
import useSceneStore from "@/store/scene-store";
import useWorkspaceStore from "@/store/workspace-store";

function App() {
  const checkAuth = useAuthStore((state) => state.checkAuth);
  const isAuthenticated = useAuthStore((state) => state.isAuthenticated);
  const isLoading = useAuthStore((state) => state.isLoading);
  const workspaces = useWorkspaceStore((state) => state.workspaces);
  const activeWorkspaceId = useWorkspaceStore((state) => state.activeWorkspaceId);
  const currentFolderId = useFolderStore((state) => state.currentFolderId);
  const fetchFolders = useFolderStore((state) => state.fetchFolders);
  const resetFolders = useFolderStore((state) => state.reset);
  const activeScene = useSceneStore((state) => state.activeScene);
  const fetchScenes = useSceneStore((state) => state.fetchScenes);
  const createScene = useSceneStore((state) => state.createScene);
  const resetScenes = useSceneStore((state) => state.reset);

  useEffect(() => {
    checkAuth();
  }, [checkAuth]);

  useEffect(() => {
    if (!activeWorkspaceId) {
      resetFolders();
      resetScenes();
      return;
    }
    resetFolders();
    resetScenes();
    void fetchFolders(activeWorkspaceId).catch(() => undefined);
  }, [activeWorkspaceId, fetchFolders, resetFolders, resetScenes]);

  useEffect(() => {
    if (!activeWorkspaceId) return;
    void fetchScenes(activeWorkspaceId, currentFolderId).catch(() => undefined);
  }, [activeWorkspaceId, currentFolderId, fetchScenes]);

  if (isLoading) return <main className="grid min-h-screen place-items-center">Loading...</main>;
  if (!isAuthenticated) return <AdminLogin />;
  if (activeScene) return <ExcalidrawWrapper />;

  const activeWorkspace = workspaces.find((workspace) => workspace.id === activeWorkspaceId);

  return (
    <main className="grid min-h-screen grid-cols-[248px_minmax(0,1fr)] bg-background max-md:grid-cols-1">
      <aside className="flex flex-col gap-6 border-r bg-card px-4 py-6 max-md:border-b max-md:border-r-0" aria-label="Workspace navigation">
        <WorkspaceSwitcher />
        <div className="max-h-[calc(100vh-7rem)] overflow-y-auto pr-2 max-md:max-h-none">
          <FolderTree />
        </div>
      </aside>

      <section className="grid content-start gap-7 p-7 max-sm:p-4">
        <FolderBreadcrumbs />
        <header className="flex items-center justify-between gap-5 max-md:flex-col max-md:items-stretch">
          <div>
            <p className="mb-1 text-xs font-bold uppercase text-muted-foreground">Workspace</p>
            <h1 className="text-3xl font-semibold leading-tight">{activeWorkspace?.name || "Select Workspace"}</h1>
          </div>
          <div className="flex items-center gap-3 max-md:flex-col max-md:items-stretch">
            <label className="relative block w-80 max-w-full">
              <Search className="absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground" size={16} aria-hidden="true" />
              <Input aria-label="Search scenes" className="pl-9" placeholder="Search scenes" />
            </label>
            <Button
              disabled={!activeWorkspaceId}
              onClick={() => {
                if (activeWorkspaceId) void createScene(activeWorkspaceId, currentFolderId);
              }}
              type="button"
            >
              <Plus size={16} aria-hidden="true" /> New scene
            </Button>
          </div>
        </header>

        <FolderList />
        <SceneList />
      </section>
    </main>
  );
}

export default App;
