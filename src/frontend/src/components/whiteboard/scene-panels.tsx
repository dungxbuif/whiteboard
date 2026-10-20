import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useRef,
  useState,
  type ReactNode,
} from 'react';
import { X } from 'lucide-react';

import { Button } from '@/components/ui/button';

export type ScenePanelId = 'comments' | 'activity' | 'history' | 'library' | 'ai';

type ScenePanelsContextValue = {
  active: ScenePanelId | null;
  open: (id: ScenePanelId, trigger?: HTMLElement | null) => void;
  close: () => void;
};

const ScenePanelsContext = createContext<ScenePanelsContextValue | null>(null);

export function ScenePanelsProvider({ children }: { children: ReactNode }) {
  const [active, setActive] = useState<ScenePanelId | null>(null);
  const returnFocus = useRef<HTMLElement | null>(null);

  const open = useCallback((id: ScenePanelId, trigger?: HTMLElement | null) => {
    const candidate = trigger ?? (document.activeElement instanceof HTMLElement ? document.activeElement : null);
    if (candidate) returnFocus.current = candidate;
    setActive(id);
  }, []);

  const close = useCallback(() => {
    setActive(null);
    const target = returnFocus.current;
    returnFocus.current = null;
    target?.focus();
  }, []);

  useEffect(() => {
    if (!active) return;
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key !== 'Escape') return;
      event.preventDefault();
      event.stopPropagation();
      close();
    };
    window.addEventListener('keydown', onKeyDown, true);
    return () => window.removeEventListener('keydown', onKeyDown, true);
  }, [active, close]);

  const value = useMemo(() => ({ active, open, close }), [active, open, close]);
  return <ScenePanelsContext.Provider value={value}>{children}</ScenePanelsContext.Provider>;
}

export function useScenePanels() {
  const value = useContext(ScenePanelsContext);
  if (!value) throw new Error('useScenePanels must be used inside ScenePanelsProvider');
  return value;
}

export function ScenePanel({ id, title, children }: { id: ScenePanelId; title: string; children: ReactNode }) {
  const panels = useScenePanels();
  if (panels.active !== id) return null;

  return (
    <div className="pointer-events-none absolute inset-0 z-30" data-testid="scene-panel-layer">
      <section
        aria-label={title}
        aria-modal="false"
        className="pointer-events-auto absolute bottom-0 right-0 top-0 flex w-full max-w-md flex-col overflow-hidden border-l bg-card/95 shadow-2xl backdrop-blur sm:bottom-3 sm:right-3 sm:top-3 sm:rounded-xl sm:border"
        role="dialog"
      >
        <div className="flex min-h-14 items-center justify-between gap-3 border-b px-4">
          <h2 className="truncate font-semibold">{title}</h2>
          <Button
            aria-label={`Close ${title}`}
            className="min-h-11 min-w-11"
            onClick={panels.close}
            size="icon"
            type="button"
            variant="ghost"
          >
            <X aria-hidden="true" size={18} />
          </Button>
        </div>
        <div className="min-h-0 flex-1 overflow-auto p-3">{children}</div>
      </section>
    </div>
  );
}
