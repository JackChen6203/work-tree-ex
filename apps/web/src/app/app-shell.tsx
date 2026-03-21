import { Outlet } from "react-router-dom";
import { AppLogo } from "../components/app-logo";
import { LocaleSwitcher } from "../components/locale-switcher";
import { ShellNav } from "../components/shell-nav";
import { SyncStatusBar } from "../components/sync-status-bar";
import { ToastRegion } from "../components/toast-region";
import { OfflineBanner } from "../features/offline/offline-banner";
import { useI18n } from "../lib/i18n";
import { useSessionStore } from "../store/session-store";

export function AppShell() {
  const user = useSessionStore((state) => state.user);
  const { t } = useI18n();

  return (
    <div className="min-h-screen px-4 py-4 sm:px-6 lg:px-10">
      <OfflineBanner />
      <div className="mx-auto flex min-h-[calc(100vh-2rem)] max-w-7xl flex-col gap-6 rounded-[36px] border border-white/60 bg-white/35 p-4 shadow-card backdrop-blur sm:p-6">
        <header className="flex flex-col gap-4 rounded-[28px] bg-gradient-to-r from-white via-white/80 to-[#f0dfd6] p-5 lg:flex-row lg:items-center lg:justify-between">
          <div className="flex flex-col gap-4">
            <AppLogo />
            <ShellNav />
          </div>
          <div className="flex flex-wrap items-center justify-end gap-3 self-start lg:self-center">
            <LocaleSwitcher />
            <div className="flex items-center gap-3 rounded-full bg-ink px-4 py-2 text-sand">
              <div className="flex h-10 w-10 items-center justify-center rounded-full bg-sand text-sm font-bold text-ink">
                {user?.avatar ?? "TT"}
              </div>
              <div>
                <p className="text-sm font-medium">{user?.name ?? t("common.guest")}</p>
                <p className="text-xs text-sand/70">{user?.email ?? t("common.awaitingSession")}</p>
              </div>
            </div>
          </div>
        </header>
        <SyncStatusBar />
        <main className="flex-1">
          <Outlet />
        </main>
      </div>
      <ToastRegion />
    </div>
  );
}
