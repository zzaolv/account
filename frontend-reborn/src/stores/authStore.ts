// src/stores/authStore.ts
import { create } from 'zustand';
import { persist, createJSONStorage } from 'zustand/middleware';

interface UserInfo {
  username: string;
  is_admin: boolean;
}

interface AuthState {
  token: string | null;
  refreshToken: string | null;
  user: UserInfo | null;
  mustChangePassword: boolean;
  isAuthenticated: boolean;
  login: (accessToken: string, refreshToken: string | null, user: UserInfo, mustChange: boolean) => void;
  logout: () => void;
  setNewAccessToken: (newAccessToken: string) => void;
  setPasswordChanged: () => void;
}

export const useAuthStore = create<AuthState>()(
  persist(
    (set) => ({
      token: null,
      refreshToken: null,
      user: null,
      mustChangePassword: false,
      isAuthenticated: false,

      login: (accessToken, refreshToken, user, mustChange) => {
        set({
          token: accessToken,
          refreshToken: refreshToken,
          user,
          mustChangePassword: mustChange,
          isAuthenticated: true,
        });
      },

      logout: () => {
        set({
          token: null,
          refreshToken: null,
          user: null,
          mustChangePassword: false,
          isAuthenticated: false,
        });
      },
      
      setNewAccessToken: (newAccessToken: string) => {
        set({ token: newAccessToken });
      },

      setPasswordChanged: () => {
        set({ mustChangePassword: false });
      },
    }),
    {
      name: 'auth-storage',
      storage: createJSONStorage(() => localStorage),
      // 【关键修改】除了 refreshToken，也持久化 user 信息
      partialize: (state) => ({ 
        refreshToken: state.refreshToken, 
        user: state.user 
      }),
    }
  )
);

export const useIsAuthenticated = () => useAuthStore((state) => state.isAuthenticated);
export const useAuthToken = () => useAuthStore((state) => state.token);
export const useCurrentUser = () => useAuthStore((state) => state.user);
export const useMustChangePassword = () => useAuthStore((state) => state.mustChangePassword);