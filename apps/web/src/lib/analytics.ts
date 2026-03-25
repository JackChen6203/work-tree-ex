export const analyticsEventNames = {
  authLoginRequested: "auth.session.login_requested",
  authLoginFailed: "auth.session.login_failed",
  authLoginSucceeded: "auth.session.login_succeeded",
  authLoggedOut: "auth.session.logged_out",
  tripCreated: "trip.workspace.created",
  tripUpdated: "trip.workspace.updated",
  itineraryItemCreated: "itinerary.item.created",
  itineraryItemReordered: "itinerary.item.reordered",
  budgetUpserted: "budget.profile.upserted",
  expenseCreated: "budget.expense.created",
  aiPlanRequested: "ai.plan.requested",
  aiPlanAdopted: "ai.plan.adopted",
  notificationRead: "notification.inbox.read",
  syncFlushCompleted: "sync.mutation.flushed"
} as const;

export type AnalyticsEventName = (typeof analyticsEventNames)[keyof typeof analyticsEventNames];
export type AnalyticsContextValue = string | number | boolean;

export interface AnalyticsEvent {
  name: AnalyticsEventName;
  context?: Record<string, AnalyticsContextValue>;
}

// ---------- Event naming validation ----------
const EVENT_NAME_PATTERN = /^[a-z]+\.[a-z_]+\.[a-z_]+$/;

export function isValidEventName(name: string): boolean {
  return EVENT_NAME_PATTERN.test(name);
}

// ---------- Sensitive field handling ----------
export function hashSensitiveValue(value: string | number): string {
  const str = String(value);
  let hash = 0;
  for (let i = 0; i < str.length; i++) {
    hash = ((hash << 5) - hash + str.charCodeAt(i)) | 0;
  }
  return `h_${Math.abs(hash).toString(36)}`;
}

export function bucketizeAmount(amount: number): string {
  if (amount <= 0) return "0";
  if (amount <= 1000) return "1-1k";
  if (amount <= 10000) return "1k-10k";
  if (amount <= 100000) return "10k-100k";
  return "100k+";
}

// ---------- Dedupe ----------
const recentEventCache = new Map<string, number>();
const dedupeWindowMs = 1000;

function getEventKey(event: AnalyticsEvent) {
  return `${event.name}:${JSON.stringify(event.context ?? {})}`;
}

export function shouldTrackEvent(event: AnalyticsEvent, now = Date.now()) {
  const key = getEventKey(event);
  const lastSentAt = recentEventCache.get(key);
  if (lastSentAt && now - lastSentAt < dedupeWindowMs) {
    return false;
  }

  recentEventCache.set(key, now);

  for (const [cachedKey, cachedAt] of recentEventCache.entries()) {
    if (now - cachedAt >= dedupeWindowMs) {
      recentEventCache.delete(cachedKey);
    }
  }

  return true;
}

export function resetAnalyticsEventCache() {
  recentEventCache.clear();
}

// ---------- Offline event queue ----------
const offlineQueue: Array<AnalyticsEvent & { occurred_at: string }> = [];

function flushOfflineQueue() {
  while (offlineQueue.length > 0) {
    const event = offlineQueue.shift();
    if (event) {
      sendEvent(event);
    }
  }
}

function sendEvent(event: AnalyticsEvent & { occurred_at?: string }) {
  if (import.meta.env.DEV) {
    console.info("[analytics]", event.name, event.context ?? {});
  }
  // In production, POST to analytics endpoint
  // Failures are silently ignored (FE-11 edge case)
}

// ---------- Context enrichment ----------
function enrichContext(context?: Record<string, AnalyticsContextValue>): Record<string, AnalyticsContextValue> {
  return {
    session_id: sessionStorage.getItem("analytics_session_id") ?? initSessionId(),
    platform: "web",
    locale: navigator.language ?? "en",
    timezone: Intl.DateTimeFormat().resolvedOptions().timeZone ?? "UTC",
    app_version: import.meta.env.VITE_APP_VERSION ?? "0.1.0",
    ...context
  };
}

function initSessionId(): string {
  const id = crypto.randomUUID();
  sessionStorage.setItem("analytics_session_id", id);
  return id;
}

// ---------- Main tracking function ----------
export function trackEvent(event: AnalyticsEvent) {
  if (!shouldTrackEvent(event)) {
    return;
  }

  const enriched: AnalyticsEvent & { occurred_at: string } = {
    ...event,
    context: enrichContext(event.context),
    occurred_at: new Date().toISOString()
  };

  if (!navigator.onLine) {
    offlineQueue.push(enriched);
    return;
  }

  sendEvent(enriched);
}

// Listen for online to flush queued events
if (typeof window !== "undefined") {
  window.addEventListener("online", flushOfflineQueue);
}

