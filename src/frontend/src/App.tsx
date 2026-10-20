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
import { DomainSettings } from '@/components/enterprise/domain-settings';
import { SSOSettings } from '@/components/enterprise/sso-settings';
import { RetentionSettings } from '@/components/enterprise/retention-settings';
import { SCIMSettings } from '@/components/enterprise/scim-settings';
import { LibraryPanel } from '@/components/library/library-panel';
import { ActivityPanel } from '@/components/activity/activity-panel';
import { CollectionView } from '@/components/collections/collection-view';
import { PresentationView } from '@/components/presentation/presentation-view';
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
  const presentationToken = pathname.match(/^\/present\/([^/]+)\/?$/)?.[1] ?? null;
  const inviteToken = pathname.match(/^\/invite\/([^/]+)\/?$/)?.[1] ?? null;
  const sceneRoute = pathname.match(/^\/w\/([^/]+)\/scenes\/([^/]+)\/?$/);
  const dashboardRoute = pathname.match(/^\/w\/([^/]+)\/?$/);
  const activityRoute = pathname.match(/^\/w\/([^/]+)\/activity\/?$/);
  const collectionRoute = pathname.match(/^\/w\/([^/]+)\/collections\/([^/]+)\/?$/);
  const settingsRoute = pathname.match(/^\/w\/([^/]+)\/settings\/(members|agents|identity|sso|retention|scim)\/?$/);
  const libraryRoute = pathname.match(/^\/w\/([^/]+)\/libraries\/?$/);
  const routeWorkspace = sceneRoute?.[1] ?? collectionRoute?.[1] ?? settingsRoute?.[1] ?? activityRoute?.[1] ?? libraryRoute?.[1] ?? dashboardRoute?.[1];
  const routeScene = sceneRoute?.[2];
  const routeCollection = collectionRoute?.[2];
  const { checkAuth, isAuthenticated, isLoading } = useAuthStore();
  const { workspaces, activeWorkspaceId, fetchWorkspaces, setActiveWorkspaceId } = useWorkspaceStore();
  const workspace = workspaces.find(item => routeWorkspace ? item.id === routeWorkspace || item.publicId === routeWorkspace : item.id === activeWorkspaceId);
  const { currentFolderId, fetchFolders, reset: resetFolders } = useFolderStore();
  const { activeScene, scenes, fetchScenes, openSceneById, cancelSceneOpen, reset: resetScenes } = useSceneStore();
  const { view: searchView, query: searchQuery, results: searchResults, search, reset: resetSearch } = useSearchStore();
  const [ready, setReady] = useState(false); const [routeError, setRouteError] = useState('');

  useEffect(() => { if (!publicToken&&!presentationToken) void checkAuth(); }, [checkAuth, publicToken,presentationToken]);
  useEffect(() => {
    let current = true; setReady(false);
    if (isAuthenticated && !publicToken&&!presentationToken) void fetchWorkspaces().finally(() => { if (current) setReady(true); });
    return () => { current = false; };
  }, [fetchWorkspaces, isAuthenticated, publicToken,presentationToken]);
  useEffect(() => {
    resetFolders(); resetScenes(); resetSearch();
    if (activeWorkspaceId && isAuthenticated && !publicToken && !presentationToken) void fetchFolders(activeWorkspaceId).catch(() => undefined);
  }, [activeWorkspaceId, fetchFolders, resetFolders, resetScenes, resetSearch, isAuthenticated, publicToken, presentationToken]);
  useEffect(() => {
    let current = true; setRouteError(''); cancelSceneOpen();
    if (!ready || !isAuthenticated || publicToken || presentationToken || inviteToken || !routeWorkspace) return;
    if (!workspace) { setRouteError('You do not have access to this workspace.'); return; }
    if (activeWorkspaceId !== workspace.id) { setActiveWorkspaceId(workspace.id); return; }
    if (routeScene) void openSceneById(workspace.id, routeScene).catch(() => { if (current) setRouteError('This scene is unavailable or you do not have access to it.'); });
    return () => { current = false; };
  }, [pathname, ready, isAuthenticated, publicToken, presentationToken, inviteToken, workspace?.id, activeWorkspaceId, routeWorkspace, routeScene, setActiveWorkspaceId, cancelSceneOpen, openSceneById]);
  useEffect(() => { if (activeWorkspaceId && isAuthenticated && !publicToken && !presentationToken) void fetchScenes(activeWorkspaceId, currentFolderId).catch(() => undefined); }, [activeWorkspaceId, currentFolderId, fetchScenes, isAuthenticated, publicToken, presentationToken, pathname]);
  useEffect(() => { if (activeWorkspaceId && searchView !== 'all') void search(activeWorkspaceId, undefined, searchView).catch(() => undefined); }, [activeWorkspaceId, search, searchView]);
  useEffect(() => { if (activeWorkspaceId && routeCollection) void search(activeWorkspaceId, '', 'all').catch(() => undefined); }, [activeWorkspaceId, search, routeCollection]);

  if (publicToken) return <PublicView key={publicToken} token={publicToken} />;
  if (presentationToken) return <PresentationView key={presentationToken} token={presentationToken}/>;
  if (isLoading) return <main className="grid min-h-screen place-items-center">Loading...</main>;
  if (inviteToken) return <InviteAccept key={inviteToken} token={inviteToken} />;
  if (!isAuthenticated) return <PasswordLogin />;
  if (routeError) return <main className="grid min-h-screen place-items-center p-6"><p role="alert" className="text-destructive">{routeError}</p><Button onClick={() => navigate('/')}>Back to dashboard</Button></main>;
  if (sceneRoute) {
    if (!activeScene || !workspace || activeScene.workspaceId !== workspace.id || (activeScene.id !== routeScene && activeScene.publicId !== routeScene)) return <main className="grid min-h-screen place-items-center">Loading scene...</main>;
    return <ExcalidrawWrapper initialThreadId={routeThread} key={activeScene.id} />;
  }
  return <DashboardShell>{libraryRoute ? !ready || !workspace ? <p>Loading workspace...</p> : <section aria-labelledby="libraries-heading" className="grid gap-4"><div><h1 className="text-2xl font-semibold" id="libraries-heading">Workspace libraries</h1><p className="text-sm text-muted-foreground">Reusable native Excalidraw libraries for this workspace.</p></div>{scenes[0] ? <LibraryPanel canManage={workspace.role === 'owner' || workspace.role === 'editor'} currentSequence="0" onInserted={() => undefined} sceneId={scenes[0].id} workspaceId={workspace.id} /> : <p>Create a scene before inserting a library.</p>}</section> : collectionRoute ? !ready || !workspace || !routeCollection ? <p>Loading workspace...</p> : <CollectionView availableScenes={searchResults.filter(item => item.type === 'scene').map(item => ({ id: item.id, title: item.name }))} canManage={workspace.role === 'owner' || workspace.role === 'editor'} collectionId={routeCollection} key={routeCollection} workspaceId={workspace.id} workspaceSlug={workspace.publicId || workspace.id} /> : activityRoute ? !ready || !workspace ? <p>Loading workspace...</p> : <section aria-labelledby="activity-heading" className="grid gap-4"><div><h1 className="text-2xl font-semibold" id="activity-heading">Activity</h1><p className="text-sm text-muted-foreground">Recent changes visible to you in this workspace.</p></div><ActivityPanel key={workspace.id} workspaceId={workspace.id} /></section> : settingsRoute ? !ready ? <p>Loading workspace...</p> : workspace?.role !== 'owner' ? <p role="alert">Only workspace owners can manage settings.</p> : settingsRoute[2] === 'members' ? <MemberList key={workspace.id} workspaceId={workspace.id} scenes={scenes} /> : settingsRoute[2] === 'identity' ? <DomainSettings key={workspace.id} workspaceId={workspace.id} /> : settingsRoute[2] === 'sso' ? <SSOSettings key={workspace.id} workspaceId={workspace.id} /> : settingsRoute[2] === 'retention' ? <RetentionSettings key={workspace.id} workspaceId={workspace.id} /> : settingsRoute[2] === 'scim' ? <SCIMSettings key={workspace.id} workspaceId={workspace.id} /> : <AgentSettings key={workspace.id} workspaceId={workspace.id} isOwner scenes={scenes} /> : <><DashboardHeader />{searchView !== 'all' || searchQuery.trim() ? <SearchResults /> : <><FolderList /><SceneList /></>}</>}</DashboardShell>;
}
export default App;
