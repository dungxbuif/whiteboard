import { useEffect, useState } from 'react';
import type { CommentAnchor as Anchor } from '@/lib/comment-api';

export type AnchorElement = { id: string; x: number; y: number; width?: number; height?: number };

export type CommentAnchorAPI = { getSceneElements(): readonly AnchorElement[]; getAppState(): { scrollX?: number; scrollY?: number; zoom?: { value?: number }; offsetLeft?: number; offsetTop?: number } };

export function resolveCommentAnchor(anchor: Anchor, api: CommentAnchorAPI | null) {
  const element = anchor.elementId ? api?.getSceneElements().find(candidate => candidate.id === anchor.elementId) : undefined;
  const state = api?.getAppState() ?? {}, zoom = state.zoom?.value ?? 1;
  const sceneX = element ? element.x + (element.width ?? 0) / 2 : anchor.x;
  const sceneY = element ? element.y + (element.height ?? 0) / 2 : anchor.y;
  return { x: (sceneX + (state.scrollX ?? 0)) * zoom + (state.offsetLeft ?? 0), y: (sceneY + (state.scrollY ?? 0)) * zoom + (state.offsetTop ?? 0), unavailable: Boolean(anchor.elementId && !element) };
}

export function CommentAnchor({ anchor, api, label, onOpen }: { anchor: Anchor; api: CommentAnchorAPI | null; label: string; onOpen?: () => void }) {
  const [position, setPosition] = useState(() => resolveCommentAnchor(anchor, api));
  useEffect(() => {
    let frame = 0;
    const update = () => { const next = resolveCommentAnchor(anchor, api); setPosition(current => current.x === next.x && current.y === next.y && current.unavailable === next.unavailable ? current : next); frame = requestAnimationFrame(update); };
    update(); return () => cancelAnimationFrame(frame);
  }, [anchor.elementId, anchor.x, anchor.y, api]);
  return <button
    aria-label={`Comment ${label}${position.unavailable ? ': Element unavailable' : ''}`}
    className="pointer-events-auto absolute z-10 flex h-7 min-w-7 items-center justify-center rounded-full bg-[#6965db] px-2 text-xs font-semibold text-white shadow-md"
    onClick={onOpen}
    style={{ left: position.x, top: position.y }}
    title={position.unavailable ? 'Element unavailable' : `Comment ${label}`}
    type="button"
  >{label}</button>;
}
