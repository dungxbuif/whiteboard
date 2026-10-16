import { Star } from "lucide-react";

import { SceneActions } from "@/components/scene-actions";
import { Button } from "@/components/ui/button";
import { ScenePreview } from "@/components/dashboard/scene-preview";
import { navigate } from "@/lib/router";
import useSceneStore from "@/store/scene-store";
import useWorkspaceStore from "@/store/workspace-store";

export function SceneList() {
  const scenes = useSceneStore((state) => state.scenes);
  const isLoading = useSceneStore((state) => state.isLoading);
  const error = useSceneStore((state) => state.error);
  const openScene = useSceneStore((state) => state.openScene);
  const activeWorkspaceId = useWorkspaceStore((state) => state.activeWorkspaceId);

  return (
    <section className="grid gap-4" aria-label="Scenes">
      <h2 className="text-lg font-semibold">Scenes</h2>
      {isLoading ? <p className="text-sm text-muted-foreground">Loading scenes...</p> : null}
      {error ? <p className="text-sm text-destructive" role="alert">{error}</p> : null}
      {!isLoading && !error && scenes.length === 0 ? (
        <div className="rounded-lg border border-dashed p-8 text-center text-sm text-muted-foreground">
          No scenes in this location.
        </div>
      ) : null}
      {scenes.length > 0 ? (
        <div className="grid grid-cols-[repeat(auto-fill,minmax(220px,1fr))] gap-4">
          {scenes.map((scene) => (
            <article key={scene.id} className="group relative overflow-hidden rounded-lg border bg-card shadow-sm transition-shadow hover:shadow-md">
              <ScenePreview content={scene.content} />
              <div className="grid gap-3 p-3">
                <div className="flex min-w-0 items-center gap-2">
                  {scene.isStarred ? <Star className="shrink-0 fill-current text-amber-500" size={15} aria-label="Starred" /> : null}
                  <h3 className="min-w-0 flex-1 truncate text-sm font-medium" title={scene.title}>{scene.title}</h3>
                  <SceneActions scene={scene} />
                </div>
                <span className="text-xs text-muted-foreground">Version {scene.contentVersion}</span>
                <Button
                  onClick={() => {
                    openScene(scene);
                    navigate(`/w/${activeWorkspaceId ?? scene.workspaceId}/scenes/${scene.id}`);
                  }}
                  type="button"
                  variant="outline"
                >Open</Button>
              </div>
            </article>
          ))}
        </div>
      ) : null}
    </section>
  );
}
