import { useMemo, useState } from "react";
import { Link } from "react-router-dom";
import { SurfaceCard } from "../../components/surface-card";
import { useI18n } from "../../lib/i18n";
import { notifications } from "../../lib/mock-data";

export function NotificationsPage() {
  const { t } = useI18n();
  const [readIds, setReadIds] = useState<string[]>([]);

  const items = useMemo(
    () =>
      notifications.map((item) => ({
        ...item,
        unread: item.unread && !readIds.includes(item.id)
      })),
    [readIds]
  );

  const markAllRead = () => {
    setReadIds(notifications.map((item) => item.id));
  };

  const markRead = (id: string) => {
    setReadIds((current) => (current.includes(id) ? current : [...current, id]));
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
