import { beforeEach, describe, expect, it, vi } from "vitest";
import { createClient } from "@supabase/supabase-js";
import {
  createSupabaseBrowserClient,
  getSupabaseBrowserClient,
  isSupabaseBrowserConfigured,
  parseSupabaseBrowserConfig,
  resetSupabaseBrowserClientForTest,
  setSupabaseBrowserConfigForTest
} from "./supabase-client";

vi.mock("@supabase/supabase-js", () => ({
  createClient: vi.fn()
}));

describe("supabase browser client", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    resetSupabaseBrowserClientForTest();
  });

  it("parses browser config only when url and anon key are both present", () => {
    expect(parseSupabaseBrowserConfig(undefined, undefined)).toBeNull();
    expect(parseSupabaseBrowserConfig("https://demo.supabase.co", undefined)).toBeNull();
    expect(parseSupabaseBrowserConfig(undefined, "anon-key")).toBeNull();
    expect(parseSupabaseBrowserConfig("https://demo.supabase.co", "anon-key")).toEqual({
      url: "https://demo.supabase.co",
      anonKey: "anon-key"
    });
  });

  it("creates no client when config is missing", () => {
    expect(createSupabaseBrowserClient(null)).toBeNull();
    expect(createClient).not.toHaveBeenCalled();
  });

  it("creates client with anon key config and caches singleton", () => {
    const fakeClient = { marker: "supabase-client" };
    vi.mocked(createClient).mockReturnValue(fakeClient as never);

    setSupabaseBrowserConfigForTest({ url: "https://demo.supabase.co", anonKey: "anon-key" });
    expect(isSupabaseBrowserConfigured()).toBe(true);

    const first = getSupabaseBrowserClient();
    const second = getSupabaseBrowserClient();

    expect(first).toBe(fakeClient);
    expect(second).toBe(fakeClient);
    expect(createClient).toHaveBeenCalledTimes(1);
    expect(createClient).toHaveBeenCalledWith(
      "https://demo.supabase.co",
      "anon-key",
      expect.objectContaining({
        auth: expect.objectContaining({
          persistSession: false,
          autoRefreshToken: false,
          detectSessionInUrl: false
        })
      })
    );
  });
});
