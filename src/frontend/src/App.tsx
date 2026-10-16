import { useEffect, useState } from "react";

import { DashboardShell } from "@/components/dashboard/dashboard-shell";
import { DashboardHeader } from "@/components/dashboard/dashboard-header";
import { PasswordLogin } from "@/components/admin-login";
import { FolderList } from "@/components/folder-list";
import { SceneList } from "@/components/scene-list";
import { SearchResults } from "@/components/search-results";
import { PublicView } from "@/components/public-view";
import { ExcalidrawWrapper } from "@/components/whiteboard/excalidraw-wrapper";
import { usePathname } from "@/lib/router";
import useAuthStore from "@/store/auth-store";
import useFolderStore from "@/store/folder-store";
import useSceneStore from "@/store/scene-store";
import useWorkspaceStore from "@/store/workspace-store";
import useSearchStore from "@/store/search-store";

function App() {
  const pathname = usePathname();
  const publicToken = pathname.match(/^\/p\/([^/]+)\/?$/)?.[1] ?? null;
  const sceneRoute = pathname.match(/^\/w\/([^/]+)\/scenes\/([^/]+)\/?$/);
  const sceneRouteWorkspaceId = sceneRoute?.[1] ?? null;
  const sceneRouteId = sceneRoute?.[2] ?? null;
  const checkAuth = useAuthStore((state) => state.checkAuth);
  const isAuthenticated = useAuthStore((state) => state.isAuthenticated);
  const isLoading = useAuthStore((state) => state.isLoading);
  const workspaces = useWorkspaceStore((state) => state.workspaces);
  const activeWorkspaceId = useWorkspaceStore((state) => state.activeWorkspaceId);
  const workspaceLoading = useWorkspaceStore((state) => state.isLoading);
  const fetchWorkspaces = useWorkspaceStore((state) => state.fetchWorkspaces);
  const setActiveWorkspaceId = useWorkspaceStore((state) => state.setActiveWorkspaceId);
  const currentFolderId = useFolderStore((state) => state.currentFolderId);
  const fetchFolders = useFolderStore((state) => state.fetchFolders);
  const resetFolders = useFolderStore((state) => state.reset);
  const activeScene = useSceneStore((state) => state.activeScene);
  const fetchScenes = useSceneStore((state) => state.fetchScenes);
  const openSceneById = useSceneStore((state) => state.openSceneById);
  const resetScenes = useSceneStore((state) => state.reset);
  const searchView = useSearchStore((state) => state.view);
  const searchQuery = useSearchStore((state) => state.query);
  const search = useSearchStore((state) => state.search);
  const resetSearch = useSearchStore((state) => state.reset);
  const [workspacesReady, setWorkspacesReady] = useState(false);
  const [sceneRouteError, setSceneRouteError] = useState<string | null>(null);

  useEffect(() => {
    if (!publicToken) void checkAuth();
  }, [checkAuth, publicToken]);

  useEffect(() => {
    if (!isAuthenticated) {
      setWorkspacesReady(false);
      return;
    }
    setWorkspacesReady(false);
    void fetchWorkspaces().finally(() => setWorkspacesReady(true));
  }, [fetchWorkspaces, isAuthenticated]);

  useEffect(() => {
    if (!sceneRouteWorkspaceId || !sceneRouteId || !isAuthenticated || !workspacesReady || workspaceLoading) return;
    if (!workspaces.some((workspace) => workspace.id === sceneRouteWorkspaceId)) {
      setSceneRouteError("You do not have access to this workspace.");
      resetScenes();
      return;
    }
    setSceneRouteError(null);
    setActiveWorkspaceId(sceneRouteWorkspaceId);
    resetScenes();
    void openSceneById(sceneRouteWorkspaceId, sceneRouteId).catch(() => {
      setSceneRouteError("This scene is unavailable or you do not have access to it.");
    });
  }, [isAuthenticated, openSceneById, resetScenes, sceneRouteId, sceneRouteWorkspaceId, setActiveWorkspaceId, workspaceLoading, workspaces, workspacesReady]);

  useEffect(() => {
    if (!activeWorkspaceId) {
      resetFolders();
      resetScenes();
      resetSearch();
      return;
    }
    resetFolders();
    resetScenes();
    void fetchFolders(activeWorkspaceId).catch(() => undefined);
  }, [activeWorkspaceId, fetchFolders, resetFolders, resetScenes, resetSearch]);

  useEffect(() => {
    if (!activeWorkspaceId || searchView === "all") return;
    void search(activeWorkspaceId, undefined, searchView).catch(() => undefined);
  }, [activeWorkspaceId, search, searchView]);

  useEffect(() => {
    if (!activeWorkspaceId) return;
    void fetchScenes(activeWorkspaceId, currentFolderId).catch(() => undefined);
  }, [activeWorkspaceId, currentFolderId, fetchScenes]);

  if (publicToken) return <PublicView token={publicToken} />;
  if (isLoading) return <main className="grid min-h-screen place-items-center">Loading...</main>;
  if (!isAuthenticated) return <PasswordLogin />;
  if (sceneRouteWorkspaceId && sceneRouteId) {
    if (sceneRouteError) return <main className="grid min-h-screen place-items-center p-6"><p className="text-destructive" role="alert">{sceneRouteError}</p></main>;
    if (!activeScene || activeWorkspaceId !== sceneRouteWorkspaceId) return <main className="grid min-h-screen place-items-center">Loading scene...</main>;
    return <ExcalidrawWrapper />;
  }
  if (activeScene) return <ExcalidrawWrapper />;

  return (
    <DashboardShell>
      <DashboardHeader />
        {searchView !== "all" || searchQuery.trim() ? <SearchResults /> : (
          <>
            <FolderList />
            <SceneList />
          </>
        )}
    </DashboardShell>
  );
}

export default App;
