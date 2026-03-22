import { beforeEach, describe, expect, it } from "vitest";
import { analyticsEventNames, resetAnalyticsEventCache, shouldTrackEvent } from "./analytics";

describe("analytics", () => {
  beforeEach(() => {
    resetAnalyticsEventCache();
  });

  it("allows first event and suppresses duplicate within dedupe window", () => {
    const event = {
      name: analyticsEventNames.authLoginRequested,
      context: { method: "oauth", provider: "google" }
    };

    expect(shouldTrackEvent(event, 1000)).toBe(true);
    expect(shouldTrackEvent(event, 1500)).toBe(false);
    expect(shouldTrackEvent(event, 2200)).toBe(true);
  });

  it("treats different context as different events", () => {
    expect(
      shouldTrackEvent({ name: analyticsEventNames.authLoginFailed, context: { method: "oauth", reason: "timeout" } }, 1000)
    ).toBe(true);
    expect(
      shouldTrackEvent({ name: analyticsEventNames.authLoginFailed, context: { method: "oauth", reason: "cancelled" } }, 1200)
    ).toBe(true);
  });
});
