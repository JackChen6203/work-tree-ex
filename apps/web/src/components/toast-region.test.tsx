// @vitest-environment jsdom

import { StrictMode } from "react";
import { act, cleanup, render, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { ToastRegion } from "./toast-region";
import { resetUiStore, useUiStore } from "../store/ui-store";

function renderToastRegion() {
  return render(
    <StrictMode>
      <ToastRegion />
    </StrictMode>
  );
}

describe("ToastRegion", () => {
  beforeEach(() => {
    vi.useFakeTimers();
    resetUiStore();
  });

  afterEach(() => {
    cleanup();
    resetUiStore();
    vi.useRealTimers();
  });

  it("renders toasts in a polite top-right live region", () => {
    useUiStore.getState().pushToast({ type: "success", message: "Saved successfully" });

    renderToastRegion();

    const liveRegion = screen.getByRole("status");
    expect(liveRegion.getAttribute("aria-live")).toBe("polite");
    expect(screen.getByText("Saved successfully")).toBeTruthy();
  });

  it("dismisses toasts after the default 3 second duration", () => {
    useUiStore.getState().pushToast("Auto-dismiss me");

    renderToastRegion();
    expect(screen.getByText("Auto-dismiss me")).toBeTruthy();

    act(() => {
      vi.advanceTimersByTime(3000);
    });

    expect(screen.queryByText("Auto-dismiss me")).toBeNull();
  });
});
