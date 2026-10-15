import { create } from 'zustand';
import api from '../lib/api';

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
  startOAuth: () => void;
  logout: () => void;
  checkAuth: () => Promise<void>;
}

const useAuthStore = create<AuthState>((set) => ({
  user: null,
  token: null,
  isAuthenticated: false,
  isLoading: true,

  startOAuth: () => {
    const baseURL = api.defaults.baseURL || '/api/v1';
    window.location.assign(`${baseURL}/auth/google/login`);
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
    const hashToken = new URLSearchParams(window.location.hash.replace(/^#/, '')).get('token');
    if (hashToken) {
      localStorage.setItem('auth_token', hashToken);
      window.history.replaceState({}, document.title, window.location.pathname + window.location.search);
    }
    const token = hashToken || localStorage.getItem('auth_token');
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
