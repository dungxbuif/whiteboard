import { fireEvent, render, screen } from '@testing-library/react';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { WorkspaceSwitcher } from '../components/workspace-switcher';
import useWorkspaceStore from '../store/workspace-store';
import React from 'react';

// Mock workspace store
vi.mock('../store/workspace-store', () => ({
  default: vi.fn(),
}));

describe('WorkspaceSwitcher', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    globalThis.ResizeObserver = class { observe() {} unobserve() {} disconnect() {} };
  });

  it('should render the active workspace name', () => {
    const mockStore = {
      workspaces: [{ id: '1', name: 'Test WS' }],
      activeWorkspaceId: '1',
      setActiveWorkspaceId: vi.fn(),
      fetchWorkspaces: vi.fn(),
    };
    (useWorkspaceStore as any).mockReturnValue(mockStore);

    render(<WorkspaceSwitcher />);

    expect(screen.getByText('Test WS')).toBeInTheDocument();
  });

  it('should show "Select Workspace" when no active workspace', () => {
    const mockStore = {
      workspaces: [],
      activeWorkspaceId: null,
      setActiveWorkspaceId: vi.fn(),
      fetchWorkspaces: vi.fn(),
    };
    (useWorkspaceStore as any).mockReturnValue(mockStore);

    render(<WorkspaceSwitcher />);

    expect(screen.getByText('Select Workspace')).toBeInTheDocument();
  });

  it('only offers rename workspace to the owner', () => {
    const mockStore = { workspaces: [{ id: '1', name: 'Team', role: 'editor' }], activeWorkspaceId: '1', createWorkspace: vi.fn(), renameWorkspace: vi.fn() };
    (useWorkspaceStore as any).mockImplementation((selector?: (state: typeof mockStore) => unknown) => selector ? selector(mockStore) : mockStore);
    const view = render(<WorkspaceSwitcher />);
    fireEvent.pointerDown(screen.getByRole('combobox'), { button: 0, ctrlKey: false, pointerType: 'mouse' });
    expect(screen.queryByText('Rename workspace')).not.toBeInTheDocument();
    view.unmount();
    mockStore.workspaces[0].role = 'owner';
    render(<WorkspaceSwitcher />);
    fireEvent.pointerDown(screen.getByRole('combobox'), { button: 0, ctrlKey: false, pointerType: 'mouse' });
    expect(screen.getByText('Rename workspace')).toBeInTheDocument();
  });
});
