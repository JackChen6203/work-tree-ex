import { apiRequest } from "./api";

export interface PlaceSearchItem {
  providerPlaceId: string;
  name: string;
  address: string;
  lat: number;
  lng: number;
  categories: string[];
}

export interface RouteEstimate {
  mode: string;
  distanceMeters: number;
  durationSeconds: number;
  estimatedCostAmount?: number;
  estimatedCostCurrency?: string;
  provider: string;
  snapshotToken: string;
}

export function searchPlaces(query: string) {
  const q = encodeURIComponent(query);
  return apiRequest<PlaceSearchItem[]>(`/api/v1/maps/search?q=${q}`);
}

export function estimateRoute(input: {
  origin: { lat: number; lng: number };
  destination: { lat: number; lng: number };
  mode: "walk" | "transit" | "drive" | "taxi";
}) {
  return apiRequest<RouteEstimate>("/api/v1/maps/routes", {
    method: "POST",
    body: JSON.stringify(input)
  });
}
