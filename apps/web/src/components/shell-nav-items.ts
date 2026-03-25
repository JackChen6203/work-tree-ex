import { useI18n } from "../lib/i18n";
import { useTripsQuery } from "../lib/queries";

export interface ShellNavItem {
  id: string;
  label: string;
  to?: string;
}

export function useShellNavItems(): ShellNavItem[] {
  const { t } = useI18n();
  const { data: trips = [] } = useTripsQuery();
  const activeTripId = trips[0]?.id;
  const tripBase = activeTripId ? `/trips/${activeTripId}` : undefined;

  return [
    { id: "overview", to: "/", label: t("nav.overview") },
    { id: "trip", to: tripBase, label: t("nav.trip") },
    { id: "itinerary", to: tripBase ? `${tripBase}/itinerary` : undefined, label: t("nav.itinerary") },
    { id: "budget", to: tripBase ? `${tripBase}/budget` : undefined, label: t("nav.budget") },
    { id: "map", to: tripBase ? `${tripBase}/map` : undefined, label: t("nav.map") },
    { id: "ai-planner", to: tripBase ? `${tripBase}/ai-planner` : undefined, label: t("nav.aiPlanner") },
    { id: "inbox", to: "/notifications", label: t("nav.inbox") },
    { id: "settings", to: "/settings", label: t("nav.settings") }
  ];
}
