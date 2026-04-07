import { expect, test } from "@playwright/test";
import { installApiMocks } from "./helpers/mock-api";

test("login magic link flow", async ({ page }) => {
  await installApiMocks(page, { authenticated: false });

  await page.goto("/login");
  await page.getByLabel(/Email/i).fill("e2e@example.com");
  await page.getByRole("button", { name: /Send sign-in link|寄送登入連結/i }).click();

  await expect(page.getByText(/Check your sign-in link|檢查你的登入連結/i)).toBeVisible();
});

test("trip create -> itinerary edit -> budget setup flow", async ({ page }) => {
  await installApiMocks(page, { authenticated: true });

  await page.goto("/");
  await page.getByRole("button", { name: /Create trip|建立旅程/i }).click();

  await page.locator('input[name="name"]').fill("E2E Kyoto Trip");
  await page.locator('input[name="departureText"]').fill("Taipei");
  await page.locator('input[name="destinationText"]').fill("Kyoto");
  await page.getByRole("button", { name: /Next|下一步/i }).click();

  await page.locator('input[name="startDate"]').fill("2026-07-01");
  await page.locator('input[name="endDate"]').fill("2026-07-03");
  await page.locator('select[name="timezone"]').selectOption("Asia/Tokyo");
  await page.locator('input[name="travelersCount"]').fill("2");
  await page.getByRole("button", { name: /Next|下一步/i }).click();

  await page.locator('select[name="currency"]').selectOption("JPY");
  await page.locator('input[name="totalBudget"]').fill("120000");
  await page.getByRole("button", { name: /Submit|送出/i }).click();

  await expect(page).toHaveURL(/\/trips\/trip-e2e$/);

  await page.goto("/trips/trip-e2e/itinerary");
  await page.getByRole("button", { name: /Add item|新增行程項目/i }).click();
  await expect(page.getByRole("button", { name: /Edit|編輯/i }).first()).toBeVisible();

  await page.goto("/trips/trip-e2e/budget");
  await page.locator('input[name="totalBudget"]').fill("100000");
  await page.getByRole("button", { name: /Save budget|儲存預算/i }).click();

  await page.locator('input[name="amount"]').fill("3200");
  await page.getByRole("button", { name: /Add expense|新增支出/i }).click();
  await expect(page.getByText(/3,200/).first()).toBeVisible();
});

test("offline to online connection toasts", async ({ page }) => {
  await installApiMocks(page, { authenticated: true });

  await page.goto("/");
  await page.evaluate(() => {
    window.dispatchEvent(new Event("offline"));
  });
  await expect(page.getByText(/Offline mode|已切換離線模式/i).first()).toBeVisible();

  await page.evaluate(() => {
    window.dispatchEvent(new Event("online"));
  });
  await expect(page.getByText(/Connection restored|已恢復連線/i).first()).toBeVisible();
});

test("push notification mock (FCM foreground)", async ({ page }) => {
  await installApiMocks(page, { authenticated: true });

  await page.goto("/");
  await expect(page.getByRole("button", { name: /Notifications/i })).toBeVisible();
  await page.waitForTimeout(300);
  await page.evaluate(() => {
    window.dispatchEvent(
      new CustomEvent("mock-fcm-message", {
        detail: {
          title: "E2E Push",
          body: "Foreground mock message"
        }
      })
    );
  });

  await expect(page.getByText(/E2E Push/)).toBeVisible();
});

test("responsive screenshots (mobile and desktop)", async ({ page }) => {
  await installApiMocks(page, { authenticated: true });

  await page.setViewportSize({ width: 390, height: 844 });
  await page.goto("/");
  const mobileShot = await page.screenshot({ fullPage: true });
  expect(mobileShot.byteLength).toBeGreaterThan(10_000);

  await page.setViewportSize({ width: 1440, height: 900 });
  await page.reload();
  const desktopShot = await page.screenshot({ fullPage: true });
  expect(desktopShot.byteLength).toBeGreaterThan(10_000);
});
