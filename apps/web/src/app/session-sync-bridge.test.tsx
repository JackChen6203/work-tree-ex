// @vitest-environment jsdom

import { StrictMode } from "react";
import { cleanup, render } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it } from "vitest";
import { SessionSyncBridge } from "./session-sync-bridge";
import { resetSessionStore, useSessionStore } from "../store/session-store";
import { broadcastSessionSignedIn, broadcastSessionSignedOut, parseSessionSyncPayload, sessionSyncStorageKey } from "../lib/session-sync";

function renderSessionSyncBridge() {
  return render(
    <StrictMode>
      <SessionSyncBridge />
    </StrictMode>
  );
}

describe("SessionSyncBridge", () => {
  beforeEach(() => {
    resetSessionStore();
  });

  afterEach(() => {
    cleanup();
    resetSessionStore();
    window.localStorage.removeItem(sessionSyncStorageKey);
  });

  it("applies signed-in events from another tab", () => {
    renderSessionSyncBridge();
    broadcastSessionSignedIn({ id: "u1", name: "Demo", email: "demo@example.com", avatar: "DM" }, ["owner"]);

    const payload = parseSessionSyncPayload(window.localStorage.getItem(sessionSyncStorageKey));
    window.dispatchEvent(
      new StorageEvent("storage", {
        key: sessionSyncStorageKey,
        newValue: JSON.stringify(payload)
      })
    );

    expect(useSessionStore.getState().user?.email).toBe("demo@example.com");
    expect(useSessionStore.getState().roles).toEqual(["owner"]);
    expect(useSessionStore.getState().hydrated).toBe(true);
  });

  it("clears the session when another tab signs out", () => {
    useSessionStore.getState().setUser({ id: "u1", name: "Demo", email: "demo@example.com", avatar: "DM" }, ["owner"]);
    renderSessionSyncBridge();
    broadcastSessionSignedOut();

    const payload = parseSessionSyncPayload(window.localStorage.getItem(sessionSyncStorageKey));
    window.dispatchEvent(
      new StorageEvent("storage", {
        key: sessionSyncStorageKey,
        newValue: JSON.stringify(payload)
      })
    );

    expect(useSessionStore.getState().user).toBeNull();
    expect(useSessionStore.getState().roles).toEqual([]);
  });
});
