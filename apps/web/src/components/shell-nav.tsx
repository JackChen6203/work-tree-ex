import { NavLink } from "react-router-dom";
import clsx from "clsx";

const items = [
  { to: "/", label: "Overview" },
  { to: "/trips/kyoto-2026", label: "Trip" },
  { to: "/trips/kyoto-2026/itinerary", label: "Itinerary" },
  { to: "/trips/kyoto-2026/budget", label: "Budget" },
  { to: "/trips/kyoto-2026/map", label: "Map" },
  { to: "/trips/kyoto-2026/ai-planner", label: "AI Planner" },
  { to: "/notifications", label: "Inbox" }
];

export function ShellNav() {
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
