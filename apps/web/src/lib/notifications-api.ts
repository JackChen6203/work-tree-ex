import { apiRequest } from "./api";

export interface NotificationItemApi {
  id: string;
  type: string;
  title: string;
  body: string;
  link: string;
  tripId?: string;
  readAt?: string;
  createdAt?: string;
}

export interface CleanupReadNotificationsResponse {
  deletedCount: number;
}

export interface FcmTokenInput {
  token: string;
  platform: "web";
  locale?: string;
  timezone?: string;
  userAgent?: string;
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

export function markNotificationRead(notificationId: string) {
  return apiRequest<void>(`/api/v1/notifications/${notificationId}/read`, {
    method: "POST"
  });
}

export function markNotificationUnread(notificationId: string) {
  return apiRequest<void>(`/api/v1/notifications/${notificationId}/unread`, {
    method: "POST"
  });
}

export function markAllNotificationsRead() {
  return apiRequest<void>("/api/v1/notifications/read-all", {
    method: "POST"
  });
}

export function cleanupReadNotifications() {
  return apiRequest<CleanupReadNotificationsResponse>("/api/v1/notifications/cleanup-read", {
    method: "POST"
  });
}

export function deleteNotification(notificationId: string) {
  return apiRequest<void>(`/api/v1/notifications/${notificationId}`, {
    method: "DELETE"
  });
}

export function registerFcmToken(input: FcmTokenInput) {
  return apiRequest<void>("/api/v1/fcm-tokens", {
    method: "POST",
    headers: {
      "Idempotency-Key": crypto.randomUUID()
    },
    body: JSON.stringify(input)
  });
}
