import { FilePenLine, Folder, RotateCcw, Star, Trash2 } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import useFolderStore from "@/store/folder-store";
import useSceneStore from "@/store/scene-store";
import useSearchStore, { SEARCH_VIEW_TRASH, type SearchItem } from "@/store/search-store";
import useWorkspaceStore from "@/store/workspace-store";

function SearchResultCard({ item }: { item: SearchItem }) {
  const activeWorkspaceId = useWorkspaceStore((state) => state.activeWorkspaceId);
  const view = useSearchStore((state) => state.view);
  const restore = useSearchStore((state) => state.restore);
  const setCurrentFolderId = useFolderStore((state) => state.setCurrentFolderId);
  const setView = useSearchStore((state) => state.setView);
  const openSceneById = useSceneStore((state) => state.openSceneById);

  const open = () => {
    if (item.type === "folder") {
      setView("all");
      setCurrentFolderId(item.id);
      return;
    }
    if (activeWorkspaceId) void openSceneById(activeWorkspaceId, item.id).catch(() => undefined);
  };

  return (
    <Card>
      <CardHeader className="space-y-3">
        <div className="flex items-center justify-between text-muted-foreground">
          {item.type === "folder" ? <Folder size={18} aria-hidden="true" /> : <FilePenLine size={18} aria-hidden="true" />}
          {item.isStarred ? <Star className="fill-current text-amber-500" size={15} aria-label="Starred" /> : null}
        </div>
        <CardTitle className="truncate text-lg">{item.name}</CardTitle>
      </CardHeader>
      <CardContent className="flex gap-2">
        {view === SEARCH_VIEW_TRASH ? (
          <Button onClick={() => activeWorkspaceId && void restore(activeWorkspaceId, item)} type="button" variant="outline">
            <RotateCcw size={16} aria-hidden="true" /> Restore
          </Button>
        ) : (
          <Button onClick={open} type="button" variant="outline">
            Open
          </Button>
        )}
      </CardContent>
    </Card>
  );
}

export function SearchResults() {
  const results = useSearchStore((state) => state.results);
  const isLoading = useSearchStore((state) => state.isLoading);
  const error = useSearchStore((state) => state.error);
  const view = useSearchStore((state) => state.view);

  const heading = view === "starred" ? "Starred" : view === "trash" ? "Trash" : "Search results";

  return (
    <section className="grid gap-4" aria-label={heading}>
      <div className="flex items-center gap-2">
        {view === "trash" ? <Trash2 size={18} aria-hidden="true" /> : null}
        <h2 className="text-lg font-semibold">{heading}</h2>
      </div>
      {isLoading ? <p className="text-sm text-muted-foreground">Loading results...</p> : null}
      {error ? <p className="text-sm text-destructive" role="alert">{error}</p> : null}
      {!isLoading && !error && results.length === 0 ? (
        <div className="rounded-lg border border-dashed p-8 text-center text-sm text-muted-foreground">No matching items.</div>
      ) : null}
      {results.length > 0 ? (
        <div className="grid grid-cols-[repeat(auto-fill,minmax(220px,1fr))] gap-4">
          {results.map((item) => <SearchResultCard key={`${item.type}-${item.id}`} item={item} />)}
        </div>
      ) : null}
    </section>
  );
}
