import { apiBaseUrl, apiRequest } from "./api";

export interface NotificationItemApi {
  id: string;
  type: string;
  title: string;
  body: string;
  link: string;
  readAt?: string;
  createdAt?: string;
}

export function listNotifications(unreadOnly = false) {
  const params = new URLSearchParams();
  if (unreadOnly) {
    params.set("unreadOnly", "true");
  }

  return apiRequest<NotificationItemApi[]>(`/api/v1/notifications${params.toString() ? `?${params.toString()}` : ""}`);
}

export async function markNotificationRead(notificationId: string) {
  const response = await fetch(`${apiBaseUrl}/api/v1/notifications/${notificationId}/read`, {
    method: "POST",
    credentials: "include"
  });

  if (!response.ok) {
    throw new Error(`Mark read failed with status ${response.status}`);
  }
}

export async function markNotificationUnread(notificationId: string) {
  const response = await fetch(`${apiBaseUrl}/api/v1/notifications/${notificationId}/unread`, {
    method: "POST",
    credentials: "include"
  });

  if (!response.ok) {
    throw new Error(`Mark unread failed with status ${response.status}`);
  }
}

export async function markAllNotificationsRead() {
  const response = await fetch(`${apiBaseUrl}/api/v1/notifications/read-all`, {
    method: "POST",
    credentials: "include"
  });

  if (!response.ok) {
    throw new Error(`Mark all read failed with status ${response.status}`);
  }
}

export async function deleteNotification(notificationId: string) {
  const response = await fetch(`${apiBaseUrl}/api/v1/notifications/${notificationId}`, {
    method: "DELETE",
    credentials: "include"
  });

  if (!response.ok) {
    throw new Error(`Delete notification failed with status ${response.status}`);
  }
}
