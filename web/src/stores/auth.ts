import { create } from 'zustand';
import { persist, createJSONStorage } from 'zustand/middleware';

interface AuthState {
  localAddress: string | null;
  token: string | null;
  address: string | null;
}

interface AuthActions {
  setLocalAuth: (address: string, token: string) => void;
  setToken: (token: string | null) => void;
  setAddress: (address: string | null) => void;
  clearAuth: () => void;
  isLocalMode: () => boolean;
  isAuthenticated: () => boolean;
}

type AuthStore = AuthState & AuthActions;

export const useAuthStore = create<AuthStore>()(
  persist(
    (set, get) => ({
      localAddress: null,
      token: null,
      address: null,

      setLocalAuth: (address: string, token: string) => set({ localAddress: address, token }),

      setToken: (token: string | null) => set({ token }),

      setAddress: (address: string | null) => set({ address }),

      clearAuth: () => set({ localAddress: null, token: null, address: null }),

      isLocalMode: () => get().localAddress !== null,

      isAuthenticated: () => get().token !== null
    }),
    {
      name: 'auth-storage',
      storage: createJSONStorage(() => localStorage)
    }
  )
);

// Export helper functions for compatibility
export const getToken = () => useAuthStore.getState().token;
export const getAddress = () => useAuthStore.getState().address;
