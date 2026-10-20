import { cleanup, render, screen, waitFor } from '@testing-library/react';
import type { SceneContent } from '@collab/contracts';
import type { ReactNode } from 'react';
import { afterEach, expect, it, vi } from 'vitest';

const empty: SceneContent = { elements: [], appState: {}, files: {}, presentationOrder: [] };
const mock = vi.hoisted(() => ({ props: null as Record<string, unknown> | null }));

vi.mock('@/collab/provider', () => ({
  CollabProvider: ({ children }: { children: unknown }) => children,
  useCollab: () => ({
    adapter: { content: () => empty, reorderPresentation: vi.fn() },
    transport: {
      scope: { principalId: 'user', workspaceId: crypto.randomUUID(), sceneId: crypto.randomUUID(), documentId: 'doc' },
      state: { status: 'synced', canEdit: true, pending: 0, recovery: false },
      isActive: () => true,
      sendAwareness: vi.fn(),
      retry: vi.fn(),
    },
    state: { status: 'synced', canEdit: true, pending: 0, recovery: false, connectionId: 'connection' },
    content: empty,
    legacy: null,
    sceneRef: { workspaceId: crypto.randomUUID(), sceneId: crypto.randomUUID() },
    undo: { canUndo: () => false, canRedo: () => false, undo: vi.fn(), redo: vi.fn() },
    principal: { current: { kind: 'user', id: 'user' } },
    presence: null,
    voice: null,
    post: vi.fn(),
    exportRecovery: vi.fn(async () => ({})),
    applyLocalScene: vi.fn(),
    applyAgentElements: vi.fn(),
  }),
}));

vi.mock('@/components/whiteboard/excalidraw-module', () => ({
  loadExcalidrawModule: async () => {
  const MainMenu = Object.assign(
    ({ children }: { children?: ReactNode }) => <div data-testid="native-main-menu">{children}</div>,
    {
      Item: ({ children }: { children?: ReactNode }) => <button type="button">{children}</button>,
      ItemCustom: ({ children }: { children?: ReactNode }) => <div>{children}</div>,
      Separator: () => <hr />,
    },
  );
  return {
    Excalidraw: (props: Record<string, unknown>) => {
      mock.props = props;
      const renderTopRightUI = props.renderTopRightUI as undefined | (() => ReactNode);
      return <div data-testid="native-canvas">{renderTopRightUI?.()}{props.children as ReactNode}</div>;
    },
    MainMenu,
    restoreElements: (elements: unknown) => elements,
    CaptureUpdateAction: { NEVER: 'never' },
  };
  },
}));

import { CollaborativeCanvas } from '@/components/whiteboard/excalidraw-wrapper';

afterEach(() => { cleanup(); mock.props = null; });

it('uses a full viewport canvas with native top-right and main-menu controls', async () => {
  const { container } = render(<CollaborativeCanvas onBack={vi.fn()} title="Scene title" />);
  await waitFor(() => expect(screen.getByTestId('native-canvas')).toBeInTheDocument());
  expect(container.querySelector('header')).toBeNull();
  expect(container.querySelector('main')).toHaveClass('h-[100dvh]');
  expect(mock.props?.renderTopRightUI).toEqual(expect.any(Function));
  expect(screen.getByTestId('native-main-menu')).toBeInTheDocument();
  expect(screen.getByRole('button', { name: 'Back to dashboard' })).toBeInTheDocument();
  expect(screen.getByRole('button', { name: 'Open AI assistant' })).toBeInTheDocument();
});
