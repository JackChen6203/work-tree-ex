// @vitest-environment jsdom

import { StrictMode } from "react";
import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { I18nProvider } from "../lib/i18n";
import { getSession } from "../lib/auth-api";
import { PublicOnlyGate } from "./public-only-gate";
import { resetSessionStore } from "../store/session-store";

vi.mock("../lib/auth-api", () => ({
  getSession: vi.fn()
}));

function renderPublicOnlyGate(initialEntry = "/login") {
  return render(
    <StrictMode>
      <I18nProvider>
        <MemoryRouter initialEntries={[initialEntry]}>
          <Routes>
            <Route
              path="/login"
              element={
                <PublicOnlyGate>
                  <div>auth screen</div>
                </PublicOnlyGate>
              }
            />
            <Route
              path="/welcome"
              element={
                <PublicOnlyGate>
                  <div>welcome screen</div>
                </PublicOnlyGate>
              }
            />
            <Route path="/" element={<div>workspace</div>} />
          </Routes>
        </MemoryRouter>
      </I18nProvider>
    </StrictMode>
  );
}

describe("PublicOnlyGate", () => {
  beforeEach(() => {
    window.localStorage.setItem("tt.locale", "en");
    resetSessionStore();
    vi.clearAllMocks();
  });

  afterEach(() => {
    cleanup();
  });

  it("keeps guests on public routes after hydration", async () => {
    vi.mocked(getSession).mockResolvedValue({ user: null, roles: [] });

    renderPublicOnlyGate("/login");

    expect(await screen.findByText("auth screen")).toBeTruthy();
  });

  it("redirects authenticated users away from public routes", async () => {
    vi.mocked(getSession).mockResolvedValue({
      user: { id: "u1", name: "Demo", email: "demo@example.com", avatar: "DM" },
      roles: ["owner"]
    });

    renderPublicOnlyGate("/welcome");

    expect(await screen.findByText("workspace")).toBeTruthy();
  });
});
