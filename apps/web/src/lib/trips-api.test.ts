import { beforeEach, describe, expect, it, vi } from "vitest";
import { createTrip, getTrip, listTrips, mapTrip, patchTrip } from "./trips-api";

const apiTrip = {
  id: "trip-1",
  name: "Kyoto",
  destinationText: "Kyoto, Japan",
  startDate: "2026-04-14",
  endDate: "2026-04-19",
  timezone: "Asia/Tokyo",
  currency: "JPY",
  travelersCount: 3,
  status: "active" as const,
  version: 4,
  createdAt: "2026-03-21T00:00:00Z",
  updatedAt: "2026-03-21T00:00:00Z"
};

describe("trips api mapping", () => {
  beforeEach(() => {
    vi.restoreAllMocks();
  });

  it("maps backend trip payload to frontend summary fields", () => {
    expect(mapTrip(apiTrip)).toMatchObject({
      id: "trip-1",
      destination: "Kyoto, Japan",
      dateRange: "2026/04/14 - 2026/04/19",
      travelersCount: 3,
      version: 4
    });
  });

  it("lists trips through the API envelope", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({
        ok: true,
        json: async () => ({ data: [apiTrip] })
      })
    );

    const trips = await listTrips();
    expect(trips).toHaveLength(1);
    expect(trips[0].name).toBe("Kyoto");
  });

  it("sends idempotency and version headers on write operations", async () => {
    vi.stubGlobal("crypto", { randomUUID: () => "11111111-1111-1111-1111-111111111111" } as unknown as Crypto);
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({ data: apiTrip })
    });
    vi.stubGlobal("fetch", fetchMock);

    await createTrip({
      name: "Kyoto",
      destinationText: "Kyoto, Japan",
      startDate: "2026-04-14",
      endDate: "2026-04-19",
      timezone: "Asia/Tokyo",
      currency: "JPY",
      travelersCount: 3
    });
    await patchTrip("trip-1", 4, { name: "Kyoto Updated" });
    await getTrip("trip-1");

    expect(fetchMock).toHaveBeenNthCalledWith(
      1,
        "http://localhost:8080/api/v1/trips",
      expect.objectContaining({
        method: "POST",
        headers: expect.objectContaining({ "Idempotency-Key": "11111111-1111-1111-1111-111111111111" })
      })
    );
    expect(fetchMock).toHaveBeenNthCalledWith(
      2,
      "http://localhost:8080/api/v1/trips/trip-1",
      expect.objectContaining({
        method: "PATCH",
        headers: expect.objectContaining({ "If-Match-Version": "4" })
      })
    );
    expect(fetchMock).toHaveBeenNthCalledWith(3, "http://localhost:8080/api/v1/trips/trip-1", expect.anything());
  });
});
