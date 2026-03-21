import { useMemo } from "react";
import { Link } from "react-router-dom";
import { SurfaceCard } from "../../components/surface-card";
import { useI18n } from "../../lib/i18n";
import { useMarkNotificationReadMutation, useNotificationsQuery } from "../../lib/queries";

export function NotificationsPage() {
  const { t } = useI18n();
  const { data: notifications = [], isLoading } = useNotificationsQuery();
  const markReadMutation = useMarkNotificationReadMutation();

  const items = useMemo(
    () =>
      notifications.map((item) => ({
        id: item.id,
        title: item.title,
        detail: item.body,
        href: item.link,
        unread: !item.readAt,
        time: item.createdAt ? new Date(item.createdAt).toLocaleString() : ""
      })),
    [notifications]
  );

  const markAllRead = () => {
    for (const item of items.filter((candidate) => candidate.unread)) {
      void markReadMutation.mutateAsync(item.id);
    }
  };

  const markRead = (id: string) => {
    void markReadMutation.mutateAsync(id);
  };

  return (
    <SurfaceCard
      eyebrow="Notification Module"
      title="In-app inbox"
      action={
        <button className="rounded-full bg-ink px-4 py-2 text-sm font-medium text-sand" onClick={markAllRead} type="button">
          {t("notifications.markAllRead")}
        </button>
      }
    >
      {isLoading ? <div className="rounded-[24px] bg-sand p-4 text-sm text-ink/65">Loading notifications...</div> : null}
      <div className="space-y-3">
        {items.map((item) => (
          <Link
            key={item.id}
            to={item.href}
            onClick={() => markRead(item.id)}
            className={`block rounded-[24px] p-4 transition ${item.unread ? "bg-[#fff1ed]" : "bg-sand"}`}
          >
            <div className="flex items-start justify-between gap-4">
              <div>
                <p className="font-medium text-ink">{item.title}</p>
                <p className="mt-2 text-sm text-ink/65">{item.detail}</p>
                <p className="mt-2 text-xs uppercase tracking-[0.2em] text-ink/45">{item.unread ? t("notifications.unread") : t("notifications.read")}</p>
              </div>
              <span className="text-xs uppercase tracking-[0.2em] text-ink/45">{item.time}</span>
            </div>
          </Link>
        ))}
      </div>
    </SurfaceCard>
  );
}
