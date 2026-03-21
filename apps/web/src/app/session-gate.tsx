import { useEffect } from "react";
import type { ReactNode } from "react";
import { useI18n } from "../lib/i18n";
import { useSessionStore } from "../store/session-store";

interface SessionGateProps {
  children: ReactNode;
}

export function SessionGate({ children }: SessionGateProps) {
  const hydrated = useSessionStore((state) => state.hydrated);
  const hydrate = useSessionStore((state) => state.hydrate);
  const { t } = useI18n();

  useEffect(() => {
    if (!hydrated) {
      void hydrate();
    }
  }, [hydrate, hydrated]);

  if (!hydrated) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-sand px-6">
        <div className="rounded-[28px] bg-white px-8 py-7 shadow-card">
          <p className="text-xs uppercase tracking-[0.24em] text-ink/45">{t("session.hydrationLabel")}</p>
          <h1 className="mt-3 font-display text-3xl font-bold text-ink">{t("session.preparing")}</h1>
          <p className="mt-2 text-sm text-ink/70">{t("session.hydrationDescription")}</p>
        </div>
      </div>
    );
  }

  return <>{children}</>;
}
