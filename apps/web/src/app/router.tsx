import { createBrowserRouter } from "react-router-dom";
import { AppShell } from "./app-shell";
import { SessionGate } from "./session-gate";
import { AuthPage } from "../features/auth/auth-page";
import { DashboardPage } from "../features/dashboard/dashboard-page";
import { TripOverviewPage } from "../features/trip/trip-overview-page";
import { ItineraryPage } from "../features/itinerary/itinerary-page";
import { BudgetPage } from "../features/budget/budget-page";
import { MapPage } from "../features/map/map-page";
import { AiPlannerPage } from "../features/ai-planner/ai-planner-page";
import { NotificationsPage } from "../features/notifications/notifications-page";

export const router = createBrowserRouter([
  {
    path: "/login",
    element: <AuthPage />
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
      { path: "notifications", element: <NotificationsPage /> }
    ]
  }
]);
