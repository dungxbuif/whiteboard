import { describe, it, expect, beforeEach, vi } from 'vitest';
import useAuthStore from '../store/auth-store';
import api from '../lib/api';

vi.mock('../lib/api', () => ({
  default: {
    defaults: { baseURL: '/api/v1' },
    get: vi.fn(),
  },
}));

describe('AuthStore', () => {
  beforeEach(() => {
    localStorage.clear();
    useAuthStore.setState({
      user: null,
      token: null,
      isAuthenticated: false,
      isLoading: true,
    });
    vi.clearAllMocks();
  });

  it('should initialize with default state', () => {
    const state = useAuthStore.getState();
    expect(state.user).toBeNull();
    expect(state.isAuthenticated).toBe(false);
  });

  it('should logout correctly', () => {
    localStorage.setItem('auth_token', 'old-token');
    useAuthStore.setState({ token: 'old-token', isAuthenticated: true });

    useAuthStore.getState().logout();

    const state = useAuthStore.getState();
    expect(state.token).toBeNull();
    expect(state.isAuthenticated).toBe(false);
    expect(localStorage.getItem('auth_token')).toBeNull();
  });

  it('should restore session in checkAuth', async () => {
    const mockUser = { id: '1', email: 'test@example.com', name: 'Test User' };
    localStorage.setItem('auth_token', 'valid-token');
    (api.get as any).mockResolvedValue({ data: mockUser });

    await useAuthStore.getState().checkAuth();

    const state = useAuthStore.getState();
    expect(state.user).toEqual(mockUser);
    expect(state.isAuthenticated).toBe(true);
  });

  it('should consume an OAuth token from the callback hash', async () => {
    const mockUser = { id: '1', email: 'test@example.com', name: 'Test User' };
    window.history.replaceState({}, '', '/#token=oauth-token');
    (api.get as any).mockResolvedValue({ data: mockUser });

    await useAuthStore.getState().checkAuth();

    const state = useAuthStore.getState();
    expect(state.token).toBe('oauth-token');
    expect(state.user).toEqual(mockUser);
    expect(localStorage.getItem('auth_token')).toBe('oauth-token');
    expect(window.location.hash).toBe('');
  });
});
