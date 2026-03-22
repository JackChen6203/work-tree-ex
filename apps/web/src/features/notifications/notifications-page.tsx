import { useMemo, useState } from "react";
import { Link } from "react-router-dom";
import { SurfaceCard } from "../../components/surface-card";
import { useI18n } from "../../lib/i18n";
import { useDeleteNotificationMutation, useMarkAllNotificationsReadMutation, useMarkNotificationReadMutation, useNotificationsQuery } from "../../lib/queries";

export function NotificationsPage() {
  const { t } = useI18n();
  const [unreadOnly, setUnreadOnly] = useState(false);
  const { data: notifications = [], isLoading } = useNotificationsQuery(unreadOnly);
  const markReadMutation = useMarkNotificationReadMutation();
  const markAllReadMutation = useMarkAllNotificationsReadMutation();
  const deleteMutation = useDeleteNotificationMutation();

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
    void markAllReadMutation.mutateAsync();
  };

  const markRead = (id: string) => {
    void markReadMutation.mutateAsync(id);
  };

  const removeItem = (id: string) => {
    void deleteMutation.mutateAsync(id);
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
      <div className="mb-3 flex items-center justify-end">
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
      {isLoading ? <div className="rounded-[24px] bg-sand p-4 text-sm text-ink/65">Loading notifications...</div> : null}
      {!isLoading && items.length === 0 ? <div className="rounded-[24px] bg-sand p-4 text-sm text-ink/65">No notifications found.</div> : null}
      <div className="space-y-3">
        {items.map((item) => (
          <div key={item.id} className={`rounded-[24px] p-4 transition ${item.unread ? "bg-[#fff1ed]" : "bg-sand"}`}>
            <div className="flex items-start justify-between gap-4">
              <div>
                <Link className="font-medium text-ink underline-offset-4 hover:underline" onClick={() => markRead(item.id)} to={item.href}>
                  {item.title}
                </Link>
                <p className="mt-2 text-sm text-ink/65">{item.detail}</p>
                <p className="mt-2 text-xs uppercase tracking-[0.2em] text-ink/45">{item.unread ? t("notifications.unread") : t("notifications.read")}</p>
              </div>
              <div className="flex flex-col items-end gap-3">
                <span className="text-xs uppercase tracking-[0.2em] text-ink/45">{item.time}</span>
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
