// @vitest-environment jsdom

import { StrictMode } from "react";
import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { MemoryRouter } from "react-router-dom";
import { I18nProvider } from "../lib/i18n";
import { ShellNav } from "./shell-nav";
import { useTripsQuery } from "../lib/queries";

vi.mock("../lib/queries", () => ({
  useTripsQuery: vi.fn()
}));

function mockTripsQuery(data: Array<{ id: string }>) {
  vi.mocked(useTripsQuery).mockReturnValue({
    data
  } as unknown as ReturnType<typeof useTripsQuery>);
}

function renderShellNav(initialEntry = "/") {
  return render(
    <StrictMode>
      <I18nProvider>
        <MemoryRouter initialEntries={[initialEntry]}>
          <ShellNav />
        </MemoryRouter>
      </I18nProvider>
    </StrictMode>
  );
}

describe("ShellNav", () => {
  beforeEach(() => {
    window.localStorage.setItem("tt.locale", "en");
    vi.clearAllMocks();
  });

  afterEach(() => {
    cleanup();
  });

  it("keeps trip sections disabled when no active trip is available", () => {
    mockTripsQuery([]);

    renderShellNav("/");

    expect(screen.getByRole("link", { name: "Overview" }).getAttribute("href")).toBe("/");
    expect(screen.queryByRole("link", { name: "Trip" })).toBeNull();
    expect(screen.getByText("Trip").getAttribute("aria-disabled")).toBe("true");
    expect(screen.queryByRole("link", { name: "Itinerary" })).toBeNull();
    expect(screen.queryByRole("link", { name: "Budget" })).toBeNull();
  });

  it("marks only the exact trip child route as active", () => {
    mockTripsQuery([{ id: "trip-1" }]);

    renderShellNav("/trips/trip-1/itinerary");

    expect(screen.getByRole("link", { name: "Itinerary" }).getAttribute("aria-current")).toBe("page");
    expect(screen.getByRole("link", { name: "Trip" }).getAttribute("aria-current")).toBeNull();
    expect(screen.getByRole("link", { name: "Overview" }).getAttribute("aria-current")).toBeNull();
  });
});
