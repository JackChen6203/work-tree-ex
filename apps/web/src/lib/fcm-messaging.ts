import { getApps, initializeApp, type FirebaseOptions } from "firebase/app";
import {
  getMessaging,
  getToken,
  isSupported,
  onMessage,
  type MessagePayload
} from "firebase/messaging";
import { registerFcmToken } from "./notifications-api";

const TOKEN_CACHE_KEY = "fcm_token_cache";

export type PushSetupStatus =
  | "configured"
  | "not_configured"
  | "unsupported"
  | "permission_required"
  | "denied"
  | "error";

export interface SetupPushMessagingOptions {
  promptForPermission?: boolean;
  forceUpload?: boolean;
  onForegroundMessage?: (payload: MessagePayload) => void;
}

export interface SetupPushMessagingResult {
  status: PushSetupStatus;
  token?: string;
}

let unsubscribeForegroundMessage: (() => void) | null = null;

function getFirebaseConfig(): FirebaseOptions | null {
  const apiKey = import.meta.env.VITE_FIREBASE_API_KEY as string | undefined;
  const authDomain = import.meta.env.VITE_FIREBASE_AUTH_DOMAIN as string | undefined;
  const projectId = import.meta.env.VITE_FIREBASE_PROJECT_ID as string | undefined;
  const storageBucket = import.meta.env.VITE_FIREBASE_STORAGE_BUCKET as string | undefined;
  const messagingSenderId = import.meta.env.VITE_FIREBASE_MESSAGING_SENDER_ID as string | undefined;
  const appId = import.meta.env.VITE_FIREBASE_APP_ID as string | undefined;

  if (!apiKey || !projectId || !messagingSenderId || !appId) {
    return null;
  }

  return {
    apiKey,
    authDomain,
    projectId,
    storageBucket,
    messagingSenderId,
    appId
  };
}

function getVapidKey() {
  return (import.meta.env.VITE_FIREBASE_VAPID_KEY as string | undefined)?.trim() ?? "";
}

function getOrCreateFirebaseApp(config: FirebaseOptions) {
  const existing = getApps().find((app) => app.name === "travel-planner-web");
  if (existing) {
    return existing;
  }

  return initializeApp(config, "travel-planner-web");
}

async function registerFcmServiceWorker() {
  if (!("serviceWorker" in navigator)) {
    return null;
  }

  return navigator.serviceWorker.register("/firebase-messaging-sw.js", {
    scope: "/firebase-cloud-messaging-push-scope"
  });
}

async function uploadToken(token: string) {
  await registerFcmToken({
    token,
    platform: "web",
    locale: navigator.language,
    timezone: Intl.DateTimeFormat().resolvedOptions().timeZone,
    userAgent: navigator.userAgent
  });
}

export async function setupPushMessaging(options: SetupPushMessagingOptions = {}): Promise<SetupPushMessagingResult> {
  if (typeof window === "undefined" || typeof Notification === "undefined") {
    return { status: "unsupported" };
  }

  const firebaseConfig = getFirebaseConfig();
  const vapidKey = getVapidKey();
  if (!firebaseConfig || !vapidKey) {
    return { status: "not_configured" };
  }

  if (!(await isSupported())) {
    return { status: "unsupported" };
  }

  let permission = Notification.permission;
  if (permission === "default" && options.promptForPermission) {
    permission = await Notification.requestPermission();
  }

  if (permission === "default") {
    return { status: "permission_required" };
  }

  if (permission === "denied") {
    return { status: "denied" };
  }

  try {
    const firebaseApp = getOrCreateFirebaseApp(firebaseConfig);
    const messaging = getMessaging(firebaseApp);
    const serviceWorkerRegistration = await registerFcmServiceWorker();
    if (!serviceWorkerRegistration) {
      return { status: "unsupported" };
    }

    const token = await getToken(messaging, {
      vapidKey,
      serviceWorkerRegistration
    });
    if (!token) {
      return { status: "error" };
    }

    const cachedToken = window.localStorage.getItem(TOKEN_CACHE_KEY);
    if (options.forceUpload || cachedToken !== token) {
      await uploadToken(token);
      window.localStorage.setItem(TOKEN_CACHE_KEY, token);
    }

    if (options.onForegroundMessage) {
      unsubscribeForegroundMessage?.();
      unsubscribeForegroundMessage = onMessage(messaging, options.onForegroundMessage);
    }

    return { status: "configured", token };
  } catch {
    return { status: "error" };
  }
}

export function isPushConfigured() {
  const config = getFirebaseConfig();
  return Boolean(config && getVapidKey());
}
