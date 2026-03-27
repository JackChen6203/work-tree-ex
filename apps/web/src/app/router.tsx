import { lazy, Suspense } from "react";
import { createBrowserRouter } from "react-router-dom";
import { AppShell } from "./app-shell";
import { PublicOnlyGate } from "./public-only-gate";
import { SessionGate } from "./session-gate";
import { OAuthBridgePage } from "./oauth-bridge-page";

const AuthPage = lazy(async () => {
  const module = await import("../features/auth/auth-page");
  return { default: module.AuthPage };
});

const WelcomePage = lazy(async () => {
  const module = await import("../features/auth/welcome-page");
  return { default: module.WelcomePage };
});

const DashboardPage = lazy(async () => {
  const module = await import("../features/dashboard/dashboard-page");
  return { default: module.DashboardPage };
});

const TripOverviewPage = lazy(async () => {
  const module = await import("../features/trip/trip-overview-page");
  return { default: module.TripOverviewPage };
});

const ItineraryPage = lazy(async () => {
  const module = await import("../features/itinerary/itinerary-page");
  return { default: module.ItineraryPage };
});

const BudgetPage = lazy(async () => {
  const module = await import("../features/budget/budget-page");
  return { default: module.BudgetPage };
});

const MapPage = lazy(async () => {
  const module = await import("../features/map/map-page");
  return { default: module.MapPage };
});

const AiPlannerPage = lazy(async () => {
  const module = await import("../features/ai-planner/ai-planner-page");
  return { default: module.AiPlannerPage };
});

const NotificationsPage = lazy(async () => {
  const module = await import("../features/notifications/notifications-page");
  return { default: module.NotificationsPage };
});

const SettingsPage = lazy(async () => {
  const module = await import("../features/settings/settings-page");
  return { default: module.SettingsPage };
});

function RouteFallback() {
  return <div className="rounded-[24px] bg-sand/70 p-5 text-sm text-ink/65">Loading...</div>;
}

function withSuspense(element: JSX.Element) {
  return <Suspense fallback={<RouteFallback />}>{element}</Suspense>;
}

export const router = createBrowserRouter([
  {
    path: "/api/v1/auth/oauth/:provider/start",
    element: <OAuthBridgePage />
  },
  {
    path: "/api/v1/auth/oauth/:provider/callback",
    element: <OAuthBridgePage />
  },
  {
    path: "/welcome",
    element: (
      <PublicOnlyGate>
        {withSuspense(<WelcomePage />)}
      </PublicOnlyGate>
    )
  },
  {
    path: "/login",
    element: (
      <PublicOnlyGate>
        {withSuspense(<AuthPage />)}
      </PublicOnlyGate>
    )
  },
  {
    path: "/",
    element: (
      <SessionGate>
        <AppShell />
      </SessionGate>
    ),
    children: [
      { index: true, element: withSuspense(<DashboardPage />) },
      { path: "trips/:tripId", element: withSuspense(<TripOverviewPage />) },
      { path: "trips/:tripId/itinerary", element: withSuspense(<ItineraryPage />) },
      { path: "trips/:tripId/budget", element: withSuspense(<BudgetPage />) },
      { path: "trips/:tripId/map", element: withSuspense(<MapPage />) },
      { path: "trips/:tripId/ai-planner", element: withSuspense(<AiPlannerPage />) },
      { path: "notifications", element: withSuspense(<NotificationsPage />) },
      { path: "settings", element: withSuspense(<SettingsPage />) }
    ]
  }
]);

