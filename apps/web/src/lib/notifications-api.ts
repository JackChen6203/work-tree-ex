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

export interface CleanupReadNotificationsResponse {
  deletedCount: number;
}

export interface ListNotificationsOptions {
  unreadOnly?: boolean;
  cursor?: string;
  limit?: number;
}

export function listNotifications(options: ListNotificationsOptions = {}) {
  const params = new URLSearchParams();
  if (options.unreadOnly) {
    params.set("unreadOnly", "true");
  }
  if (options.cursor) {
    params.set("cursor", options.cursor);
  }
  if (typeof options.limit === "number") {
    params.set("limit", String(options.limit));
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

export function cleanupReadNotifications() {
  return apiRequest<CleanupReadNotificationsResponse>("/api/v1/notifications/cleanup-read", {
    method: "POST"
  });
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
