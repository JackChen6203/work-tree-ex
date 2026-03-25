import { describe, expect, it } from "vitest";
import { useTripPermission } from "./use-trip-permission";

describe("useTripPermission", () => {
  it("grants full control to owners", () => {
    expect(useTripPermission("owner")).toEqual({
      canEdit: true,
      canComment: true,
      canViewOnly: false,
      canManageMembers: true,
      role: "owner"
    });
  });

  it("grants editing without member management to editors", () => {
    expect(useTripPermission("editor")).toEqual({
      canEdit: true,
      canComment: true,
      canViewOnly: false,
      canManageMembers: false,
      role: "editor"
    });
  });

  it("falls back to view-only permissions", () => {
    expect(useTripPermission()).toEqual({
      canEdit: false,
      canComment: false,
      canViewOnly: true,
      canManageMembers: false,
      role: "viewer"
    });
  });
});
