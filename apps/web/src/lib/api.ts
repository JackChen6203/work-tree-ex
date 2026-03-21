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

export async function apiRequest<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(`${apiBaseUrl}${path}`, {
    ...init,
    headers: {
      "Content-Type": "application/json",
      ...(init?.headers ?? {})
    }
  });

  if (!response.ok) {
    const payload = (await response.json().catch(() => null)) as ApiErrorPayload | null;
    throw new Error(payload?.error?.message ?? `Request failed with status ${response.status}`);
  }

  const payload = (await response.json()) as { data: T };
  return payload.data;
}
