interface AnalyticsEvent {
  name: string;
  context?: Record<string, string | number | boolean>;
}

export function trackEvent(event: AnalyticsEvent) {
  if (import.meta.env.DEV) {
    console.info("[analytics]", event.name, event.context ?? {});
  }
}
