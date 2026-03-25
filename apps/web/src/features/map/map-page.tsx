import { useEffect, useMemo, useState } from "react";
import { SurfaceCard } from "../../components/surface-card";
import { useEstimateRouteMutation, useMapPlacesQuery } from "../../lib/queries";
import { useUiStore } from "../../store/ui-store";
import { useI18n } from "../../lib/i18n";

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

export function MapPage() {
  const { t } = useI18n();
  const pushToast = useUiStore((state) => state.pushToast);
  const [searchInput, setSearchInput] = useState("kyoto");
  const [searchKeyword, setSearchKeyword] = useState("kyoto");
  const [originId, setOriginId] = useState("");
  const [destinationId, setDestinationId] = useState("");
  const [travelMode, setTravelMode] = useState<"walk" | "transit" | "drive" | "taxi">("transit");
  const { data: places = [], isLoading } = useMapPlacesQuery(searchKeyword);
  const estimateRoute = useEstimateRouteMutation();
  const [estimatedRoutes, setEstimatedRoutes] = useState<EstimatedRouteCard[]>([]);

  const points = useMemo(
    () =>
      places.map((place, index) => ({
        id: place.providerPlaceId,
        title: place.name,
        location: place.address,
        transit: place.categories.join(" / "),
        lat: place.lat,
        lng: place.lng,
        index
      })),
    [places]
  );

  const originPoint = useMemo(() => points.find((point) => point.id === originId), [points, originId]);
  const destinationPoint = useMemo(() => points.find((point) => point.id === destinationId), [points, destinationId]);

  useEffect(() => {
    if (points.length === 0) {
      setOriginId("");
      setDestinationId("");
      return;
    }
    setOriginId((current) => current || points[0].id);
    setDestinationId((current) => current || points[Math.min(1, points.length - 1)].id);
  }, [points]);

  const canEstimate = Boolean(originPoint && destinationPoint && originPoint.id !== destinationPoint.id);

  const runEstimate = async () => {
    if (!originPoint || !destinationPoint || originPoint.id === destinationPoint.id) {
      return;
    }

    const result = await estimateRoute.mutateAsync({
      origin: { lat: originPoint.lat, lng: originPoint.lng },
      destination: { lat: destinationPoint.lat, lng: destinationPoint.lng },
      mode: travelMode
    });

    setEstimatedRoutes((prev) => [
      {
        id: crypto.randomUUID(),
        originTitle: originPoint.title,
        destinationTitle: destinationPoint.title,
        distanceKm: Math.round((result.distanceMeters / 1000) * 10) / 10,
        durationMin: Math.round(result.durationSeconds / 60),
        provider: result.provider,
        estimatedCostAmount: result.estimatedCostAmount,
        estimatedCostCurrency: result.estimatedCostCurrency
      },
      ...prev
    ]);
    pushToast(`${t("map.distance")} ${Math.round(result.distanceMeters / 1000)}km / ${t("map.duration")} ${Math.round(result.durationSeconds / 60)}min`);
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

        <div className="relative min-h-[380px] overflow-hidden rounded-[28px] bg-ink">
          <div className="absolute inset-0 bg-[radial-gradient(circle_at_20%_20%,rgba(218,106,78,0.55),transparent_20%),radial-gradient(circle_at_80%_25%,rgba(244,239,230,0.18),transparent_15%),radial-gradient(circle_at_55%_70%,rgba(45,90,74,0.55),transparent_20%),linear-gradient(180deg,#12202d_0%,#1a2f2c_100%)]" />
          {points.slice(0, 5).map((item, index) => (
            <div
              key={item.id}
              className="absolute flex h-12 w-12 items-center justify-center rounded-full border border-white/20 bg-white/15 text-xs font-bold text-white"
              style={{
                left: `${18 + index * 14}%`,
                top: `${18 + (index % 3) * 18}%`
              }}
            >
              {index + 1}
            </div>
          ))}
          <div className="absolute bottom-5 left-5 rounded-2xl bg-white/10 px-4 py-3 text-sm text-white backdrop-blur">
            {t("map.sdkFailed")} — {t("map.fallbackList")}
          </div>
        </div>
      </SurfaceCard>
      <SurfaceCard eyebrow={t("map.linkedPois")} title={t("map.dailyPois")}>
        {isLoading ? <div className="mb-3 rounded-[20px] bg-sand p-3 text-sm text-ink/65">{t("map.searching")}</div> : null}
        {points.length >= 2 ? (
          <div className="mb-4 grid gap-3 rounded-[20px] border border-ink/10 bg-white p-3 md:grid-cols-2">
            <label>
              <span className="mb-1 block text-xs uppercase tracking-[0.18em] text-ink/55">{t("trip.startDate")}</span>
              <select
                className="w-full rounded-xl border border-ink/10 bg-sand px-3 py-2 text-sm text-ink"
                value={originId}
                onChange={(event) => setOriginId(event.target.value)}
              >
                {points.map((point) => (
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
                {points.map((point) => (
                  <option key={point.id} value={point.id}>
                    {point.title}
                  </option>
                ))}
              </select>
            </label>
          </div>
        ) : null}
        <div className="space-y-3">
          {points.map((item) => (
            <div key={item.id} className="rounded-[24px] bg-sand p-4">
              <p className="font-medium text-ink">{item.title}</p>
              <p className="mt-1 text-sm text-ink/60">{item.location}</p>
              <p className="mt-2 text-sm text-pine">{item.transit}</p>
            </div>
          ))}
          {!isLoading && points.length === 0 ? <div className="rounded-[24px] bg-sand p-4 text-sm text-ink/65">{t("map.noResults")}</div> : null}
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
                <p className="text-sm font-medium text-ink">{route.originTitle}{" → "}{route.destinationTitle}</p>
                <p className="mt-1 text-xs text-ink/65">{route.distanceKm} km / {route.durationMin} min</p>
                {route.estimatedCostAmount ? <p className="mt-1 text-xs text-ink/65">{route.estimatedCostCurrency} {route.estimatedCostAmount.toLocaleString()}</p> : null}
              </div>
            ))}
          </div>
        </div>
      </SurfaceCard>
    </div>
  );
}
