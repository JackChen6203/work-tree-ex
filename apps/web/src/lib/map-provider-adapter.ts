// ---------- MapProviderAdapter abstraction (FE-07) ----------

export interface LatLng {
  lat: number;
  lng: number;
}

export interface PlaceSearchResult {
  providerPlaceId: string;
  name: string;
  address: string;
  lat: number;
  lng: number;
  categories: string[];
}

export type TransportMode = "walk" | "transit" | "drive" | "taxi";

export interface RouteEstimate {
  mode: TransportMode;
  distanceMeters: number;
  durationSeconds: number;
  estimatedCostAmount?: number;
  estimatedCostCurrency?: string;
  provider: string;
  snapshotToken: string;
}

export interface MapRenderOptions {
  center: LatLng;
  zoom: number;
  markers?: Array<LatLng & { label?: string }>;
  enableClustering?: boolean;
}

export interface MapInstance {
  setCenter: (center: LatLng) => void;
  setZoom: (zoom: number) => void;
  addMarker: (position: LatLng, label?: string) => string;
  removeMarker: (markerId: string) => void;
  clearMarkers: () => void;
  fitBounds: (points: LatLng[]) => void;
  destroy: () => void;
}

/**
 * Provider-agnostic map adapter interface.
 * Swap implementations without changing consumer code.
 */
export interface MapProviderAdapter {
  searchPlaces(query: string, lat?: number, lng?: number): Promise<PlaceSearchResult[]>;
  estimateRoute(origin: LatLng, destination: LatLng, mode: TransportMode): Promise<RouteEstimate>;
  renderMap(containerId: string, options: MapRenderOptions): MapInstance;
}

// ---------- Mock adapter (uses backend API) ----------

import { apiRequest } from "./api";

export class BackendMapAdapter implements MapProviderAdapter {
  async searchPlaces(query: string): Promise<PlaceSearchResult[]> {
    return apiRequest<PlaceSearchResult[]>(`/api/v1/maps/search?q=${encodeURIComponent(query)}`);
  }

  async estimateRoute(origin: LatLng, destination: LatLng, mode: TransportMode): Promise<RouteEstimate> {
    return apiRequest<RouteEstimate>("/api/v1/maps/routes", {
      method: "POST",
      body: JSON.stringify({ origin, destination, mode })
    });
  }

  renderMap(containerId: string, options: MapRenderOptions): MapInstance {
    const markers = new Map<string, { position: LatLng; label?: string }>();

    if (options.markers) {
      for (const m of options.markers) {
        markers.set(crypto.randomUUID(), { position: m, label: m.label });
      }
    }

    return {
      setCenter: () => { /* no-op in mock */ },
      setZoom: () => { /* no-op in mock */ },
      addMarker: (position, label) => {
        const id = crypto.randomUUID();
        markers.set(id, { position, label });
        return id;
      },
      removeMarker: (id) => { markers.delete(id); },
      clearMarkers: () => { markers.clear(); },
      fitBounds: () => { /* no-op in mock */ },
      destroy: () => { markers.clear(); }
    };
  }
}

// ---------- Coordinate validation (edge case: bad coords) ----------

export function isValidCoordinate(lat: number, lng: number): boolean {
  return (
    Number.isFinite(lat) &&
    Number.isFinite(lng) &&
    lat >= -90 && lat <= 90 &&
    lng >= -180 && lng <= 180 &&
    !(lat === 0 && lng === 0)
  );
}

// ---------- Simple marker clustering ----------

export interface Cluster {
  center: LatLng;
  points: Array<LatLng & { label?: string }>;
  count: number;
}

export function clusterMarkers(
  points: Array<LatLng & { label?: string }>,
  gridSize = 0.01
): Cluster[] {
  const grid = new Map<string, Cluster>();

  for (const p of points) {
    if (!isValidCoordinate(p.lat, p.lng)) continue;

    const gridKey = `${Math.floor(p.lat / gridSize)}_${Math.floor(p.lng / gridSize)}`;
    const existing = grid.get(gridKey);

    if (existing) {
      existing.points.push(p);
      existing.count++;
      existing.center = {
        lat: existing.points.reduce((s, pt) => s + pt.lat, 0) / existing.count,
        lng: existing.points.reduce((s, pt) => s + pt.lng, 0) / existing.count
      };
    } else {
      grid.set(gridKey, {
        center: { lat: p.lat, lng: p.lng },
        points: [p],
        count: 1
      });
    }
  }

  return Array.from(grid.values());
}
