import { describe, it, expect, beforeEach, vi } from 'vitest';
import useAuthStore from '../store/auth-store';
import api from '../lib/api';
import { ssoClient } from '../lib/sso-api';

vi.mock('../lib/api', () => ({
  default: {
    defaults: { baseURL: '/api/v1' },
    get: vi.fn(),
  },
}));
vi.mock('../lib/sso-api', () => ({ ssoClient: { begin: vi.fn(), exchange: vi.fn() } }));

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

  it('should consume a browser-bound one-time SSO exchange code', async () => {
    const mockUser = { id: '1', email: 'test@example.com', name: 'Test User' };
    window.history.replaceState({}, '', '/auth/sso/exchange?code=one-time&return=%2Fw%2Fdemo');
    vi.mocked(ssoClient.exchange).mockResolvedValue({ token: 'sso-token', user: mockUser });
    (api.get as any).mockResolvedValue({ data: mockUser });

    await useAuthStore.getState().checkAuth();

    const state = useAuthStore.getState();
    expect(state.token).toBe('sso-token');
    expect(state.user).toEqual(mockUser);
    expect(localStorage.getItem('auth_token')).toBe('sso-token');
    expect(window.location.pathname).toBe('/w/demo');
  });
});
