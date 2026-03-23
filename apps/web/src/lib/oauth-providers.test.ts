import { describe, expect, it } from "vitest";
import { isMagicLinkAuthEnabled, parseEnabledOAuthProviderIds } from "./oauth-providers";

describe("oauth provider configuration", () => {
  it("defaults production builds to google only", () => {
    expect(parseEnabledOAuthProviderIds(undefined, false)).toEqual(["google"]);
  });

  it("keeps all providers in local development by default", () => {
    const providers = parseEnabledOAuthProviderIds(undefined, true);
    expect(providers).toContain("google");
    expect(providers).toContain("github");
  });

  it("parses configured providers without duplicates", () => {
    expect(parseEnabledOAuthProviderIds("google, line,google", false)).toEqual(["google", "line"]);
  });

  it("disables magic link auth outside development unless explicitly enabled", () => {
    expect(isMagicLinkAuthEnabled(undefined, false)).toBe(false);
    expect(isMagicLinkAuthEnabled(undefined, true)).toBe(true);
    expect(isMagicLinkAuthEnabled("false", true)).toBe(false);
  });
});
