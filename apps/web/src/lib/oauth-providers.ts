export interface OAuthProvider {
  id: string;
  label: string;
  category: "social" | "travel";
}

const allOAuthProviders: OAuthProvider[] = [
  { id: "google", label: "Google", category: "social" },
  { id: "apple", label: "Apple", category: "social" },
  { id: "facebook", label: "Facebook", category: "social" },
  { id: "x", label: "X", category: "social" },
  { id: "github", label: "GitHub", category: "social" },
  { id: "line", label: "LINE", category: "social" },
  { id: "kakao", label: "Kakao", category: "social" },
  { id: "wechat", label: "WeChat", category: "social" },
  { id: "tripadvisor", label: "Tripadvisor", category: "travel" },
  { id: "booking", label: "Booking.com", category: "travel" }
];

export function parseEnabledOAuthProviderIds(configValue?: string, isDev = false) {
  const requested = (configValue ?? "")
    .split(",")
    .map((item) => item.trim().toLowerCase())
    .filter(Boolean);

  if (requested.length > 0) {
    return requested.filter((providerId, index) => requested.indexOf(providerId) === index);
  }

  if (isDev) {
    return allOAuthProviders.map((provider) => provider.id);
  }

  return ["google"];
}

export function isMagicLinkAuthEnabled(configValue?: string, isDev = false) {
  if (configValue === "true") {
    return true;
  }
  if (configValue === "false") {
    return false;
  }
  return isDev;
}

const enabledProviderIds = new Set(parseEnabledOAuthProviderIds(import.meta.env.VITE_OAUTH_PROVIDERS, import.meta.env.DEV));

export const oauthProviders = allOAuthProviders.filter((provider) => enabledProviderIds.has(provider.id));
export const magicLinkAuthEnabled = isMagicLinkAuthEnabled(import.meta.env.VITE_ENABLE_MAGIC_LINK_AUTH, import.meta.env.DEV);
