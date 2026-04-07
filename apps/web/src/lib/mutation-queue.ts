import { useSessionStore } from "../store/session-store";
import { apiFetch } from "./api";
import {
  clearOfflineExpiredData,
  deletePersistedMutation,
  listPersistedMutations,
  savePersistedMutation
} from "./offline-db";

export interface MutationQueueItem {
  id: string;
  idempotencyKey: string;
  method: "POST" | "PATCH" | "DELETE";
  endpoint: string;
  payload: unknown;
  version?: number;
  enqueuedAt: number;
  retryCount: number;
  status: "pending" | "syncing" | "failed";
  failureReason?: string;
}

// ---------- In-memory queue + IndexedDB persistence ----------
const mutationQueue: MutationQueueItem[] = [];
let queueHydrationPromise: Promise<void> | null = null;

// API keys must never enter persistent cache
const SENSITIVE_KEYS = ["apiKey", "secretKey", "token", "password", "accessToken", "refreshToken"];

function sanitizePayload(payload: unknown): unknown {
  if (!payload || typeof payload !== "object") {
    return payload;
  }

  if (Array.isArray(payload)) {
    return payload.map((item) => sanitizePayload(item));
  }

  const sanitized: Record<string, unknown> = {};
  for (const [key, value] of Object.entries(payload as Record<string, unknown>)) {
    if (SENSITIVE_KEYS.some((sensitiveKey) => key.toLowerCase().includes(sensitiveKey.toLowerCase()))) {
      continue;
    }
    sanitized[key] = sanitizePayload(value);
  }
  return sanitized;
}

async function hydrateQueueFromIndexedDb() {
  if (queueHydrationPromise) {
    return queueHydrationPromise;
  }

  queueHydrationPromise = (async () => {
    try {
      await clearOfflineExpiredData();
      const persisted = await listPersistedMutations();
      mutationQueue.splice(0, mutationQueue.length, ...persisted);
      syncPendingCount();
    } catch {
      // Keep queue in-memory only when IndexedDB is unavailable.
    }
  })();

  return queueHydrationPromise;
}

export function enqueueMutation(item: Omit<MutationQueueItem, "id" | "enqueuedAt" | "retryCount" | "status">): string {
  void hydrateQueueFromIndexedDb();

  const entry: MutationQueueItem = {
    ...item,
    id: crypto.randomUUID(),
    payload: sanitizePayload(item.payload),
    enqueuedAt: Date.now(),
    retryCount: 0,
    status: "pending"
  };
  mutationQueue.push(entry);
  void savePersistedMutation(entry);
  syncPendingCount();
  return entry.id;
}

export function getPendingMutations(): MutationQueueItem[] {
  return mutationQueue.filter((m) => m.status === "pending");
}

export function getFailedMutations(): MutationQueueItem[] {
  return mutationQueue.filter((m) => m.status === "failed");
}

export function resolveMutation(id: string) {
  const index = mutationQueue.findIndex((m) => m.id === id);
  if (index >= 0) {
    mutationQueue.splice(index, 1);
    void deletePersistedMutation(id);
  }
  syncPendingCount();
}

export function markMutationFailed(id: string, reason: string) {
  const item = mutationQueue.find((m) => m.id === id);
  if (item) {
    item.status = "failed";
    item.failureReason = reason;
    item.retryCount++;
    void savePersistedMutation(item);
  }
  syncPendingCount();
}

function syncPendingCount() {
  const pending = mutationQueue.filter((m) => m.status === "pending" || m.status === "syncing").length;
  // Update session store pending count for badge display
  const state = useSessionStore.getState();
  if (state.pendingMutations !== pending) {
    useSessionStore.setState({ pendingMutations: pending });
  }
}

// ---------- Sequential replay on reconnect ----------
export async function replayPendingMutations(
  executor: (item: MutationQueueItem) => Promise<void> = executeQueuedMutation
): Promise<{ succeeded: number; failed: number }> {
  await hydrateQueueFromIndexedDb();

  let succeeded = 0;
  let failed = 0;

  const pending = getPendingMutations();
  for (const item of pending) {
    item.status = "syncing";
    void savePersistedMutation(item);
    try {
      await executor(item);
      resolveMutation(item.id);
      succeeded++;
    } catch (error) {
      const reason = error instanceof Error ? error.message : "Unknown error";
      markMutationFailed(item.id, reason);
      failed++;
    }
  }

  return { succeeded, failed };
}

async function executeQueuedMutation(item: MutationQueueItem) {
  const response = await apiFetch(item.endpoint, {
    method: item.method,
    headers: {
      "Content-Type": "application/json",
      "Idempotency-Key": item.idempotencyKey,
      ...(typeof item.version === "number" ? { "If-Match-Version": String(item.version) } : {})
    },
    body: item.method === "DELETE" ? undefined : JSON.stringify(item.payload)
  });

  if (!response.ok) {
    throw new Error(`Queued mutation failed with status ${response.status}`);
  }
}

// ---------- Original simple wrapper (backward compat) ----------
export async function trackQueuedMutation<T>(scope: string, run: () => Promise<T>) {
  const state = useSessionStore.getState();
  const mutationId = state.enqueuePendingMutation(scope);

  try {
    return await run();
  } finally {
    useSessionStore.getState().resolvePendingMutation(mutationId);
  }
}

// Listen for online → auto-replay
if (typeof window !== "undefined") {
  void hydrateQueueFromIndexedDb();

  window.addEventListener("online", () => {
    void replayPendingMutations();
  });
}
