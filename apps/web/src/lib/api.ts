const inferredBaseUrl =
  typeof window === "undefined" || window.location.port === "5173" ? "http://localhost:8080" : "";

export const apiBaseUrl = (import.meta.env.VITE_API_BASE_URL ?? inferredBaseUrl).replace(/\/$/, "");

export interface ApiErrorPayload {
  error?: {
    code?: string;
    message?: string;
    details?: unknown;
    requestId?: string;
  };
}

// ---------- Token refresh lock ----------
let refreshPromise: Promise<boolean> | null = null;

async function tryRefreshToken(): Promise<boolean> {
  if (refreshPromise) return refreshPromise;

  refreshPromise = (async () => {
    try {
      const res = await fetch(`${apiBaseUrl}/api/v1/auth/refresh`, {
        method: "POST",
        credentials: "include"
      });
      return res.ok;
    } catch {
      return false;
    } finally {
      refreshPromise = null;
    }
  })();

  return refreshPromise;
}

function handleSessionExpired() {
  // Lazy import to avoid circular dependency
  const currentPath = window.location.pathname + window.location.search;
  localStorage.setItem("redirect_after_login", currentPath);

  // Clear session and redirect — uses dynamic import to avoid circular deps
  import("../store/session-store").then(({ useSessionStore }) => {
    useSessionStore.getState().clearUser();
  });

  window.location.href = "/login";
}

// ---------- Invite context persistence ----------
const INVITE_CONTEXT_KEY = "invite_context";

export function saveInviteContext(token: string, redirectTo: string) {
  localStorage.setItem(INVITE_CONTEXT_KEY, JSON.stringify({ token, redirectTo, savedAt: Date.now() }));
}

export function consumeInviteContext(): { token: string; redirectTo: string } | null {
  const raw = localStorage.getItem(INVITE_CONTEXT_KEY);
  if (!raw) return null;
  localStorage.removeItem(INVITE_CONTEXT_KEY);
  try {
    const parsed = JSON.parse(raw) as { token: string; redirectTo: string; savedAt: number };
    // Expire after 1 hour
    if (Date.now() - parsed.savedAt > 3600_000) return null;
    return parsed;
  } catch {
    return null;
  }
}

// ---------- Core API request ----------
export async function apiRequest<T>(path: string, init?: RequestInit): Promise<T> {
  const doFetch = () =>
    fetch(`${apiBaseUrl}${path}`, {
      ...init,
      credentials: init?.credentials ?? "include",
      headers: {
        "Content-Type": "application/json",
        ...(init?.headers ?? {})
      }
    });

  let response = await doFetch();

  // On 401, attempt silent token refresh then retry once
  if (response.status === 401) {
    const refreshed = await tryRefreshToken();
    if (refreshed) {
      response = await doFetch();
    } else {
      handleSessionExpired();
      throw new Error("Session expired");
    }
  }

  if (!response.ok) {
    const payload = (await response.json().catch(() => null)) as ApiErrorPayload | null;
    const code = payload?.error?.code ?? "";
    const message = payload?.error?.message ?? `Request failed with status ${response.status}`;
    const error = new Error(message) as Error & { status: number; code: string };
    error.status = response.status;
    error.code = code;
    throw error;
  }

  const payload = (await response.json()) as { data: T };
  return payload.data;
}

