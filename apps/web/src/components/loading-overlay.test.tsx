// @vitest-environment jsdom

import { StrictMode } from "react";
import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it } from "vitest";
import { LoadingOverlay } from "./loading-overlay";
import { resetUiStore, useUiStore } from "../store/ui-store";

function renderLoadingOverlay() {
  return render(
    <StrictMode>
      <LoadingOverlay />
    </StrictMode>
  );
}

describe("LoadingOverlay", () => {
  beforeEach(() => {
    resetUiStore();
  });

  afterEach(() => {
    cleanup();
    resetUiStore();
  });

  it("stays hidden until the global loading state is enabled", () => {
    renderLoadingOverlay();

    expect(screen.queryByRole("status")).toBeNull();
  });

  it("renders the loading message when the overlay is active", () => {
    useUiStore.getState().showLoadingOverlay("Signing out...");

    renderLoadingOverlay();

    expect(screen.getByRole("status")).toBeTruthy();
    expect(screen.getByText("Signing out...")).toBeTruthy();
  });
});
