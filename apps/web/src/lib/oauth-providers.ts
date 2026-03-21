export interface OAuthProvider {
  id: string;
  label: string;
  category: "social" | "travel";
}

export const oauthProviders: OAuthProvider[] = [
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
