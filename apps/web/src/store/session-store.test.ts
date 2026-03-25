import { beforeEach, describe, expect, it, vi } from "vitest";
import { getSession } from "../lib/auth-api";
import { resetSessionStore, useSessionStore } from "./session-store";

vi.mock("../lib/auth-api", () => ({
  getSession: vi.fn()
}));

describe("session store", () => {
  beforeEach(() => {
    resetSessionStore();
    vi.clearAllMocks();
  });

  it("hydrates the active session once for concurrent calls", async () => {
    const getSessionMock = vi.mocked(getSession).mockResolvedValue({
      user: { id: "u1", name: "Demo", email: "demo@example.com", avatar: "DM" },
      roles: ["owner"]
    });

    const firstHydration = useSessionStore.getState().hydrate();
    const secondHydration = useSessionStore.getState().hydrate();

    await Promise.all([firstHydration, secondHydration]);

    expect(getSessionMock).toHaveBeenCalledTimes(1);
    expect(useSessionStore.getState().hydrated).toBe(true);
    expect(useSessionStore.getState().user).toEqual({
      id: "u1",
      name: "Demo",
      email: "demo@example.com",
      avatar: "DM"
    });
    expect(useSessionStore.getState().roles).toEqual(["owner"]);
  });

  it("falls back to a signed-out state when hydration fails", async () => {
    vi.mocked(getSession).mockRejectedValue(new Error("session failed"));

    await useSessionStore.getState().hydrate();

    expect(useSessionStore.getState().hydrated).toBe(true);
    expect(useSessionStore.getState().user).toBeNull();
    expect(useSessionStore.getState().roles).toEqual([]);
  });
});
