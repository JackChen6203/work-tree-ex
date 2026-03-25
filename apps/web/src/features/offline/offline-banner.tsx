import { useEffect } from "react";
import { useSessionStore } from "../../store/session-store";
import { useUiStore } from "../../store/ui-store";
import { useI18n } from "../../lib/i18n";

export function OfflineBanner() {
  const setOnline = useSessionStore((state) => state.setOnline);
  const pushToast = useUiStore((state) => state.pushToast);
  const { t } = useI18n();

  useEffect(() => {
    const handleOnline = () => {
      setOnline(true);
      pushToast(t("sync.connectionRestored"));
    };
    const handleOffline = () => {
      setOnline(false);
      pushToast(t("sync.offlineMode"));
    };

    window.addEventListener("online", handleOnline);
    window.addEventListener("offline", handleOffline);

    return () => {
      window.removeEventListener("online", handleOnline);
      window.removeEventListener("offline", handleOffline);
    };
  }, [pushToast, setOnline, t]);

  return null;
}
