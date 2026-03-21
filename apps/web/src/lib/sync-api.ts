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

export function getSyncBootstrap(tripId: string, sinceVersion = 0) {
  const params = new URLSearchParams();
  params.set("sinceVersion", String(sinceVersion));
  if (tripId) {
    params.set("tripId", tripId);
  }

  return apiRequest<SyncBootstrapResponse>(`/api/v1/sync/bootstrap?${params.toString()}`);
}
