import { beforeEach, describe, expect, it, vi } from "vitest";
import { apiRequest } from "./api";

function createMemoryStorage() {
  const values = new Map<string, string>();
  return {
    clear: () => values.clear(),
    getItem: (key: string) => values.get(key) ?? null,
    removeItem: (key: string) => values.delete(key),
    setItem: (key: string, value: string) => values.set(key, value)
  } as unknown as Storage;
}

describe("api request", () => {
  beforeEach(() => {
    vi.restoreAllMocks();
    vi.stubGlobal("document", { cookie: "" } as unknown as Document);
    vi.stubGlobal("sessionStorage", createMemoryStorage());
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

  it("adds csrf token to unsafe session requests", async () => {
    vi.stubGlobal("crypto", { randomUUID: () => "csrf-token-1" } as unknown as Crypto);
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      json: async () => ({ data: { ok: true } })
    });
    vi.stubGlobal("fetch", fetchMock);

    await apiRequest<{ ok: boolean }>("/api/v1/trips", {
      method: "POST",
      body: JSON.stringify({ name: "Kyoto" })
    });

    expect(fetchMock).toHaveBeenCalledWith(
      "http://localhost:8080/api/v1/trips",
      expect.objectContaining({
        method: "POST",
        headers: expect.objectContaining({ "X-CSRF-Token": "csrf-token-1" })
      })
    );
    expect(document.cookie).toContain("tt_csrf=csrf-token-1");
  });
});
