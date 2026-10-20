import { fireEvent, render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';

import {
  ScenePanel,
  ScenePanelsProvider,
  useScenePanels,
} from '@/components/whiteboard/scene-panels';

function Fixture() {
  const panels = useScenePanels();
  return (
    <div data-testid="canvas" style={{ height: 640, width: 960 }}>
      <button onClick={(event) => panels.open('comments', event.currentTarget)} type="button">Open comments</button>
      <button onClick={(event) => panels.open('ai', event.currentTarget)} type="button">Open AI</button>
      <ScenePanel id="comments" title="Comments"><textarea aria-label="Comment draft" /></ScenePanel>
      <ScenePanel id="ai" title="AI assistant"><input aria-label="AI prompt" /></ScenePanel>
    </div>
  );
}

function renderFixture() {
  return render(<ScenePanelsProvider><Fixture /></ScenePanelsProvider>);
}

describe('ScenePanelsProvider', () => {
  it('onlyOnePanelOpen', () => {
    renderFixture();
    fireEvent.click(screen.getByRole('button', { name: 'Open comments' }));
    expect(screen.getByRole('dialog', { name: 'Comments' })).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: 'Open AI' }));
    expect(screen.queryByRole('dialog', { name: 'Comments' })).not.toBeInTheDocument();
    expect(screen.getByRole('dialog', { name: 'AI assistant' })).toBeInTheDocument();
  });

  it('escapeReturnsFocus', () => {
    renderFixture();
    const trigger = screen.getByRole('button', { name: 'Open comments' });
    trigger.focus();
    fireEvent.click(trigger);
    fireEvent.keyDown(document, { key: 'Escape' });
    expect(screen.queryByRole('dialog', { name: 'Comments' })).not.toBeInTheDocument();
    expect(trigger).toHaveFocus();
  });

  it('textareaUndoIsNative', () => {
    renderFixture();
    fireEvent.click(screen.getByRole('button', { name: 'Open comments' }));
    const textarea = screen.getByRole('textbox', { name: 'Comment draft' });
    const undo = new KeyboardEvent('keydown', { bubbles: true, cancelable: true, key: 'z', ctrlKey: true });
    expect(textarea.dispatchEvent(undo)).toBe(true);
    expect(undo.defaultPrevented).toBe(false);
  });

  it('panelDoesNotResizeCanvas', () => {
    renderFixture();
    const canvas = screen.getByTestId('canvas');
    const before = canvas.getAttribute('style');
    fireEvent.click(screen.getByRole('button', { name: 'Open comments' }));
    expect(screen.getByTestId('scene-panel-layer')).toHaveClass('absolute', 'inset-0', 'pointer-events-none');
    expect(canvas.getAttribute('style')).toBe(before);
  });

  it('mobilePanelHas44pxControls', () => {
    renderFixture();
    fireEvent.click(screen.getByRole('button', { name: 'Open comments' }));
    expect(screen.getByRole('button', { name: 'Close Comments' })).toHaveClass('min-h-11', 'min-w-11');
  });
});
