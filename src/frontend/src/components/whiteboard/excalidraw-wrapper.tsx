import { ArrowLeft, Cloud, CloudAlert, CloudUpload } from "lucide-react";
import { useEffect, useMemo, useRef, useState } from "react";
import type { ComponentType } from "react";

import "@excalidraw/excalidraw/index.css";

import { Button } from "@/components/ui/button";
import { navigate } from "@/lib/router";
import type { SceneContent } from "@/store/scene-store";
import useSceneStore, { syncPresentationOrder } from "@/store/scene-store";
import usePresentationStore from "@/store/presentation-store";
import useWorkspaceStore from "@/store/workspace-store";
import api from "@/lib/api";
import { PresentationControls } from "@/components/whiteboard/presentation-controls";
import { SlideNavigator } from "@/components/whiteboard/slide-navigator";
import { CollaborationToolbar } from "@/components/whiteboard/collaboration-toolbar";
import { AICommandPanel } from "@/components/whiteboard/ai-command-panel";
import useRealtimeStore from "@/store/realtime-store";

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
  const updatePresentationOrder = useSceneStore((state) => state.updatePresentationOrder);
  const isPresenting = usePresentationStore((state) => state.isPresenting);
  const activeSlideIndex = usePresentationStore((state) => state.activeSlideIndex);
  const enterPresentation = usePresentationStore((state) => state.enter);
  const exitPresentation = usePresentationStore((state) => state.exit);
  const setActiveSlide = usePresentationStore((state) => state.setActiveSlide);
  const connectRealtime = useRealtimeStore((state) => state.connect);
  const disconnectRealtime = useRealtimeStore((state) => state.disconnect);
  const publishRealtime = useRealtimeStore((state) => state.publish);
  const latestSceneUpdate = useRealtimeStore((state) => state.latestSceneUpdate);
  const clearLatestSceneUpdate = useRealtimeStore((state) => state.clearLatestSceneUpdate);
  
  const [module, setModule] = useState<ExcalidrawModule | null>(null);
  const [hydratedContent, setHydratedContent] = useState<SceneContent | null>(null);
  
  const lastSerialized = useRef(JSON.stringify(activeScene?.content));
  const isInitialized = useRef(false);
  const uploadedFiles = useRef<Set<string>>(new Set());
  const excalidrawAPI = useRef<{
    scrollToContent: (target?: unknown, options?: Record<string, unknown>) => void;
    updateScene?: (scene: Record<string, unknown>) => void;
  } | null>(null);
  const applyingRemote = useRef(false);

  useEffect(() => {
    if (!activeWorkspaceId || !activeScene?.id) return;
    connectRealtime(activeWorkspaceId, activeScene.id);
    return () => disconnectRealtime();
  }, [activeScene?.id, activeWorkspaceId, connectRealtime, disconnectRealtime]);

  useEffect(() => {
    if (!latestSceneUpdate || !excalidrawAPI.current?.updateScene) return;
    applyingRemote.current = true;
    excalidrawAPI.current.updateScene({
      elements: latestSceneUpdate.elements,
      appState: latestSceneUpdate.appState,
      files: latestSceneUpdate.files,
    });
    clearLatestSceneUpdate();
  }, [clearLatestSceneUpdate, latestSceneUpdate]);

  const frameElements = useMemo(
    () => (activeScene?.content.elements ?? []).filter((element): element is { id: string; type: string } => {
      if (!element || typeof element !== "object") return false;
      const candidate = element as { id?: unknown; type?: unknown };
      return candidate.type === "frame" && typeof candidate.id === "string";
    }),
    [activeScene?.content.elements],
  );
  const slideIds = useMemo(
    () => syncPresentationOrder(frameElements, activeScene?.content.presentationOrder ?? []),
    [activeScene?.content.presentationOrder, frameElements],
  );

  useEffect(() => {
    setActiveSlide(activeSlideIndex, slideIds.length);
  }, [activeSlideIndex, setActiveSlide, slideIds.length]);

  useEffect(() => {
    if (!isPresenting) return;
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.target instanceof HTMLElement && ["INPUT", "TEXTAREA", "BUTTON"].includes(event.target.tagName)) return;
      if (event.key === "Escape") {
        event.preventDefault();
        exitPresentation();
      } else if (event.key === "ArrowRight" || event.key === "ArrowDown" || event.key === " ") {
        event.preventDefault();
        setActiveSlide(activeSlideIndex + 1, slideIds.length);
      } else if (event.key === "ArrowLeft" || event.key === "ArrowUp") {
        event.preventDefault();
        setActiveSlide(activeSlideIndex - 1, slideIds.length);
      }
    };
    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  }, [activeSlideIndex, exitPresentation, isPresenting, setActiveSlide, slideIds.length]);

  useEffect(() => {
    if (!isPresenting || !excalidrawAPI.current || slideIds.length === 0) return;
    const frame = frameElements.find((element) => element.id === slideIds[activeSlideIndex]);
    if (frame) {
      excalidrawAPI.current.scrollToContent(frame, { fitToViewport: true, viewportZoomFactor: 0.8, animate: true, duration: 300 });
    }
  }, [activeSlideIndex, frameElements, isPresenting, slideIds]);

  useEffect(() => {
    void import("@excalidraw/excalidraw").then(setModule);
  }, []);

  // Hydrate scene content with S3 presigned URLs for files without dataURLs
  useEffect(() => {
    if (!activeScene) return;

    let isMounted = true;
    const hydrate = async () => {
      const content = activeScene.content;
      if (!content || !content.files) {
        if (isMounted) setHydratedContent(content || { elements: [], appState: {}, files: {}, presentationOrder: [] });
        return;
      }

      const missingUrls: string[] = [];
      Object.values(content.files).forEach((file: any) => {
        if (!file.dataURL) {
          missingUrls.push(file.id);
        }
      });

      if (missingUrls.length === 0) {
        if (isMounted) setHydratedContent(content);
        return;
      }

      try {
        const res = await api.post('/assets/presign-view', { fileIds: missingUrls }, {
          headers: { "X-Workspace-ID": activeWorkspaceId }
        });
        const urls = res.data.urls || {};

        const hydratedFiles = { ...content.files };
        for (const fileId of missingUrls) {
          if (urls[fileId] && urls[fileId].viewUrl) {
            (hydratedFiles as any)[fileId] = { ...(hydratedFiles as any)[fileId], dataURL: urls[fileId].viewUrl };
          }
        }

        if (isMounted) setHydratedContent({ ...content, files: hydratedFiles });
      } catch (err) {
        console.error("Failed to hydrate S3 URLs", err);
        if (isMounted) setHydratedContent(content);
      }
    };

    void hydrate();
    return () => { isMounted = false; };
  }, [activeScene?.id, activeWorkspaceId]);

  useEffect(() => {
    if (!module || !hydratedContent) return;
    isInitialized.current = false;
    const timer = window.setTimeout(() => {
      isInitialized.current = true;
    }, EXCALIDRAW_INITIALIZATION_MS);
    return () => window.clearTimeout(timer);
  }, [hydratedContent, module]);

  if (!activeScene || !activeWorkspaceId) return null;

  if (!module || !hydratedContent) {
    return <main className="grid min-h-screen place-items-center">Loading whiteboard...</main>;
  }

  const Excalidraw = module.Excalidraw as ComponentType<Record<string, unknown>>;
  const SaveIcon = saveStatus === "error" ? CloudAlert : saveStatus === "pending" || saveStatus === "saving" ? CloudUpload : Cloud;

  return (
    <main className="grid h-screen min-w-0 grid-rows-[56px_minmax(0,1fr)] overflow-hidden bg-background">
      <header className="flex min-w-0 flex-wrap items-center justify-between gap-4 border-b bg-card px-4 max-md:gap-2">
        <div className="flex min-w-0 items-center gap-3">
          <Button aria-label="Back to dashboard" onClick={() => void closeScene().then(() => navigate("/")).catch(() => undefined)} size="icon" type="button" variant="ghost">
            <ArrowLeft size={18} aria-hidden="true" />
          </Button>
          <h1 className="truncate font-semibold">{activeScene.title}</h1>
        </div>
        <div className="flex min-w-0 flex-wrap items-center justify-end gap-3 max-md:gap-2">
          <Button onClick={() => (isPresenting ? exitPresentation() : enterPresentation())} type="button" variant={isPresenting ? "secondary" : "outline"}>
            {isPresenting ? "Exit presentation" : "Present"}
          </Button>
          <CollaborationToolbar />
          <p className={`flex items-center gap-2 text-sm ${saveStatus === "error" ? "text-destructive" : "text-muted-foreground"}`} role="status">
            <SaveIcon size={16} aria-hidden="true" /> {saveLabels[saveStatus]}
          </p>
        </div>
      </header>
      <section className="min-h-0" aria-label="Excalidraw canvas">
        <Excalidraw
          excalidrawAPI={(apiInstance: { scrollToContent: (target?: unknown, options?: Record<string, unknown>) => void; updateScene?: (scene: Record<string, unknown>) => void }) => { excalidrawAPI.current = apiInstance; }}
          initialData={hydratedContent}
          name={activeScene.title}
          viewModeEnabled={isPresenting}
          onChange={async (elements: unknown[], appState: Record<string, unknown>, files: Record<string, unknown>) => {
            const presentationOrder = syncPresentationOrder(elements, activeScene.content.presentationOrder ?? []);
            const contentToSave: SceneContent = { elements: [...elements], appState, files: { ...files }, presentationOrder };
            if (applyingRemote.current) {
              applyingRemote.current = false;
              return;
            }
            publishRealtime("scene:update", contentToSave as unknown as Record<string, unknown>);
            const filesToUpload: any[] = [];

            // 1. Process files: intercept new data URIs and strip them for DB save
            for (const [id, f] of Object.entries(files)) {
              const file = f as any;
              if (file.dataURL && file.dataURL.startsWith("data:") && !uploadedFiles.current.has(id)) {
                filesToUpload.push(file);
                uploadedFiles.current.add(id);
              }
              if (contentToSave.files[id]) {
                (contentToSave.files as any)[id] = { ...(contentToSave.files as any)[id], dataURL: "" };
              }
            }

            // 2. Upload to S3 asynchronously
            if (filesToUpload.length > 0) {
              for (const file of filesToUpload) {
                try {
                  const base64Len = file.dataURL.length - file.dataURL.indexOf(',') - 1;
                  const byteSize = Math.floor((base64Len * 3) / 4);
                  
                  const res = await api.post('/assets/presign-upload', {
                    fileId: file.id,
                    mimeType: file.mimeType,
                    byteSize: byteSize
                  }, { headers: { "X-Workspace-ID": activeWorkspaceId } });
                  
                  const uploadUrl = res.data.urls?.[file.id]?.uploadUrl;
                  if (uploadUrl) {
                    const resBlob = await fetch(file.dataURL);
                    const blob = await resBlob.blob();
                    await fetch(uploadUrl, {
                      method: 'PUT',
                      body: blob,
                      headers: { 'Content-Type': file.mimeType }
                    });
                  }
                } catch (e) {
                  console.error("Failed to upload file to S3", file.id, e);
                  uploadedFiles.current.delete(file.id);
                }
              }
            }

            // 3. Save scene (without embedded dataURLs)
            const serialized = JSON.stringify(contentToSave);
            if (!isInitialized.current) {
              lastSerialized.current = serialized;
              return;
            }
            if (serialized === lastSerialized.current) return;
            lastSerialized.current = serialized;
            scheduleSave(activeWorkspaceId, activeScene.id, contentToSave);
          }}
        />
        {isPresenting ? (
          <>
            <SlideNavigator
              activeIndex={activeSlideIndex}
              onMove={(index, direction) => {
                const nextOrder = [...slideIds];
                const swapIndex = index + direction;
                [nextOrder[index], nextOrder[swapIndex]] = [nextOrder[swapIndex], nextOrder[index]];
                updatePresentationOrder(activeWorkspaceId, activeScene.id, nextOrder);
                setActiveSlide(swapIndex, nextOrder.length);
              }}
              onSelect={(index) => setActiveSlide(index, slideIds.length)}
              slideIds={slideIds}
            />
            <PresentationControls
              activeIndex={activeSlideIndex}
              onExit={exitPresentation}
              onNext={() => setActiveSlide(activeSlideIndex + 1, slideIds.length)}
              onPrevious={() => setActiveSlide(activeSlideIndex - 1, slideIds.length)}
              slideCount={slideIds.length}
            />
          </>
        ) : null}
        {!isPresenting ? (
          <AICommandPanel
            onApplyDiagram={(generated) => {
              if (!excalidrawAPI.current?.updateScene) return;
              applyingRemote.current = true;
              excalidrawAPI.current.updateScene({ elements: generated.elements, appState: generated.appState, files: generated.files });
            }}
            scene={activeScene.content}
          />
        ) : null}
      </section>
    </main>
  );
}
