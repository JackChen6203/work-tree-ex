import { useEffect } from "react";
import { useSessionStore } from "../../store/session-store";
import { useUiStore } from "../../store/ui-store";

export function OfflineBanner() {
  const setOnline = useSessionStore((state) => state.setOnline);
  const pushToast = useUiStore((state) => state.pushToast);

  useEffect(() => {
    const handleOnline = () => {
      setOnline(true);
      pushToast("Connection restored. Resuming sync queue.");
    };
    const handleOffline = () => {
      setOnline(false);
      pushToast("Offline mode active. Mutations will be queued locally.");
    };

    window.addEventListener("online", handleOnline);
    window.addEventListener("offline", handleOffline);

    return () => {
      window.removeEventListener("online", handleOnline);
      window.removeEventListener("offline", handleOffline);
    };
  }, [pushToast, setOnline]);

  return null;
}
