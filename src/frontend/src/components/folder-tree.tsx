import { ChevronDown, ChevronRight, Folder, Star, Trash2 } from "lucide-react";
import { useMemo, useState } from "react";

import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import useFolderStore, { buildFolderTree, type FolderNode } from "@/store/folder-store";
import useSearchStore from "@/store/search-store";

function TreeItem({ node }: { node: FolderNode }) {
  const currentFolderId = useFolderStore((state) => state.currentFolderId);
  const setCurrentFolderId = useFolderStore((state) => state.setCurrentFolderId);
  const setView = useSearchStore((state) => state.setView);
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
          onClick={() => { setView("all"); setCurrentFolderId(node.id); }}
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
  const folders = useFolderStore((state) => state.folders);
  const setCurrentFolderId = useFolderStore((state) => state.setCurrentFolderId);
  const setView = useSearchStore((state) => state.setView);
  const tree = useMemo(() => buildFolderTree(folders), [folders]);

  return (
    <nav aria-label="Folders" className="grid gap-1">
      <Button className="justify-start" onClick={() => { setView("all"); setCurrentFolderId(null); }} type="button" variant="ghost">
        <Folder /> All files
      </Button>
      <Button className="justify-start" onClick={() => setView("starred")} type="button" variant="ghost">
        <Star /> Starred
      </Button>
      <Button className="justify-start" onClick={() => setView("trash")} type="button" variant="ghost">
        <Trash2 /> Trash
      </Button>
      {tree.length ? <ul>{tree.map((node) => <TreeItem key={node.id} node={node} />)}</ul> : (
        <p className="px-2 py-3 text-sm text-muted-foreground">No folders yet</p>
      )}
    </nav>
  );
}
