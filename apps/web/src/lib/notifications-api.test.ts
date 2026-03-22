import { beforeEach, describe, expect, it, vi } from "vitest";
import {
  cleanupReadNotifications,
  deleteNotification,
  listNotifications,
  markAllNotificationsRead,
  markNotificationRead,
  markNotificationUnread
} from "./notifications-api";

describe("notifications api", () => {
  beforeEach(() => {
    vi.restoreAllMocks();
  });

  it("builds list endpoint with optional unreadOnly query", async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce({ ok: true, json: async () => ({ data: [] }) })
      .mockResolvedValueOnce({ ok: true, json: async () => ({ data: [] }) });
    vi.stubGlobal("fetch", fetchMock);

    await listNotifications(false);
    await listNotifications(true);

    expect(fetchMock).toHaveBeenNthCalledWith(1, "http://localhost:8080/api/v1/notifications", expect.anything());
    expect(fetchMock).toHaveBeenNthCalledWith(2, "http://localhost:8080/api/v1/notifications?unreadOnly=true", expect.anything());
  });

  it("posts and deletes notification actions", async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce({ ok: true, json: async () => ({ data: null }) })
      .mockResolvedValueOnce({ ok: true })
      .mockResolvedValueOnce({ ok: true })
      .mockResolvedValueOnce({ ok: true })
      .mockResolvedValueOnce({ ok: true, json: async () => ({ data: { deletedCount: 2 } }) })
      .mockResolvedValueOnce({ ok: true });
    vi.stubGlobal("fetch", fetchMock);

    await listNotifications();
    await markNotificationRead("n-1");
    await markNotificationUnread("n-1");
    await markAllNotificationsRead();
    await cleanupReadNotifications();
    await deleteNotification("n-1");

    expect(fetchMock).toHaveBeenNthCalledWith(
      2,
      "http://localhost:8080/api/v1/notifications/n-1/read",
      expect.objectContaining({ method: "POST" })
    );
    expect(fetchMock).toHaveBeenNthCalledWith(
      3,
      "http://localhost:8080/api/v1/notifications/n-1/unread",
      expect.objectContaining({ method: "POST" })
    );
    expect(fetchMock).toHaveBeenNthCalledWith(
      4,
      "http://localhost:8080/api/v1/notifications/read-all",
      expect.objectContaining({ method: "POST" })
    );
    expect(fetchMock).toHaveBeenNthCalledWith(
      5,
      "http://localhost:8080/api/v1/notifications/cleanup-read",
      expect.objectContaining({ method: "POST" })
    );
    expect(fetchMock).toHaveBeenNthCalledWith(
      6,
      "http://localhost:8080/api/v1/notifications/n-1",
      expect.objectContaining({ method: "DELETE" })
    );
  });
});
