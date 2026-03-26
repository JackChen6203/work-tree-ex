import { apiRequest } from "./api";
import type { Feature, FeatureCollection, LineString, Point } from "geojson";

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
  onMarkerClick?: (markerIndex: number) => void;
}

export interface MapInstance {
  setCenter: (center: LatLng) => void;
  setZoom: (zoom: number) => void;
  setMarkers: (points: Array<LatLng & { label?: string }>) => void;
  addMarker: (position: LatLng, label?: string) => string;
  removeMarker: (markerId: string) => void;
  clearMarkers: () => void;
  fitBounds: (points: LatLng[]) => void;
  setRoutePath: (points: LatLng[]) => void;
  destroy: () => void;
}

export interface MapProviderAdapter {
  searchPlaces(query: string, lat?: number, lng?: number): Promise<PlaceSearchResult[]>;
  estimateRoute(origin: LatLng, destination: LatLng, mode: TransportMode): Promise<RouteEstimate>;
  renderMap(containerId: string, options: MapRenderOptions): MapInstance;
}

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

  renderMap(_containerId: string, options: MapRenderOptions): MapInstance {
    const markers = new Map<string, { position: LatLng; label?: string }>();
    let routePath: LatLng[] = [];

    if (options.markers) {
      for (const marker of options.markers) {
        markers.set(crypto.randomUUID(), { position: marker, label: marker.label });
      }
    }

    return {
      setCenter: () => { /* no-op in fallback mode */ },
      setZoom: () => { /* no-op in fallback mode */ },
      setMarkers: (points) => {
        markers.clear();
        for (const marker of points) {
          markers.set(crypto.randomUUID(), { position: marker, label: marker.label });
        }
      },
      addMarker: (position, label) => {
        const id = crypto.randomUUID();
        markers.set(id, { position, label });
        return id;
      },
      removeMarker: (id) => {
        markers.delete(id);
      },
      clearMarkers: () => {
        markers.clear();
      },
      fitBounds: () => { /* no-op in fallback mode */ },
      setRoutePath: (points) => {
        routePath = points;
      },
      destroy: () => {
        markers.clear();
        routePath = [];
      }
    };
  }
}

type MapboxModule = typeof import("mapbox-gl");
function buildPointFeatureCollection(
  points: Array<LatLng & { label?: string }>
): FeatureCollection<Point, { index: number; label: string }> {
  return {
    type: "FeatureCollection",
    features: points.map((point, index): Feature<Point, { index: number; label: string }> => ({
      type: "Feature",
      geometry: {
        type: "Point",
        coordinates: [point.lng, point.lat]
      },
      properties: {
        index,
        label: point.label ?? String(index + 1)
      }
    }))
  };
}

function buildLineFeatureCollection(points: LatLng[]): FeatureCollection<LineString> {
  return {
    type: "FeatureCollection",
    features: [
      {
        type: "Feature",
        geometry: {
          type: "LineString",
          coordinates: points.map((point) => [point.lng, point.lat])
        },
        properties: {}
      }
    ]
  };
}

export class MapboxAdapter implements MapProviderAdapter {
  private constructor(private readonly mapboxModule: MapboxModule) {}

  static async create(accessToken: string): Promise<MapboxAdapter> {
    if (!accessToken) {
      throw new Error("Missing mapbox token");
    }

    const module = await import("mapbox-gl");
    module.default.accessToken = accessToken;
    return new MapboxAdapter(module);
  }

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
    const mapboxgl = this.mapboxModule.default;
    const map = new mapboxgl.Map({
      container: containerId,
      style: "mapbox://styles/mapbox/streets-v12",
      center: [options.center.lng, options.center.lat],
      zoom: options.zoom
    });

    const markerRefs = new Map<string, import("mapbox-gl").Marker>();
    const sourceId = "poi-source";
    const unclusteredLayerId = "poi-unclustered";
    const clusterLayerId = "poi-clusters";
    const clusterCountLayerId = "poi-cluster-count";
    const routeSourceId = "route-source";
    const routeLayerId = "route-layer";

    const seedMarkers = options.markers ?? [];
    let markerSnapshot = [...seedMarkers];

    const setPointSourceData = (points: Array<LatLng & { label?: string }>) => {
      markerSnapshot = [...points];
      const source = map.getSource(sourceId) as import("mapbox-gl").GeoJSONSource | undefined;
      if (source) {
        source.setData(buildPointFeatureCollection(points));
      }
    };

    map.on("load", () => {
      const data = buildPointFeatureCollection(markerSnapshot);
      map.addSource(sourceId, {
        type: "geojson",
        data,
        cluster: options.enableClustering ?? true,
        clusterMaxZoom: 14,
        clusterRadius: 50
      });

      map.addLayer({
        id: clusterLayerId,
        type: "circle",
        source: sourceId,
        filter: ["has", "point_count"],
        paint: {
          "circle-color": "#2d5a4a",
          "circle-radius": [
            "step",
            ["get", "point_count"],
            18,
            10,
            24,
            30,
            32
          ],
          "circle-opacity": 0.85
        }
      });

      map.addLayer({
        id: clusterCountLayerId,
        type: "symbol",
        source: sourceId,
        filter: ["has", "point_count"],
        layout: {
          "text-field": ["get", "point_count_abbreviated"],
          "text-size": 12
        },
        paint: {
          "text-color": "#ffffff"
        }
      });

      map.addLayer({
        id: unclusteredLayerId,
        type: "circle",
        source: sourceId,
        filter: ["!", ["has", "point_count"]],
        paint: {
          "circle-color": "#da6a4e",
          "circle-radius": 8,
          "circle-stroke-width": 2,
          "circle-stroke-color": "#ffffff"
        }
      });

      map.on("click", unclusteredLayerId, (event) => {
        const feature = event.features?.[0];
        const markerIndex = Number(feature?.properties?.index);
        if (Number.isFinite(markerIndex)) {
          options.onMarkerClick?.(markerIndex);
        }
      });

      map.on("click", clusterLayerId, (event) => {
        const features = map.queryRenderedFeatures(event.point, { layers: [clusterLayerId] });
        const clusterId = features[0]?.properties?.cluster_id;
        if (clusterId === undefined) {
          return;
        }

        const source = map.getSource(sourceId) as import("mapbox-gl").GeoJSONSource;
        source.getClusterExpansionZoom(clusterId, (error, zoom) => {
          if (error) {
            return;
          }

          map.easeTo({
            center: (features[0]?.geometry as { coordinates?: number[] })?.coordinates as [number, number] | undefined,
            zoom: typeof zoom === "number" ? zoom : map.getZoom()
          });
        });
      });
    });

    const setRoutePath = (points: LatLng[]) => {
      if (!map.getSource(routeSourceId)) {
        map.addSource(routeSourceId, {
          type: "geojson",
          data: buildLineFeatureCollection(points)
        });

        map.addLayer({
          id: routeLayerId,
          type: "line",
          source: routeSourceId,
          paint: {
            "line-color": "#da6a4e",
            "line-width": 4
          }
        });
        return;
      }

      const source = map.getSource(routeSourceId) as import("mapbox-gl").GeoJSONSource;
      source.setData(buildLineFeatureCollection(points));
    };

    return {
      setCenter: (center) => {
        map.easeTo({ center: [center.lng, center.lat], duration: 350 });
      },
      setZoom: (zoom) => {
        map.setZoom(zoom);
      },
      setMarkers: (points) => {
        if (map.isStyleLoaded()) {
          setPointSourceData(points);
          return;
        }

        map.once("load", () => {
          setPointSourceData(points);
        });
      },
      addMarker: (position, label) => {
        const marker = new mapboxgl.Marker({ color: "#da6a4e" })
          .setLngLat([position.lng, position.lat])
          .setPopup(new mapboxgl.Popup({ offset: 10 }).setText(label ?? "POI"))
          .addTo(map);

        const markerId = crypto.randomUUID();
        markerRefs.set(markerId, marker);
        return markerId;
      },
      removeMarker: (markerId) => {
        const marker = markerRefs.get(markerId);
        if (!marker) {
          return;
        }
        marker.remove();
        markerRefs.delete(markerId);
      },
      clearMarkers: () => {
        for (const marker of markerRefs.values()) {
          marker.remove();
        }
        markerRefs.clear();
      },
      fitBounds: (points) => {
        if (points.length === 0) {
          return;
        }

        const bounds = new mapboxgl.LngLatBounds([points[0].lng, points[0].lat], [points[0].lng, points[0].lat]);
        for (const point of points.slice(1)) {
          bounds.extend([point.lng, point.lat]);
        }
        map.fitBounds(bounds, { padding: 48, duration: 450, maxZoom: 14 });
      },
      setRoutePath: (points) => {
        if (points.length < 2) {
          return;
        }

        if (map.isStyleLoaded()) {
          setRoutePath(points);
          return;
        }

        map.once("load", () => {
          setRoutePath(points);
        });
      },
      destroy: () => {
        for (const marker of markerRefs.values()) {
          marker.remove();
        }
        markerRefs.clear();
        map.remove();
      }
    };
  }
}

export function isValidCoordinate(lat: number, lng: number): boolean {
  return (
    Number.isFinite(lat) &&
    Number.isFinite(lng) &&
    lat >= -90 && lat <= 90 &&
    lng >= -180 && lng <= 180 &&
    !(lat === 0 && lng === 0)
  );
}

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

  for (const point of points) {
    if (!isValidCoordinate(point.lat, point.lng)) {
      continue;
    }

    const gridKey = `${Math.floor(point.lat / gridSize)}_${Math.floor(point.lng / gridSize)}`;
    const existing = grid.get(gridKey);

    if (existing) {
      existing.points.push(point);
      existing.count += 1;
      existing.center = {
        lat: existing.points.reduce((sum, row) => sum + row.lat, 0) / existing.count,
        lng: existing.points.reduce((sum, row) => sum + row.lng, 0) / existing.count
      };
      continue;
    }

    grid.set(gridKey, {
      center: { lat: point.lat, lng: point.lng },
      points: [point],
      count: 1
    });
  }

  return Array.from(grid.values());
}
