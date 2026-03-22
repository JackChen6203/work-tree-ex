import { beforeEach, describe, expect, it, vi } from "vitest";
import { estimateRoute, searchPlaces } from "./maps-api";

describe("maps api", () => {
  beforeEach(() => {
    vi.restoreAllMocks();
  });

  it("builds search query with optional coordinates and limit", async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce({ ok: true, json: async () => ({ data: [] }) })
      .mockResolvedValueOnce({ ok: true, json: async () => ({ data: [] }) });
    vi.stubGlobal("fetch", fetchMock);

    await searchPlaces("kyoto");
    await searchPlaces("kyoto", { lat: 35.01, lng: 135.76, limit: 1 });

    expect(fetchMock).toHaveBeenNthCalledWith(1, "http://localhost:8080/api/v1/maps/search?q=kyoto", expect.anything());
    expect(fetchMock).toHaveBeenNthCalledWith(
      2,
      "http://localhost:8080/api/v1/maps/search?q=kyoto&lat=35.01&lng=135.76&limit=1",
      expect.anything()
    );
  });

  it("posts route estimate payload", async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({
        data: {
          mode: "transit",
          distanceMeters: 1000,
          durationSeconds: 300,
          provider: "mock-map-adapter",
          snapshotToken: "rt_1"
        }
      })
    });
    vi.stubGlobal("fetch", fetchMock);

    await estimateRoute({ origin: { lat: 35, lng: 135 }, destination: { lat: 35.1, lng: 135.2 }, mode: "transit" });

    expect(fetchMock).toHaveBeenCalledWith(
      "http://localhost:8080/api/v1/maps/routes",
      expect.objectContaining({ method: "POST" })
    );
  });
});
