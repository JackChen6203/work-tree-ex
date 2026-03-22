import { create } from "zustand";
import { getSession } from "../lib/auth-api";
import type { SessionUser } from "../types/domain";

interface PendingMutationRecord {
  id: string;
  scope: string;
  createdAt: number;
}

interface SessionState {
  hydrated: boolean;
  user: SessionUser | null;
  isOnline: boolean;
  pendingMutations: number;
  pendingMutationRecords: PendingMutationRecord[];
  hydrate: () => Promise<void>;
  setUser: (user: SessionUser | null) => void;
  clearUser: () => void;
  setOnline: (isOnline: boolean) => void;
  enqueuePendingMutation: (scope: string, id?: string) => string;
  resolvePendingMutation: (id: string) => void;
  clearPendingMutations: () => void;
}

export const useSessionStore = create<SessionState>((set) => ({
  hydrated: false,
  user: null,
  isOnline: true,
  pendingMutations: 0,
  pendingMutationRecords: [],
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
  setOnline: (isOnline) => set({ isOnline }),
  enqueuePendingMutation: (scope, id = crypto.randomUUID()) => {
    set((state) => {
      const pendingMutationRecords = [...state.pendingMutationRecords, { id, scope, createdAt: Date.now() }];
      return {
        pendingMutationRecords,
        pendingMutations: pendingMutationRecords.length
      };
    });
    return id;
  },
  resolvePendingMutation: (id) =>
    set((state) => {
      const pendingMutationRecords = state.pendingMutationRecords.filter((item) => item.id !== id);
      return {
        pendingMutationRecords,
        pendingMutations: pendingMutationRecords.length
      };
    }),
  clearPendingMutations: () => set({ pendingMutationRecords: [], pendingMutations: 0 })
}));
