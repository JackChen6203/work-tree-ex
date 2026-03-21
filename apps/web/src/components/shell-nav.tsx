import { NavLink } from "react-router-dom";
import clsx from "clsx";
import { useI18n } from "../lib/i18n";

export function ShellNav() {
  const { t } = useI18n();
  const items = [
    { to: "/", label: t("nav.overview") },
    { to: "/trips/kyoto-2026", label: t("nav.trip") },
    { to: "/trips/kyoto-2026/itinerary", label: t("nav.itinerary") },
    { to: "/trips/kyoto-2026/budget", label: t("nav.budget") },
    { to: "/trips/kyoto-2026/map", label: t("nav.map") },
    { to: "/trips/kyoto-2026/ai-planner", label: t("nav.aiPlanner") },
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
