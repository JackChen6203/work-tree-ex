import { beforeEach, describe, expect, it, vi } from "vitest";
import {
  deleteMyAccount,
  getMyNotificationPreferences,
  listMyLlmProviders,
  patchMyProfile,
  putMyNotificationPreferences,
  testMyLlmProviderConnection
} from "./users-api";

describe("users api", () => {
  beforeEach(() => {
    vi.restoreAllMocks();
  });

  it("reads notification preferences through envelope", async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({
        data: {
          pushEnabled: true,
          emailEnabled: false,
          digestFrequency: "daily",
          quietHoursStart: "22:00",
          quietHoursEnd: "07:00",
          tripUpdates: true,
          budgetAlerts: true,
          aiPlanReadyAlerts: true,
          version: 2
        }
      })
    });
    vi.stubGlobal("fetch", fetchMock);

    const result = await getMyNotificationPreferences();

    expect(result.digestFrequency).toBe("daily");
    expect(fetchMock).toHaveBeenCalledWith("http://localhost:8080/api/v1/users/me/notification-preferences", expect.anything());
  });

  it("sends put payload for notification preferences", async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({
        data: {
          pushEnabled: true,
          emailEnabled: true,
          digestFrequency: "weekly",
          quietHoursStart: "23:00",
          quietHoursEnd: "07:30",
          tripUpdates: true,
          budgetAlerts: false,
          aiPlanReadyAlerts: true,
          version: 3
        }
      })
    });
    vi.stubGlobal("fetch", fetchMock);

    await putMyNotificationPreferences({
      pushEnabled: true,
      emailEnabled: true,
      digestFrequency: "weekly",
      quietHoursStart: "23:00",
      quietHoursEnd: "07:30",
      tripUpdates: true,
      budgetAlerts: false,
      aiPlanReadyAlerts: true
    });

    expect(fetchMock).toHaveBeenCalledWith(
      "http://localhost:8080/api/v1/users/me/notification-preferences",
      expect.objectContaining({
        method: "PUT",
        body: JSON.stringify({
          pushEnabled: true,
          emailEnabled: true,
          digestFrequency: "weekly",
          quietHoursStart: "23:00",
          quietHoursEnd: "07:30",
          tripUpdates: true,
          budgetAlerts: false,
          aiPlanReadyAlerts: true
        })
      })
    );
  });

  it("keeps profile patch path working", async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({
        data: {
          id: "u_01",
          email: "ariel@example.com",
          displayName: "Ariel",
          locale: "en",
          timezone: "Asia/Tokyo",
          currency: "JPY"
        }
      })
    });
    vi.stubGlobal("fetch", fetchMock);

    await patchMyProfile({ displayName: "Ariel" });

    expect(fetchMock).toHaveBeenCalledWith(
      "http://localhost:8080/api/v1/users/me",
      expect.objectContaining({ method: "PATCH", body: JSON.stringify({ displayName: "Ariel" }) })
    );
  });

  it("lists llm providers with optional provider filter", async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce({ ok: true, json: async () => ({ data: [] }) })
      .mockResolvedValueOnce({ ok: true, json: async () => ({ data: [] }) });
    vi.stubGlobal("fetch", fetchMock);

    await listMyLlmProviders();
    await listMyLlmProviders("openai");

    expect(fetchMock).toHaveBeenNthCalledWith(1, "http://localhost:8080/api/v1/users/me/llm-providers", expect.anything());
    expect(fetchMock).toHaveBeenNthCalledWith(2, "http://localhost:8080/api/v1/users/me/llm-providers?provider=openai", expect.anything());
  });

  it("tests llm provider connection", async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({
        data: {
          provider: "openai",
          model: "gpt-4.1-mini",
          status: "ok",
          latencyMs: 120,
          message: "provider connection verified",
          checkedAt: "2026-04-07T07:10:00Z"
        }
      })
    });
    vi.stubGlobal("fetch", fetchMock);

    const result = await testMyLlmProviderConnection({
      provider: "openai",
      label: "Personal",
      model: "gpt-4.1-mini",
      encryptedApiKeyEnvelope: "enc_test_1234567890"
    });

    expect(result.status).toBe("ok");
    expect(fetchMock).toHaveBeenCalledWith(
      "http://localhost:8080/api/v1/users/me/llm-providers/test",
      expect.objectContaining({
        method: "POST",
        body: JSON.stringify({
          provider: "openai",
          model: "gpt-4.1-mini",
          encryptedApiKeyEnvelope: "enc_test_1234567890"
        })
      })
    );
  });

  it("deletes current account", async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({ data: null })
    });
    vi.stubGlobal("fetch", fetchMock);

    await deleteMyAccount();

    expect(fetchMock).toHaveBeenCalledWith(
      "http://localhost:8080/api/v1/users/me",
      expect.objectContaining({ method: "DELETE" })
    );
  });
});
