import { useMemo, useState } from "react";
import { useSessionStore } from "../store/session-store";
import { useI18n } from "../lib/i18n";
import { StatusPill } from "./status-pill";
import { buildSyncStatusSnapshot } from "./sync-status";
import { useFlushSyncMutationsMutation, useSyncBootstrapQuery, useTripsQuery } from "../lib/queries";

export function SyncStatusBar() {
  const { isOnline, pendingMutations, pendingMutationRecords } = useSessionStore();
  const { t } = useI18n();
  const { data: trips = [] } = useTripsQuery();
  const primaryTripId = trips[0]?.id ?? "";
  const { data: syncData, isLoading: syncLoading } = useSyncBootstrapQuery(primaryTripId, 0);
  const flushMutation = useFlushSyncMutationsMutation();
  const [baseVersion, setBaseVersion] = useState(0);

  const syncSnapshot = useMemo(
    () =>
      buildSyncStatusSnapshot({
        changedTrips: syncData?.changedTrips,
        changedDays: syncData?.changedDays,
        changedNotifications: syncData?.changedNotifications,
        pendingMutationRecords,
        flushData: flushMutation.data
      }),
    [flushMutation.data, pendingMutationRecords, syncData?.changedDays, syncData?.changedNotifications, syncData?.changedTrips]
  );

  const flushNow = async () => {
    if (!primaryTripId) {
      return;
    }
    const result = await flushMutation.mutateAsync({
      tripId: primaryTripId,
      mutations: [
        {
          id: crypto.randomUUID(),
          entityType: "trip",
          entityId: primaryTripId,
          baseVersion
        }
      ]
    });
    setBaseVersion(result.nextVersion);
  };

  return (
    <div className="rounded-[28px] border border-ink/10 bg-white/75 px-4 py-3 text-sm text-ink/70">
      <div className="flex flex-wrap items-center gap-3">
        <StatusPill tone={isOnline ? "success" : "danger"}>{isOnline ? t("sync.online") : t("sync.offline")}</StatusPill>
        {pendingMutations > 0 ? (
          <span>
            {t("sync.queue")}: {pendingMutations}
          </span>
        ) : null}
        <span>{syncLoading ? t("sync.syncing") : `${t("sync.serverChanges")}: ${syncSnapshot.changedTotal}`}</span>
        {syncData?.serverTime ? <span>{t("sync.lastSync")}: {new Date(syncData.serverTime).toLocaleTimeString()}</span> : null}
        {pendingMutations > 0 ? (
          <button
            className="rounded-full border border-ink/20 px-3 py-1 text-xs font-medium text-ink disabled:opacity-50"
            disabled={!isOnline || !primaryTripId || flushMutation.isPending}
            onClick={() => {
              void flushNow();
            }}
            type="button"
          >
            {flushMutation.isPending ? t("sync.flushing") : t("sync.flushNow")}
          </button>
        ) : null}
      </div>

      {flushMutation.data ? (
        <div className={`mt-3 rounded-2xl px-4 py-3 ${syncSnapshot.hasConflicts ? "bg-coral/10 text-coral" : "bg-sand text-ink/70"}`}>
          <div>
            {t("sync.flushResult")}: {syncSnapshot.acceptedCount} {t("sync.accepted")} / {syncSnapshot.conflictCount} {t("sync.conflicts")}
          </div>
        </div>
      ) : null}
    </div>
  );
}
