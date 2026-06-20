import { create } from "zustand";
import { api } from "../api";

// Minimal auth state. The token itself lives in localStorage (see api.ts); this
// store mirrors "are we logged in" so React re-renders on login/logout, and
// caches the current user's admin flag for conditional UI.

interface AuthState {
  authed: boolean;
  isAdmin: boolean;
  username: string | null;
  setSession: (isAdmin: boolean) => void;
  setUser: (username: string, isAdmin: boolean) => void;
  logout: () => void;
}

export const useAuth = create<AuthState>((set) => ({
  authed: api.isAuthenticated(),
  isAdmin: false,
  username: null,
  setSession: (isAdmin) => set({ authed: true, isAdmin }),
  setUser: (username, isAdmin) => set({ username, isAdmin }),
  logout: () => {
    api.logout();
    set({ authed: false, isAdmin: false, username: null });
  },
}));
