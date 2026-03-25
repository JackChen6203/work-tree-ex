// @vitest-environment jsdom

import { StrictMode } from "react";
import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { MemoryRouter } from "react-router-dom";
import { I18nProvider } from "../../lib/i18n";
import { AuthPage } from "./auth-page";
import { resetSessionStore } from "../../store/session-store";
import { resetUiStore } from "../../store/ui-store";
import { getSession, oauthStartUrl } from "../../lib/auth-api";
import { useRequestMagicLinkMutation, useVerifyMagicLinkMutation } from "../../lib/queries";

vi.mock("../../lib/queries", () => ({
  useRequestMagicLinkMutation: vi.fn(),
  useVerifyMagicLinkMutation: vi.fn()
}));

vi.mock("../../lib/auth-api", () => ({
  getSession: vi.fn(),
  oauthStartUrl: vi.fn((provider: string) => `/oauth/${provider}`)
}));

function renderAuthPage(initialEntry = "/login") {
  return render(
    <StrictMode>
      <I18nProvider>
        <MemoryRouter initialEntries={[initialEntry]}>
          <AuthPage />
        </MemoryRouter>
      </I18nProvider>
    </StrictMode>
  );
}

describe("AuthPage", () => {
  beforeEach(() => {
    window.localStorage.setItem("tt.locale", "en");
    resetSessionStore();
    resetUiStore();
    vi.clearAllMocks();
    vi.mocked(getSession).mockResolvedValue({ user: null, roles: [] });
    vi.mocked(useVerifyMagicLinkMutation).mockReturnValue({
      mutateAsync: vi.fn(),
      isPending: false,
      error: null
    } as unknown as ReturnType<typeof useVerifyMagicLinkMutation>);
  });

  afterEach(() => {
    cleanup();
    resetSessionStore();
    resetUiStore();
  });

  it("switches to the waiting view after requesting a magic link", async () => {
    vi.mocked(useRequestMagicLinkMutation).mockReturnValue({
      mutateAsync: vi.fn().mockResolvedValue({ sent: true, expiresIn: 600, previewCode: "123456" }),
      isPending: false,
      error: null
    } as unknown as ReturnType<typeof useRequestMagicLinkMutation>);

    renderAuthPage();

    fireEvent.change(screen.getByLabelText("Email"), { target: { value: "demo@example.com" } });
    fireEvent.click(screen.getByRole("button", { name: "Send sign-in link" }));

    await waitFor(() => {
      expect(screen.getByText("Check your sign-in link")).toBeTruthy();
    });

    expect(screen.getByText("demo@example.com")).toBeTruthy();
    expect(screen.getByRole("button", { name: "Request another link" })).toBeTruthy();
    expect(screen.getByText(/Dev preview code/)).toBeTruthy();
  });

  it("renders grouped oauth providers", () => {
    vi.mocked(useRequestMagicLinkMutation).mockReturnValue({
      mutateAsync: vi.fn(),
      isPending: false,
      error: null
    } as unknown as ReturnType<typeof useRequestMagicLinkMutation>);

    renderAuthPage();

    expect(screen.getByText("Social providers")).toBeTruthy();
    expect(screen.getByText("Travel providers")).toBeTruthy();
    expect(screen.getByRole("link", { name: "Google" }).getAttribute("href")).toBe("/oauth/google");
    expect(vi.mocked(oauthStartUrl)).toHaveBeenCalledWith("google");
  });
});
