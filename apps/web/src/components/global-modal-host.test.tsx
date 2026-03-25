// @vitest-environment jsdom

import { StrictMode } from "react";
import { act, cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { GlobalModalHost } from "./global-modal-host";
import { resetUiStore, useUiStore } from "../store/ui-store";
import { I18nProvider } from "../lib/i18n";

function renderModalHost() {
  return render(
    <StrictMode>
      <I18nProvider>
        <GlobalModalHost />
      </I18nProvider>
    </StrictMode>
  );
}

describe("GlobalModalHost", () => {
  beforeEach(() => {
    localStorage.setItem("tt.locale", "en");
    resetUiStore();
  });

  afterEach(() => {
    cleanup();
    localStorage.removeItem("tt.locale");
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

  it("renders adopt-draft modal and passes hasWarnings to confirm handler", async () => {
    const onConfirm = vi.fn(async () => Promise.resolve());
    useUiStore.getState().openAdoptDraftModal({
      draftId: "draft-1",
      tripId: "trip-1",
      draftTitle: "Budget Balanced",
      hasWarnings: true,
      onConfirm
    });

    renderModalHost();

    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "Adopt this plan" }));
    });

    expect(onConfirm).toHaveBeenCalledWith(true);
    expect(screen.queryByRole("dialog")).toBeNull();
  });
});
