import { useEffect } from "react";
import { parseSessionSyncPayload, sessionSyncStorageKey } from "../lib/session-sync";
import { useSessionStore } from "../store/session-store";

export function SessionSyncBridge() {
  const setUser = useSessionStore((state) => state.setUser);
  const clearUser = useSessionStore((state) => state.clearUser);

  useEffect(() => {
    const onStorage = (event: StorageEvent) => {
      if (event.key !== sessionSyncStorageKey) {
        return;
      }

      const payload = parseSessionSyncPayload(event.newValue);
      if (!payload) {
        return;
      }

      if (payload.type === "signed_out") {
        clearUser();
        return;
      }

      if (payload.user) {
        setUser(payload.user, payload.roles);
      }
    };

    window.addEventListener("storage", onStorage);
    return () => {
      window.removeEventListener("storage", onStorage);
    };
  }, [clearUser, setUser]);

  return null;
}
