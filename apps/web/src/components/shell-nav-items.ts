import { useI18n } from "../lib/i18n";
import { useTripsQuery } from "../lib/queries";
import { useLocation } from "react-router-dom";

export interface ShellNavItem {
  id: string;
  label: string;
  to?: string;
}

export function useShellNavItems(): ShellNavItem[] {
  const { t } = useI18n();
  const location = useLocation();
  const { data: trips = [] } = useTripsQuery();
  const routeTripId = location.pathname.match(/^\/trips\/([^/]+)/)?.[1];
  const activeTripId = routeTripId || trips[0]?.id;
  const tripBase = activeTripId ? `/trips/${activeTripId}` : undefined;
  const createTripTarget = "/?openCreateTrip=1";

  return [
    { id: "overview", to: "/", label: t("nav.overview") },
    { id: "trip", to: tripBase ?? createTripTarget, label: t("nav.trip") },
    { id: "itinerary", to: tripBase ? `${tripBase}/itinerary` : createTripTarget, label: t("nav.itinerary") },
    { id: "budget", to: tripBase ? `${tripBase}/budget` : createTripTarget, label: t("nav.budget") },
    { id: "map", to: tripBase ? `${tripBase}/map` : createTripTarget, label: t("nav.map") },
    { id: "ai-planner", to: tripBase ? `${tripBase}/ai-planner` : createTripTarget, label: t("nav.aiPlanner") },
    { id: "inbox", to: "/notifications", label: t("nav.inbox") },
    { id: "settings", to: "/settings", label: t("nav.settings") }
  ];
}
