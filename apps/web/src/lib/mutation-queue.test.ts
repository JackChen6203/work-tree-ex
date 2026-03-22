import { beforeEach, describe, expect, it } from "vitest";
import { trackQueuedMutation } from "./mutation-queue";
import { useSessionStore } from "../store/session-store";

describe("mutation queue", () => {
  beforeEach(() => {
    useSessionStore.getState().clearPendingMutations();
  });

  it("tracks pending mutation lifecycle on success", async () => {
    const pendingPromise = trackQueuedMutation("trips.create", async () => {
      expect(useSessionStore.getState().pendingMutations).toBe(1);
      return "ok";
    });

    await expect(pendingPromise).resolves.toBe("ok");
    expect(useSessionStore.getState().pendingMutations).toBe(0);
  });

  it("cleans up pending mutation on failure", async () => {
    const pendingPromise = trackQueuedMutation("sync.flush", async () => {
      expect(useSessionStore.getState().pendingMutations).toBe(1);
      throw new Error("boom");
    });

    await expect(pendingPromise).rejects.toThrow("boom");
    expect(useSessionStore.getState().pendingMutations).toBe(0);
  });
});
