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

export function listNotifications() {
  return apiRequest<NotificationItemApi[]>("/api/v1/notifications");
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
