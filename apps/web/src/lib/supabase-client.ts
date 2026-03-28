import { createClient, type SupabaseClient } from "@supabase/supabase-js";

export type SupabaseBrowserConfig = {
  url: string;
  anonKey: string;
};

let cachedClient: SupabaseClient | null | undefined;
let configOverride: SupabaseBrowserConfig | null | undefined;

export function parseSupabaseBrowserConfig(
  urlValue: string | undefined,
  anonKeyValue: string | undefined
): SupabaseBrowserConfig | null {
  const url = urlValue?.trim() ?? "";
  const anonKey = anonKeyValue?.trim() ?? "";
  if (!url || !anonKey) {
    return null;
  }
  return { url, anonKey };
}

export function readSupabaseBrowserConfig(): SupabaseBrowserConfig | null {
  return parseSupabaseBrowserConfig(
    import.meta.env.VITE_SUPABASE_URL as string | undefined,
    import.meta.env.VITE_SUPABASE_ANON_KEY as string | undefined
  );
}

function resolveSupabaseBrowserConfig(): SupabaseBrowserConfig | null {
  if (configOverride !== undefined) {
    return configOverride;
  }
  return readSupabaseBrowserConfig();
}

export function isSupabaseBrowserConfigured(): boolean {
  return resolveSupabaseBrowserConfig() !== null;
}

export function createSupabaseBrowserClient(config: SupabaseBrowserConfig | null): SupabaseClient | null {
  if (!config) {
    return null;
  }
  return createClient(config.url, config.anonKey, {
    auth: {
      persistSession: false,
      autoRefreshToken: false,
      detectSessionInUrl: false
    },
    global: {
      headers: {
        "X-App-Client": "time-tree-web"
      }
    }
  });
}

export function getSupabaseBrowserClient(): SupabaseClient | null {
  if (cachedClient !== undefined) {
    return cachedClient;
  }
  cachedClient = createSupabaseBrowserClient(resolveSupabaseBrowserConfig());
  return cachedClient;
}

export function setSupabaseBrowserConfigForTest(config: SupabaseBrowserConfig | null): void {
  configOverride = config;
  cachedClient = undefined;
}

export function resetSupabaseBrowserClientForTest(): void {
  configOverride = undefined;
  cachedClient = undefined;
}
