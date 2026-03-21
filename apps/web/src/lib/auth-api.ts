import { apiBaseUrl, apiRequest } from "./api";
import type { SessionUser } from "../types/domain";

interface RequestMagicLinkResponse {
  sent: boolean;
  expiresIn: number;
  previewCode?: string;
}

interface VerifyMagicLinkResponse {
  user: SessionUser;
  roles: string[];
}

interface SessionResponse {
  user: SessionUser | null;
  roles: string[];
}

export function requestMagicLink(email: string) {
  return apiRequest<RequestMagicLinkResponse>("/api/v1/auth/request-magic-link", {
    method: "POST",
    body: JSON.stringify({ email })
  });
}

export function verifyMagicLink(email: string, code: string) {
  return apiRequest<VerifyMagicLinkResponse>("/api/v1/auth/verify-magic-link", {
    method: "POST",
    body: JSON.stringify({ email, code })
  });
}

export function getSession() {
  return apiRequest<SessionResponse>("/api/v1/auth/session");
}

export function oauthStartUrl(provider: string) {
  return `${apiBaseUrl}/api/v1/auth/oauth/${provider}/start`;
}

export async function logout() {
  const response = await fetch(`${apiBaseUrl}/api/v1/auth/logout`, {
    method: "POST",
    credentials: "include"
  });

  if (!response.ok) {
    throw new Error(`Logout failed with status ${response.status}`);
  }
}
