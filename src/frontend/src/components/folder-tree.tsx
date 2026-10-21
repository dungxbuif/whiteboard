import { ChevronDown, ChevronRight, Folder, Star, Trash2 } from "lucide-react";
import { useMemo, useState } from "react";

import { Button } from "@/components/ui/button";
import { navigateDashboard } from "@/lib/dashboard-navigation";
import { cn } from "@/lib/utils";
import useFolderStore, { buildFolderTree, type FolderNode } from "@/store/folder-store";
import useWorkspaceStore from "@/store/workspace-store";

function TreeItem({ node }: { node: FolderNode }) {
  const currentFolderId = useFolderStore((state) => state.currentFolderId);
  const [expanded, setExpanded] = useState(true);
  const hasChildren = node.children.length > 0;

  return (
    <li>
      <div className="flex items-center" style={{ paddingLeft: `${node.depth * 12}px` }}>
        <Button
          aria-label={`${expanded ? "Collapse" : "Expand"} ${node.name}`}
          className={cn("h-7 w-7", !hasChildren && "invisible")}
          onClick={() => setExpanded((value) => !value)}
          size="icon"
          type="button"
          variant="ghost"
        >
          {expanded ? <ChevronDown /> : <ChevronRight />}
        </Button>
        <Button
          className={cn("h-8 min-w-0 flex-1 justify-start px-2", currentFolderId === node.id && "bg-accent")}
          onClick={() => navigateDashboard({ folderId: node.id })}
          type="button"
          variant="ghost"
        >
          <Folder />
          <span className="truncate">{node.name}</span>
        </Button>
      </div>
      {hasChildren && expanded ? (
        <ul>{node.children.map((child) => <TreeItem key={child.id} node={child} />)}</ul>
      ) : null}
    </li>
  );
}

export function FolderTree() {
  const role = useWorkspaceStore(state => state.workspaces.find(workspace => workspace.id === state.activeWorkspaceId)?.role);
  const folders = useFolderStore((state) => state.folders);
  const tree = useMemo(() => buildFolderTree(folders), [folders]);

  return (
    <nav aria-label="Folders" className="mt-3 grid gap-1 border-t border-sidebar-border pt-3">
      <p className="px-2 pb-1 text-[11px] font-semibold uppercase tracking-wide text-muted-foreground">Files</p>
      <Button className="dashboard-nav-item h-9 justify-start rounded-md px-2 text-sm" onClick={() => navigateDashboard()} type="button" variant="ghost">
        <Folder /> All files
      </Button>
      <Button className="dashboard-nav-item h-9 justify-start rounded-md px-2 text-sm" onClick={() => navigateDashboard({ view: "starred" })} type="button" variant="ghost">
        <Star /> Starred
      </Button>
      {role === "owner" && <Button className="dashboard-nav-item h-9 justify-start rounded-md px-2 text-sm" onClick={() => navigateDashboard({ view: "trash" })} type="button" variant="ghost">
        <Trash2 /> Trash
      </Button>}
      {tree.length ? <ul>{tree.map((node) => <TreeItem key={node.id} node={node} />)}</ul> : (
        <p className="px-2 py-2 text-xs text-muted-foreground">No folders yet</p>
      )}
    </nav>
  );
}
