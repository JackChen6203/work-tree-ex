import { apiRequest } from "./api";

interface SyncEntityVersion {
  id?: string;
  tripId?: string;
  dayId?: string;
  version: number;
}

export interface SyncBootstrapResponse {
  serverTime: string;
  sinceVersion: number;
  tripId: string;
  fullResyncRequired: boolean;
  changedTrips: SyncEntityVersion[];
  changedDays: SyncEntityVersion[];
  changedNotifications: SyncEntityVersion[];
}

export interface SyncFlushMutationInput {
  id: string;
  entityType: string;
  entityId: string;
  baseVersion: number;
}

export interface SyncFlushResponse {
  tripId: string;
  acceptedCount: number;
  conflictCount: number;
  conflicts: Array<{ id: string; reason: string; entityId: string; expectedVersion?: number }>;
  nextVersion: number;
  serverTime: string;
}

export function getSyncBootstrap(tripId: string, sinceVersion = 0) {
  const params = new URLSearchParams();
  params.set("sinceVersion", String(sinceVersion));
  if (tripId) {
    params.set("tripId", tripId);
  }

  return apiRequest<SyncBootstrapResponse>(`/api/v1/sync/bootstrap?${params.toString()}`);
}

export function flushSyncMutations(tripId: string, mutations: SyncFlushMutationInput[]) {
  return apiRequest<SyncFlushResponse>("/api/v1/sync/mutations/flush", {
    method: "POST",
    body: JSON.stringify({
      tripId,
      mutations
    })
  });
}
