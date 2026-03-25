import type { SessionUser } from "../types/domain";

export const sessionSyncStorageKey = "tt.session.sync";

export interface SessionSyncPayload {
  type: "signed_in" | "signed_out";
  user: SessionUser | null;
  roles: string[];
  occurredAt: number;
}

function writeSessionSyncPayload(payload: SessionSyncPayload) {
  if (typeof window === "undefined") {
    return;
  }

  window.localStorage.setItem(sessionSyncStorageKey, JSON.stringify(payload));
}

export function broadcastSessionSignedIn(user: SessionUser, roles: string[]) {
  writeSessionSyncPayload({
    type: "signed_in",
    user,
    roles,
    occurredAt: Date.now()
  });
}

export function broadcastSessionSignedOut() {
  writeSessionSyncPayload({
    type: "signed_out",
    user: null,
    roles: [],
    occurredAt: Date.now()
  });
}

export function parseSessionSyncPayload(rawValue: string | null): SessionSyncPayload | null {
  if (!rawValue) {
    return null;
  }

  try {
    const parsed = JSON.parse(rawValue) as SessionSyncPayload;
    if (parsed.type !== "signed_in" && parsed.type !== "signed_out") {
      return null;
    }
    return parsed;
  } catch {
    return null;
  }
}
