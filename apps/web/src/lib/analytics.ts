export const analyticsEventNames = {
  authLoginRequested: "auth.session.login_requested",
  authLoginFailed: "auth.session.login_failed",
  authLoginSucceeded: "auth.session.login_succeeded",
  authLoggedOut: "auth.session.logged_out",
  tripCreated: "trip_created"
} as const;

export type AnalyticsEventName = (typeof analyticsEventNames)[keyof typeof analyticsEventNames];
export type AnalyticsContextValue = string | number | boolean;

export interface AnalyticsEvent {
  name: AnalyticsEventName;
  context?: Record<string, AnalyticsContextValue>;
}

const recentEventCache = new Map<string, number>();
const dedupeWindowMs = 1000;

function getEventKey(event: AnalyticsEvent) {
  return `${event.name}:${JSON.stringify(event.context ?? {})}`;
}

export function shouldTrackEvent(event: AnalyticsEvent, now = Date.now()) {
  const key = getEventKey(event);
  const lastSentAt = recentEventCache.get(key);
  if (lastSentAt && now-lastSentAt < dedupeWindowMs) {
    return false;
  }

  recentEventCache.set(key, now);

  for (const [cachedKey, cachedAt] of recentEventCache.entries()) {
    if (now-cachedAt >= dedupeWindowMs) {
      recentEventCache.delete(cachedKey);
    }
  }

  return true;
}

export function resetAnalyticsEventCache() {
  recentEventCache.clear();
}

export function trackEvent(event: AnalyticsEvent) {
  if (!shouldTrackEvent(event)) {
    return;
  }

  if (import.meta.env.DEV) {
    console.info("[analytics]", event.name, event.context ?? {});
  }
}
