import { useSessionStore } from "../store/session-store";
import { useI18n } from "../lib/i18n";
import { StatusPill } from "./status-pill";

export function SyncStatusBar() {
  const { isOnline, pendingMutations } = useSessionStore();
  const { t } = useI18n();

  return (
    <div className="flex flex-wrap items-center gap-3 rounded-full border border-ink/10 bg-white/75 px-4 py-3 text-sm text-ink/70">
      <StatusPill tone={isOnline ? "success" : "danger"}>{isOnline ? t("sync.online") : t("sync.offline")}</StatusPill>
      <span>
        {t("sync.queue")}: {pendingMutations}
      </span>
      <span>{t("sync.authoritative")}</span>
    </div>
  );
}
