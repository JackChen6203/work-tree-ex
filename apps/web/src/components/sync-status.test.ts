import { describe, expect, it } from "vitest";
import { buildSyncStatusSnapshot } from "./sync-status";

describe("sync status snapshot", () => {
  it("builds queue scopes and conflict summary", () => {
    const snapshot = buildSyncStatusSnapshot({
      changedTrips: [{ id: "trip-1" }],
      changedDays: [],
      changedNotifications: [{ id: "n-1" }],
      pendingMutationRecords: [
        { id: "1", scope: "trips.create", createdAt: 1 },
        { id: "2", scope: "sync.flush", createdAt: 2 },
        { id: "3", scope: "sync.flush", createdAt: 3 }
      ],
      flushData: {
        acceptedCount: 1,
        conflictCount: 1,
        conflicts: [{ id: "m-1", entityId: "trip-1", reason: "version_conflict", expectedVersion: 2 }]
      }
    });

    expect(snapshot.changedTotal).toBe(2);
    expect(snapshot.queueScopes).toEqual(["trips.create", "sync.flush"]);
    expect(snapshot.hasConflicts).toBe(true);
    expect(snapshot.acceptedCount).toBe(1);
    expect(snapshot.conflictCount).toBe(1);
    expect(snapshot.conflicts[0]?.entityId).toBe("trip-1");
  });
});
