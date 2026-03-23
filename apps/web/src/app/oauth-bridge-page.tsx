import { useEffect, useState } from "react";
import { useLocation } from "react-router-dom";
import { apiBaseUrl } from "../lib/api";

interface RedirectPayload {
  data?: {
    redirectTo?: string;
  };
}

function buildBridgeUrl(pathname: string, search: string) {
  const params = new URLSearchParams(search);
  params.set("transport", "json");
  const query = params.toString();
  return `${apiBaseUrl}${pathname}${query ? `?${query}` : ""}`;
}

export function OAuthBridgePage() {
  const location = useLocation();
  const [message, setMessage] = useState("Redirecting authentication...");

  useEffect(() => {
    const controller = new AbortController();

    void (async () => {
      try {
        const response = await fetch(buildBridgeUrl(location.pathname, location.search), {
          credentials: "include",
          signal: controller.signal
        });
        if (!response.ok) {
          throw new Error(`Bridge request failed with status ${response.status}`);
        }

        const payload = (await response.json()) as RedirectPayload;
        const redirectTo = payload.data?.redirectTo;
        if (!redirectTo) {
          throw new Error("Bridge response did not include a redirect target");
        }

        window.location.replace(redirectTo);
      } catch (error) {
        if (controller.signal.aborted) {
          return;
        }

        setMessage(error instanceof Error ? error.message : "Authentication redirect failed.");
      }
    })();

    return () => controller.abort();
  }, [location.pathname, location.search]);

  return (
    <div className="flex min-h-screen items-center justify-center px-4 py-10">
      <div className="rounded-[24px] border border-ink/10 bg-white px-6 py-5 text-sm text-ink/70 shadow-[0_24px_80px_rgba(15,23,42,0.08)]">
        {message}
      </div>
    </div>
  );
}
