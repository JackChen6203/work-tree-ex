import { NavLink } from "react-router-dom";
import clsx from "clsx";
import { useShellNavItems } from "./shell-nav-items";

export function ShellNav() {
  const items = useShellNavItems();

  return (
    <nav className="flex flex-wrap gap-2">
      {items.map((item) => (
        item.to ? (
          <NavLink
            key={item.id}
            end
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
        ) : (
          <span
            key={item.id}
            aria-disabled="true"
            className="cursor-not-allowed rounded-full border border-ink/8 bg-white/40 px-4 py-2 text-sm font-medium text-ink/35"
            title="Create or open a trip to unlock this section."
          >
            {item.label}
          </span>
        )
      ))}
    </nav>
  );
}
