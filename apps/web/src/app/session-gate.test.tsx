// @vitest-environment jsdom

import { StrictMode } from "react";
import { cleanup, render, screen, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { I18nProvider } from "../lib/i18n";
import { getSession } from "../lib/auth-api";
import { SessionGate } from "./session-gate";
import { resetSessionStore } from "../store/session-store";

vi.mock("../lib/auth-api", () => ({
  getSession: vi.fn()
}));

function renderSessionGate() {
  return render(
    <StrictMode>
      <I18nProvider>
        <MemoryRouter initialEntries={["/"]}>
          <Routes>
            <Route
              path="/"
              element={
                <SessionGate>
                  <div>workspace</div>
                </SessionGate>
              }
            />
            <Route path="/welcome" element={<div>welcome screen</div>} />
          </Routes>
        </MemoryRouter>
      </I18nProvider>
    </StrictMode>
  );
}

describe("SessionGate", () => {
  beforeEach(() => {
    window.localStorage.setItem("tt.locale", "en");
    resetSessionStore();
    vi.clearAllMocks();
  });

  afterEach(() => {
    cleanup();
  });

  it("shows hydration UI and avoids duplicate session requests in StrictMode", async () => {
    const deferredSession = {} as {
      resolve: (value: { user: { id: string; name: string; email: string; avatar: string }; roles: string[] }) => void;
      promise: Promise<{ user: { id: string; name: string; email: string; avatar: string }; roles: string[] }>;
    };
    deferredSession.promise = new Promise((resolve) => {
      deferredSession.resolve = resolve;
    });
    const getSessionMock = vi.mocked(getSession).mockReturnValue(deferredSession.promise);

    renderSessionGate();

    expect(screen.getByText("Preparing workspace")).toBeTruthy();

    await waitFor(() => {
      expect(getSessionMock).toHaveBeenCalledTimes(1);
    });

    deferredSession.resolve({
      user: { id: "u1", name: "Demo", email: "demo@example.com", avatar: "DM" },
      roles: ["owner"]
    });

    expect(await screen.findByText("workspace")).toBeTruthy();
  });

  it("redirects to the welcome page when hydration fails", async () => {
    vi.mocked(getSession).mockRejectedValue(new Error("unauthorized"));

    renderSessionGate();

    expect(await screen.findByText("welcome screen")).toBeTruthy();
  });
});
