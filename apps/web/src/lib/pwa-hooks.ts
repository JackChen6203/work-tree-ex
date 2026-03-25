import { useEffect } from "react";
import { useUiStore } from "../store/ui-store";

// ---------- PWA Install Prompt ----------

let deferredInstallPrompt: BeforeInstallPromptEvent | null = null;

interface BeforeInstallPromptEvent extends Event {
  prompt: () => Promise<void>;
  userChoice: Promise<{ outcome: "accepted" | "dismissed" }>;
}

export function usePwaInstallPrompt() {
  const setPwaInstallable = useUiStore((state) => state.setPwaInstallable);
  const pwaInstallable = useUiStore((state) => state.pwaInstallable);

  useEffect(() => {
    const handler = (event: Event) => {
      event.preventDefault();
      deferredInstallPrompt = event as BeforeInstallPromptEvent;
      setPwaInstallable(true);
    };

    window.addEventListener("beforeinstallprompt", handler);
    return () => window.removeEventListener("beforeinstallprompt", handler);
  }, [setPwaInstallable]);

  const promptInstall = async () => {
    if (!deferredInstallPrompt) return;
    await deferredInstallPrompt.prompt();
    const choice = await deferredInstallPrompt.userChoice;
    if (choice.outcome === "accepted") {
      setPwaInstallable(false);
    }
    deferredInstallPrompt = null;
  };

  return { pwaInstallable, promptInstall };
}

// ---------- Service Worker Update Notification ----------

export function useSwUpdateNotification() {
  const setSwUpdateAvailable = useUiStore((state) => state.setSwUpdateAvailable);
  const swUpdateAvailable = useUiStore((state) => state.swUpdateAvailable);

  useEffect(() => {
    if (!("serviceWorker" in navigator)) return;

    const checkForUpdates = async () => {
      try {
        const registration = await navigator.serviceWorker.getRegistration();
        if (!registration) return;

        registration.addEventListener("updatefound", () => {
          const newWorker = registration.installing;
          if (!newWorker) return;

          newWorker.addEventListener("statechange", () => {
            if (newWorker.state === "installed" && navigator.serviceWorker.controller) {
              setSwUpdateAvailable(true);
            }
          });
        });
      } catch {
        // SW update check failed — silent retry next time
      }
    };

    void checkForUpdates();
  }, [setSwUpdateAvailable]);

  const applyUpdate = () => {
    window.location.reload();
  };

  return { swUpdateAvailable, applyUpdate };
}
