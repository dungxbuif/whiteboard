import { navigate } from '@/lib/router';
import useWorkspaceStore, { type Workspace } from '@/store/workspace-store';
import useFolderStore from '@/store/folder-store';
import useSearchStore, { type SearchView } from '@/store/search-store';

type DashboardSelection = { folderId?: string | null; view?: SearchView; query?: string };

export function workspaceDashboardPath(workspace?: Pick<Workspace, 'id' | 'publicId'>): string {
  return workspace ? `/w/${encodeURIComponent(workspace.publicId || workspace.id)}` : '/';
}

export function navigateDashboard({ folderId = null, view = 'all', query = '' }: DashboardSelection = {}) {
  const { activeWorkspaceId, workspaces } = useWorkspaceStore.getState();
  useFolderStore.getState().setCurrentFolderId(folderId);
  useSearchStore.getState().setView(view);
  useSearchStore.getState().setQuery(query);
  navigate(workspaceDashboardPath(workspaces.find(workspace => workspace.id === activeWorkspaceId)));
}

export function selectWorkspace(workspace: Workspace) {
  useWorkspaceStore.getState().setActiveWorkspaceId(workspace.id);
  navigateDashboard();
}
