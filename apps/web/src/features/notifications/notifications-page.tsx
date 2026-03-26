import { useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";
import { SurfaceCard } from "../../components/surface-card";
import { useI18n } from "../../lib/i18n";
import { useUiStore } from "../../store/ui-store";
import {
  useCleanupReadNotificationsMutation,
  useDeleteNotificationMutation,
  useMarkAllNotificationsReadMutation,
  useMarkNotificationReadMutation,
  useMarkNotificationUnreadMutation,
  useNotificationsQuery,
  useTripsQuery
} from "../../lib/queries";

function resolveNotificationHref(
  input: { link?: string; type?: string },
  fallbackTripId: string | null
) {
  const raw = (input.link ?? "").trim();

  if (raw.startsWith("/trips/")) {
    return raw;
  }

  if (raw === "/dashboard") {
    return "/";
  }

  if (raw === "/trips" && fallbackTripId) {
    return `/trips/${fallbackTripId}`;
  }

  if (raw.length > 0) {
    return raw;
  }

  if (!fallbackTripId) {
    return "/notifications";
  }

  if (input.type?.includes("budget")) {
    return `/trips/${fallbackTripId}/budget`;
  }
  if (input.type?.includes("itinerary")) {
    return `/trips/${fallbackTripId}/itinerary`;
  }
  if (input.type?.includes("ai")) {
    return `/trips/${fallbackTripId}/ai-planner`;
  }
  return `/trips/${fallbackTripId}`;
}

function extractTripIdFromPath(path: string) {
  const match = path.match(/^\/trips\/([^/]+)/);
  return match?.[1] ?? null;
}

export function NotificationsPage() {
  const { t } = useI18n();
  const navigate = useNavigate();
  const pushToast = useUiStore((state) => state.pushToast);
  const [unreadOnly, setUnreadOnly] = useState(false);
  const { data: notifications = [], isLoading } = useNotificationsQuery(unreadOnly);
  const { data: trips = [] } = useTripsQuery();
  const markReadMutation = useMarkNotificationReadMutation();
  const markUnreadMutation = useMarkNotificationUnreadMutation();
  const markAllReadMutation = useMarkAllNotificationsReadMutation();
  const cleanupReadMutation = useCleanupReadNotificationsMutation();
  const deleteMutation = useDeleteNotificationMutation();
  const fallbackTripId = trips[0]?.id ?? null;
  const activeTripIds = useMemo(() => new Set(trips.map((trip) => trip.id)), [trips]);

  const items = useMemo(
    () =>
      notifications.map((item) => ({
        id: item.id,
        type: item.type,
        title: item.title,
        detail: item.body,
        href: resolveNotificationHref({ link: item.link, type: item.type }, fallbackTripId),
        unread: !item.readAt,
        time: item.createdAt ? new Date(item.createdAt).toLocaleString() : ""
      })),
    [fallbackTripId, notifications]
  );

  const markAllRead = () => {
    void markAllReadMutation.mutateAsync();
  };

  const markRead = (id: string) => {
    void markReadMutation.mutateAsync(id);
  };

  const markUnread = (id: string) => {
    void markUnreadMutation.mutateAsync(id);
  };

  const removeItem = (id: string) => {
    void deleteMutation.mutateAsync(id);
  };

  const cleanupRead = () => {
    void cleanupReadMutation.mutateAsync().then((result) => {
      pushToast(`${t("notifications.cleanupReadDone")}: ${result.deletedCount}`);
    });
  };

  const openNotification = (item: (typeof items)[number]) => {
    const targetTripId = extractTripIdFromPath(item.href);
    if (targetTripId && !activeTripIds.has(targetTripId)) {
      pushToast(t("notifications.tripDeleted"));
      void markReadMutation.mutateAsync(item.id);
      return;
    }

    void markReadMutation.mutateAsync(item.id).finally(() => {
      navigate(item.href);
    });
  };

  return (
    <SurfaceCard
      eyebrow={t("nav.inbox")}
      title={t("notifications.title")}
      action={
        <button className="rounded-full bg-ink px-4 py-2 text-sm font-medium text-sand" onClick={markAllRead} type="button">
          {t("notifications.markAllRead")}
        </button>
      }
    >
      <div className="mb-3 flex items-center justify-end gap-2">
        <button
          className="rounded-full border border-ink/20 px-3 py-1 text-xs font-medium text-ink"
          disabled={cleanupReadMutation.isPending}
          onClick={cleanupRead}
          type="button"
        >
          {cleanupReadMutation.isPending ? t("notifications.cleaning") : t("notifications.cleanupRead")}
        </button>
        <button
          className="rounded-full border border-ink/20 px-3 py-1 text-xs font-medium text-ink"
          onClick={() => {
            setUnreadOnly((prev) => !prev);
          }}
          type="button"
        >
          {unreadOnly ? t("notifications.showAll") : t("notifications.showUnread")}
        </button>
      </div>
      {isLoading ? <div className="rounded-[24px] bg-sand p-4 text-sm text-ink/65">{t("common.loading")}</div> : null}
      {!isLoading && items.length === 0 ? <div className="rounded-[24px] bg-sand p-4 text-sm text-ink/65">{t("notifications.empty")}</div> : null}
      <div className="space-y-3">
        {items.map((item) => (
          <div key={item.id} className={`rounded-[24px] p-4 transition ${item.unread ? "bg-[#fff1ed]" : "bg-sand"}`}>
            <div className="flex items-start justify-between gap-4">
              <div>
                <button
                  className="font-medium text-ink underline-offset-4 hover:underline"
                  onClick={() => {
                    openNotification(item);
                  }}
                  type="button"
                >
                  {item.title}
                </button>
                <p className="mt-2 text-sm text-ink/65">{item.detail}</p>
                <p className="mt-2 text-xs uppercase tracking-[0.2em] text-ink/45">{item.unread ? t("notifications.unread") : t("notifications.read")}</p>
              </div>
              <div className="flex flex-col items-end gap-3">
                <span className="text-xs uppercase tracking-[0.2em] text-ink/45">{item.time}</span>
                <button
                  className="rounded-full border border-ink/20 px-3 py-1 text-xs font-medium text-ink transition hover:bg-white/70"
                  disabled={markReadMutation.isPending || markUnreadMutation.isPending}
                  onClick={() => {
                    if (item.unread) {
                      markRead(item.id);
                      return;
                    }
                    markUnread(item.id);
                  }}
                  type="button"
                >
                  {item.unread ? t("notifications.markRead") : t("notifications.markUnread")}
                </button>
                <button
                  className="rounded-full border border-ink/20 px-3 py-1 text-xs font-medium text-ink transition hover:bg-white/70"
                  disabled={deleteMutation.isPending}
                  onClick={() => removeItem(item.id)}
                  type="button"
                >
                  {t("notifications.delete")}
                </button>
              </div>
            </div>
          </div>
        ))}
      </div>
    </SurfaceCard>
  );
}
