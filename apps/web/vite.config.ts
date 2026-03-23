import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import { VitePWA } from "vite-plugin-pwa";

export default defineConfig({
  plugins: [
    react(),
    VitePWA({
      registerType: "autoUpdate",
      includeAssets: ["favicon.svg"],
      workbox: {
        navigateFallbackDenylist: [/^\/api\//, /^\/healthz$/]
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
});
