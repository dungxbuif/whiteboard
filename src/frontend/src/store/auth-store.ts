import { create } from 'zustand';
import api from '../lib/api';
import { ssoClient } from '../lib/sso-api';

export interface User {
  id: string;
  email: string;
  name: string;
  avatarUrl?: string;
}

interface AuthState {
  user: User | null;
  token: string | null;
  isAuthenticated: boolean;
  isLoading: boolean;
  login: (email: string, password: string) => Promise<void>;
  loginWithSSO: (connectionId: string, returnPath?: string) => Promise<void>;
  logout: () => void;
  checkAuth: () => Promise<void>;
}

const useAuthStore = create<AuthState>((set) => ({
  user: null,
  token: null,
  isAuthenticated: false,
  isLoading: true,

  login: async (email, password) => {
    const response = await api.post<{ token: string }>('/auth/login', { email, password });
    localStorage.setItem('auth_token', response.data.token);
    set({ token: response.data.token, isLoading: true });
    try {
      const userResponse = await api.get<User>('/users/me');
      set({ user: userResponse.data, isAuthenticated: true, isLoading: false });
    } catch (error) {
      localStorage.removeItem('auth_token');
      set({ user: null, token: null, isAuthenticated: false, isLoading: false });
      throw error;
    }
  },

  loginWithSSO: async (connectionId, returnPath = window.location.pathname + window.location.search) => {
    const result = await ssoClient.begin(connectionId, returnPath);
    window.location.assign(result.url);
  },

  logout: () => {
    localStorage.removeItem('auth_token');
    set({
      user: null,
      token: null,
      isAuthenticated: false,
      isLoading: false,
    });
  },

  checkAuth: async () => {
    let token = localStorage.getItem('auth_token');
    const query = new URLSearchParams(window.location.search);
    const exchangeCode = window.location.pathname === '/auth/sso/exchange' ? query.get('code') : null;
    if (exchangeCode) {
      try {
        const result = await ssoClient.exchange(exchangeCode);
        token = result.token;
        localStorage.setItem('auth_token', token);
        const target = query.get('return');
        const safeTarget = target?.startsWith('/') && !target.startsWith('//') ? target : '/';
        window.history.replaceState({}, document.title, safeTarget);
      } catch {
        localStorage.removeItem('auth_token');
        window.history.replaceState({}, document.title, '/');
        set({ user: null, token: null, isAuthenticated: false, isLoading: false });
        return;
      }
    }
    if (!token) {
      set({ isLoading: false });
      return;
    }

    set({ token, isLoading: true });
    try {
      const response = await api.get<User>('/users/me');
      set({
        user: response.data,
        isAuthenticated: true,
        isLoading: false,
      });
    } catch (error) {
      localStorage.removeItem('auth_token');
      set({
        user: null,
        token: null,
        isAuthenticated: false,
        isLoading: false,
      });
    }
  },
}));

export default useAuthStore;
