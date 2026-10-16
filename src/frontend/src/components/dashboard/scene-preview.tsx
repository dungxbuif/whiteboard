import { FilePenLine } from "lucide-react";
import { useEffect, useRef, useState } from "react";
import type { SceneContent } from "@/store/scene-store";

export function ScenePreview({ content }: { content: SceneContent }) {
  const container = useRef<HTMLDivElement>(null);
  const [visible, setVisible] = useState(false);
  const [url, setUrl] = useState<string>();
  useEffect(() => {
    const observer = new IntersectionObserver(entries => { if (entries.some(entry => entry.isIntersecting)) { setVisible(true); observer.disconnect(); } });
    if (container.current) observer.observe(container.current);
    return () => observer.disconnect();
  }, []);
  useEffect(() => {
    if (!visible || !content.elements.length) return;
    let cancelled = false;
    let objectURL: string | undefined;
    void import("@excalidraw/excalidraw").then(async ({ exportToSvg, restoreElements }) => {
      const background = typeof content.appState.viewBackgroundColor === "string" ? content.appState.viewBackgroundColor : "#ffffff";
      const svg = await exportToSvg({ elements: restoreElements(content.elements as Parameters<typeof restoreElements>[0], null), appState: { exportBackground: true, viewBackgroundColor: background }, files: content.files as Parameters<typeof exportToSvg>[0]["files"] });
      if (cancelled) return;
      objectURL = URL.createObjectURL(new Blob([svg.outerHTML], { type: "image/svg+xml" }));
      setUrl(objectURL);
    }).catch(() => undefined);
    return () => { cancelled = true; if (objectURL) URL.revokeObjectURL(objectURL); };
  }, [content, visible]);
  return <div ref={container} className="flex aspect-video items-center justify-center overflow-hidden bg-muted">
    {url ? <img alt="" src={url} className="h-full w-full object-contain" /> : <FilePenLine size={32} className="text-muted-foreground/50" aria-hidden="true" />}
  </div>;
}
