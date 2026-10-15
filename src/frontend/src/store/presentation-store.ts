import { create } from "zustand";

interface PresentationState {
  isPresenting: boolean;
  activeSlideIndex: number;
  enter: () => void;
  exit: () => void;
  toggle: () => void;
  setActiveSlide: (index: number, slideCount: number) => void;
  reset: () => void;
}

const clampIndex = (index: number, slideCount: number) =>
  slideCount > 0 ? Math.max(0, Math.min(index, slideCount - 1)) : 0;

const usePresentationStore = create<PresentationState>((set) => ({
  isPresenting: false,
  activeSlideIndex: 0,
  enter: () => set({ isPresenting: true, activeSlideIndex: 0 }),
  exit: () => set({ isPresenting: false }),
  toggle: () => set((state) => ({ isPresenting: !state.isPresenting, activeSlideIndex: 0 })),
  setActiveSlide: (index, slideCount) => set({ activeSlideIndex: clampIndex(index, slideCount) }),
  reset: () => set({ isPresenting: false, activeSlideIndex: 0 }),
}));

export default usePresentationStore;
