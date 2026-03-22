import { beforeEach, describe, expect, it, vi } from "vitest";
import { adoptAiPlan, createAiPlan, getAiPlan, listAiPlans } from "./ai-planner-api";

describe("ai planner api", () => {
  beforeEach(() => {
    vi.restoreAllMocks();
  });

  it("calls list and get endpoints for plan retrieval", async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce({
        ok: true,
        json: async () => ({ data: [] })
      })
      .mockResolvedValueOnce({
        ok: true,
        json: async () => ({
          data: {
            id: "plan-1",
            tripId: "trip-1",
            title: "Draft",
            status: "valid",
            summary: "ok",
            warnings: [],
            totalEstimated: 100,
            budget: 120,
            currency: "JPY",
            createdAt: "2026-03-22T00:00:00Z"
          }
        })
      });
    vi.stubGlobal("fetch", fetchMock);

    await listAiPlans("trip-1");
    await getAiPlan("trip-1", "plan-1");

    expect(fetchMock).toHaveBeenNthCalledWith(1, "http://localhost:8080/api/v1/trips/trip-1/ai/plans", expect.anything());
    expect(fetchMock).toHaveBeenNthCalledWith(2, "http://localhost:8080/api/v1/trips/trip-1/ai/plans/plan-1", expect.anything());
  });

  it("sends idempotency headers for create and adopt", async () => {
    vi.stubGlobal("crypto", { randomUUID: () => "33333333-3333-3333-3333-333333333333" } as unknown as Crypto);
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce({
        ok: true,
        json: async () => ({ data: { jobId: "plan-1", status: "succeeded", acceptedAt: "2026-03-22T00:00:00Z" } })
      })
      .mockResolvedValueOnce({
        ok: true,
        json: async () => ({ data: { tripId: "trip-1", planId: "plan-1", adopted: true, status: "valid", warnings: [] } })
      });
    vi.stubGlobal("fetch", fetchMock);

    await createAiPlan("trip-1", {
      providerConfigId: "cfg_1",
      title: "Draft",
      constraints: {
        totalBudget: 20000,
        currency: "JPY",
        pace: "balanced",
        transportPreference: "transit",
        mustVisit: [],
        avoid: []
      }
    });

    await adoptAiPlan("trip-1", "plan-1");

    expect(fetchMock).toHaveBeenNthCalledWith(
      1,
      "http://localhost:8080/api/v1/trips/trip-1/ai/plans",
      expect.objectContaining({
        method: "POST",
        headers: expect.objectContaining({ "Idempotency-Key": "33333333-3333-3333-3333-333333333333" })
      })
    );
    expect(fetchMock).toHaveBeenNthCalledWith(
      2,
      "http://localhost:8080/api/v1/trips/trip-1/ai/plans/plan-1/adopt",
      expect.objectContaining({
        method: "POST",
        headers: expect.objectContaining({ "Idempotency-Key": "33333333-3333-3333-3333-333333333333" })
      })
    );
  });
});
