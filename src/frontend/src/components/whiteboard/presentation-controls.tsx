import { ChevronLeft, ChevronRight, Play, X } from "lucide-react";

import { Button } from "@/components/ui/button";

interface PresentationControlsProps {
  activeIndex: number;
  slideCount: number;
  onPrevious: () => void;
  onNext: () => void;
  onExit: () => void;
}

export function PresentationControls({ activeIndex, slideCount, onPrevious, onNext, onExit }: PresentationControlsProps) {
  return (
    <div aria-label="Presentation controls" className="absolute bottom-5 left-1/2 z-20 flex -translate-x-1/2 items-center gap-2 rounded-full border bg-card/95 px-3 py-2 shadow-lg backdrop-blur" role="region">
      <Play size={14} aria-hidden="true" />
      <span className="min-w-16 text-center text-sm font-medium">{slideCount ? `${activeIndex + 1} / ${slideCount}` : "No slides"}</span>
      <Button aria-label="Previous slide" disabled={activeIndex === 0} onClick={onPrevious} size="icon" type="button" variant="ghost">
        <ChevronLeft size={16} aria-hidden="true" />
      </Button>
      <Button aria-label="Next slide" disabled={activeIndex >= slideCount - 1} onClick={onNext} size="icon" type="button" variant="ghost">
        <ChevronRight size={16} aria-hidden="true" />
      </Button>
      <Button aria-label="Exit presentation" onClick={onExit} size="icon" type="button" variant="ghost">
        <X size={16} aria-hidden="true" />
      </Button>
    </div>
  );
}
