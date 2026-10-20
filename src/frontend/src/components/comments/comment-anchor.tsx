import type { CommentAnchor as Anchor } from '@/lib/comment-api';

export type AnchorElement = { id: string; x: number; y: number; width?: number; height?: number };

export function resolveCommentAnchor(anchor: Anchor, elements: readonly AnchorElement[]) {
  if (!anchor.elementId) return { x: anchor.x, y: anchor.y, unavailable: false };
  const element = elements.find(candidate => candidate.id === anchor.elementId);
  if (!element) return { x: anchor.x, y: anchor.y, unavailable: true };
  return { x: element.x + (element.width ?? 0) / 2, y: element.y + (element.height ?? 0) / 2, unavailable: false };
}

export function CommentAnchor({ anchor, elements, label, onOpen }: { anchor: Anchor; elements: readonly AnchorElement[]; label: string; onOpen?: () => void }) {
  const position = resolveCommentAnchor(anchor, elements);
  return <button
    aria-label={`Comment ${label}${position.unavailable ? ': Element unavailable' : ''}`}
    className="pointer-events-auto absolute z-10 flex h-7 min-w-7 items-center justify-center rounded-full bg-[#6965db] px-2 text-xs font-semibold text-white shadow-md"
    onClick={onOpen}
    style={{ left: position.x, top: position.y }}
    title={position.unavailable ? 'Element unavailable' : `Comment ${label}`}
    type="button"
  >{label}</button>;
}
