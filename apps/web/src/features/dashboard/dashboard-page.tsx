import { useState } from "react";
import { Link, useNavigate } from "react-router-dom";
import { useForm } from "react-hook-form";
import { SurfaceCard } from "../../components/surface-card";
import { StatusPill } from "../../components/status-pill";
import { useCreateTripMutation, useTripsQuery } from "../../lib/queries";
import { analyticsEventNames, trackEvent } from "../../lib/analytics";
import { useI18n } from "../../lib/i18n";
import { useUiStore } from "../../store/ui-store";

interface CreateTripFormValues {
  name: string;
  destinationText: string;
  startDate: string;
  endDate: string;
  timezone: string;
  currency: string;
  travelersCount: number;
}

export function DashboardPage() {
  const { t } = useI18n();
  const navigate = useNavigate();
  const pushToast = useUiStore((state) => state.pushToast);
  const [showForm, setShowForm] = useState(false);
  const { data: trips = [], isLoading, error } = useTripsQuery();
  const createTrip = useCreateTripMutation();
  const form = useForm<CreateTripFormValues>({
    defaultValues: {
      name: "",
      destinationText: "",
      startDate: "2026-04-14",
      endDate: "2026-04-19",
      timezone: "Asia/Tokyo",
      currency: "JPY",
      travelersCount: 2
    }
  });

  const onSubmit = form.handleSubmit(async (values) => {
    const trip = await createTrip.mutateAsync(values);
    trackEvent({ name: analyticsEventNames.tripCreated, context: { trip_id: trip.id } });
    pushToast(`Trip created: ${trip.name}`);
    setShowForm(false);
    navigate(`/trips/${trip.id}`);
  });

  return (
    <div className="grid gap-6 xl:grid-cols-[1.25fr_0.75fr]">
      <SurfaceCard
        eyebrow="Workspace"
        title="Shared travel workspaces"
        action={
          <button
            className="rounded-full bg-ink px-4 py-2 text-sm font-medium text-sand"
            onClick={() => setShowForm((current) => !current)}
            type="button"
          >
            {showForm ? "Close" : "Create trip"}
          </button>
        }
      >
        {showForm ? (
          <form className="mb-6 grid gap-4 rounded-[28px] bg-sand/75 p-5 md:grid-cols-2" onSubmit={onSubmit}>
            <label className="block">
              <span className="mb-2 block text-sm font-medium text-ink">Trip name</span>
              <input className="w-full rounded-2xl border border-ink/10 bg-white px-4 py-3" {...form.register("name")} />
            </label>
            <label className="block">
              <span className="mb-2 block text-sm font-medium text-ink">Destination</span>
              <input className="w-full rounded-2xl border border-ink/10 bg-white px-4 py-3" {...form.register("destinationText")} />
            </label>
            <label className="block">
              <span className="mb-2 block text-sm font-medium text-ink">Start date</span>
              <input className="w-full rounded-2xl border border-ink/10 bg-white px-4 py-3" type="date" {...form.register("startDate")} />
            </label>
            <label className="block">
              <span className="mb-2 block text-sm font-medium text-ink">End date</span>
              <input className="w-full rounded-2xl border border-ink/10 bg-white px-4 py-3" type="date" {...form.register("endDate")} />
            </label>
            <label className="block">
              <span className="mb-2 block text-sm font-medium text-ink">Timezone</span>
              <input className="w-full rounded-2xl border border-ink/10 bg-white px-4 py-3" {...form.register("timezone")} />
            </label>
            <label className="block">
              <span className="mb-2 block text-sm font-medium text-ink">Currency</span>
              <input className="w-full rounded-2xl border border-ink/10 bg-white px-4 py-3" {...form.register("currency")} />
            </label>
            <label className="block">
              <span className="mb-2 block text-sm font-medium text-ink">Travelers</span>
              <input
                className="w-full rounded-2xl border border-ink/10 bg-white px-4 py-3"
                type="number"
                min={1}
                {...form.register("travelersCount", { valueAsNumber: true })}
              />
            </label>
            <div className="flex items-end">
              <button className="rounded-full bg-coral px-5 py-3 text-sm font-medium text-white" disabled={createTrip.isPending} type="submit">
                {createTrip.isPending ? "Creating..." : "Submit"}
              </button>
            </div>
          </form>
        ) : null}
        {error ? (
          <div className="mb-6 rounded-[24px] bg-coral/10 p-4 text-sm text-coral">
            Unable to load trips right now. Please refresh or try again shortly.
          </div>
        ) : null}
        {isLoading ? <div className="rounded-[24px] bg-sand/70 p-5 text-sm text-ink/60">Loading trips from API...</div> : null}
        <div className="grid gap-4">
          {trips.map((trip) => (
            <Link
              key={trip.id}
              to={`/trips/${trip.id}`}
              className={`rounded-[28px] bg-gradient-to-r ${trip.coverGradient} p-5 text-white transition hover:scale-[0.99]`}
            >
              <div className="flex flex-wrap items-center justify-between gap-3">
                <div>
                  <p className="text-xs uppercase tracking-[0.24em] text-white/70">{trip.destination}</p>
                  <h2 className="mt-2 font-display text-3xl font-bold">{trip.name}</h2>
                </div>
                <StatusPill tone="accent">{trip.role}</StatusPill>
              </div>
              <div className="mt-6 flex flex-wrap gap-5 text-sm text-white/85">
                <span>{trip.dateRange}</span>
                <span>{trip.timezone}</span>
                <span>{trip.members} members</span>
                <span>{trip.currency}</span>
              </div>
            </Link>
          ))}
        </div>
      </SurfaceCard>
      <SurfaceCard eyebrow="App Shell" title="Frontend foundation">
        <div className="space-y-4 text-sm leading-7 text-ink/72">
          <p>Session hydration, route protection, offline sync hint, toast region, and responsive navigation are in place.</p>
          <p>Module surfaces cover Trip, Itinerary, Budget, Map, AI Planner, and Notification entry points.</p>
          <p>State is split between TanStack Query for server cache and Zustand for app/session UI state.</p>
        </div>
        <div className="mt-6 rounded-[24px] border border-ink/10 bg-white p-5">
          <h3 className="font-display text-2xl font-bold text-ink">{t("auth.quickAccess")}</h3>
          <p className="mt-2 text-sm text-ink/70">{t("auth.accessHint")}</p>
          <div className="mt-4 flex flex-wrap gap-3">
            <Link className="rounded-full bg-ink px-4 py-2 text-sm font-medium text-sand" to="/welcome">
              {t("auth.openWelcome")}
            </Link>
            <Link className="rounded-full border border-ink/15 bg-sand px-4 py-2 text-sm font-medium text-ink" to="/login">
              {t("auth.openLogin")}
            </Link>
          </div>
        </div>
        <div className="mt-6 rounded-[24px] border border-ink/10 bg-sand/70 p-5">
          <h3 className="font-display text-2xl font-bold text-ink">{t("dashboard.remainingTitle")}</h3>
          <p className="mt-2 text-sm text-ink/70">{t("dashboard.remainingDescription")}</p>
          <ul className="mt-4 space-y-2 text-sm text-ink/75">
            <li>{t("dashboard.remainingItem.1")}</li>
            <li>{t("dashboard.remainingItem.2")}</li>
            <li>{t("dashboard.remainingItem.3")}</li>
            <li>{t("dashboard.remainingItem.4")}</li>
          </ul>
        </div>
      </SurfaceCard>
    </div>
  );
}
