import { useEffect } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { setupPushMessaging } from "../../lib/fcm-messaging";
import { useI18n } from "../../lib/i18n";
import { useSessionStore } from "../../store/session-store";
import { useUiStore } from "../../store/ui-store";

const TOKEN_REFRESH_INTERVAL_MS = 6 * 60 * 60 * 1000;

export function useFcmPush() {
  const user = useSessionStore((state) => state.user);
  const pushToast = useUiStore((state) => state.pushToast);
  const { t } = useI18n();
  const queryClient = useQueryClient();

  useEffect(() => {
    if (!user) {
      return;
    }

    let cancelled = false;
    const onMockMessage = (event: Event) => {
      if (cancelled) {
        return;
      }

      const detail = (event as CustomEvent<{ title?: string; body?: string }>).detail;
      const title = detail?.title || t("notifications.pushForeground");
      const body = detail?.body;
      pushToast({
        type: "info",
        message: body ? `${title} · ${body}` : title
      });
      void queryClient.invalidateQueries({ queryKey: ["notifications"] });
    };

    if (import.meta.env.DEV) {
      window.addEventListener("mock-fcm-message", onMockMessage as EventListener);
    }

    const initPush = async () => {
      const result = await setupPushMessaging({
        promptForPermission: false,
        onForegroundMessage: (payload) => {
          if (cancelled) {
            return;
          }

          const title = payload.notification?.title || payload.data?.title || t("notifications.pushForeground");
          const body = payload.notification?.body || payload.data?.body;
          pushToast({
            type: "info",
            message: body ? `${title} · ${body}` : title
          });
          void queryClient.invalidateQueries({ queryKey: ["notifications"] });
        }
      });

      if (!cancelled && result.status === "denied") {
        pushToast(t("notifications.pushDenied"));
      }
    };

    void initPush();

    const intervalId = window.setInterval(() => {
      void setupPushMessaging({
        promptForPermission: false
      });
    }, TOKEN_REFRESH_INTERVAL_MS);

    return () => {
      cancelled = true;
      window.clearInterval(intervalId);
      if (import.meta.env.DEV) {
        window.removeEventListener("mock-fcm-message", onMockMessage as EventListener);
      }
    };
  }, [pushToast, queryClient, t, user]);
}
