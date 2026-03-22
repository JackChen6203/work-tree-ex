export interface PendingMutationRecord {
  id: string;
  scope: string;
  createdAt: number;
}

export interface SyncConflictItem {
  id: string;
  entityId: string;
  reason: string;
  expectedVersion?: number;
}

export interface SyncStatusSnapshotInput {
  changedTrips?: Array<unknown>;
  changedDays?: Array<unknown>;
  changedNotifications?: Array<unknown>;
  pendingMutationRecords: PendingMutationRecord[];
  flushData?: {
    acceptedCount: number;
    conflictCount: number;
    conflicts: SyncConflictItem[];
  } | null;
}

export function buildSyncStatusSnapshot(input: SyncStatusSnapshotInput) {
  const changedTotal = (input.changedTrips?.length ?? 0) + (input.changedDays?.length ?? 0) + (input.changedNotifications?.length ?? 0);
  const queueScopes = Array.from(new Set(input.pendingMutationRecords.map((item) => item.scope)));
  const hasConflicts = (input.flushData?.conflictCount ?? 0) > 0;

  return {
    changedTotal,
    queueScopes,
    hasConflicts,
    acceptedCount: input.flushData?.acceptedCount ?? 0,
    conflictCount: input.flushData?.conflictCount ?? 0,
    conflicts: input.flushData?.conflicts ?? []
  };
}
