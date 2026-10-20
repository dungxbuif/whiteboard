import { fireEvent, render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';

import { SceneControls } from '@/components/whiteboard/scene-controls';

const status = { text: 'Saved', severity: 'neutral' as const };

describe('SceneControls', () => {
  it('fillsViewportWithoutHeader', () => {
    const { container } = render(
      <div data-testid="canvas" className="h-[100dvh]">
        <SceneControls
          canRedo={false}
          canUndo={false}
          onExportRecovery={vi.fn()}
          onPresent={vi.fn()}
          onRedo={vi.fn()}
          onRetry={vi.fn()}
          onUndo={vi.fn()}
          status={status}
          title="Untitled scene"
        >
          <button type="button">People (1)</button>
        </SceneControls>
      </div>,
    );

    expect(screen.getByTestId('scene-controls')).toHaveClass('pointer-events-none');
    expect(screen.getByTestId('scene-title-island')).toHaveTextContent('Untitled scene');
    expect(container.querySelector('header')).toBeNull();
    expect(screen.getByTestId('canvas')).toHaveClass('h-[100dvh]');
  });

  it('controlsAvoidNativeTools', () => {
    render(
      <SceneControls
        canRedo={false}
        canUndo
        onExportRecovery={vi.fn()}
        onPresent={vi.fn()}
        onRedo={vi.fn()}
        onRetry={vi.fn()}
        onUndo={vi.fn()}
        status={status}
        title="Canvas"
      >
        <button type="button">People (1)</button>
        <button type="button">Join voice</button>
      </SceneControls>,
    );

    expect(screen.getByTestId('scene-control-cluster')).toHaveClass('pointer-events-auto');
    expect(screen.getByRole('button', { name: 'People (1)' })).toHaveClass('min-h-11');
    expect(screen.getByRole('button', { name: 'Join voice' })).toHaveClass('min-h-11');
    expect(screen.queryByText('Voice 0/10')).not.toBeInTheDocument();
  });

  it('keepsSavedRecoveryAndBackAccessible', () => {
    const onBack = vi.fn();
    const onExportRecovery = vi.fn();
    const onRetry = vi.fn();
    render(
      <SceneControls
        canRedo={false}
        canUndo
        onBack={onBack}
        onExportRecovery={onExportRecovery}
        onPresent={vi.fn()}
        onRedo={vi.fn()}
        onRetry={onRetry}
        onUndo={vi.fn()}
        status={{ text: 'Recovery required', severity: 'error' }}
        title="Canvas"
      />,
    );

    expect(screen.getByRole('alert')).toHaveTextContent('Recovery required');
    fireEvent.click(screen.getByRole('button', { name: 'More scene actions' }));
    fireEvent.click(screen.getByRole('button', { name: 'Back to dashboard' }));
    fireEvent.click(screen.getByRole('button', { name: 'Export recovery' }));
    fireEvent.click(screen.getByRole('button', { name: 'Retry connection / uploads' }));
    expect(onBack).toHaveBeenCalledOnce();
    expect(onExportRecovery).toHaveBeenCalledOnce();
    expect(onRetry).toHaveBeenCalledOnce();
  });
});
