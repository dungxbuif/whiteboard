import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react';
import type { ReactNode } from 'react';
import { afterEach, expect, it, vi } from 'vitest';

const mock = vi.hoisted(() => {
  const content = { elements: [], appState: {}, files: {}, presentationOrder: [] };
  const state = { status: 'synced', canEdit: true, pending: 0, recovery: false, connectionId: 'connection' };
  const exportRecovery = vi.fn(async () => ({}));
  const transport = {
    scope: { principalId: 'user', workspaceId: 'workspace', sceneId: 'scene', documentId: 'doc' },
    state,
    isActive: () => true,
    sendAwareness: vi.fn(),
    retry: vi.fn(),
  };
  return {
    props: null as Record<string, unknown> | null,
    commentPrincipal: null as Record<string, unknown> | null,
    exportRecovery,
    collab: {
      adapter: { content: () => content, reorderPresentation: vi.fn() },
      transport,
      state,
      content,
      legacy: null,
      sceneRef: { workspaceId: 'workspace', sceneId: 'scene' },
      undo: { canUndo: () => false, canRedo: () => false, undo: vi.fn(), redo: vi.fn() },
      principal: { current: { kind: 'user', id: 'user' } },
      presence: null,
      voice: null,
      post: vi.fn(),
      exportRecovery,
      applyLocalScene: vi.fn(),
      applyAgentElements: vi.fn(),
    },
  };
});

vi.mock('@/collab/provider', () => ({
  CollabProvider: ({ children }: { children: unknown }) => children,
  useCollab: () => mock.collab,
}));

vi.mock('@/components/whiteboard/excalidraw-module', () => ({
  loadExcalidrawModule: async () => {
  const MainMenu = Object.assign(
    ({ children }: { children?: ReactNode }) => <div data-testid="native-main-menu">{children}</div>,
    {
      Item: ({ children, onSelect }: { children?: ReactNode; onSelect?: () => void }) => <button onClick={onSelect} type="button">{children}</button>,
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

vi.mock('@/components/activity/activity-panel', () => ({
  ActivityPanel: ({ sceneId, workspaceId }: { sceneId: string; workspaceId: string }) => <div>{`${workspaceId}:${sceneId}`}</div>,
}));

vi.mock('@/components/comments/comment-panel', () => ({
  CommentPanel: ({ principal }: { principal: Record<string, unknown> }) => {
    mock.commentPrincipal = principal;
    return <div data-testid="comment-panel-stub" />;
  },
}));

import { CollaborativeCanvas } from '@/components/whiteboard/excalidraw-wrapper';
import useWorkspaceStore from '@/store/workspace-store';

afterEach(() => {
  cleanup();
  mock.props = null;
  mock.commentPrincipal = null;
  mock.collab.state.status = 'synced';
  mock.collab.state.canEdit = true;
  useWorkspaceStore.setState({ workspaces: [], activeWorkspaceId: null });
  mock.exportRecovery.mockReset();
  mock.exportRecovery.mockResolvedValue({});
  vi.unstubAllGlobals();
});

it('uses a full viewport canvas with native top-right and main-menu controls', async () => {
  const { container } = render(<CollaborativeCanvas onBack={vi.fn()} title="Scene title" />);
  await waitFor(() => expect(screen.getByTestId('native-canvas')).toBeInTheDocument());
  expect(container.querySelector('header')).toBeNull();
  expect(container.querySelector('main')).toHaveClass('h-[100dvh]');
  expect(mock.props?.renderTopRightUI).toEqual(expect.any(Function));
  expect(screen.getByTestId('native-main-menu')).toBeInTheDocument();
  expect(screen.getByRole('button', { name: 'Back to dashboard' })).toBeInTheDocument();
  expect(screen.getByRole('button', { name: 'Open AI assistant' })).toBeInTheDocument();
  fireEvent.click(screen.getByRole('button', { name: 'Open comments' }));
  expect(screen.getByTestId('scene-panel-layer')).toHaveClass('absolute', 'inset-0');
  expect(container.querySelector('main')).toHaveClass('h-[100dvh]');
});

it('waits for the recovery export before rejoining a revoked scene', async () => {
  let finishExport!: (value: object) => void;
  mock.collab.state.status = 'revoked';
  mock.exportRecovery.mockReturnValue(new Promise(resolve => { finishExport = resolve; }));
  vi.stubGlobal('URL', { ...URL, createObjectURL: vi.fn(() => 'blob:recovery'), revokeObjectURL: vi.fn() });
  const onRejoin = vi.fn();
  render(<CollaborativeCanvas onRejoin={onRejoin} title="Revoked scene" />);
  const action = await screen.findByRole('button', { name: 'Export recovery and rejoin' });
  fireEvent.click(action);
  expect(mock.exportRecovery).toHaveBeenCalledOnce();
  expect(onRejoin).not.toHaveBeenCalled();
  finishExport({ format: 'recovery' });
  await waitFor(() => expect(onRejoin).toHaveBeenCalledOnce());
});

it('opens scene activity as an overlay and returns focus on Escape', async () => {
  render(<CollaborativeCanvas title="Activity scene" />);
  const trigger = await screen.findByRole('button', { name: 'Open activity' });
  fireEvent.click(trigger);
  expect(screen.getByRole('dialog', { name: 'Activity' })).toBeInTheDocument();
  expect(screen.getByText('workspace:scene')).toBeInTheDocument();
  fireEvent.keyDown(document, { key: 'Escape' });
  expect(screen.queryByRole('dialog', { name: 'Activity' })).not.toBeInTheDocument();
  expect(trigger).toHaveFocus();
});

it('does not grant comment moderation when a workspace editor is a scene viewer', async () => {
  useWorkspaceStore.setState({
    activeWorkspaceId: 'workspace',
    workspaces: [{ id: 'workspace', name: 'Workspace', ownerId: 'owner', role: 'editor', createdAt: '2026-10-20T00:00:00Z' }],
  });
  mock.collab.state.canEdit = false;
  render(<CollaborativeCanvas title="Read-only scene" />);
  fireEvent.click(await screen.findByRole('button', { name: 'Open comments' }));
  await waitFor(() => expect(mock.commentPrincipal).toMatchObject({ kind: 'user', canResolve: false }));
});
