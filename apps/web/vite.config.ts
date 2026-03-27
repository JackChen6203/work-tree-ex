import { defineConfig } from "vitest/config";
import react from "@vitejs/plugin-react";
import { VitePWA } from "vite-plugin-pwa";

function bundleReportPlugin(enabled: boolean) {
  return {
    apply: "build",
    generateBundle(_options: unknown, bundle: Record<string, { type: string; fileName?: string; code?: string; modules?: Record<string, { renderedLength?: number }> }>) {
      if (!enabled) {
        return;
      }

      const chunkRows = Object.values(bundle)
        .filter((entry) => entry.type === "chunk")
        .map((entry) => {
          const modules = entry.modules ?? {};
          const moduleRows = Object.values(modules);
          return {
            fileName: entry.fileName ?? "unknown",
            bytes: entry.code?.length ?? 0,
            moduleCount: moduleRows.length,
            treeShakenModules: moduleRows.filter((module) => module.renderedLength === 0).length
          };
        })
        .sort((a, b) => b.bytes - a.bytes);

      this.emitFile({
        type: "asset",
        fileName: "bundle-report.json",
        source: JSON.stringify(
          {
            generatedAt: new Date().toISOString(),
            chunks: chunkRows
          },
          null,
          2
        )
      });
    },
    name: "bundle-report-plugin"
  };
}

export default defineConfig(({ mode }) => {
  const isAnalyze = mode === "analyze";

  return {
    test: {
      exclude: ["e2e/**", "node_modules/**", "dist/**"]
    },
    plugins: [
      react(),
      bundleReportPlugin(isAnalyze),
      VitePWA({
        registerType: "autoUpdate",
        includeAssets: ["favicon.svg"],
        workbox: {
          cleanupOutdatedCaches: true,
          navigateFallbackDenylist: [/^\/api\//, /^\/healthz$/],
          runtimeCaching: [
            {
              urlPattern: /\/api\/.*$/i,
              handler: "NetworkFirst",
              options: {
                cacheName: "api-runtime-cache",
                networkTimeoutSeconds: 5,
                cacheableResponse: {
                  statuses: [0, 200]
                },
                expiration: {
                  maxEntries: 80,
                  maxAgeSeconds: 60 * 60 * 24
                }
              }
            },
            {
              urlPattern: /\.(?:js|css|woff2?|png|jpg|jpeg|gif|svg|webp|avif)$/i,
              handler: "CacheFirst",
              options: {
                cacheName: "static-runtime-cache",
                cacheableResponse: {
                  statuses: [0, 200]
                },
                expiration: {
                  maxEntries: 180,
                  maxAgeSeconds: 60 * 60 * 24 * 30
                }
              }
            }
          ]
        },
        manifest: {
          name: "TimeTree Travel Planner",
          short_name: "TravelPlanner",
          description: "Offline-friendly collaborative travel planning PWA.",
          theme_color: "#0f172a",
          background_color: "#f4efe6",
          display: "standalone",
          start_url: "/",
          icons: [
            {
              src: "/favicon.svg",
              sizes: "any",
              type: "image/svg+xml",
              purpose: "any"
            }
          ]
        }
      })
    ]
  };
});
