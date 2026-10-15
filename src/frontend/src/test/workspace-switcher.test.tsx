import { render, screen } from '@testing-library/react';
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
});
