// @vitest-environment jsdom

import { StrictMode } from "react";
import { act, cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { GlobalModalHost } from "./global-modal-host";
import { resetUiStore, useUiStore } from "../store/ui-store";

function renderModalHost() {
  return render(
    <StrictMode>
      <GlobalModalHost />
    </StrictMode>
  );
}

describe("GlobalModalHost", () => {
  beforeEach(() => {
    resetUiStore();
  });

  afterEach(() => {
    cleanup();
    resetUiStore();
  });

  it("renders a confirm dialog and closes when cancelled", () => {
    useUiStore.getState().openConfirmModal({
      title: "Confirm logout",
      description: "This will end the current session.",
      confirmLabel: "Logout",
      cancelLabel: "Cancel",
      onConfirm: vi.fn()
    });

    renderModalHost();

    expect(screen.getByRole("dialog")).toBeTruthy();
    fireEvent.click(screen.getByRole("button", { name: "Cancel" }));
    expect(screen.queryByRole("dialog")).toBeNull();
  });

  it("awaits async confirmation and then closes the dialog", async () => {
    const onConfirm = vi.fn(async () => Promise.resolve());
    useUiStore.getState().openConfirmModal({
      title: "Confirm logout",
      description: "This will end the current session.",
      confirmLabel: "Logout",
      cancelLabel: "Cancel",
      onConfirm
    });

    renderModalHost();

    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "Logout" }));
    });

    expect(onConfirm).toHaveBeenCalledTimes(1);
    expect(screen.queryByRole("dialog")).toBeNull();
  });
});
