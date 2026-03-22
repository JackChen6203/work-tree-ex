import { apiRequest } from "./api";

export interface UserProfile {
  id: string;
  email: string;
  displayName: string;
  locale: string;
  timezone: string;
  currency: string;
}

export interface PatchUserProfileInput {
  displayName?: string;
  locale?: string;
  timezone?: string;
  currency?: string;
}

export interface UserPreferences {
  tripPace: string;
  wakePattern: string;
  transportPreference: string;
  foodPreference: string[];
  avoidTags: string[];
  version: number;
}

export interface UpsertUserPreferencesInput {
  tripPace: string;
  wakePattern: string;
  transportPreference: string;
  foodPreference: string[];
  avoidTags: string[];
}

export interface UserNotificationPreferences {
  pushEnabled: boolean;
  emailEnabled: boolean;
  digestFrequency: "instant" | "daily" | "weekly";
  quietHoursStart: string;
  quietHoursEnd: string;
  tripUpdates: boolean;
  budgetAlerts: boolean;
  aiPlanReadyAlerts: boolean;
  version: number;
}

export interface UpsertUserNotificationPreferencesInput {
  pushEnabled: boolean;
  emailEnabled: boolean;
  digestFrequency: "instant" | "daily" | "weekly";
  quietHoursStart: string;
  quietHoursEnd: string;
  tripUpdates: boolean;
  budgetAlerts: boolean;
  aiPlanReadyAlerts: boolean;
}

export interface LlmProviderConfig {
  id: string;
  provider: string;
  label: string;
  model: string;
  maskedKey: string;
  createdAt: string;
}

export interface CreateLlmProviderInput {
  provider: string;
  label: string;
  model: string;
  encryptedApiKeyEnvelope: string;
}

export function getMyProfile() {
  return apiRequest<UserProfile>("/api/v1/users/me");
}

export function patchMyProfile(input: PatchUserProfileInput) {
  return apiRequest<UserProfile>("/api/v1/users/me", {
    method: "PATCH",
    body: JSON.stringify(input)
  });
}

export function getMyPreferences() {
  return apiRequest<UserPreferences>("/api/v1/users/me/preferences");
}

export function putMyPreferences(input: UpsertUserPreferencesInput) {
  return apiRequest<UserPreferences>("/api/v1/users/me/preferences", {
    method: "PUT",
    body: JSON.stringify(input)
  });
}

export function getMyNotificationPreferences() {
  return apiRequest<UserNotificationPreferences>("/api/v1/users/me/notification-preferences");
}

export function putMyNotificationPreferences(input: UpsertUserNotificationPreferencesInput) {
  return apiRequest<UserNotificationPreferences>("/api/v1/users/me/notification-preferences", {
    method: "PUT",
    body: JSON.stringify(input)
  });
}

export function listMyLlmProviders() {
  return apiRequest<LlmProviderConfig[]>("/api/v1/users/me/llm-providers");
}

export function createMyLlmProvider(input: CreateLlmProviderInput) {
  return apiRequest<LlmProviderConfig>("/api/v1/users/me/llm-providers", {
    method: "POST",
    body: JSON.stringify(input)
  });
}

export function deleteMyLlmProvider(providerId: string) {
  return apiRequest<void>(`/api/v1/users/me/llm-providers/${providerId}`, {
    method: "DELETE"
  });
}
