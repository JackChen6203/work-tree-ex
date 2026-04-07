import { NavLink } from "react-router-dom";
import clsx from "clsx";
import { useShellNavItems } from "./shell-nav-items";

export function ShellNav() {
  const items = useShellNavItems();

  return (
    <nav className="flex items-center gap-2 overflow-x-auto whitespace-nowrap scrollbar-hide">
      {items.map((item) => (
        <NavLink
          key={item.id}
          end
          to={item.to ?? "/"}
          className={({ isActive }) =>
            clsx(
              "shrink-0 rounded-full px-4 py-2 text-sm font-medium transition",
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
