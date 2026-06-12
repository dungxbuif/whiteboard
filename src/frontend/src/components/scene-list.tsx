import { FilePenLine, Star } from "lucide-react";

import { SceneActions } from "@/components/scene-actions";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import useSceneStore from "@/store/scene-store";

export function SceneList() {
  const scenes = useSceneStore((state) => state.scenes);
  const isLoading = useSceneStore((state) => state.isLoading);
  const error = useSceneStore((state) => state.error);
  const openScene = useSceneStore((state) => state.openScene);

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
            <Card key={scene.id}>
              <CardHeader className="space-y-4">
                <div className="flex items-center justify-between text-muted-foreground">
                  <FilePenLine size={18} aria-hidden="true" />
                  <SceneActions scene={scene} />
                </div>
                <CardTitle className="flex items-center gap-2 text-lg">
                  {scene.isStarred ? <Star className="fill-current text-amber-500" size={15} aria-label="Starred" /> : null}
                  {scene.title}
                </CardTitle>
              </CardHeader>
              <CardContent className="grid gap-3 text-sm text-muted-foreground">
                <span>Version {scene.contentVersion}</span>
                <Button onClick={() => openScene(scene)} type="button" variant="outline">Open</Button>
              </CardContent>
            </Card>
          ))}
        </div>
      ) : null}
    </section>
  );
}
