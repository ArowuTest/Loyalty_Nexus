import { create } from "zustand";
import { persist } from "zustand/middleware";
import { api } from "@/lib/api";

interface User {
  id: string;
  phone_number: string;
  display_name?: string;
  email?: string;
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
  // wallet is intentionally NOT persisted — it is always fetched live from the
  // API via SWR so that stale cached balances are never shown to users.
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
      wallet: null,           // runtime-only; cleared on every cold start
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
      // ⚠️  wallet is deliberately excluded from this list.
      // Only the auth token and lightweight user profile are persisted.
      // Wallet balances (pulse_points, spin_credits, etc.) must always be
      // fetched fresh from GET /api/v1/user/wallet to avoid stale-cache bugs
      // where a user sees 0 points after an admin top-up.
      partialize: (state) => ({
        token: state.token,
        user: state.user,
        isAuthenticated: state.isAuthenticated,
      }),
      onRehydrateStorage: () => (state) => {
        // After localStorage hydration, mark store as ready and fix any stale flags.
        if (state) {
          if (state.token && !state.isAuthenticated) {
            state.isAuthenticated = true;
          }
          // Sync the API client token so requests work immediately after page reload
          // without waiting for a component to call api.setToken() manually.
          if (state.token) {
            api.setToken(state.token);
          }
          // Ensure wallet always starts null on rehydration — SWR will populate it.
          state.wallet = null;
          state._hasHydrated = true;
        }
      },
    }
  )
);
