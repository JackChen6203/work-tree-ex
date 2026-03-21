import { create } from "zustand";
import { getSession } from "../lib/auth-api";
import type { SessionUser } from "../types/domain";

interface SessionState {
  hydrated: boolean;
  user: SessionUser | null;
  isOnline: boolean;
  pendingMutations: number;
  hydrate: () => Promise<void>;
  setUser: (user: SessionUser | null) => void;
  clearUser: () => void;
  setOnline: (isOnline: boolean) => void;
}

export const useSessionStore = create<SessionState>((set) => ({
  hydrated: false,
  user: null,
  isOnline: true,
  pendingMutations: 0,
  hydrate: async () => {
    await new Promise((resolve) => window.setTimeout(resolve, 400));
    try {
      const session = await getSession();
      set({ hydrated: true, user: session.user });
    } catch {
      set({ hydrated: true, user: null });
    }
  },
  setUser: (user) => set({ user }),
  clearUser: () => set({ user: null }),
  setOnline: (isOnline) => set({ isOnline })
}));
