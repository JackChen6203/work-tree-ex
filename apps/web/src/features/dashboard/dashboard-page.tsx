import { useMemo, useState } from "react";
import { Link, useNavigate } from "react-router-dom";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { SurfaceCard } from "../../components/surface-card";
import { StatusPill } from "../../components/status-pill";
import { useCreateTripMutation, useNotificationsQuery, useTripsQuery } from "../../lib/queries";
import { analyticsEventNames, trackEvent } from "../../lib/analytics";
import { useI18n } from "../../lib/i18n";
import { useUiStore } from "../../store/ui-store";
import { createTripSchema, validationMessages } from "../../lib/schemas";
import type { CreateTripFormValues } from "../../lib/schemas";
import type { Locale } from "../../lib/translations";

export function DashboardPage() {
  const { t, locale } = useI18n();
  const msgs = validationMessages[locale as Locale] ?? validationMessages.en;
  const navigate = useNavigate();
  const pushToast = useUiStore((state) => state.pushToast);
  const [showForm, setShowForm] = useState(false);
  const { data: trips = [], isLoading, error } = useTripsQuery();
  const { data: notifications = [], isLoading: loadingNotifications } = useNotificationsQuery(false);
  const createTrip = useCreateTripMutation();
  const form = useForm<CreateTripFormValues>({
    resolver: zodResolver(createTripSchema),
    defaultValues: {
      name: "",
      destinationText: "",
      startDate: "",
      endDate: "",
      timezone: Intl.DateTimeFormat().resolvedOptions().timeZone,
      currency: "TWD",
      travelersCount: 2
    }
  });
  const { formState: { errors } } = form;
  const today = new Date().toISOString().slice(0, 10);
  const upcomingTrip = useMemo(
    () =>
      trips
        .filter((trip) => trip.startDate > today)
        .sort((left, right) => left.startDate.localeCompare(right.startDate))[0],
    [trips, today]
  );
  const currentTrip = useMemo(
    () => trips.find((trip) => trip.startDate <= today && trip.endDate >= today),
    [trips, today]
  );
  const activityFeed = useMemo(
    () =>
      notifications.slice(0, 5).map((notification) => ({
        id: notification.id,
        title: notification.title,
        detail: notification.body,
        href: notification.link,
        createdAt: notification.createdAt
      })),
    [notifications]
  );
  const quickAccessTrip = useMemo(() => {
    for (const notification of notifications) {
      const match = notification.link.match(/^\/trips\/([^/]+)/);
      if (!match) {
        continue;
      }
      const matchedTrip = trips.find((trip) => trip.id === match[1]);
      if (matchedTrip) {
        return matchedTrip;
      }
    }
    return trips[0] ?? null;
  }, [notifications, trips]);

  const onSubmit = form.handleSubmit(async (values) => {
    const trip = await createTrip.mutateAsync(values);
    trackEvent({ name: analyticsEventNames.tripCreated, context: { trip_id: trip.id } });
    pushToast(t("trip.created"));
    setShowForm(false);
    navigate(`/trips/${trip.id}`);
  });

  return (
    <div className="grid gap-6 xl:grid-cols-[1.25fr_0.75fr]">
      <SurfaceCard
        eyebrow={t("dashboard.workspace")}
        title={t("dashboard.title")}
        action={
          <button
            className="rounded-full bg-ink px-4 py-2 text-sm font-medium text-sand"
            onClick={() => setShowForm((current) => !current)}
            type="button"
          >
            {showForm ? t("dashboard.closeForm") : t("dashboard.createTrip")}
          </button>
        }
      >
        {showForm ? (
          <form className="mb-6 grid gap-4 rounded-[28px] bg-sand/75 p-5 md:grid-cols-2" onSubmit={onSubmit}>
            <label className="block">
              <span className="mb-2 block text-sm font-medium text-ink">{t("trip.name")}</span>
              <input className="w-full rounded-2xl border border-ink/10 bg-white px-4 py-3" {...form.register("name")} />
              {errors.name ? <p className="mt-1 text-xs text-coral">{msgs[errors.name.message ?? ""] ?? errors.name.message}</p> : null}
            </label>
            <label className="block">
              <span className="mb-2 block text-sm font-medium text-ink">{t("trip.destination")}</span>
              <input className="w-full rounded-2xl border border-ink/10 bg-white px-4 py-3" {...form.register("destinationText")} />
              {errors.destinationText ? <p className="mt-1 text-xs text-coral">{msgs[errors.destinationText.message ?? ""] ?? errors.destinationText.message}</p> : null}
            </label>
            <label className="block">
              <span className="mb-2 block text-sm font-medium text-ink">{t("trip.startDate")}</span>
              <input className="w-full rounded-2xl border border-ink/10 bg-white px-4 py-3" type="date" {...form.register("startDate")} />
              {errors.startDate ? <p className="mt-1 text-xs text-coral">{msgs[errors.startDate.message ?? ""] ?? errors.startDate.message}</p> : null}
            </label>
            <label className="block">
              <span className="mb-2 block text-sm font-medium text-ink">{t("trip.endDate")}</span>
              <input className="w-full rounded-2xl border border-ink/10 bg-white px-4 py-3" type="date" {...form.register("endDate")} />
              {errors.endDate ? <p className="mt-1 text-xs text-coral">{msgs[errors.endDate.message ?? ""] ?? errors.endDate.message}</p> : null}
            </label>
            <label className="block">
              <span className="mb-2 block text-sm font-medium text-ink">{t("trip.timezone")}</span>
              <input className="w-full rounded-2xl border border-ink/10 bg-white px-4 py-3" {...form.register("timezone")} />
            </label>
            <label className="block">
              <span className="mb-2 block text-sm font-medium text-ink">{t("trip.currency")}</span>
              <input className="w-full rounded-2xl border border-ink/10 bg-white px-4 py-3" {...form.register("currency")} />
            </label>
            <label className="block">
              <span className="mb-2 block text-sm font-medium text-ink">{t("trip.travelers")}</span>
              <input
                className="w-full rounded-2xl border border-ink/10 bg-white px-4 py-3"
                type="number"
                min={1}
                {...form.register("travelersCount", { valueAsNumber: true })}
              />
              {errors.travelersCount ? <p className="mt-1 text-xs text-coral">{msgs[errors.travelersCount.message ?? ""] ?? errors.travelersCount.message}</p> : null}
            </label>
            <div className="flex items-end">
              <button className="rounded-full bg-coral px-5 py-3 text-sm font-medium text-white" disabled={createTrip.isPending} type="submit">
                {createTrip.isPending ? t("common.creating") : t("common.submit")}
              </button>
            </div>
          </form>
        ) : null}
        {error ? (
          <div className="mb-6 rounded-[24px] bg-coral/10 p-4 text-sm text-coral">
            {t("dashboard.tripLoadError")}
          </div>
        ) : null}
        {isLoading ? <div className="rounded-[24px] bg-sand/70 p-5 text-sm text-ink/60">{t("dashboard.loadingTrips")}</div> : null}
        {!isLoading && trips.length === 0 ? (
          <div className="rounded-[24px] bg-sand/70 p-8 text-center">
            <p className="text-lg font-semibold text-ink/80">{t("dashboard.noTrips")}</p>
            <p className="mt-2 text-sm text-ink/60">{t("dashboard.noTripsHint")}</p>
          </div>
        ) : null}
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
                <span>{trip.members} {t("common.members")}</span>
                <span>{trip.currency}</span>
              </div>
            </Link>
          ))}
        </div>
      </SurfaceCard>
      <div className="space-y-6">
        <SurfaceCard eyebrow={t("dashboard.workspace")} title={t("dashboard.upcomingTrip")}>
          {(() => {
            if (currentTrip) {
              return (
                <div className="rounded-[24px] bg-gradient-to-br from-pine/10 to-pine/5 p-5">
                  <p className="text-xs font-medium uppercase tracking-[0.2em] text-pine">{t("dashboard.tripInProgress")}</p>
                  <p className="mt-2 text-lg font-bold text-ink">{currentTrip.name}</p>
                  <p className="mt-1 text-sm text-ink/60">{currentTrip.destination}</p>
                </div>
              );
            }
            if (upcomingTrip) {
              const diffMs = new Date(upcomingTrip.startDate).getTime() - Date.now();
              const diffDays = Math.max(0, Math.ceil(diffMs / (1000 * 60 * 60 * 24)));
              return (
                <div className="rounded-[24px] bg-gradient-to-br from-coral/10 to-coral/5 p-5">
                  <p className="font-display text-5xl font-bold text-coral">{diffDays}</p>
                  <p className="mt-2 text-sm font-medium text-ink/70">{t("dashboard.daysUntil").replace("{n}", String(diffDays))}</p>
                  <p className="mt-1 text-sm text-ink/60">{upcomingTrip.name} · {upcomingTrip.destination}</p>
                </div>
              );
            }
            return <p className="text-sm text-ink/60">{t("dashboard.noUpcoming")}</p>;
          })()}
        </SurfaceCard>
        <SurfaceCard eyebrow={t("dashboard.workspace")} title={t("dashboard.recentActivity")}>
          {loadingNotifications ? <p className="text-sm text-ink/60">{t("common.loading")}</p> : null}
          {!loadingNotifications && activityFeed.length === 0 ? <p className="text-sm text-ink/60">{t("dashboard.noRecentActivity")}</p> : null}
          <div className="space-y-3">
            {activityFeed.map((item) => (
              <Link key={item.id} to={item.href} className="block rounded-[20px] border border-ink/10 bg-sand px-4 py-3 transition hover:bg-white">
                <p className="text-sm font-medium text-ink">{item.title}</p>
                <p className="mt-1 text-xs text-ink/65">{item.detail}</p>
                <p className="mt-2 text-[11px] uppercase tracking-[0.18em] text-ink/45">
                  {item.createdAt ? new Date(item.createdAt).toLocaleString() : ""}
                </p>
              </Link>
            ))}
          </div>
        </SurfaceCard>
        <SurfaceCard eyebrow={t("dashboard.workspace")} title={t("dashboard.quickAccess")}>
          {!quickAccessTrip ? <p className="text-sm text-ink/60">{t("dashboard.noTripsHint")}</p> : null}
          {quickAccessTrip ? (
            <div className="rounded-[24px] bg-gradient-to-br from-ink/5 to-pine/10 p-5">
              <p className="text-xs uppercase tracking-[0.2em] text-ink/50">{t("dashboard.quickAccessHint")}</p>
              <p className="mt-2 text-lg font-semibold text-ink">{quickAccessTrip.name}</p>
              <p className="text-sm text-ink/60">{quickAccessTrip.destination}</p>
              <div className="mt-4 flex flex-wrap gap-2">
                <Link to={`/trips/${quickAccessTrip.id}`} className="rounded-full border border-ink/15 px-3 py-1 text-xs font-medium text-ink">{t("dashboard.openTrip")}</Link>
                <Link to={`/trips/${quickAccessTrip.id}/itinerary`} className="rounded-full border border-ink/15 px-3 py-1 text-xs font-medium text-ink">{t("dashboard.openItinerary")}</Link>
                <Link to={`/trips/${quickAccessTrip.id}/budget`} className="rounded-full border border-ink/15 px-3 py-1 text-xs font-medium text-ink">{t("dashboard.openBudget")}</Link>
              </div>
            </div>
          ) : null}
        </SurfaceCard>
      </div>
    </div>
  );
}
