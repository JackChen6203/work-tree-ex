import { NavLink } from "react-router-dom";
import clsx from "clsx";
import { useShellNavItems } from "./shell-nav-items";
import { useI18n } from "../lib/i18n";

export function ShellNav() {
  const items = useShellNavItems();
  const { t } = useI18n();

  return (
    <nav className="flex items-center gap-2 overflow-x-auto whitespace-nowrap scrollbar-hide">
      {items.map((item) =>
        item.to ? (
          <NavLink
            key={item.id}
            end
            to={item.to}
            className={({ isActive }) =>
              clsx(
                "shrink-0 rounded-full px-4 py-2 text-sm font-medium transition",
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
            className="shrink-0 cursor-not-allowed rounded-full border border-ink/8 bg-white/40 px-4 py-2 text-sm font-medium text-ink/35"
            title={t("nav.disabledHint")}
          >
            {item.label}
          </span>
        )
      )}
    </nav>
  );
}
