import { beforeEach, describe, expect, it, vi } from "vitest";
import { flushSyncMutations, getSyncBootstrap } from "./sync-api";

describe("sync api", () => {
  beforeEach(() => {
    vi.restoreAllMocks();
  });

  it("builds bootstrap query with tripId and sinceVersion", async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({
        data: {
          serverTime: "2026-03-22T00:00:00Z",
          sinceVersion: 3,
          tripId: "trip-1",
          fullResyncRequired: false,
          changedTrips: [],
          changedDays: [],
          changedNotifications: []
        }
      })
    });
    vi.stubGlobal("fetch", fetchMock);

    await getSyncBootstrap("trip-1", 3);

    expect(fetchMock).toHaveBeenCalledWith(
      "http://localhost:8080/api/v1/sync/bootstrap?sinceVersion=3&tripId=trip-1",
      expect.anything()
    );
  });

  it("posts sync mutations flush payload", async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({
        data: {
          tripId: "trip-1",
          acceptedCount: 1,
          conflictCount: 0,
          conflicts: [],
          nextVersion: 2,
          serverTime: "2026-03-22T00:00:00Z"
        }
      })
    });
    vi.stubGlobal("fetch", fetchMock);

    await flushSyncMutations("trip-1", [
      {
        id: "m-1",
        entityType: "itinerary_item",
        entityId: "i-1",
        baseVersion: 0
      }
    ]);

    expect(fetchMock).toHaveBeenCalledWith(
      "http://localhost:8080/api/v1/sync/mutations/flush",
      expect.objectContaining({
        method: "POST",
        body: JSON.stringify({
          tripId: "trip-1",
          mutations: [
            {
              id: "m-1",
              entityType: "itinerary_item",
              entityId: "i-1",
              baseVersion: 0
            }
          ]
        })
      })
    );
  });
});
