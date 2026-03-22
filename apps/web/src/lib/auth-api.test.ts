import { beforeEach, describe, expect, it, vi } from "vitest";
import { getSession, logout, oauthStartUrl, requestMagicLink, verifyMagicLink } from "./auth-api";

describe("auth api", () => {
  beforeEach(() => {
    vi.restoreAllMocks();
  });

  it("posts magic link request and verification payloads", async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce({ ok: true, json: async () => ({ data: { sent: true, expiresIn: 600, previewCode: "123456" } }) })
      .mockResolvedValueOnce({ ok: true, json: async () => ({ data: { user: { id: "u1", name: "Demo", email: "demo@example.com", avatar: "DM" }, roles: ["owner"] } }) })
      .mockResolvedValueOnce({ ok: true, json: async () => ({ data: { user: { id: "u1", name: "Demo", email: "demo@example.com", avatar: "DM" }, roles: ["owner"] } }) })
      .mockResolvedValueOnce({ ok: true });
    vi.stubGlobal("fetch", fetchMock);

    await requestMagicLink("demo@example.com");
    await verifyMagicLink("demo@example.com", "123456");
    await getSession();
    await logout();

    expect(fetchMock).toHaveBeenNthCalledWith(
      1,
      "http://localhost:8080/api/v1/auth/request-magic-link",
      expect.objectContaining({ method: "POST", body: JSON.stringify({ email: "demo@example.com" }) })
    );
    expect(fetchMock).toHaveBeenNthCalledWith(
      2,
      "http://localhost:8080/api/v1/auth/verify-magic-link",
      expect.objectContaining({ method: "POST", body: JSON.stringify({ email: "demo@example.com", code: "123456" }) })
    );
    expect(fetchMock).toHaveBeenNthCalledWith(3, "http://localhost:8080/api/v1/auth/session", expect.anything());
    expect(fetchMock).toHaveBeenNthCalledWith(
      4,
      "http://localhost:8080/api/v1/auth/logout",
      expect.objectContaining({ method: "POST" })
    );
  });

  it("builds oauth start url", () => {
    expect(oauthStartUrl("google")).toBe("http://localhost:8080/api/v1/auth/oauth/google/start");
  });
});
