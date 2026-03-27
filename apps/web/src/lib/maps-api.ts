import { apiRequest } from "./api";

export interface PlaceSearchItem {
  providerPlaceId: string;
  name: string;
  address: string;
  lat: number;
  lng: number;
  categories: string[];
  openingHours?: string;
}

export interface RouteEstimate {
  mode: string;
  distanceMeters: number;
  durationSeconds: number;
  estimatedCostAmount?: number;
  estimatedCostCurrency?: string;
  provider?: string;
  snapshotToken?: string;
}

export interface PlaceDetail extends PlaceSearchItem {
  warnings?: string[];
}

export function searchPlaces(query: string, options?: { lat?: number; lng?: number; limit?: number }) {
  const params = new URLSearchParams();
  params.set("q", query);
  if (typeof options?.lat === "number") {
    params.set("lat", String(options.lat));
  }
  if (typeof options?.lng === "number") {
    params.set("lng", String(options.lng));
  }
  if (typeof options?.limit === "number") {
    params.set("limit", String(options.limit));
  }
  return apiRequest<PlaceSearchItem[]>(`/api/v1/maps/search?${params.toString()}`);
}

export function getPlaceDetail(placeId: string) {
  return apiRequest<PlaceDetail>(`/api/v1/maps/places/${placeId}`);
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
