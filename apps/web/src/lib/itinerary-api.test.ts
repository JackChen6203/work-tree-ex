import { beforeEach, describe, expect, it, vi } from "vitest";
import { createItineraryItem, deleteItineraryItem, patchItineraryItem, reorderItineraryItems } from "./itinerary-api";

describe("itinerary api write operations", () => {
  beforeEach(() => {
    vi.restoreAllMocks();
  });

  it("sends idempotency header for create and reorder", async () => {
    vi.stubGlobal("crypto", { randomUUID: () => "22222222-2222-2222-2222-222222222222" } as unknown as Crypto);
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({ data: { id: "i-1", dayId: "day-1", title: "Morning", itemType: "custom", allDay: false, sortOrder: 1, version: 1 } })
    });
    vi.stubGlobal("fetch", fetchMock);

    await createItineraryItem("trip-1", {
      dayId: "day-1",
      title: "Morning",
      itemType: "custom",
      allDay: false,
      note: "hello"
    });

    await reorderItineraryItems("trip-1", {
      operations: [{ itemId: "i-1", targetDayId: "day-1", targetSortOrder: 1 }]
    });

    expect(fetchMock).toHaveBeenNthCalledWith(
      1,
      "http://localhost:8080/api/v1/trips/trip-1/items",
      expect.objectContaining({
        method: "POST",
        headers: expect.objectContaining({ "Idempotency-Key": "22222222-2222-2222-2222-222222222222" })
      })
    );

    expect(fetchMock).toHaveBeenNthCalledWith(
      2,
      "http://localhost:8080/api/v1/trips/trip-1/items/reorder",
      expect.objectContaining({
        method: "POST",
        headers: expect.objectContaining({ "Idempotency-Key": "22222222-2222-2222-2222-222222222222" })
      })
    );
  });

  it("sends if-match version header for patch updates", async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({ data: { id: "i-1", dayId: "day-1", title: "Edited", itemType: "custom", allDay: false, sortOrder: 1, version: 2 } })
    });
    vi.stubGlobal("fetch", fetchMock);

    await patchItineraryItem("trip-1", "i-1", 1, {
      title: "Edited",
      note: "updated note"
    });

    expect(fetchMock).toHaveBeenCalledWith(
      "http://localhost:8080/api/v1/trips/trip-1/items/i-1",
      expect.objectContaining({
        method: "PATCH",
        headers: expect.objectContaining({ "If-Match-Version": "1" }),
        body: JSON.stringify({ title: "Edited", note: "updated note" })
      })
    );
  });

  it("sends delete request for itinerary item removal", async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({ data: null })
    });
    vi.stubGlobal("fetch", fetchMock);

    await deleteItineraryItem("trip-1", "i-1");

    expect(fetchMock).toHaveBeenCalledWith(
      "http://localhost:8080/api/v1/trips/trip-1/items/i-1",
      expect.objectContaining({
        method: "DELETE"
      })
    );
  });

  it("normalizes create payload warnings into item response", async () => {
    vi.stubGlobal("crypto", { randomUUID: () => "22222222-2222-2222-2222-222222222222" } as unknown as Crypto);
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({
        data: {
          item: {
            id: "i-1",
            dayId: "day-1",
            title: "Morning",
            itemType: "custom",
            allDay: false,
            sortOrder: 1,
            version: 1
          },
          warnings: ["time overlap between 'A' and 'B'"]
        }
      })
    });
    vi.stubGlobal("fetch", fetchMock);

    const created = await createItineraryItem("trip-1", {
      dayId: "day-1",
      title: "Morning",
      itemType: "custom",
      allDay: false
    });

    expect(created.warnings).toEqual(["time overlap between 'A' and 'B'"]);
  });
});
