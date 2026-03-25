import { useState } from "react";
import { Link, useNavigate } from "react-router-dom";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { SurfaceCard } from "../../components/surface-card";
import { StatusPill } from "../../components/status-pill";
import { useCreateTripMutation, useTripsQuery } from "../../lib/queries";
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
            const today = new Date().toISOString().slice(0, 10);
            const upcomingTrip = trips.find((trip) => trip.dateRange && trip.dateRange > today);
            const currentTrip = trips.find((trip) => {
              const parts = (trip.dateRange ?? "").split(" – ");
              return parts.length === 2 && parts[0] <= today && parts[1] >= today;
            });
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
              const diffMs = new Date(upcomingTrip.dateRange.split(" – ")[0] ?? "").getTime() - Date.now();
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
      </div>
    </div>
  );
}
