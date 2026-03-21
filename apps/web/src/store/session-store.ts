import { create } from "zustand";
import { sessionUser } from "../lib/mock-data";
import type { SessionUser } from "../types/domain";

interface SessionState {
  hydrated: boolean;
  user: SessionUser | null;
  isOnline: boolean;
  pendingMutations: number;
  hydrate: () => Promise<void>;
  setOnline: (isOnline: boolean) => void;
}

export const useSessionStore = create<SessionState>((set) => ({
  hydrated: false,
  user: null,
  isOnline: true,
  pendingMutations: 3,
  hydrate: async () => {
    await new Promise((resolve) => window.setTimeout(resolve, 600));
    set({ hydrated: true, user: sessionUser });
  },
  setOnline: (isOnline) => set({ isOnline })
}));
