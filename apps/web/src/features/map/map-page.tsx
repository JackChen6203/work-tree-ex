import { useMemo, useState } from "react";
import { SurfaceCard } from "../../components/surface-card";
import { useEstimateRouteMutation, useMapPlacesQuery } from "../../lib/queries";
import { useUiStore } from "../../store/ui-store";

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
  const pushToast = useUiStore((state) => state.pushToast);
  const { data: places = [], isLoading } = useMapPlacesQuery("kyoto");
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

  const runEstimate = async () => {
    if (points.length < 2) {
      return;
    }

    const result = await estimateRoute.mutateAsync({
      origin: { lat: points[0].lat, lng: points[0].lng },
      destination: { lat: points[1].lat, lng: points[1].lng },
      mode: "transit"
    });

    setEstimatedRoutes((prev) => [
      {
        id: crypto.randomUUID(),
        originTitle: points[0].title,
        destinationTitle: points[1].title,
        distanceKm: Math.round((result.distanceMeters / 1000) * 10) / 10,
        durationMin: Math.round(result.durationSeconds / 60),
        provider: result.provider,
        estimatedCostAmount: result.estimatedCostAmount,
        estimatedCostCurrency: result.estimatedCostCurrency
      },
      ...prev
    ]);
    pushToast(`Route ${Math.round(result.distanceMeters / 1000)}km / ${Math.round(result.durationSeconds / 60)}min`);
  };

  return (
    <div className="grid gap-6 lg:grid-cols-[1.1fr_0.9fr]">
      <SurfaceCard
        eyebrow="Map Module"
        title="Provider-agnostic route preview"
        action={
          <button
            className="rounded-full bg-ink px-4 py-2 text-sm font-medium text-sand"
            disabled={estimateRoute.isPending || points.length < 2}
            onClick={() => {
              void runEstimate();
            }}
            type="button"
          >
            {estimateRoute.isPending ? "Estimating..." : "Estimate route"}
          </button>
        }
      >
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
            SDK adapter layer connected to search/route endpoints. UI model remains detached from provider response shape.
          </div>
        </div>
      </SurfaceCard>
      <SurfaceCard eyebrow="Places" title="Daily linked POIs">
        {isLoading ? <div className="mb-3 rounded-[20px] bg-sand p-3 text-sm text-ink/65">Loading places...</div> : null}
        <div className="space-y-3">
          {points.map((item) => (
            <div key={item.id} className="rounded-[24px] bg-sand p-4">
              <p className="font-medium text-ink">{item.title}</p>
              <p className="mt-1 text-sm text-ink/60">{item.location}</p>
              <p className="mt-2 text-sm text-pine">{item.transit}</p>
            </div>
          ))}
          {!isLoading && points.length === 0 ? <div className="rounded-[24px] bg-sand p-4 text-sm text-ink/65">No places returned from provider.</div> : null}
        </div>

        <div className="mt-6 border-t border-ink/10 pt-5">
          <div className="mb-3 flex items-center justify-between gap-3">
            <p className="text-sm font-semibold text-ink">Estimated route snapshots</p>
            <button
              className="rounded-full border border-ink/15 px-3 py-1 text-xs font-medium text-ink disabled:opacity-40"
              disabled={estimatedRoutes.length === 0}
              onClick={() => {
                setEstimatedRoutes([]);
              }}
              type="button"
            >
              Clear
            </button>
          </div>
          {estimatedRoutes.length === 0 ? <p className="text-sm text-ink/60">No route estimates yet.</p> : null}
          <div className="space-y-3">
            {estimatedRoutes.map((route) => (
              <div className="rounded-[20px] border border-ink/10 bg-white p-3" key={route.id}>
                <p className="text-sm font-medium text-ink">{route.originTitle}{" -> "}{route.destinationTitle}</p>
                <p className="mt-1 text-xs text-ink/65">{route.distanceKm} km / {route.durationMin} min / provider: {route.provider}</p>
                {route.estimatedCostAmount ? <p className="mt-1 text-xs text-ink/65">Cost: {route.estimatedCostCurrency} {route.estimatedCostAmount.toLocaleString()}</p> : null}
              </div>
            ))}
          </div>
        </div>
      </SurfaceCard>
    </div>
  );
}
