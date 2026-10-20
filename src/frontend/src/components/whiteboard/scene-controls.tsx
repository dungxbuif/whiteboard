import {
  ArrowLeft,
  Download,
  MoreHorizontal,
  Presentation,
  Redo2,
  RefreshCcw,
  Undo2,
} from 'lucide-react';
import {
  Children,
  cloneElement,
  isValidElement,
  type ReactElement,
  type ReactNode,
} from 'react';

import { Button } from '@/components/ui/button';
import { cn } from '@/lib/utils';

export type SceneControlStatus = {
  text: string;
  severity: 'neutral' | 'pending' | 'warning' | 'error';
};

export interface SceneControlsProps {
  title: string;
  status: SceneControlStatus;
  canUndo: boolean;
  canRedo: boolean;
  onBack?: () => void;
  onUndo: () => void;
  onRedo: () => void;
  onPresent?: () => void;
  onExportRecovery: () => void;
  onRetry: () => void;
  children?: ReactNode;
}

function touchSizedChildren(children: ReactNode) {
  return Children.map(children, (child) => {
    if (!isValidElement(child)) return child;
    const element = child as ReactElement<{ className?: string }>;
    return cloneElement(element, { className: cn('min-h-11', element.props.className) });
  });
}

export function SceneControlCluster({ children }: { children: ReactNode }) {
  return (
    <div
      className="pointer-events-auto flex max-w-[min(75vw,52rem)] flex-wrap items-center justify-end gap-2"
      data-testid="scene-control-cluster"
    >
      {touchSizedChildren(children)}
    </div>
  );
}

export function SceneControls({
  title,
  status,
  canUndo,
  canRedo,
  onBack,
  onUndo,
  onRedo,
  onPresent,
  onExportRecovery,
  onRetry,
  children,
}: SceneControlsProps) {
  const needsRecovery = status.severity === 'warning' || status.severity === 'error';

  return (
    <div
      className="pointer-events-none absolute inset-0 z-20 overflow-hidden"
      data-testid="scene-controls"
    >
      <h1 className="sr-only">{title}</h1>
      <div
        className="pointer-events-none absolute left-[max(3.75rem,calc(env(safe-area-inset-left)+3.75rem))] top-[max(0.75rem,env(safe-area-inset-top))] hidden max-w-[min(36vw,28rem)] truncate rounded-lg border bg-card/90 px-3 py-2 text-sm font-semibold shadow-sm backdrop-blur md:block"
        data-testid="scene-title-island"
        title={title}
      >
        {title}
      </div>

      {children ? (
        <div className="absolute right-[max(0.75rem,env(safe-area-inset-right))] top-[max(0.75rem,env(safe-area-inset-top))]">
          <SceneControlCluster>{children}</SceneControlCluster>
        </div>
      ) : null}

      <div className="pointer-events-auto absolute bottom-[max(0.75rem,env(safe-area-inset-bottom))] right-[max(0.75rem,env(safe-area-inset-right))] flex max-w-[calc(100vw-1.5rem)] items-center gap-2">
        <span
          className={cn(
            'max-w-[min(56vw,28rem)] truncate rounded-full border bg-card/90 px-3 py-2 text-xs shadow-sm backdrop-blur',
            status.severity === 'error' && 'border-destructive text-destructive',
            status.severity === 'warning' && 'border-amber-500 text-amber-800 dark:text-amber-300',
          )}
          role={status.severity === 'error' ? 'alert' : 'status'}
        >
          {status.text}
        </span>
        <details className="relative">
          <summary
            aria-label="More scene actions"
            className="flex min-h-11 min-w-11 cursor-pointer list-none items-center justify-center rounded-lg border bg-card/95 shadow-sm backdrop-blur focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
            role="button"
          >
            <MoreHorizontal aria-hidden="true" size={18} />
          </summary>
          <div className="absolute bottom-full right-0 mb-2 grid min-w-56 gap-1 rounded-lg border bg-popover p-1 text-popover-foreground shadow-lg">
            {onBack ? <Button className="justify-start" onClick={onBack} variant="ghost"><ArrowLeft aria-hidden="true" size={16} />Back to dashboard</Button> : null}
            {onPresent ? <Button className="justify-start" onClick={onPresent} variant="ghost"><Presentation aria-hidden="true" size={16} />Present</Button> : null}
            <Button className="justify-start" disabled={!canUndo} onClick={onUndo} variant="ghost"><Undo2 aria-hidden="true" size={16} />Undo my edit</Button>
            <Button className="justify-start" disabled={!canRedo} onClick={onRedo} variant="ghost"><Redo2 aria-hidden="true" size={16} />Redo my edit</Button>
            {needsRecovery ? <Button className="justify-start" onClick={onExportRecovery} variant="ghost"><Download aria-hidden="true" size={16} />Export recovery</Button> : null}
            {needsRecovery ? <Button className="justify-start" onClick={onRetry} variant="ghost"><RefreshCcw aria-hidden="true" size={16} />Retry connection / uploads</Button> : null}
          </div>
        </details>
      </div>
    </div>
  );
}
