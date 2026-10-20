import { useEffect, useState } from 'react';
import { DashboardShell } from '@/components/dashboard/dashboard-shell';
import { DashboardHeader } from '@/components/dashboard/dashboard-header';
import { PasswordLogin } from '@/components/admin-login';
import { FolderList } from '@/components/folder-list';
import { SceneList } from '@/components/scene-list';
import { SearchResults } from '@/components/search-results';
import { PublicView } from '@/components/public-view';
import { InviteAccept } from '@/components/collaboration/invite-accept';
import { MemberList } from '@/components/collaboration/member-list';
import { AgentSettings } from '@/components/collaboration/agent-settings';
import { ExcalidrawWrapper } from '@/components/whiteboard/excalidraw-wrapper';
import { useLocation, navigate } from '@/lib/router';
import { Button } from '@/components/ui/button';
import useAuthStore from '@/store/auth-store';
import useFolderStore from '@/store/folder-store';
import useSceneStore from '@/store/scene-store';
import useWorkspaceStore from '@/store/workspace-store';
import useSearchStore from '@/store/search-store';

function App() {
  const location = useLocation();
  const pathname = location.pathname;
  const routeThread = new URLSearchParams(location.search).get('thread') ?? undefined;
  const publicToken = pathname.match(/^\/p\/([^/]+)\/?$/)?.[1] ?? null;
  const inviteToken = pathname.match(/^\/invite\/([^/]+)\/?$/)?.[1] ?? null;
  const sceneRoute = pathname.match(/^\/w\/([^/]+)\/scenes\/([^/]+)\/?$/);
  const dashboardRoute = pathname.match(/^\/w\/([^/]+)\/?$/);
  const settingsRoute = pathname.match(/^\/w\/([^/]+)\/settings\/(members|agents)\/?$/);
  const routeWorkspace = sceneRoute?.[1] ?? settingsRoute?.[1] ?? dashboardRoute?.[1];
  const routeScene = sceneRoute?.[2];
  const { checkAuth, isAuthenticated, isLoading } = useAuthStore();
  const { workspaces, activeWorkspaceId, fetchWorkspaces, setActiveWorkspaceId } = useWorkspaceStore();
  const workspace = workspaces.find(item => routeWorkspace ? item.id === routeWorkspace || item.publicId === routeWorkspace : item.id === activeWorkspaceId);
  const { currentFolderId, fetchFolders, reset: resetFolders } = useFolderStore();
  const { activeScene, scenes, fetchScenes, openSceneById, cancelSceneOpen, reset: resetScenes } = useSceneStore();
  const { view: searchView, query: searchQuery, search, reset: resetSearch } = useSearchStore();
  const [ready, setReady] = useState(false); const [routeError, setRouteError] = useState('');

  useEffect(() => { if (!publicToken) void checkAuth(); }, [checkAuth, publicToken]);
  useEffect(() => {
    let current = true; setReady(false);
    if (isAuthenticated && !publicToken) void fetchWorkspaces().finally(() => { if (current) setReady(true); });
    return () => { current = false; };
  }, [fetchWorkspaces, isAuthenticated, publicToken]);
  useEffect(() => {
    resetFolders(); resetScenes(); resetSearch();
    if (activeWorkspaceId && isAuthenticated && !publicToken) void fetchFolders(activeWorkspaceId).catch(() => undefined);
  }, [activeWorkspaceId, fetchFolders, resetFolders, resetScenes, resetSearch, isAuthenticated, publicToken]);
  useEffect(() => {
    let current = true; setRouteError(''); cancelSceneOpen();
    if (!ready || !isAuthenticated || publicToken || inviteToken || !routeWorkspace) return;
    if (!workspace) { setRouteError('You do not have access to this workspace.'); return; }
    if (activeWorkspaceId !== workspace.id) { setActiveWorkspaceId(workspace.id); return; }
    if (routeScene) void openSceneById(workspace.id, routeScene).catch(() => { if (current) setRouteError('This scene is unavailable or you do not have access to it.'); });
    return () => { current = false; };
  }, [pathname, ready, isAuthenticated, publicToken, inviteToken, workspace?.id, activeWorkspaceId, routeWorkspace, routeScene, setActiveWorkspaceId, cancelSceneOpen, openSceneById]);
  useEffect(() => { if (activeWorkspaceId && isAuthenticated && !publicToken) void fetchScenes(activeWorkspaceId, currentFolderId).catch(() => undefined); }, [activeWorkspaceId, currentFolderId, fetchScenes, isAuthenticated, publicToken, pathname]);
  useEffect(() => { if (activeWorkspaceId && searchView !== 'all') void search(activeWorkspaceId, undefined, searchView).catch(() => undefined); }, [activeWorkspaceId, search, searchView]);

  if (publicToken) return <PublicView key={publicToken} token={publicToken} />;
  if (isLoading) return <main className="grid min-h-screen place-items-center">Loading...</main>;
  if (inviteToken) return <InviteAccept key={inviteToken} token={inviteToken} />;
  if (!isAuthenticated) return <PasswordLogin />;
  if (routeError) return <main className="grid min-h-screen place-items-center p-6"><p role="alert" className="text-destructive">{routeError}</p><Button onClick={() => navigate('/')}>Back to dashboard</Button></main>;
  if (sceneRoute) {
    if (!activeScene || !workspace || activeScene.workspaceId !== workspace.id || (activeScene.id !== routeScene && activeScene.publicId !== routeScene)) return <main className="grid min-h-screen place-items-center">Loading scene...</main>;
    return <ExcalidrawWrapper initialThreadId={routeThread} key={activeScene.id} />;
  }
  return <DashboardShell>{settingsRoute ? !ready ? <p>Loading workspace...</p> : workspace?.role !== 'owner' ? <p role="alert">Only workspace owners can manage settings.</p> : settingsRoute[2] === 'members' ? <MemberList key={workspace.id} workspaceId={workspace.id} scenes={scenes} /> : <AgentSettings key={workspace.id} workspaceId={workspace.id} isOwner scenes={scenes} /> : <><DashboardHeader />{searchView !== 'all' || searchQuery.trim() ? <SearchResults /> : <><FolderList /><SceneList /></>}</>}</DashboardShell>;
}
export default App;
