import { Activity, BookOpen, LayoutDashboard, Settings } from "lucide-react";
import { navigateDashboard, workspaceDashboardPath } from "@/lib/dashboard-navigation";
import { navigate } from "@/lib/router";
import { FolderTree } from "@/components/folder-tree";
import { SearchBar } from "@/components/search-bar";
import { WorkspaceSwitcher } from "@/components/workspace-switcher";
import type { SurfaceTheme } from "@/lib/surface-theme";
import useWorkspaceStore from "@/store/workspace-store";
import { UserMenu } from "./user-menu";
import { NotificationInbox } from "@/components/comments/notification-inbox";
import { CollectionList } from "@/components/collections/collection-list";

export function DashboardSidebar({ theme, onThemeChange, onNavigate }: { theme: SurfaceTheme; onThemeChange: (value: SurfaceTheme) => void; onNavigate?: () => void }) {
  const workspaceId = useWorkspaceStore(state => state.activeWorkspaceId);
  const workspace = useWorkspaceStore(state => state.workspaces.find(item => item.id === state.activeWorkspaceId));
  return (
    <nav aria-label="Workspace navigation" className="dashboard-sidebar flex h-full min-h-0 w-64 flex-col border-r border-sidebar-border">
      <div className="grid min-w-0 gap-3 p-2"><WorkspaceSwitcher /><SearchBar workspaceId={workspaceId} /></div>
      <div className="min-h-0 flex-1 overflow-y-auto px-2 py-3">
        <p className="px-2 pb-2 text-[11px] font-semibold uppercase tracking-wide text-muted-foreground">Your workspace</p>
        <div className="grid gap-1">
          <a className="dashboard-nav-item flex h-9 items-center gap-2 rounded-md px-2 text-sm font-medium" href={workspaceDashboardPath(workspace)} onClick={event => { event.preventDefault(); navigateDashboard(); onNavigate?.(); }}><LayoutDashboard />Dashboard</a>
          {workspace ? <a className="dashboard-nav-item flex h-9 items-center gap-2 rounded-md px-2 text-sm font-medium" href={`/w/${workspace.publicId || workspace.id}/activity`} onClick={event => { event.preventDefault(); navigate(event.currentTarget.getAttribute("href")!); onNavigate?.(); }}><Activity />Activity</a> : null}
          {workspace ? <a className="dashboard-nav-item flex h-9 items-center gap-2 rounded-md px-2 text-sm font-medium" href={`/w/${workspace.publicId || workspace.id}/libraries`} onClick={event => { event.preventDefault(); navigate(event.currentTarget.getAttribute("href")!); onNavigate?.(); }}><BookOpen />Libraries</a> : null}
        </div>
        <FolderTree />
        {workspace ? <CollectionList canManage={workspace.role === "owner" || workspace.role === "editor"} onNavigate={onNavigate} workspaceId={workspace.id} workspaceSlug={workspace.publicId || workspace.id} /> : null}
        {workspace?.role === "owner" && <div className="mt-5 grid gap-1 border-t border-sidebar-border pt-4"><p className="px-2 pb-1 text-[11px] font-semibold uppercase tracking-wide text-muted-foreground">Workspace settings</p>{[["members", "Members"], ["agents", "Agent keys"], ["identity", "Identity & domains"], ["sso", "Single sign-on"], ["scim", "SCIM provisioning"], ["retention", "Retention"]].map(([path, label]) => <a key={path} className="dashboard-nav-item flex h-9 items-center gap-2 rounded-md px-2 text-sm" href={`/w/${workspace.publicId || workspace.id}/settings/${path}`} onClick={event => { event.preventDefault(); navigate(event.currentTarget.getAttribute("href")!); onNavigate?.(); }}><Settings />{label}</a>)}</div>}
      </div>
      <div className="grid gap-1 border-t border-sidebar-border p-2"><NotificationInbox /><UserMenu theme={theme} onThemeChange={onThemeChange} /></div>
    </nav>
  );
}
