import { useNavigate } from "react-router-dom";
import { useNotificationsQuery } from "../lib/queries";

export function NotificationBell() {
  const navigate = useNavigate();
  const { data: unreadNotifications = [] } = useNotificationsQuery(true);
  const unreadCount = unreadNotifications.length;

  return (
    <button
      className="relative flex h-10 w-10 items-center justify-center rounded-full bg-white/80 text-ink transition hover:bg-sand"
      onClick={() => navigate("/notifications")}
      type="button"
      aria-label="Notifications"
    >
      <svg
        className="h-5 w-5 text-ink/70"
        fill="none"
        stroke="currentColor"
        strokeWidth={1.8}
        viewBox="0 0 24 24"
      >
        <path
          strokeLinecap="round"
          strokeLinejoin="round"
          d="M14.857 17.082a23.848 23.848 0 0 0 5.454-1.31A8.967 8.967 0 0 1 18 9.75V9A6 6 0 0 0 6 9v.75a8.967 8.967 0 0 1-2.312 6.022c1.733.64 3.56 1.085 5.455 1.31m5.714 0a24.255 24.255 0 0 1-5.714 0m5.714 0a3 3 0 1 1-5.714 0"
        />
      </svg>
      {unreadCount > 0 ? (
        <span className="absolute -right-0.5 -top-0.5 flex h-5 min-w-5 items-center justify-center rounded-full bg-coral px-1 text-[10px] font-bold text-white shadow-sm">
          {unreadCount > 99 ? "99+" : unreadCount}
        </span>
      ) : null}
    </button>
  );
}
