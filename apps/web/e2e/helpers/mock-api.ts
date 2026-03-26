import type { Page, Route } from "@playwright/test";

interface MockTrip {
  id: string;
  name: string;
  destinationText: string;
  startDate: string;
  endDate: string;
  timezone: string;
  currency: string;
  travelersCount: number;
  status: "draft" | "active" | "archived";
  version: number;
  createdAt: string;
  updatedAt: string;
}

interface MockItineraryDay {
  dayId: string;
  date: string;
  sortOrder: number;
  items: Array<{
    id: string;
    dayId: string;
    title: string;
    itemType: string;
    allDay: boolean;
    sortOrder: number;
    version: number;
    note?: string;
  }>;
}

interface MockBudgetProfile {
  tripId: string;
  totalBudget?: number;
  perPersonBudget?: number;
  perDayBudget?: number;
  currency: string;
  categories: Array<{ category: string; plannedAmount: number }>;
  version: number;
  actualSpend: number;
  overBudget: boolean;
  createdAt: string;
  updatedAt: string;
}

interface MockExpense {
  id: string;
  tripId: string;
  category: string;
  amount: number;
  currency: string;
  expenseAt?: string;
  note?: string;
  createdAt: string;
}

interface MockNotification {
  id: string;
  type: string;
  title: string;
  body: string;
  link: string;
  createdAt: string;
  readAt?: string;
}

interface InstallApiMocksOptions {
  authenticated?: boolean;
}

function json(data: unknown, status = 200) {
  return {
    status,
    contentType: "application/json",
    body: JSON.stringify({ data })
  };
}

async function parseBody(route: Route) {
  const raw = route.request().postData() ?? "{}";
  try {
    return JSON.parse(raw) as Record<string, unknown>;
  } catch {
    return {};
  }
}

export async function installApiMocks(page: Page, options: InstallApiMocksOptions = {}) {
  const authenticated = options.authenticated ?? true;
  const now = new Date().toISOString();
  const user = {
    id: "u-e2e",
    name: "E2E User",
    email: "e2e@example.com",
    avatar: "E2"
  };

  const trips: MockTrip[] = [];
  const itineraryByTrip = new Map<string, MockItineraryDay[]>();
  const budgetByTrip = new Map<string, MockBudgetProfile>();
  const expensesByTrip = new Map<string, MockExpense[]>();
  const notifications: MockNotification[] = [
    {
      id: "n-e2e-1",
      type: "ai.plan.succeeded",
      title: "AI draft ready",
      body: "A new draft is ready to review.",
      link: "/notifications",
      createdAt: now
    }
  ];

  await page.route("**/api/v1/**", async (route) => {
    const url = new URL(route.request().url());
    const method = route.request().method();
    const path = url.pathname;

    if (method === "GET" && path === "/api/v1/auth/session") {
      await route.fulfill(json({ user: authenticated ? user : null, roles: authenticated ? ["owner"] : [] }));
      return;
    }

    if (method === "POST" && path === "/api/v1/auth/request-magic-link") {
      await route.fulfill(json({ sent: true, expiresIn: 300, previewCode: "123456" }));
      return;
    }

    if (method === "POST" && path === "/api/v1/auth/refresh") {
      await route.fulfill(json({ accessToken: "token", expiresAt: Date.now() + 60_000 }));
      return;
    }

    if (method === "GET" && path === "/api/v1/trips") {
      await route.fulfill(json(trips));
      return;
    }

    if (method === "POST" && path === "/api/v1/trips") {
      const body = await parseBody(route);
      const created: MockTrip = {
        id: "trip-e2e",
        name: String(body.name ?? "E2E Trip"),
        destinationText: String(body.destinationText ?? "Tokyo"),
        startDate: String(body.startDate ?? "2026-06-01"),
        endDate: String(body.endDate ?? "2026-06-03"),
        timezone: String(body.timezone ?? "Asia/Tokyo"),
        currency: String(body.currency ?? "JPY"),
        travelersCount: Number(body.travelersCount ?? 2),
        status: "draft",
        version: 1,
        createdAt: now,
        updatedAt: now
      };
      trips.push(created);
      itineraryByTrip.set(created.id, [
        {
          dayId: "day-1",
          date: created.startDate,
          sortOrder: 1,
          items: []
        }
      ]);
      budgetByTrip.set(created.id, {
        tripId: created.id,
        totalBudget: 0,
        perPersonBudget: 0,
        perDayBudget: 0,
        currency: created.currency,
        categories: [
          { category: "lodging", plannedAmount: 0 },
          { category: "transit", plannedAmount: 0 },
          { category: "food", plannedAmount: 0 },
          { category: "attraction", plannedAmount: 0 },
          { category: "shopping", plannedAmount: 0 },
          { category: "misc", plannedAmount: 0 }
        ],
        version: 1,
        actualSpend: 0,
        overBudget: false,
        createdAt: now,
        updatedAt: now
      });
      expensesByTrip.set(created.id, []);
      await route.fulfill(json(created, 201));
      return;
    }

    if (method === "GET" && /^\/api\/v1\/trips\/[^/]+$/.test(path)) {
      const tripId = path.split("/")[4];
      const trip = trips.find((item) => item.id === tripId);
      if (!trip) {
        await route.fulfill(json({ message: "not found" }, 404));
        return;
      }
      await route.fulfill(json(trip));
      return;
    }

    if (method === "GET" && /^\/api\/v1\/trips\/[^/]+\/days$/.test(path)) {
      const tripId = path.split("/")[4];
      await route.fulfill(json(itineraryByTrip.get(tripId) ?? []));
      return;
    }

    if (method === "POST" && /^\/api\/v1\/trips\/[^/]+\/items$/.test(path)) {
      const tripId = path.split("/")[4];
      const body = await parseBody(route);
      const days = itineraryByTrip.get(tripId) ?? [];
      const dayId = String(body.dayId ?? days[0]?.dayId ?? "day-1");
      const targetDay = days.find((item) => item.dayId === dayId);
      if (!targetDay) {
        await route.fulfill(json({ message: "day not found" }, 404));
        return;
      }

      const createdItem = {
        id: `item-${targetDay.items.length + 1}`,
        dayId,
        title: String(body.title ?? "New item"),
        itemType: String(body.itemType ?? "custom"),
        allDay: Boolean(body.allDay ?? false),
        sortOrder: targetDay.items.length + 1,
        version: 1,
        note: String(body.note ?? "")
      };
      targetDay.items.push(createdItem);
      await route.fulfill(json(createdItem, 201));
      return;
    }

    if (method === "GET" && /^\/api\/v1\/trips\/[^/]+\/budget$/.test(path)) {
      const tripId = path.split("/")[4];
      const budget = budgetByTrip.get(tripId);
      if (!budget) {
        await route.fulfill(json({ message: "not found" }, 404));
        return;
      }
      await route.fulfill(json(budget));
      return;
    }

    if (method === "PUT" && /^\/api\/v1\/trips\/[^/]+\/budget$/.test(path)) {
      const tripId = path.split("/")[4];
      const body = await parseBody(route);
      const existing = budgetByTrip.get(tripId);
      if (!existing) {
        await route.fulfill(json({ message: "not found" }, 404));
        return;
      }
      const next = {
        ...existing,
        totalBudget: Number(body.totalBudget ?? existing.totalBudget ?? 0),
        perPersonBudget: Number(body.perPersonBudget ?? existing.perPersonBudget ?? 0),
        perDayBudget: Number(body.perDayBudget ?? existing.perDayBudget ?? 0),
        currency: String(body.currency ?? existing.currency),
        categories: Array.isArray(body.categories) ? body.categories as Array<{ category: string; plannedAmount: number }> : existing.categories,
        updatedAt: new Date().toISOString()
      };
      budgetByTrip.set(tripId, next);
      await route.fulfill(json(next));
      return;
    }

    if (method === "GET" && /^\/api\/v1\/trips\/[^/]+\/expenses$/.test(path)) {
      const tripId = path.split("/")[4];
      await route.fulfill(json(expensesByTrip.get(tripId) ?? []));
      return;
    }

    if (method === "POST" && /^\/api\/v1\/trips\/[^/]+\/expenses$/.test(path)) {
      const tripId = path.split("/")[4];
      const body = await parseBody(route);
      const expenses = expensesByTrip.get(tripId) ?? [];
      const created: MockExpense = {
        id: `expense-${expenses.length + 1}`,
        tripId,
        category: String(body.category ?? "food"),
        amount: Number(body.amount ?? 0),
        currency: String(body.currency ?? "JPY"),
        note: String(body.note ?? ""),
        expenseAt: body.expenseAt ? String(body.expenseAt) : undefined,
        createdAt: new Date().toISOString()
      };
      expenses.push(created);
      expensesByTrip.set(tripId, expenses);
      await route.fulfill(json(created, 201));
      return;
    }

    if (method === "GET" && path === "/api/v1/maps/search") {
      const q = url.searchParams.get("q") ?? "Kyoto";
      await route.fulfill(
        json([
          {
            providerPlaceId: "place-1",
            name: q,
            address: `${q} Station`,
            lat: 35.0116,
            lng: 135.7681,
            categories: ["city"]
          }
        ])
      );
      return;
    }

    if (method === "GET" && path === "/api/v1/notifications") {
      const unreadOnly = url.searchParams.get("unreadOnly") === "true";
      const filtered = unreadOnly ? notifications.filter((item) => !item.readAt) : notifications;
      await route.fulfill(json(filtered));
      return;
    }

    if (method === "POST" && /^\/api\/v1\/notifications\/[^/]+\/read$/.test(path)) {
      const notificationId = path.split("/")[4];
      const target = notifications.find((item) => item.id === notificationId);
      if (target) {
        target.readAt = new Date().toISOString();
      }
      await route.fulfill({ status: 204, body: "" });
      return;
    }

    if (method === "POST" && path === "/api/v1/notifications/read-all") {
      for (const item of notifications) {
        item.readAt = new Date().toISOString();
      }
      await route.fulfill({ status: 204, body: "" });
      return;
    }

    if (method === "POST" && path === "/api/v1/notifications/cleanup-read") {
      const before = notifications.length;
      for (let i = notifications.length - 1; i >= 0; i -= 1) {
        if (notifications[i].readAt) {
          notifications.splice(i, 1);
        }
      }
      await route.fulfill(json({ deletedCount: before - notifications.length }));
      return;
    }

    if (method === "POST" && path === "/api/v1/fcm-tokens") {
      await route.fulfill(json(null, 201));
      return;
    }

    if (method === "GET" && path === "/api/v1/sync/bootstrap") {
      await route.fulfill(
        json({
          serverTime: new Date().toISOString(),
          sinceVersion: 0,
          tripId: trips[0]?.id ?? "",
          fullResyncRequired: false,
          changedTrips: [],
          changedDays: [],
          changedNotifications: []
        })
      );
      return;
    }

    if (method === "POST" && path === "/api/v1/sync/mutations/flush") {
      await route.fulfill(
        json({
          tripId: trips[0]?.id ?? "",
          acceptedCount: 1,
          conflictCount: 0,
          conflicts: [],
          nextVersion: 1,
          serverTime: new Date().toISOString()
        })
      );
      return;
    }

    if (path.startsWith("/api/v1/users/")) {
      if (path === "/api/v1/users/me" && method === "GET") {
        await route.fulfill(
          json({
            id: user.id,
            email: user.email,
            displayName: user.name,
            locale: "en",
            timezone: "Asia/Taipei",
            currency: "JPY"
          })
        );
        return;
      }

      if (path === "/api/v1/users/preferences" && method === "GET") {
        await route.fulfill(
          json({
            tripPace: "balanced",
            wakePattern: "normal",
            transportPreference: "transit",
            foodPreference: [],
            avoidTags: []
          })
        );
        return;
      }

      if (path === "/api/v1/users/notification-preferences" && method === "GET") {
        await route.fulfill(
          json({
            pushEnabled: false,
            emailEnabled: false,
            digestFrequency: "daily",
            quietHoursStart: "22:00",
            quietHoursEnd: "07:00",
            tripUpdates: true,
            budgetAlerts: true,
            aiPlanReadyAlerts: true
          })
        );
        return;
      }

      if (method === "PUT" || method === "PATCH" || method === "POST" || method === "DELETE") {
        await route.fulfill(json(null));
        return;
      }
    }

    await route.fulfill(json(null));
  });
}
