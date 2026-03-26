// @vitest-environment jsdom

import { StrictMode } from "react";
import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { MemoryRouter } from "react-router-dom";
import { I18nProvider } from "../lib/i18n";
import { useTripsQuery } from "../lib/queries";
import { BottomSheetHost } from "./bottom-sheet-host";
import { resetSessionStore, useSessionStore } from "../store/session-store";
import { resetUiStore, useUiStore } from "../store/ui-store";

vi.mock("../lib/queries", () => ({
  useTripsQuery: vi.fn()
}));

function mockTripsQuery(data: Array<{ id: string }>) {
  vi.mocked(useTripsQuery).mockReturnValue({
    data
  } as unknown as ReturnType<typeof useTripsQuery>);
}

function renderBottomSheetHost(initialEntry = "/") {
  return render(
    <StrictMode>
      <I18nProvider>
        <MemoryRouter initialEntries={[initialEntry]}>
          <BottomSheetHost />
        </MemoryRouter>
      </I18nProvider>
    </StrictMode>
  );
}

describe("BottomSheetHost", () => {
  beforeEach(() => {
    window.localStorage.setItem("tt.locale", "en");
    resetUiStore();
    resetSessionStore();
    useSessionStore.setState({
      user: { id: "u1", name: "Demo User", email: "demo@example.com", avatar: "DU" },
      hydrated: true
    });
    vi.clearAllMocks();
  });

  afterEach(() => {
    cleanup();
    resetUiStore();
    resetSessionStore();
  });

  it("renders the mobile navigation sheet and closes it", () => {
    mockTripsQuery([{ id: "trip-1" }]);
    useUiStore.getState().openSheet("mobile-nav");

    renderBottomSheetHost("/trips/trip-1");

    expect(screen.getByRole("dialog")).toBeTruthy();
    expect(screen.getByRole("heading", { name: "Navigation" })).toBeTruthy();
    fireEvent.click(screen.getAllByRole("button", { name: "Close" })[1]!);
    expect(screen.queryByRole("dialog")).toBeNull();
  });

  it("shows disabled trip sections when no trip is available", () => {
    mockTripsQuery([]);
    useUiStore.getState().openSheet("mobile-nav");

    renderBottomSheetHost("/");

    expect(screen.queryByRole("link", { name: "Trip" })).toBeNull();
    expect(screen.getByText("Trip").getAttribute("aria-disabled")).toBe("true");
  });
});
