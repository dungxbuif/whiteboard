import { ArrowLeft, Cloud, CloudAlert, CloudUpload } from "lucide-react";
import { useEffect, useRef, useState } from "react";
import type { ComponentType } from "react";

import "@excalidraw/excalidraw/index.css";

import { Button } from "@/components/ui/button";
import type { SceneContent } from "@/store/scene-store";
import useSceneStore from "@/store/scene-store";
import useWorkspaceStore from "@/store/workspace-store";

type ExcalidrawModule = typeof import("@excalidraw/excalidraw");
const EXCALIDRAW_INITIALIZATION_MS = 250;

const saveLabels = {
  idle: "Saved",
  pending: "Waiting to save",
  saving: "Saving",
  saved: "Saved",
  error: "Save failed",
} as const;

export function ExcalidrawWrapper() {
  const activeWorkspaceId = useWorkspaceStore((state) => state.activeWorkspaceId);
  const activeScene = useSceneStore((state) => state.activeScene);
  const saveStatus = useSceneStore((state) => state.saveStatus);
  const scheduleSave = useSceneStore((state) => state.scheduleSave);
  const closeScene = useSceneStore((state) => state.closeScene);
  const [module, setModule] = useState<ExcalidrawModule | null>(null);
  const lastSerialized = useRef(JSON.stringify(activeScene?.content));
  const isInitialized = useRef(false);

  useEffect(() => {
    void import("@excalidraw/excalidraw").then(setModule);
  }, []);

  useEffect(() => {
    if (!module) return;
    isInitialized.current = false;
    const timer = window.setTimeout(() => {
      isInitialized.current = true;
    }, EXCALIDRAW_INITIALIZATION_MS);
    return () => window.clearTimeout(timer);
  }, [activeScene?.id, module]);

  if (!activeScene || !activeWorkspaceId) return null;

  if (!module) {
    return <main className="grid min-h-screen place-items-center">Loading whiteboard...</main>;
  }

  const Excalidraw = module.Excalidraw as ComponentType<Record<string, unknown>>;
  const SaveIcon = saveStatus === "error" ? CloudAlert : saveStatus === "pending" || saveStatus === "saving" ? CloudUpload : Cloud;

  return (
    <main className="grid h-screen grid-rows-[56px_minmax(0,1fr)] bg-background">
      <header className="flex items-center justify-between gap-4 border-b bg-card px-4">
        <div className="flex min-w-0 items-center gap-3">
          <Button aria-label="Back to dashboard" onClick={() => void closeScene().catch(() => undefined)} size="icon" type="button" variant="ghost">
            <ArrowLeft size={18} aria-hidden="true" />
          </Button>
          <h1 className="truncate font-semibold">{activeScene.title}</h1>
        </div>
        <p className={`flex items-center gap-2 text-sm ${saveStatus === "error" ? "text-destructive" : "text-muted-foreground"}`} role="status">
          <SaveIcon size={16} aria-hidden="true" /> {saveLabels[saveStatus]}
        </p>
      </header>
      <section className="min-h-0" aria-label="Excalidraw canvas">
        <Excalidraw
          initialData={activeScene.content}
          name={activeScene.title}
          onChange={(elements: unknown[], appState: Record<string, unknown>, files: Record<string, unknown>) => {
            const content: SceneContent = { elements: [...elements], appState, files };
            const serialized = JSON.stringify(content);
            if (!isInitialized.current) {
              lastSerialized.current = serialized;
              return;
            }
            if (serialized === lastSerialized.current) return;
            lastSerialized.current = serialized;
            scheduleSave(activeWorkspaceId, activeScene.id, content);
          }}
        />
      </section>
    </main>
  );
}
