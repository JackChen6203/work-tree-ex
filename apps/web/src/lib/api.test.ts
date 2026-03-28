import { beforeEach, describe, expect, it, vi } from "vitest";
import { apiRequest } from "./api";

describe("api request", () => {
  beforeEach(() => {
    vi.restoreAllMocks();
  });

  it("refreshes token and retries once on 401", async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce({
        ok: false,
        status: 401,
        json: async () => ({ error: { message: "unauthorized" } })
      })
      .mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: async () => ({ data: { accessToken: "token", expiresAt: Date.now() + 60_000 } })
      })
      .mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: async () => ({ data: { ok: true } })
      });
    vi.stubGlobal("fetch", fetchMock);

    const result = await apiRequest<{ ok: boolean }>("/api/v1/test-endpoint");

    expect(result.ok).toBe(true);
    expect(fetchMock).toHaveBeenNthCalledWith(1, "http://localhost:8080/api/v1/test-endpoint", expect.anything());
    expect(fetchMock).toHaveBeenNthCalledWith(
      2,
      "http://localhost:8080/api/v1/auth/refresh",
      expect.objectContaining({ method: "POST" })
    );
    expect(fetchMock).toHaveBeenNthCalledWith(3, "http://localhost:8080/api/v1/test-endpoint", expect.anything());
  });
});
