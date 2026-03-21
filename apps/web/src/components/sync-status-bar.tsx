import { useSessionStore } from "../store/session-store";
import { useI18n } from "../lib/i18n";
import { StatusPill } from "./status-pill";
import { useSyncBootstrapQuery, useTripsQuery } from "../lib/queries";

export function SyncStatusBar() {
  const { isOnline, pendingMutations } = useSessionStore();
  const { t } = useI18n();
  const { data: trips = [] } = useTripsQuery();
  const primaryTripId = trips[0]?.id ?? "";
  const { data: syncData, isLoading: syncLoading } = useSyncBootstrapQuery(primaryTripId, 0);

  const changedTotal = (syncData?.changedTrips.length ?? 0) + (syncData?.changedDays.length ?? 0) + (syncData?.changedNotifications.length ?? 0);

  return (
    <div className="flex flex-wrap items-center gap-3 rounded-full border border-ink/10 bg-white/75 px-4 py-3 text-sm text-ink/70">
      <StatusPill tone={isOnline ? "success" : "danger"}>{isOnline ? t("sync.online") : t("sync.offline")}</StatusPill>
      <span>
        {t("sync.queue")}: {pendingMutations}
      </span>
      <span>{syncLoading ? "Syncing..." : `Server changes: ${changedTotal}`}</span>
      {syncData?.serverTime ? <span>Last sync: {new Date(syncData.serverTime).toLocaleTimeString()}</span> : null}
      <span>{t("sync.authoritative")}</span>
    </div>
  );
}
