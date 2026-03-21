import { NavLink } from "react-router-dom";
import clsx from "clsx";
import { useI18n } from "../lib/i18n";
import { useTripsQuery } from "../lib/queries";

export function ShellNav() {
  const { t } = useI18n();
  const { data: trips = [] } = useTripsQuery();
  const activeTripId = trips[0]?.id;
  const tripBase = activeTripId ? `/trips/${activeTripId}` : "/";

  const items = [
    { to: "/", label: t("nav.overview") },
    { to: tripBase, label: t("nav.trip") },
    { to: activeTripId ? `${tripBase}/itinerary` : "/", label: t("nav.itinerary") },
    { to: activeTripId ? `${tripBase}/budget` : "/", label: t("nav.budget") },
    { to: activeTripId ? `${tripBase}/map` : "/", label: t("nav.map") },
    { to: activeTripId ? `${tripBase}/ai-planner` : "/", label: t("nav.aiPlanner") },
    { to: "/notifications", label: t("nav.inbox") },
    { to: "/settings", label: t("nav.settings") }
  ];

  return (
    <nav className="flex flex-wrap gap-2">
      {items.map((item) => (
        <NavLink
          key={item.to}
          to={item.to}
          className={({ isActive }) =>
            clsx(
              "rounded-full px-4 py-2 text-sm font-medium transition",
              isActive ? "bg-ink text-sand" : "bg-white/70 text-ink/70 hover:bg-white"
            )
          }
        >
          {item.label}
        </NavLink>
      ))}
    </nav>
  );
}
