import { apiRequest } from "./api";

export interface TripShareLink {
  id: string;
  tripId: string;
  token?: string;
  accessScope: string;
  expiresAt?: string;
  revokedAt?: string;
  createdAt: string;
}

export interface TripInvitation {
  id: string;
  tripId: string;
  invitedByUserId: string;
  inviteeEmail: string;
  role: "editor" | "commenter" | "viewer";
  status: "pending" | "accepted" | "revoked" | "expired";
  token?: string;
  expiresAt: string;
  acceptedAt?: string;
  createdAt: string;
}

export function listTripShareLinks(tripId: string) {
  return apiRequest<TripShareLink[] | null>(`/api/v1/trips/${tripId}/share-links`).then((data) => data ?? []);
}

export function createTripShareLink(tripId: string) {
  return apiRequest<TripShareLink>(`/api/v1/trips/${tripId}/share-links`, {
    method: "POST",
    headers: {
      "Idempotency-Key": crypto.randomUUID()
    }
  });
}

export function revokeTripShareLink(tripId: string, linkId: string) {
  return apiRequest<TripShareLink>(`/api/v1/trips/${tripId}/share-links/${linkId}/revoke`, {
    method: "POST"
  });
}

export function listTripInvitations(tripId: string) {
  return apiRequest<TripInvitation[] | null>(`/api/v1/trips/${tripId}/invitations`).then((data) => data ?? []);
}

export function createTripInvitation(
  tripId: string,
  input: { inviteeEmail: string; role: "editor" | "commenter" | "viewer" }
) {
  return apiRequest<TripInvitation>(`/api/v1/trips/${tripId}/invitations`, {
    method: "POST",
    headers: {
      "Idempotency-Key": crypto.randomUUID()
    },
    body: JSON.stringify(input)
  });
}

export function revokeTripInvitation(tripId: string, invitationId: string) {
  return apiRequest<TripInvitation>(`/api/v1/trips/${tripId}/invitations/${invitationId}/revoke`, {
    method: "POST"
  });
}
