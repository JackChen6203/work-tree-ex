// @vitest-environment jsdom

import { StrictMode } from "react";
import type { ReactElement, ReactNode } from "react";
import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { I18nProvider } from "../lib/i18n";
import { GlobalErrorBoundary } from "./global-error-boundary";

function renderBoundary(ui: ReactNode) {
  return render(
    <StrictMode>
      <I18nProvider>
        <GlobalErrorBoundary>{ui}</GlobalErrorBoundary>
      </I18nProvider>
    </StrictMode>
  );
}

describe("GlobalErrorBoundary", () => {
  beforeEach(() => {
    window.localStorage.setItem("tt.locale", "en");
    vi.spyOn(console, "error").mockImplementation(() => undefined);
  });

  afterEach(() => {
    cleanup();
    vi.restoreAllMocks();
  });

  it("renders the fallback shell when a descendant throws", () => {
    function Boom(): ReactElement {
      throw new Error("Render exploded");
    }

    renderBoundary(<Boom />);

    expect(screen.getByText("Something interrupted the workspace")).toBeTruthy();
    expect(screen.getByText(/Render exploded/)).toBeTruthy();
    expect(screen.getByRole("button", { name: "Try again" })).toBeTruthy();
  });

  it("retries rendering children after reset", () => {
    let shouldThrow = true;

    function FlakyScreen(): ReactElement {
      if (shouldThrow) {
        throw new Error("Temporary crash");
      }

      return <div>recovered screen</div>;
    }

    renderBoundary(<FlakyScreen />);

    shouldThrow = false;
    fireEvent.click(screen.getByRole("button", { name: "Try again" }));

    expect(screen.getByText("recovered screen")).toBeTruthy();
  });
});
