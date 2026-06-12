import { ChevronRight, Home } from "lucide-react";
import { useMemo } from "react";

import { Button } from "@/components/ui/button";
import useFolderStore, { getFolderPath } from "@/store/folder-store";

export function FolderBreadcrumbs() {
  const folders = useFolderStore((state) => state.folders);
  const currentFolderId = useFolderStore((state) => state.currentFolderId);
  const setCurrentFolderId = useFolderStore((state) => state.setCurrentFolderId);
  const path = useMemo(() => getFolderPath(folders, currentFolderId), [folders, currentFolderId]);

  return (
    <nav aria-label="Folder breadcrumbs" className="flex flex-wrap items-center gap-1">
      <Button aria-label="All files" onClick={() => setCurrentFolderId(null)} size="sm" type="button" variant="ghost">
        <Home /> All files
      </Button>
      {path.map((folder) => (
        <span className="flex items-center gap-1" key={folder.id}>
          <ChevronRight className="h-4 w-4 text-muted-foreground" />
          <Button onClick={() => setCurrentFolderId(folder.id)} size="sm" type="button" variant="ghost">
            {folder.name}
          </Button>
        </span>
      ))}
    </nav>
  );
}
