import { useState } from "react";
import { useSessionStore } from "../store/session-store";
import { useI18n } from "../lib/i18n";
import { StatusPill } from "./status-pill";
import { useFlushSyncMutationsMutation, useSyncBootstrapQuery, useTripsQuery } from "../lib/queries";

export function SyncStatusBar() {
  const { isOnline, pendingMutations } = useSessionStore();
  const { t } = useI18n();
  const { data: trips = [] } = useTripsQuery();
  const primaryTripId = trips[0]?.id ?? "";
  const { data: syncData, isLoading: syncLoading } = useSyncBootstrapQuery(primaryTripId, 0);
  const flushMutation = useFlushSyncMutationsMutation();
  const [baseVersion, setBaseVersion] = useState(0);

  const changedTotal = (syncData?.changedTrips.length ?? 0) + (syncData?.changedDays.length ?? 0) + (syncData?.changedNotifications.length ?? 0);

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
    <div className="flex flex-wrap items-center gap-3 rounded-full border border-ink/10 bg-white/75 px-4 py-3 text-sm text-ink/70">
      <StatusPill tone={isOnline ? "success" : "danger"}>{isOnline ? t("sync.online") : t("sync.offline")}</StatusPill>
      <span>
        {t("sync.queue")}: {pendingMutations}
      </span>
      <span>{syncLoading ? "Syncing..." : `Server changes: ${changedTotal}`}</span>
      {syncData?.serverTime ? <span>Last sync: {new Date(syncData.serverTime).toLocaleTimeString()}</span> : null}
      <button
        className="rounded-full border border-ink/20 px-3 py-1 text-xs font-medium text-ink disabled:opacity-50"
        disabled={!isOnline || !primaryTripId || flushMutation.isPending}
        onClick={() => {
          void flushNow();
        }}
        type="button"
      >
        {flushMutation.isPending ? "Flushing..." : "Flush queue"}
      </button>
      {flushMutation.data ? (
        <span>
          Flush result: {flushMutation.data.acceptedCount} accepted / {flushMutation.data.conflictCount} conflicts
        </span>
      ) : null}
      <span>{t("sync.authoritative")}</span>
    </div>
  );
}
