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
  roles: string[];
  isOnline: boolean;
  pendingMutations: number;
  pendingMutationRecords: PendingMutationRecord[];
  hydrate: () => Promise<void>;
  setUser: (user: SessionUser | null, roles?: string[]) => void;
  clearUser: () => void;
  setOnline: (isOnline: boolean) => void;
  enqueuePendingMutation: (scope: string, id?: string) => string;
  resolvePendingMutation: (id: string) => void;
  clearPendingMutations: () => void;
}

const defaultSessionState = {
  hydrated: false,
  user: null,
  roles: [],
  isOnline: true,
  pendingMutations: 0,
  pendingMutationRecords: []
} satisfies Pick<SessionState, "hydrated" | "user" | "roles" | "isOnline" | "pendingMutations" | "pendingMutationRecords">;

export const useSessionStore = create<SessionState>((set, get) => {
  let inflightHydration: Promise<void> | null = null;

  return {
    ...defaultSessionState,
    hydrate: async () => {
      if (get().hydrated) {
        return;
      }

      if (inflightHydration) {
        return inflightHydration;
      }

      inflightHydration = getSession()
        .then((session) => {
          set({ hydrated: true, user: session.user, roles: session.roles });
        })
        .catch(() => {
          set({ hydrated: true, user: null, roles: [] });
        })
        .finally(() => {
          inflightHydration = null;
        });

      return inflightHydration;
    },
    setUser: (user, roles = []) => set({ hydrated: true, user, roles }),
    clearUser: () => set({ hydrated: true, user: null, roles: [] }),
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
  };
});

export function resetSessionStore() {
  useSessionStore.setState((state) => ({
    ...state,
    ...defaultSessionState
  }));
}
