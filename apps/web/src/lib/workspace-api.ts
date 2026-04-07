import type { TripSummary } from "../types/domain";
import { apiRequest } from "./api";

interface TripApiModel {
  id: string;
  name: string;
  destination?: string;
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

export interface WorkspaceActivity {
  id: string;
  type: string;
  title: string;
  body: string;
  link: string;
  readAt?: string;
  createdAt?: string;
}

interface WorkspaceSummaryApi {
  upcomingTrip?: TripApiModel;
  recentActivities?: WorkspaceActivity[];
  quickAccessTrips?: TripApiModel[];
}

export interface WorkspaceSummary {
  upcomingTrip: TripSummary | null;
  recentActivities: WorkspaceActivity[];
  quickAccessTrips: TripSummary[];
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

function mapTrip(apiTrip: TripApiModel, index = 0): TripSummary {
  const destination = apiTrip.destinationText || apiTrip.destination || "Destination TBD";
  return {
    id: apiTrip.id,
    name: apiTrip.name,
    destination,
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

export async function getWorkspaceSummary() {
  const data = await apiRequest<WorkspaceSummaryApi>("/api/v1/workspace/summary");
  return {
    upcomingTrip: data?.upcomingTrip ? mapTrip(data.upcomingTrip) : null,
    recentActivities: data?.recentActivities ?? [],
    quickAccessTrips: (data?.quickAccessTrips ?? []).map((trip, index) => mapTrip(trip, index))
  } satisfies WorkspaceSummary;
}

