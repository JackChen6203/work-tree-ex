import { createBrowserRouter } from "react-router-dom";
import { AppShell } from "./app-shell";
import { PublicOnlyGate } from "./public-only-gate";
import { SessionGate } from "./session-gate";
import { AuthPage } from "../features/auth/auth-page";
import { WelcomePage } from "../features/auth/welcome-page";
import { DashboardPage } from "../features/dashboard/dashboard-page";
import { TripOverviewPage } from "../features/trip/trip-overview-page";
import { ItineraryPage } from "../features/itinerary/itinerary-page";
import { BudgetPage } from "../features/budget/budget-page";
import { MapPage } from "../features/map/map-page";
import { AiPlannerPage } from "../features/ai-planner/ai-planner-page";
import { NotificationsPage } from "../features/notifications/notifications-page";
import { SettingsPage } from "../features/settings/settings-page";

export const router = createBrowserRouter([
  {
    path: "/welcome",
    element: (
      <PublicOnlyGate>
        <WelcomePage />
      </PublicOnlyGate>
    )
  },
  {
    path: "/login",
    element: (
      <PublicOnlyGate>
        <AuthPage />
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
      { index: true, element: <DashboardPage /> },
      { path: "trips/:tripId", element: <TripOverviewPage /> },
      { path: "trips/:tripId/itinerary", element: <ItineraryPage /> },
      { path: "trips/:tripId/budget", element: <BudgetPage /> },
      { path: "trips/:tripId/map", element: <MapPage /> },
      { path: "trips/:tripId/ai-planner", element: <AiPlannerPage /> },
      { path: "notifications", element: <NotificationsPage /> },
      { path: "settings", element: <SettingsPage /> }
    ]
  }
]);
