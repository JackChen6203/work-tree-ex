import { useForm } from "react-hook-form";
import { useParams } from "react-router-dom";
import { SurfaceCard } from "../../components/surface-card";
import { StatusPill } from "../../components/status-pill";
import { usePatchTripMutation, useTripQuery } from "../../lib/queries";
import { useUiStore } from "../../store/ui-store";

interface TripPatchValues {
  name: string;
  destinationText: string;
  startDate: string;
  endDate: string;
  timezone: string;
  currency: string;
  travelersCount: number;
  status: "draft" | "active" | "archived";
}

export function TripOverviewPage() {
  const { tripId } = useParams();
  const pushToast = useUiStore((state) => state.pushToast);
  const { data: trip, isLoading, error } = useTripQuery(tripId ?? "");
  const patchTrip = usePatchTripMutation(tripId ?? "");
  const form = useForm<TripPatchValues>({
    values: trip
      ? {
          name: trip.name,
          destinationText: trip.destination,
          startDate: trip.startDate,
          endDate: trip.endDate,
          timezone: trip.timezone,
          currency: trip.currency,
          travelersCount: trip.travelersCount,
          status: trip.status
        }
      : undefined
  });

  if (isLoading) {
    return <div className="rounded-[28px] bg-white/80 p-6 text-sm text-ink/65">Loading trip detail from API...</div>;
  }

  if (error || !trip) {
    return <div className="rounded-[28px] bg-coral/10 p-6 text-sm text-coral">Trip detail could not be loaded from backend.</div>;
  }

  const onSubmit = form.handleSubmit(async (values) => {
    const updated = await patchTrip.mutateAsync({
      version: trip.version,
      input: values
    });
    pushToast(`Trip updated: ${updated.name}`);
  });

  return (
    <div className="grid gap-6 lg:grid-cols-[1.1fr_0.9fr]">
      <SurfaceCard eyebrow="Trip Module" title={trip.name}>
        <div className={`rounded-[28px] bg-gradient-to-br ${trip.coverGradient} p-6 text-white`}>
          <p className="text-xs uppercase tracking-[0.24em] text-white/70">{trip.destination}</p>
          <div className="mt-4 flex flex-wrap items-center gap-3">
            <StatusPill tone="accent">{trip.status}</StatusPill>
            <StatusPill tone="accent">{trip.role}</StatusPill>
          </div>
          <div className="mt-6 grid gap-4 sm:grid-cols-2">
            <div>
              <p className="text-sm text-white/70">Date range</p>
              <p className="mt-1 text-lg font-medium">{trip.dateRange}</p>
            </div>
            <div>
              <p className="text-sm text-white/70">Timezone</p>
              <p className="mt-1 text-lg font-medium">{trip.timezone}</p>
            </div>
            <div>
              <p className="text-sm text-white/70">Members</p>
              <p className="mt-1 text-lg font-medium">{trip.travelersCount}</p>
            </div>
            <div>
              <p className="text-sm text-white/70">Version</p>
              <p className="mt-1 text-lg font-medium">v{trip.version}</p>
            </div>
          </div>
        </div>
      </SurfaceCard>
      <SurfaceCard eyebrow="Server Data" title="Patch trip metadata">
        <form className="grid gap-4" onSubmit={onSubmit}>
          <label className="block">
            <span className="mb-2 block text-sm font-medium text-ink">Trip name</span>
            <input className="w-full rounded-2xl border border-ink/10 bg-sand px-4 py-3" {...form.register("name")} />
          </label>
          <label className="block">
            <span className="mb-2 block text-sm font-medium text-ink">Destination</span>
            <input className="w-full rounded-2xl border border-ink/10 bg-sand px-4 py-3" {...form.register("destinationText")} />
          </label>
          <div className="grid gap-4 sm:grid-cols-2">
            <label className="block">
              <span className="mb-2 block text-sm font-medium text-ink">Start date</span>
              <input className="w-full rounded-2xl border border-ink/10 bg-sand px-4 py-3" type="date" {...form.register("startDate")} />
            </label>
            <label className="block">
              <span className="mb-2 block text-sm font-medium text-ink">End date</span>
              <input className="w-full rounded-2xl border border-ink/10 bg-sand px-4 py-3" type="date" {...form.register("endDate")} />
            </label>
          </div>
          <div className="grid gap-4 sm:grid-cols-2">
            <label className="block">
              <span className="mb-2 block text-sm font-medium text-ink">Timezone</span>
              <input className="w-full rounded-2xl border border-ink/10 bg-sand px-4 py-3" {...form.register("timezone")} />
            </label>
            <label className="block">
              <span className="mb-2 block text-sm font-medium text-ink">Currency</span>
              <input className="w-full rounded-2xl border border-ink/10 bg-sand px-4 py-3" {...form.register("currency")} />
            </label>
          </div>
          <div className="grid gap-4 sm:grid-cols-2">
            <label className="block">
              <span className="mb-2 block text-sm font-medium text-ink">Travelers</span>
              <input
                className="w-full rounded-2xl border border-ink/10 bg-sand px-4 py-3"
                type="number"
                min={1}
                {...form.register("travelersCount", { valueAsNumber: true })}
              />
            </label>
            <label className="block">
              <span className="mb-2 block text-sm font-medium text-ink">Status</span>
              <select className="w-full rounded-2xl border border-ink/10 bg-sand px-4 py-3" {...form.register("status")}>
                <option value="draft">draft</option>
                <option value="active">active</option>
                <option value="archived">archived</option>
              </select>
            </label>
          </div>
          <button className="rounded-full bg-pine px-5 py-3 text-sm font-medium text-white" disabled={patchTrip.isPending} type="submit">
            {patchTrip.isPending ? "Saving..." : "Save trip"}
          </button>
        </form>
      </SurfaceCard>
    </div>
  );
}
