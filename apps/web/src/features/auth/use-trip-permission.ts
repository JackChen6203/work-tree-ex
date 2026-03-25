import type { UserRole } from "../../types/domain";

export interface TripPermission {
  canEdit: boolean;
  canComment: boolean;
  canViewOnly: boolean;
  canManageMembers: boolean;
  role: UserRole;
}

export function useTripPermission(role?: UserRole | null): TripPermission {
  const resolvedRole = role ?? "viewer";

  return {
    canEdit: resolvedRole === "owner" || resolvedRole === "editor",
    canComment: resolvedRole === "owner" || resolvedRole === "editor" || resolvedRole === "commenter",
    canViewOnly: resolvedRole === "viewer",
    canManageMembers: resolvedRole === "owner",
    role: resolvedRole
  };
}
