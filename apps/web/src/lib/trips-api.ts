import type { TripSummary } from "../types/domain";
import { apiRequest } from "./api";

interface TripApiModel {
  id: string;
  name: string;
  destinationText?: string;
  startDate: string;
  endDate: string;
  timezone: string;
  currency: string;
  travelersCount: number;
  status: "draft" | "active" | "archived";
  version: number;
  createdAt: string;
  updatedAt: string;
}

export interface CreateTripInput {
  name: string;
  destinationText: string;
  startDate: string;
  endDate: string;
  timezone: string;
  currency: string;
  travelersCount: number;
}

export interface PatchTripInput extends Partial<CreateTripInput> {
  status?: "draft" | "active" | "archived";
}

const gradients = [
  "from-[#24403a] via-[#376052] to-[#b4cdc2]",
  "from-[#36243a] via-[#6e4d63] to-[#f0d6ce]",
  "from-[#1f3657] via-[#305f8f] to-[#d7e8f6]",
  "from-[#4a3025] via-[#8b5d46] to-[#f2dccb]"
];

function toDisplayRange(startDate: string, endDate: string) {
  return `${startDate.replace(/-/g, "/")} - ${endDate.replace(/-/g, "/")}`;
}

export function mapTrip(apiTrip: TripApiModel, index = 0): TripSummary {
  return {
    id: apiTrip.id,
    name: apiTrip.name,
    destination: apiTrip.destinationText || "Destination TBD",
    dateRange: toDisplayRange(apiTrip.startDate, apiTrip.endDate),
    timezone: apiTrip.timezone,
    coverGradient: gradients[index % gradients.length],
    status: apiTrip.status,
    role: "owner",
    pendingInvites: 0,
    members: Math.max(apiTrip.travelersCount, 1),
    currency: apiTrip.currency,
    travelersCount: apiTrip.travelersCount,
    version: apiTrip.version,
    startDate: apiTrip.startDate,
    endDate: apiTrip.endDate
  };
}

export async function listTrips() {
  const data = await apiRequest<TripApiModel[]>("/api/v1/trips");
  return data.map((trip, index) => mapTrip(trip, index));
}

export async function getTrip(tripId: string) {
  const data = await apiRequest<TripApiModel>(`/api/v1/trips/${tripId}`);
  return mapTrip(data);
}

export async function createTrip(input: CreateTripInput) {
  const data = await apiRequest<TripApiModel>("/api/v1/trips", {
    method: "POST",
    headers: {
      "Idempotency-Key": crypto.randomUUID()
    },
    body: JSON.stringify(input)
  });
  return mapTrip(data);
}

export async function patchTrip(tripId: string, version: number, input: PatchTripInput) {
  const data = await apiRequest<TripApiModel>(`/api/v1/trips/${tripId}`, {
    method: "PATCH",
    headers: {
      "If-Match-Version": String(version)
    },
    body: JSON.stringify(input)
  });
  return mapTrip(data);
}
