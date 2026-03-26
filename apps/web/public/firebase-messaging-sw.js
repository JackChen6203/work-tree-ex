self.addEventListener("push", (event) => {
  let payload = {};
  try {
    payload = event.data ? event.data.json() : {};
  } catch {
    payload = {};
  }

  const notification = payload.notification || payload.data || {};
  const title = notification.title || "Travel Planner";
  const options = {
    body: notification.body || "",
    icon: "/favicon.svg",
    data: {
      link: notification.link || notification.click_action || "/notifications"
    }
  };

  event.waitUntil(self.registration.showNotification(title, options));
});

self.addEventListener("notificationclick", (event) => {
  event.notification.close();
  const link = event.notification.data?.link || "/notifications";

  event.waitUntil(
    self.clients.matchAll({ type: "window", includeUncontrolled: true }).then((clients) => {
      for (const client of clients) {
        if ("focus" in client && client.url.includes(self.location.origin)) {
          client.navigate(link);
          return client.focus();
        }
      }

      if (self.clients.openWindow) {
        return self.clients.openWindow(link);
      }

      return undefined;
    })
  );
});
