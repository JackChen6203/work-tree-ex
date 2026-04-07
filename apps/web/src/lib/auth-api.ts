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

interface RefreshTokenResponse {
  accessToken: string;
  expiresAt: number;
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

export function refreshAccessToken() {
  return apiRequest<RefreshTokenResponse>("/api/v1/auth/refresh", {
    method: "POST"
  });
}

export function oauthStartUrl(provider: string) {
  return `${apiBaseUrl}/api/v1/auth/oauth/${provider}/start`;
}

export function logout() {
  return apiRequest<void>("/api/v1/auth/logout", {
    method: "POST"
  });
}

// ---------- Redirect hint helpers ----------
const REDIRECT_KEY = "redirect_after_login";

export function getRedirectAfterLogin(): string | null {
  return localStorage.getItem(REDIRECT_KEY);
}

export function clearRedirectAfterLogin(): void {
  localStorage.removeItem(REDIRECT_KEY);
}
