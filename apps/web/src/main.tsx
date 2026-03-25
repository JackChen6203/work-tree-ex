import React from "react";
import ReactDOM from "react-dom/client";
import { QueryClientProvider } from "@tanstack/react-query";
import { RouterProvider } from "react-router-dom";
import { GlobalErrorBoundary } from "./app/global-error-boundary";
import { SessionSyncBridge } from "./app/session-sync-bridge";
import { queryClient } from "./lib/query-client";
import { I18nProvider } from "./lib/i18n";
import { router } from "./app/router";
import "./styles.css";

if (typeof window !== "undefined" && "serviceWorker" in navigator) {
  void navigator.serviceWorker.getRegistrations().then(async (registrations) => {
    if (registrations.length === 0) {
      return;
    }

    await Promise.all(registrations.map((registration) => registration.unregister()));
    if (navigator.serviceWorker.controller) {
      window.location.reload();
    }
  });
}

ReactDOM.createRoot(document.getElementById("root")!).render(
  <React.StrictMode>
    <I18nProvider>
      <QueryClientProvider client={queryClient}>
        <GlobalErrorBoundary>
          <SessionSyncBridge />
          <RouterProvider router={router} />
        </GlobalErrorBoundary>
      </QueryClientProvider>
    </I18nProvider>
  </React.StrictMode>
);
