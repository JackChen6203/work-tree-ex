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

const csrfCookieName = "tt_csrf";
const csrfHeaderName = "X-CSRF-Token";
const csrfStorageKey = "tt.csrf-token";
const csrfCookieMaxAgeSeconds = 24 * 60 * 60;
const unsafeMethods = new Set(["POST", "PUT", "PATCH", "DELETE"]);

function normalizeHeaders(headers?: HeadersInit): Record<string, string> {
  const normalized: Record<string, string> = {};
  if (!headers) return normalized;

  if (headers instanceof Headers) {
    headers.forEach((value, key) => {
      normalized[key] = value;
    });
    return normalized;
  }

  if (Array.isArray(headers)) {
    for (const [key, value] of headers) {
      normalized[key] = value;
    }
    return normalized;
  }

  return { ...headers };
}

function hasHeader(headers: Record<string, string>, name: string) {
  return Object.keys(headers).some((key) => key.toLowerCase() === name.toLowerCase());
}

function setHeader(headers: Record<string, string>, name: string, value: string) {
  const existing = Object.keys(headers).find((key) => key.toLowerCase() === name.toLowerCase());
  headers[existing ?? name] = value;
}

function isFormDataBody(body: BodyInit | null | undefined) {
  return typeof FormData !== "undefined" && body instanceof FormData;
}

function isUnsafeRequest(method?: string) {
  return unsafeMethods.has((method ?? "GET").toUpperCase());
}

function readCsrfCookie() {
  if (typeof document === "undefined") return "";

  const prefix = `${csrfCookieName}=`;
  const rawCookie = document.cookie
    .split(";")
    .map((part) => part.trim())
    .find((part) => part.startsWith(prefix));
  if (!rawCookie) return "";

  try {
    return decodeURIComponent(rawCookie.slice(prefix.length));
  } catch {
    return "";
  }
}

function writeCsrfCookie(token: string) {
  if (typeof document === "undefined") return;

  const secure = typeof window !== "undefined" && window.location.protocol === "https:" ? "; Secure" : "";
  document.cookie = `${csrfCookieName}=${encodeURIComponent(token)}; Path=/; SameSite=Lax; Max-Age=${csrfCookieMaxAgeSeconds}${secure}`;
}

function readStoredCsrfToken() {
  if (typeof sessionStorage === "undefined") return "";
  try {
    return sessionStorage.getItem(csrfStorageKey) ?? "";
  } catch {
    return "";
  }
}

function writeStoredCsrfToken(token: string) {
  if (typeof sessionStorage === "undefined") return;
  try {
    sessionStorage.setItem(csrfStorageKey, token);
  } catch {
    // Storage may be unavailable in private or embedded contexts.
  }
}

function createCsrfToken() {
  const webCrypto = globalThis.crypto;
  if (typeof webCrypto?.randomUUID === "function") {
    return webCrypto.randomUUID();
  }

  if (typeof webCrypto?.getRandomValues === "function") {
    const bytes = new Uint8Array(16);
    webCrypto.getRandomValues(bytes);
    return Array.from(bytes, (byte) => byte.toString(16).padStart(2, "0")).join("");
  }

  return `${Date.now().toString(36)}-${Math.random().toString(36).slice(2)}`;
}

export function getCsrfToken() {
  const cookieToken = readCsrfCookie();
  if (cookieToken) {
    writeStoredCsrfToken(cookieToken);
    return cookieToken;
  }

  const token = readStoredCsrfToken() || createCsrfToken();
  writeStoredCsrfToken(token);
  writeCsrfCookie(token);
  return token;
}

function buildRequestInit(init?: RequestInit): RequestInit {
  const headers = normalizeHeaders(init?.headers);
  const credentials = init?.credentials ?? "include";

  if (!hasHeader(headers, "Content-Type") && !isFormDataBody(init?.body)) {
    setHeader(headers, "Content-Type", "application/json");
  }
  if (credentials !== "omit" && isUnsafeRequest(init?.method) && !hasHeader(headers, csrfHeaderName)) {
    setHeader(headers, csrfHeaderName, getCsrfToken());
  }

  return {
    ...init,
    credentials,
    headers
  };
}

function buildApiUrl(pathOrUrl: string) {
  if (/^https?:\/\//i.test(pathOrUrl)) {
    return pathOrUrl;
  }
  return `${apiBaseUrl}${pathOrUrl}`;
}

export function apiFetch(pathOrUrl: string, init?: RequestInit) {
  return fetch(buildApiUrl(pathOrUrl), buildRequestInit(init));
}

// ---------- Token refresh lock ----------
let refreshPromise: Promise<boolean> | null = null;

async function tryRefreshToken(): Promise<boolean> {
  if (refreshPromise) return refreshPromise;

  refreshPromise = (async () => {
    try {
      const res = await apiFetch("/api/v1/auth/refresh", { method: "POST" });
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
  const doFetch = () => apiFetch(path, init);

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

  if (response.status === 204 || typeof response.json !== "function") {
    return undefined as T;
  }

  const payload = (await response.json().catch(() => null)) as { data?: T } | null;
  return payload?.data as T;
}
