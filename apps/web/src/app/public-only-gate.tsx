import { useEffect, type ReactNode } from "react";
import { Navigate } from "react-router-dom";
import { useSessionStore } from "../store/session-store";
import { SessionHydrationScreen } from "./session-hydration-screen";

interface PublicOnlyGateProps {
  children: ReactNode;
}

export function PublicOnlyGate({ children }: PublicOnlyGateProps) {
  const hydrated = useSessionStore((state) => state.hydrated);
  const user = useSessionStore((state) => state.user);
  const hydrate = useSessionStore((state) => state.hydrate);

  useEffect(() => {
    if (!hydrated) {
      void hydrate();
    }
  }, [hydrate, hydrated]);

  if (!hydrated) {
    return <SessionHydrationScreen />;
  }

  if (user) {
    return <Navigate to="/" replace />;
  }

  return <>{children}</>;
}
