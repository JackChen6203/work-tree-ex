import { beforeEach, describe, expect, it, vi } from "vitest";
import {
  createTripInvitation,
  createTripShareLink,
  revokeTripInvitation,
  revokeTripShareLink
} from "./trips-collaboration-api";

describe("trip collaboration api", () => {
  beforeEach(() => {
    vi.restoreAllMocks();
  });

  it("sends idempotency key for create invitation/share-link", async () => {
    vi.stubGlobal("crypto", { randomUUID: () => "33333333-3333-3333-3333-333333333333" } as unknown as Crypto);
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({ data: { id: "id-1" } })
    });
    vi.stubGlobal("fetch", fetchMock);

    await createTripInvitation("trip-1", { inviteeEmail: "qa@example.com", role: "viewer" });
    await createTripShareLink("trip-1");

    expect(fetchMock).toHaveBeenNthCalledWith(
      1,
      "http://localhost:8080/api/v1/trips/trip-1/invitations",
      expect.objectContaining({
        method: "POST",
        headers: expect.objectContaining({ "Idempotency-Key": "33333333-3333-3333-3333-333333333333" })
      })
    );
    expect(fetchMock).toHaveBeenNthCalledWith(
      2,
      "http://localhost:8080/api/v1/trips/trip-1/share-links",
      expect.objectContaining({
        method: "POST",
        headers: expect.objectContaining({ "Idempotency-Key": "33333333-3333-3333-3333-333333333333" })
      })
    );
  });

  it("calls revoke endpoints", async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({ data: { id: "id-1" } })
    });
    vi.stubGlobal("fetch", fetchMock);

    await revokeTripInvitation("trip-1", "inv-1");
    await revokeTripShareLink("trip-1", "sl-1");

    expect(fetchMock).toHaveBeenNthCalledWith(
      1,
      "http://localhost:8080/api/v1/trips/trip-1/invitations/inv-1/revoke",
      expect.objectContaining({ method: "POST" })
    );
    expect(fetchMock).toHaveBeenNthCalledWith(
      2,
      "http://localhost:8080/api/v1/trips/trip-1/share-links/sl-1/revoke",
      expect.objectContaining({ method: "POST" })
    );
  });
});

