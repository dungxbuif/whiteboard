import { FolderTree } from "@/components/folder-tree";
import { SearchBar } from "@/components/search-bar";
import { WorkspaceSwitcher } from "@/components/workspace-switcher";
import type { SurfaceTheme } from "@/lib/surface-theme";
import useWorkspaceStore from "@/store/workspace-store";
import { UserMenu } from "./user-menu";

export function DashboardSidebar({ theme, onThemeChange }: { theme: SurfaceTheme; onThemeChange: (value: SurfaceTheme) => void }) {
  const workspaceId = useWorkspaceStore(state => state.activeWorkspaceId);
  return (
    <nav aria-label="Workspace navigation" className="dashboard-sidebar flex h-full min-h-0 w-64 flex-col">
      <div className="grid min-w-0 grid-cols-1 gap-3 p-2"><WorkspaceSwitcher /><SearchBar workspaceId={workspaceId} /></div>
      <div className="min-h-0 flex-1 overflow-y-auto px-2 py-4">
        <p className="px-2 pb-2 text-xs font-medium text-muted-foreground">Your workspace</p>
        <FolderTree />
      </div>
      <div className="p-2"><UserMenu theme={theme} onThemeChange={onThemeChange} /></div>
    </nav>
  );
}
