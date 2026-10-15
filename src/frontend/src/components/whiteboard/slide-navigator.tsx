import { ChevronLeft, ChevronRight } from "lucide-react";

import { Button } from "@/components/ui/button";

interface SlideNavigatorProps {
  slideIds: string[];
  activeIndex: number;
  onSelect: (index: number) => void;
  onMove: (index: number, direction: -1 | 1) => void;
}

export function SlideNavigator({ slideIds, activeIndex, onSelect, onMove }: SlideNavigatorProps) {
  if (slideIds.length === 0) return null;

  return (
    <aside aria-label="Slide navigator" className="absolute left-4 top-20 z-10 grid max-h-[calc(100vh-7rem)] w-44 gap-2 overflow-y-auto rounded-lg border bg-card/95 p-2 shadow-lg backdrop-blur">
      <p className="px-2 text-xs font-semibold uppercase tracking-wide text-muted-foreground">Slides</p>
      {slideIds.map((slideId, index) => (
        <div className="grid grid-cols-[1fr_auto_auto] items-center gap-1" key={slideId}>
          <Button
            aria-current={index === activeIndex ? "true" : undefined}
            className="justify-start truncate"
            onClick={() => onSelect(index)}
            size="sm"
            type="button"
            variant={index === activeIndex ? "secondary" : "ghost"}
          >
            <span className="mr-1 text-xs text-muted-foreground">{index + 1}</span>
            <span className="truncate">{slideId.slice(0, 8)}</span>
          </Button>
          <Button aria-label={`Move slide ${index + 1} left`} disabled={index === 0} onClick={() => onMove(index, -1)} size="icon" type="button" variant="ghost">
            <ChevronLeft size={14} aria-hidden="true" />
          </Button>
          <Button aria-label={`Move slide ${index + 1} right`} disabled={index === slideIds.length - 1} onClick={() => onMove(index, 1)} size="icon" type="button" variant="ghost">
            <ChevronRight size={14} aria-hidden="true" />
          </Button>
        </div>
      ))}
    </aside>
  );
}
