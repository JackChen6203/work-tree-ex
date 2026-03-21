import { Link } from "react-router-dom";
import { SurfaceCard } from "../../components/surface-card";
import { notifications } from "../../lib/mock-data";

export function NotificationsPage() {
  return (
    <SurfaceCard eyebrow="Notification Module" title="In-app inbox">
      <div className="space-y-3">
        {notifications.map((item) => (
          <Link
            key={item.id}
            to={item.href}
            className={`block rounded-[24px] p-4 transition ${item.unread ? "bg-[#fff1ed]" : "bg-sand"}`}
          >
            <div className="flex items-start justify-between gap-4">
              <div>
                <p className="font-medium text-ink">{item.title}</p>
                <p className="mt-2 text-sm text-ink/65">{item.detail}</p>
              </div>
              <span className="text-xs uppercase tracking-[0.2em] text-ink/45">{item.time}</span>
            </div>
          </Link>
        ))}
      </div>
    </SurfaceCard>
  );
}
