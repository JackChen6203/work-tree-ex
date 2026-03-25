import { NavLink } from "react-router-dom";
import clsx from "clsx";
import { LocaleSwitcher } from "./locale-switcher";
import { useShellNavItems } from "./shell-nav-items";
import { useI18n } from "../lib/i18n";
import { useSessionStore } from "../store/session-store";
import { useUiStore } from "../store/ui-store";

export function BottomSheetHost() {
  const activeSheet = useUiStore((state) => state.activeSheet);
  const closeSheet = useUiStore((state) => state.closeSheet);
  const { t } = useI18n();
  const items = useShellNavItems();
  const user = useSessionStore((state) => state.user);

  if (!activeSheet || activeSheet.type !== "mobile-nav") {
    return null;
  }

  return (
    <div className="fixed inset-0 z-[65] flex items-end bg-ink/35 backdrop-blur-sm lg:hidden">
      <button aria-label={t("common.close")} className="absolute inset-0" onClick={closeSheet} type="button" />
      <div
        aria-describedby="mobile-nav-sheet-description"
        aria-modal="true"
        className="relative w-full rounded-t-[32px] border border-white/70 bg-white/95 px-4 pb-7 pt-4 shadow-card"
        role="dialog"
      >
        <div className="mx-auto mb-4 h-1.5 w-14 rounded-full bg-ink/15" />
        <div className="flex items-start justify-between gap-4">
          <div>
            <p className="text-xs uppercase tracking-[0.24em] text-ink/45">{t("sheet.mobileNavEyebrow")}</p>
            <h2 className="mt-2 font-display text-2xl font-bold text-ink">{t("sheet.mobileNavTitle")}</h2>
            <p className="mt-2 text-sm text-ink/70" id="mobile-nav-sheet-description">
              {t("sheet.mobileNavDescription")}
            </p>
          </div>
          <button
            className="rounded-full border border-ink/12 bg-sand px-3 py-1.5 text-xs font-medium text-ink"
            onClick={closeSheet}
            type="button"
          >
            {t("common.close")}
          </button>
        </div>

        <div className="mt-5 rounded-[24px] border border-ink/10 bg-sand/70 p-4">
          <p className="text-xs uppercase tracking-[0.22em] text-ink/45">{t("sheet.accountEyebrow")}</p>
          <div className="mt-3 flex items-center gap-3">
            <div className="flex h-11 w-11 items-center justify-center rounded-full bg-ink text-sm font-bold text-sand">
              {user?.avatar ?? "TT"}
            </div>
            <div className="min-w-0">
              <p className="truncate text-sm font-medium text-ink">{user?.name ?? t("common.guest")}</p>
              <p className="truncate text-xs text-ink/60">{user?.email ?? t("common.awaitingSession")}</p>
            </div>
          </div>
        </div>

        <nav className="mt-5 grid gap-2">
          {items.map((item) =>
            item.to ? (
              <NavLink
                key={item.id}
                end
                to={item.to}
                className={({ isActive }) =>
                  clsx(
                    "rounded-[20px] px-4 py-3 text-sm font-medium transition",
                    isActive ? "bg-ink text-sand" : "border border-ink/10 bg-white text-ink/75"
                  )
                }
                onClick={closeSheet}
              >
                {item.label}
              </NavLink>
            ) : (
              <span
                key={item.id}
                aria-disabled="true"
                className="cursor-not-allowed rounded-[20px] border border-ink/8 bg-white/40 px-4 py-3 text-sm font-medium text-ink/35"
              >
                {item.label}
              </span>
            )
          )}
        </nav>

        <div className="mt-5 rounded-[24px] border border-ink/10 bg-white px-4 py-4">
          <p className="text-xs uppercase tracking-[0.22em] text-ink/45">{t("shell.language")}</p>
          <div className="mt-3">
            <LocaleSwitcher />
          </div>
        </div>
      </div>
    </div>
  );
}
