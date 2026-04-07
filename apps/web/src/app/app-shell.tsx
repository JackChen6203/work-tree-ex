import { Outlet, useNavigate } from "react-router-dom";
import { AppLogo } from "../components/app-logo";
import { BottomSheetHost } from "../components/bottom-sheet-host";
import { GlobalModalHost } from "../components/global-modal-host";
import { LoadingOverlay } from "../components/loading-overlay";
import { LocaleSwitcher } from "../components/locale-switcher";
import { NotificationBell } from "../components/notification-bell";
import { ShellNav } from "../components/shell-nav";
import { ToastRegion } from "../components/toast-region";
import { OfflineBanner } from "../features/offline/offline-banner";
import { useFcmPush } from "../features/notifications/use-fcm-push";
import { analyticsEventNames, trackEvent } from "../lib/analytics";
import { logout } from "../lib/auth-api";
import { useI18n } from "../lib/i18n";
import { broadcastSessionSignedOut } from "../lib/session-sync";
import { useSessionStore } from "../store/session-store";
import { useUiStore } from "../store/ui-store";

export function AppShell() {
  const navigate = useNavigate();
  const user = useSessionStore((state) => state.user);
  const clearUser = useSessionStore((state) => state.clearUser);
  const pushToast = useUiStore((state) => state.pushToast);
  const openConfirmModal = useUiStore((state) => state.openConfirmModal);
  const openSheet = useUiStore((state) => state.openSheet);
  const showLoadingOverlay = useUiStore((state) => state.showLoadingOverlay);
  const hideLoadingOverlay = useUiStore((state) => state.hideLoadingOverlay);
  const { t } = useI18n();
  useFcmPush();

  const onLogout = async () => {
    showLoadingOverlay(t("auth.loggingOut"));
    try {
      await logout();
      clearUser();
      broadcastSessionSignedOut();
      trackEvent({ name: analyticsEventNames.authLoggedOut });
      pushToast({ type: "success", message: t("auth.loggedOut") });
      navigate("/welcome");
    } catch {
      pushToast({ type: "error", message: t("auth.logoutError") });
    } finally {
      hideLoadingOverlay();
    }
  };

  return (
    <div className="min-h-screen px-4 py-4 sm:px-6 lg:px-10">
      <a
        className="sr-only z-[90] rounded-md bg-ink px-3 py-2 text-sm font-medium text-sand focus:not-sr-only focus:fixed focus:left-4 focus:top-4"
        href="#main-content"
      >
        {t("common.skipToMain")}
      </a>
      <OfflineBanner />
      <div className="mx-auto flex min-h-[calc(100vh-2rem)] max-w-7xl flex-col gap-6 rounded-[36px] border border-white/60 bg-white/35 p-4 shadow-card backdrop-blur sm:p-6">
        <header className="flex flex-col gap-4 rounded-[28px] bg-gradient-to-r from-white via-white/80 to-[#f0dfd6] p-5 lg:flex-row lg:items-center lg:justify-between">
          <div className="flex flex-col gap-4">
            <div className="flex items-start justify-between gap-3">
              <AppLogo />
              <button
                className="rounded-full border border-ink/12 bg-white/85 px-4 py-2 text-sm font-medium text-ink shadow-sm lg:hidden"
                onClick={() => {
                  openSheet("mobile-nav");
                }}
                type="button"
              >
                {t("shell.menu")}
              </button>
            </div>
            <div className="hidden lg:block">
              <ShellNav />
            </div>
          </div>
          <div className="flex flex-wrap items-center justify-end gap-3 self-start lg:self-center">
            <div className="flex items-center gap-2 rounded-full bg-white/80 px-3 py-2 text-xs font-medium text-ink/70">
              <span>{t("shell.language")}</span>
              <LocaleSwitcher />
            </div>
            <NotificationBell />
            <div className="flex items-center gap-3 rounded-full bg-ink px-4 py-2 text-sand">
              <div className="flex h-10 w-10 items-center justify-center rounded-full bg-sand text-sm font-bold text-ink">
                {user?.avatar ?? "TT"}
              </div>
              <div>
                <p className="text-sm font-medium">{user?.name ?? t("common.guest")}</p>
                <p className="text-xs text-sand/70">{user?.email ?? t("common.awaitingSession")}</p>
              </div>
              <button
                className="rounded-full bg-sand/20 px-3 py-1 text-xs font-medium text-sand transition hover:bg-sand/30"
                onClick={() => {
                  openConfirmModal({
                    title: t("auth.logoutConfirmTitle"),
                    description: t("auth.logoutConfirmDescription"),
                    confirmLabel: t("auth.logout"),
                    cancelLabel: t("common.cancel"),
                    tone: "danger",
                    onConfirm: onLogout
                  });
                }}
                type="button"
              >
                {t("auth.logout")}
              </button>
            </div>
          </div>
        </header>
        <main className="flex-1" id="main-content" tabIndex={-1}>
          <Outlet />
        </main>
      </div>
      <BottomSheetHost />
      <GlobalModalHost />
      <LoadingOverlay />
      <ToastRegion />
    </div>
  );
}
