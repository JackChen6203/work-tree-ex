import { useEffect, useMemo, useRef, useState } from "react";
import { useParams } from "react-router-dom";
import "mapbox-gl/dist/mapbox-gl.css";
import { SurfaceCard } from "../../components/surface-card";
import {
  useCreateItineraryItemMutation,
  useEstimateRouteMutation,
  useItineraryDaysQuery,
  useMapPlacesQuery
} from "../../lib/queries";
import { useUiStore } from "../../store/ui-store";
import { useI18n } from "../../lib/i18n";
import { MapboxAdapter, isValidCoordinate, type MapInstance } from "../../lib/map-provider-adapter";
import { extractItineraryMapPoints, toRoutePath } from "./map-itinerary-utils";

interface EstimatedRouteCard {
  id: string;
  originTitle: string;
  destinationTitle: string;
  distanceKm: number;
  durationMin: number;
  provider: string;
  estimatedCostAmount?: number;
  estimatedCostCurrency?: string;
}

const MAP_CONTAINER_ID = "trip-map-canvas";
const MAPBOX_TOKEN = import.meta.env.VITE_MAPBOX_ACCESS_TOKEN as string | undefined;

function buildStaticMapPreviewUrl(lat: number, lng: number) {
  const params = new URLSearchParams({
    center: `${lat},${lng}`,
    zoom: "13",
    size: "640x220",
    markers: `${lat},${lng},red-pushpin`
  });
  return `https://staticmap.openstreetmap.de/staticmap.php?${params.toString()}`;
}

export function MapPage() {
  const { tripId = "" } = useParams();
  const { t } = useI18n();
  const pushToast = useUiStore((state) => state.pushToast);
  const [searchInput, setSearchInput] = useState("kyoto");
  const [searchKeyword, setSearchKeyword] = useState("kyoto");
  const [originId, setOriginId] = useState("");
  const [destinationId, setDestinationId] = useState("");
  const [targetDayId, setTargetDayId] = useState("");
  const [travelMode, setTravelMode] = useState<"walk" | "transit" | "drive" | "taxi">("transit");
  const [selectedPointId, setSelectedPointId] = useState("");
  const [pointScope, setPointScope] = useState<"day" | "trip">("day");
  const [mapState, setMapState] = useState<"initializing" | "ready" | "fallback">("initializing");
  const [estimatedRoutes, setEstimatedRoutes] = useState<EstimatedRouteCard[]>([]);
  const { data: places = [], isLoading } = useMapPlacesQuery(searchKeyword);
  const { data: itineraryDays = [] } = useItineraryDaysQuery(tripId);
  const estimateRoute = useEstimateRouteMutation();
  const createItem = useCreateItineraryItemMutation(tripId);
  const mapRef = useRef<MapInstance | null>(null);
  const markerPointIdsRef = useRef<string[]>([]);

  const allItineraryPoints = useMemo(() => extractItineraryMapPoints(itineraryDays), [itineraryDays]);

  const dayOptions = useMemo(
    () =>
      itineraryDays.map((day, index) => ({
        dayId: day.dayId,
        label: `${t("itinerary.day").replace("{n}", String(index + 1))} · ${day.date}`
      })),
    [itineraryDays, t]
  );

  useEffect(() => {
    if (!targetDayId && dayOptions.length > 0) {
      setTargetDayId(dayOptions[0].dayId);
    }
  }, [dayOptions, targetDayId]);

  const displayedItineraryPoints = useMemo(() => {
    if (pointScope === "trip") {
      return allItineraryPoints;
    }
    if (!targetDayId) {
      return allItineraryPoints;
    }
    return allItineraryPoints.filter((point) => point.dayId === targetDayId);
  }, [allItineraryPoints, pointScope, targetDayId]);

  const searchPoints = useMemo(
    () =>
      places.map((place) => ({
        id: place.providerPlaceId,
        title: place.name,
        location: place.address,
        transit: place.categories.join(" / "),
        lat: place.lat,
        lng: place.lng,
        placeId: place.providerPlaceId
      })),
    [places]
  );

  const originPoint = useMemo(() => searchPoints.find((point) => point.id === originId), [searchPoints, originId]);
  const destinationPoint = useMemo(() => searchPoints.find((point) => point.id === destinationId), [searchPoints, destinationId]);

  useEffect(() => {
    if (!MAPBOX_TOKEN) {
      setMapState("fallback");
      return;
    }

    let cancelled = false;

    const initMap = async () => {
      try {
        const adapter = await MapboxAdapter.create(MAPBOX_TOKEN);
        if (cancelled) {
          return;
        }

        mapRef.current = adapter.renderMap(MAP_CONTAINER_ID, {
          center: { lat: 35.0116, lng: 135.7681 },
          zoom: 11,
          markers: displayedItineraryPoints.map((point, index) => ({
            lat: point.lat,
            lng: point.lng,
            label: `${index + 1}`
          })),
          enableClustering: true,
          onMarkerClick: (markerIndex) => {
            const pointId = markerPointIdsRef.current[markerIndex];
            if (!pointId) {
              return;
            }
            setSelectedPointId(pointId);
          }
        });
        setMapState("ready");
      } catch {
        setMapState("fallback");
      }
    };

    void initMap();

    return () => {
      cancelled = true;
      mapRef.current?.destroy();
      mapRef.current = null;
    };
  }, []);

  useEffect(() => {
    if (searchPoints.length === 0) {
      setOriginId("");
      setDestinationId("");
      return;
    }

    setOriginId((current) => current || searchPoints[0].id);
    setDestinationId((current) => current || searchPoints[Math.min(1, searchPoints.length - 1)].id);
  }, [searchPoints]);

  useEffect(() => {
    if (displayedItineraryPoints.length === 0) {
      setSelectedPointId("");
      return;
    }
    setSelectedPointId((current) => current || displayedItineraryPoints[0].id);
  }, [displayedItineraryPoints]);

  useEffect(() => {
    if (mapState !== "ready" || !mapRef.current) {
      return;
    }

    markerPointIdsRef.current = displayedItineraryPoints.map((point) => point.id);
    mapRef.current.setMarkers(
      displayedItineraryPoints.map((point, index) => ({
        lat: point.lat,
        lng: point.lng,
        label: `${index + 1}`
      }))
    );
    mapRef.current.fitBounds(displayedItineraryPoints);
    mapRef.current.setRoutePath(toRoutePath(displayedItineraryPoints));
  }, [displayedItineraryPoints, mapState]);

  const canEstimate = Boolean(originPoint && destinationPoint && originPoint.id !== destinationPoint.id);

  const focusPoint = (pointId: string) => {
    setSelectedPointId(pointId);

    const point = displayedItineraryPoints.find((item) => item.id === pointId);
    if (!point || !isValidCoordinate(point.lat, point.lng) || !mapRef.current) {
      return;
    }

    mapRef.current.setCenter({ lat: point.lat, lng: point.lng });
    mapRef.current.setZoom(13);
  };

  const runEstimate = async () => {
    if (!originPoint || !destinationPoint || originPoint.id === destinationPoint.id) {
      return;
    }

    const result = await estimateRoute.mutateAsync({
      origin: { lat: originPoint.lat, lng: originPoint.lng },
      destination: { lat: destinationPoint.lat, lng: destinationPoint.lng },
      mode: travelMode
    });

    if (isValidCoordinate(originPoint.lat, originPoint.lng) && isValidCoordinate(destinationPoint.lat, destinationPoint.lng)) {
      mapRef.current?.setRoutePath([
        { lat: originPoint.lat, lng: originPoint.lng },
        { lat: destinationPoint.lat, lng: destinationPoint.lng }
      ]);
    }

    setEstimatedRoutes((prev) => [
      {
        id: crypto.randomUUID(),
        originTitle: originPoint.title,
        destinationTitle: destinationPoint.title,
        distanceKm: Math.round((result.distanceMeters / 1000) * 10) / 10,
        durationMin: Math.round(result.durationSeconds / 60),
        provider: result.provider ?? "maps",
        estimatedCostAmount: result.estimatedCostAmount,
        estimatedCostCurrency: result.estimatedCostCurrency
      },
      ...prev
    ]);
    pushToast(`${t("map.distance")} ${Math.round(result.distanceMeters / 1000)}km / ${t("map.duration")} ${Math.round(result.durationSeconds / 60)}min`);
  };

  const addPoiToItinerary = async (pointId: string) => {
    const targetDay = targetDayId || itineraryDays[0]?.dayId;
    if (!targetDay) {
      pushToast({ type: "warning", message: t("map.noDayForAdd") });
      return;
    }

    const point = searchPoints.find((item) => item.id === pointId);
    if (!point) {
      return;
    }

    try {
      const created = await createItem.mutateAsync({
        dayId: targetDay,
        title: point.title,
        itemType: "place_visit",
        allDay: false,
        note: point.location,
        placeId: point.placeId,
        lat: point.lat,
        lng: point.lng
      });
      if (created.warnings && created.warnings.length > 0) {
        pushToast({
          type: "warning",
          message: t("itinerary.serverConflictWarning").replace("{items}", created.warnings.join(", "))
        });
      } else {
        pushToast(t("map.poiAdded"));
      }
    } catch (error) {
      pushToast({
        type: "error",
        message: error instanceof Error ? error.message : t("common.actionFailed")
      });
    }
  };

  return (
    <div className="grid gap-6 lg:grid-cols-[1.1fr_0.9fr]">
      <SurfaceCard
        eyebrow={t("nav.map")}
        title={t("map.title")}
        action={
          <button
            className="rounded-full bg-ink px-4 py-2 text-sm font-medium text-sand"
            disabled={estimateRoute.isPending || !canEstimate}
            onClick={() => {
              void runEstimate();
            }}
            type="button"
          >
            {estimateRoute.isPending ? t("map.estimating") : t("map.routeEstimate")}
          </button>
        }
      >
        <form
          className="mb-4 grid gap-3 rounded-[24px] bg-white/10 p-3 text-white md:grid-cols-4"
          onSubmit={(event) => {
            event.preventDefault();
            setSearchKeyword(searchInput.trim() || "kyoto");
          }}
        >
          <label className="md:col-span-2">
            <span className="mb-1 block text-xs uppercase tracking-[0.2em] text-white/70">{t("map.search")}</span>
            <input
              className="w-full rounded-xl border border-white/25 bg-white/10 px-3 py-2 text-sm text-white outline-none placeholder:text-white/60"
              placeholder={t("map.searchPlaceholder")}
              value={searchInput}
              onChange={(event) => setSearchInput(event.target.value)}
            />
          </label>
          <div>
            <span className="mb-1 block text-xs uppercase tracking-[0.2em] text-white/70">{t("ai.transport")}</span>
            <select
              className="w-full rounded-xl border border-white/25 bg-white/10 px-3 py-2 text-sm text-white"
              value={travelMode}
              onChange={(event) => setTravelMode(event.target.value as "walk" | "transit" | "drive" | "taxi")}
            >
              <option value="transit">{t("map.transit")}</option>
              <option value="walk">{t("map.walk")}</option>
              <option value="drive">{t("map.drive")}</option>
              <option value="taxi">{t("map.taxi")}</option>
            </select>
          </div>
          <div className="flex items-end">
            <button className="w-full rounded-xl bg-white/20 px-3 py-2 text-sm font-medium text-white hover:bg-white/30" type="submit">
              {t("map.search")}
            </button>
          </div>
        </form>

        <div className="mb-4 grid gap-3 rounded-[20px] border border-ink/10 bg-white p-3 md:grid-cols-[1fr_auto_auto]">
          <label>
            <span className="mb-1 block text-xs uppercase tracking-[0.18em] text-ink/55">{t("map.pointScope")}</span>
            <select
              className="w-full rounded-xl border border-ink/10 bg-sand px-3 py-2 text-sm text-ink"
              onChange={(event) => setPointScope(event.target.value as "day" | "trip")}
              value={pointScope}
            >
              <option value="day">{t("map.scopeDay")}</option>
              <option value="trip">{t("map.scopeTrip")}</option>
            </select>
          </label>
          <label>
            <span className="mb-1 block text-xs uppercase tracking-[0.18em] text-ink/55">{t("map.dayFilter")}</span>
            <select
              className="w-full rounded-xl border border-ink/10 bg-sand px-3 py-2 text-sm text-ink disabled:opacity-45"
              disabled={pointScope === "trip" || dayOptions.length === 0}
              onChange={(event) => setTargetDayId(event.target.value)}
              value={targetDayId}
            >
              {dayOptions.map((day) => (
                <option key={day.dayId} value={day.dayId}>
                  {day.label}
                </option>
              ))}
            </select>
          </label>
          <div className="rounded-xl border border-ink/10 bg-sand px-3 py-2 text-xs text-ink/65">
            {t("map.pointCount").replace("{n}", String(displayedItineraryPoints.length))}
          </div>
        </div>

        <div className="relative min-h-[420px] overflow-hidden rounded-[28px] border border-ink/15 bg-ink">
          {mapState === "ready" ? (
            <div className="absolute inset-0">
              <div className="h-full w-full" id={MAP_CONTAINER_ID} />
            </div>
          ) : (
            <div className="absolute inset-0 bg-[radial-gradient(circle_at_20%_20%,rgba(218,106,78,0.55),transparent_20%),radial-gradient(circle_at_80%_25%,rgba(244,239,230,0.18),transparent_15%),radial-gradient(circle_at_55%_70%,rgba(45,90,74,0.55),transparent_20%),linear-gradient(180deg,#12202d_0%,#1a2f2c_100%)]" />
          )}

          {mapState === "fallback" ? (
            <div className="absolute bottom-5 left-5 rounded-2xl bg-white/10 px-4 py-3 text-sm text-white backdrop-blur">
              {t("map.sdkFailed")} — {t("map.fallbackList")}
            </div>
          ) : null}

          {mapState === "initializing" ? (
            <div className="absolute bottom-5 left-5 rounded-2xl bg-white/10 px-4 py-3 text-sm text-white backdrop-blur">
              {t("common.loading")}
            </div>
          ) : null}
        </div>
      </SurfaceCard>

      <SurfaceCard eyebrow={t("map.linkedPois")} title={t("map.dailyPois")}>
        {displayedItineraryPoints.length === 0 ? (
          <div className="mb-4 rounded-[20px] border border-dashed border-ink/20 bg-sand p-3 text-sm text-ink/65">
            {t("map.noItineraryPoints")}
          </div>
        ) : null}

        <div className="space-y-3">
          {displayedItineraryPoints.map((point) => (
            <button
              className={`w-full rounded-[24px] p-4 text-left transition ${
                selectedPointId === point.id ? "bg-pine/10 ring-1 ring-pine/30" : "bg-sand hover:bg-sand/70"
              }`}
              key={point.id}
              onClick={() => {
                focusPoint(point.id);
              }}
              type="button"
            >
              <p className="font-medium text-ink">{point.title}</p>
              <p className="mt-1 text-sm text-ink/60">{point.itemType}</p>
              <p className="mt-2 text-xs text-pine">{point.dayDate}</p>
            </button>
          ))}
        </div>

        {searchPoints.length >= 2 ? (
          <div className="mt-6 mb-4 grid gap-3 rounded-[20px] border border-ink/10 bg-white p-3 md:grid-cols-2">
            <label>
              <span className="mb-1 block text-xs uppercase tracking-[0.18em] text-ink/55">{t("trip.startDate")}</span>
              <select
                className="w-full rounded-xl border border-ink/10 bg-sand px-3 py-2 text-sm text-ink"
                value={originId}
                onChange={(event) => setOriginId(event.target.value)}
              >
                {searchPoints.map((point) => (
                  <option key={point.id} value={point.id}>
                    {point.title}
                  </option>
                ))}
              </select>
            </label>
            <label>
              <span className="mb-1 block text-xs uppercase tracking-[0.18em] text-ink/55">{t("trip.endDate")}</span>
              <select
                className="w-full rounded-xl border border-ink/10 bg-sand px-3 py-2 text-sm text-ink"
                value={destinationId}
                onChange={(event) => setDestinationId(event.target.value)}
              >
                {searchPoints.map((point) => (
                  <option key={point.id} value={point.id}>
                    {point.title}
                  </option>
                ))}
              </select>
            </label>
          </div>
        ) : null}

        {isLoading ? <div className="mb-3 rounded-[20px] bg-sand p-3 text-sm text-ink/65">{t("map.searching")}</div> : null}

        <p className="mb-2 text-sm font-semibold text-ink">{t("map.searchResults")}</p>
        <div className="space-y-3">
          {searchPoints.map((item) => (
            <div className="rounded-[22px] border border-ink/10 bg-white p-3" key={item.id}>
              <p className="text-sm font-medium text-ink">{item.title}</p>
              <p className="mt-1 text-xs text-ink/60">{item.location}</p>
              <div className="mt-3 overflow-hidden rounded-xl border border-ink/10 bg-sand/70">
                <img
                  alt={t("map.previewAlt").replace("{title}", item.title)}
                  className="h-24 w-full object-cover"
                  decoding="async"
                  loading="lazy"
                  src={buildStaticMapPreviewUrl(item.lat, item.lng)}
                />
              </div>
              <div className="mt-3 flex items-center justify-between gap-2">
                <p className="text-xs text-pine">{item.transit}</p>
                <button
                  className="rounded-full border border-ink/15 px-3 py-1 text-xs font-medium text-ink disabled:opacity-45"
                  disabled={createItem.isPending}
                  onClick={() => {
                    void addPoiToItinerary(item.id);
                  }}
                  type="button"
                >
                  {createItem.isPending ? t("map.addingPoi") : t("map.addPoi")}
                </button>
              </div>
            </div>
          ))}
          {!isLoading && searchPoints.length === 0 ? <div className="rounded-[24px] bg-sand p-4 text-sm text-ink/65">{t("map.noResults")}</div> : null}
        </div>

        <div className="mt-6 border-t border-ink/10 pt-5">
          <div className="mb-3 flex items-center justify-between gap-3">
            <p className="text-sm font-semibold text-ink">{t("map.routeEstimate")}</p>
            <button
              className="rounded-full border border-ink/15 px-3 py-1 text-xs font-medium text-ink disabled:opacity-40"
              disabled={estimatedRoutes.length === 0}
              onClick={() => {
                setEstimatedRoutes([]);
              }}
              type="button"
            >
              {t("common.delete")}
            </button>
          </div>
          {estimatedRoutes.length === 0 ? <p className="text-sm text-ink/60">{t("common.noData")}</p> : null}
          <div className="space-y-3">
            {estimatedRoutes.map((route) => (
              <div className="rounded-[20px] border border-ink/10 bg-white p-3" key={route.id}>
                <p className="text-sm font-medium text-ink">
                  {route.originTitle}
                  {" → "}
                  {route.destinationTitle}
                </p>
                <p className="mt-1 text-xs text-ink/65">
                  {route.distanceKm} km / {route.durationMin} min · {route.provider}
                </p>
                {typeof route.estimatedCostAmount === "number" ? (
                  <p className="mt-1 text-xs text-ink/65">
                    {route.estimatedCostCurrency} {route.estimatedCostAmount.toLocaleString()}
                  </p>
                ) : null}
              </div>
            ))}
          </div>
        </div>
      </SurfaceCard>
    </div>
  );
}
