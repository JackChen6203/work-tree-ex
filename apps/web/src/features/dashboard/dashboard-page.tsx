import { useMemo, useState } from "react";
import { Link, useNavigate } from "react-router-dom";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { SurfaceCard } from "../../components/surface-card";
import { StatusPill } from "../../components/status-pill";
import { useCreateTripMutation, useMapPlacesQuery, useNotificationsQuery, useTripsQuery } from "../../lib/queries";
import { analyticsEventNames, trackEvent } from "../../lib/analytics";
import { useI18n } from "../../lib/i18n";
import { useUiStore } from "../../store/ui-store";
import { createTripSchema, validationMessages } from "../../lib/schemas";
import type { CreateTripFormValues } from "../../lib/schemas";
import type { Locale } from "../../lib/translations";
import { upsertBudgetProfile } from "../../lib/budget-api";
import { budgetSeedCategories, currencyOptions, timezoneOptions } from "../../lib/trip-form-options";

export function DashboardPage() {
  const { t, locale } = useI18n();
  const msgs = validationMessages[locale as Locale] ?? validationMessages.en;
  const navigate = useNavigate();
  const pushToast = useUiStore((state) => state.pushToast);
  const [showForm, setShowForm] = useState(false);
  const [destinationKeyword, setDestinationKeyword] = useState("");
  const [selectedDestinationPoint, setSelectedDestinationPoint] = useState<{ lat: number; lng: number } | null>(null);
  const { data: trips = [], isLoading, error } = useTripsQuery();
  const { data: notifications = [] } = useNotificationsQuery();
  const { data: destinationCandidatesData, isFetching: isDestinationSearching } = useMapPlacesQuery(destinationKeyword);
  const destinationCandidates = Array.isArray(destinationCandidatesData) ? destinationCandidatesData : [];
  const createTrip = useCreateTripMutation();
  const form = useForm<CreateTripFormValues>({
    resolver: zodResolver(createTripSchema),
    defaultValues: {
      name: "",
      departureText: "",
      destinationText: "",
      startDate: "",
      endDate: "",
      timezone: Intl.DateTimeFormat().resolvedOptions().timeZone,
      currency: "TWD",
      totalBudget: undefined,
      travelersCount: 2
    }
  });
  const destinationValue = form.watch("destinationText");
  const destinationSuggestions = useMemo(() => destinationCandidates.slice(0, 6), [destinationCandidates]);
  const destinationSearchActive = destinationValue.trim().length > 0;
  const destinationField = form.register("destinationText");
  const { formState: { errors } } = form;
  const recentActivities = notifications.slice(0, 4);

  const recentTripIds = Array.from(
    new Set(
      notifications
        .map((item) => {
          const match = item.link?.match(/\/trips\/([^/]+)/);
          return match?.[1];
        })
        .filter(Boolean) as string[]
    )
  );

  const quickAccessTrips = (recentTripIds.length > 0
    ? recentTripIds
      .map((id) => trips.find((trip) => trip.id === id))
      .filter(Boolean)
    : trips.slice(0, 3)) as typeof trips;

  const onSubmit = form.handleSubmit(async (values) => {
    const departureText = values.departureText.trim();
    const destinationText = values.destinationText.trim();
    const trip = await createTrip.mutateAsync({
      name: values.name.trim(),
      departureText,
      destinationText,
      destinations: [departureText, destinationText],
      startDate: values.startDate,
      endDate: values.endDate,
      timezone: values.timezone,
      currency: values.currency,
      travelersCount: values.travelersCount
    });

    if (typeof values.totalBudget === "number") {
      try {
        await upsertBudgetProfile(trip.id, {
          totalBudget: values.totalBudget,
          currency: values.currency,
          categories: budgetSeedCategories.map((category) => ({ category, plannedAmount: 0 }))
        });
      } catch {
        pushToast({ type: "warning", message: t("dashboard.budgetSetupFailed") });
      }
    }

    trackEvent({ name: analyticsEventNames.tripCreated, context: { trip_id: trip.id } });
    pushToast(t("trip.created"));
    setShowForm(false);
    form.reset({
      name: "",
      departureText: "",
      destinationText: "",
      startDate: "",
      endDate: "",
      timezone: Intl.DateTimeFormat().resolvedOptions().timeZone,
      currency: "TWD",
      totalBudget: undefined,
      travelersCount: 2
    });
    setDestinationKeyword("");
    setSelectedDestinationPoint(null);
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
              <span className="mb-2 block text-sm font-medium text-ink">{t("trip.departure")}</span>
              <input className="w-full rounded-2xl border border-ink/10 bg-white px-4 py-3" {...form.register("departureText")} />
              {errors.departureText ? <p className="mt-1 text-xs text-coral">{msgs[errors.departureText.message ?? ""] ?? errors.departureText.message}</p> : null}
            </label>
            <label className="block md:col-span-2">
              <span className="mb-2 block text-sm font-medium text-ink">{t("trip.destination")}</span>
              <input
                className="w-full rounded-2xl border border-ink/10 bg-white px-4 py-3"
                {...destinationField}
                onChange={(event) => {
                  destinationField.onChange(event);
                  const value = event.target.value.trim();
                  setDestinationKeyword(value);
                  setSelectedDestinationPoint(null);
                }}
              />
              {errors.destinationText ? <p className="mt-1 text-xs text-coral">{msgs[errors.destinationText.message ?? ""] ?? errors.destinationText.message}</p> : null}
              {destinationSearchActive ? (
                <div className="mt-2 rounded-2xl border border-ink/10 bg-white p-3">
                  <p className="text-xs font-medium text-ink/60">{t("trip.destinationSearchHint")}</p>
                  {isDestinationSearching ? <p className="mt-2 text-xs text-ink/55">{t("trip.destinationSearching")}</p> : null}
                  {!isDestinationSearching && destinationSuggestions.length === 0 ? (
                    <p className="mt-2 text-xs text-ink/55">{t("trip.destinationNoMatch")}</p>
                  ) : null}
                  {destinationSuggestions.length > 0 ? (
                    <div className="mt-2 grid gap-2">
                      {destinationSuggestions.map((place) => (
                        <button
                          className="rounded-xl border border-ink/10 px-3 py-2 text-left text-sm text-ink transition hover:border-ink/20 hover:bg-sand/60"
                          key={place.providerPlaceId}
                          onClick={() => {
                            form.setValue("destinationText", `${place.name}${place.address ? `, ${place.address}` : ""}`, { shouldDirty: true, shouldValidate: true });
                            setDestinationKeyword(place.name);
                            setSelectedDestinationPoint({ lat: place.lat, lng: place.lng });
                          }}
                          type="button"
                        >
                          <p className="font-medium">{place.name}</p>
                          <p className="text-xs text-ink/60">{place.address || "-"}</p>
                        </button>
                      ))}
                    </div>
                  ) : null}
                  <div className="mt-3 rounded-xl border border-ink/10 bg-sand/60 p-3">
                    <p className="text-xs text-ink/65">{t("trip.destinationExternalHelp")}</p>
                    <div className="mt-2 flex flex-wrap gap-2">
                      <a
                        className="rounded-full border border-ink/15 bg-white px-3 py-1 text-xs font-medium text-ink transition hover:bg-sand"
                        href={`https://www.google.com/maps/search/?api=1&query=${encodeURIComponent(destinationValue.trim())}`}
                        rel="noopener noreferrer"
                        target="_blank"
                      >
                        {t("trip.openGoogleMaps")}
                      </a>
                      <a
                        className="rounded-full border border-ink/15 bg-white px-3 py-1 text-xs font-medium text-ink transition hover:bg-sand"
                        href="https://gemini.google.com/app"
                        rel="noopener noreferrer"
                        target="_blank"
                      >
                        {t("trip.openGemini")}
                      </a>
                    </div>
                  </div>
                </div>
              ) : null}
              {selectedDestinationPoint ? (
                <p className="mt-2 text-xs text-ink/60">
                  {t("trip.destinationCoord")
                    .replace("{lat}", selectedDestinationPoint.lat.toFixed(6))
                    .replace("{lng}", selectedDestinationPoint.lng.toFixed(6))}
                </p>
              ) : null}
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
              <select className="w-full rounded-2xl border border-ink/10 bg-white px-4 py-3" {...form.register("timezone")}>
                {timezoneOptions.map((timezone: string) => (
                  <option key={timezone} value={timezone}>
                    {timezone}
                  </option>
                ))}
              </select>
              <p className="mt-1 text-xs text-ink/55">{t("trip.timezoneHint")}</p>
            </label>
            <label className="block">
              <span className="mb-2 block text-sm font-medium text-ink">{t("trip.currency")}</span>
              <select className="w-full rounded-2xl border border-ink/10 bg-white px-4 py-3" {...form.register("currency")}>
                {currencyOptions.map((currency) => (
                  <option key={currency.code} value={currency.code}>
                    {currency.label}
                  </option>
                ))}
              </select>
              {errors.currency ? <p className="mt-1 text-xs text-coral">{msgs[errors.currency.message ?? ""] ?? errors.currency.message}</p> : null}
            </label>
            <label className="block">
              <span className="mb-2 block text-sm font-medium text-ink">{t("budget.totalBudget")}</span>
              <input
                className="w-full rounded-2xl border border-ink/10 bg-white px-4 py-3"
                min={0}
                placeholder="0"
                step="1"
                type="number"
                {...form.register("totalBudget", {
                  setValueAs: (value) => (value === "" ? undefined : Number(value))
                })}
              />
              {errors.totalBudget ? <p className="mt-1 text-xs text-coral">{msgs[errors.totalBudget.message ?? ""] ?? errors.totalBudget.message}</p> : null}
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
            const upcomingTrip = trips.find((trip) => trip.startDate > today);
            const currentTrip = trips.find((trip) => trip.startDate <= today && trip.endDate >= today);
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

        <SurfaceCard eyebrow={t("nav.inbox")} title={t("dashboard.recentActivity")}>
          {recentActivities.length === 0 ? <p className="text-sm text-ink/60">{t("notifications.empty")}</p> : null}
          <div className="space-y-3">
            {recentActivities.map((activity) => (
              <Link className="block rounded-[18px] border border-ink/10 bg-sand/70 px-4 py-3" key={activity.id} to={activity.link || "/notifications"}>
                <p className="text-sm font-medium text-ink">{activity.title}</p>
                <p className="mt-1 text-xs text-ink/60">{activity.body}</p>
                <p className="mt-2 text-[11px] uppercase tracking-[0.18em] text-ink/45">{activity.createdAt ? new Date(activity.createdAt).toLocaleString() : "-"}</p>
              </Link>
            ))}
          </div>
        </SurfaceCard>

        <SurfaceCard eyebrow={t("dashboard.workspace")} title={t("dashboard.quickAccess")}>
          {quickAccessTrips.length === 0 ? <p className="text-sm text-ink/60">{t("dashboard.noTrips")}</p> : null}
          <div className="space-y-3">
            {quickAccessTrips.slice(0, 3).map((trip) => (
              <Link className="block rounded-[18px] border border-ink/10 bg-white px-4 py-3 transition hover:bg-sand/80" key={trip.id} to={`/trips/${trip.id}/itinerary`}>
                <p className="text-sm font-semibold text-ink">{trip.name}</p>
                <p className="mt-1 text-xs text-ink/60">{trip.destination}</p>
                <p className="mt-2 text-[11px] uppercase tracking-[0.18em] text-ink/45">{trip.dateRange}</p>
              </Link>
            ))}
          </div>
        </SurfaceCard>
      </div>
    </div>
  );
}
