import { create } from "zustand";
import { persist } from "zustand/middleware";

interface User {
  id: string;
  phone_number: string;
  tier: string;
  streak_count: number;
  is_active: boolean;
}

interface Wallet {
  pulse_points: number;
  spin_credits: number;
  lifetime_points: number;
}

interface AppState {
  token: string | null;
  user: User | null;
  wallet: Wallet | null;
  isAuthenticated: boolean;
  _hasHydrated: boolean;
  setToken: (token: string) => void;
  setUser: (user: User) => void;
  setWallet: (wallet: Wallet) => void;
  logout: () => void;
  setHasHydrated: (val: boolean) => void;
}

export const useStore = create<AppState>()(
  persist(
    (set) => ({
      token: null,
      user: null,
      wallet: null,
      isAuthenticated: false,
      _hasHydrated: false,
      setToken: (token) => set({ token, isAuthenticated: true }),
      setUser: (user) => set({ user }),
      setWallet: (wallet) => set({ wallet }),
      logout: () => set({ token: null, user: null, wallet: null, isAuthenticated: false }),
      setHasHydrated: (val) => set({ _hasHydrated: val }),
    }),
    {
      name: "nexus-store",
      partialize: (state) => ({
        token: state.token,
        user: state.user,
        isAuthenticated: state.isAuthenticated,
      }),
      onRehydrateStorage: () => (state) => {
        // After localStorage hydration, mark store as ready and fix any stale flags
        if (state) {
          if (state.token && !state.isAuthenticated) {
            state.isAuthenticated = true;
          }
          state._hasHydrated = true;
        }
      },
    }
  )
);
