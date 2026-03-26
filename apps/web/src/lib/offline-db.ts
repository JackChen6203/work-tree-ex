import { openDB, type DBSchema, type IDBPDatabase } from "idb";
import type { TripSummary } from "../types/domain";
import type { MutationQueueItem } from "./mutation-queue";

const OFFLINE_DB_NAME = "travel-planner-offline";
const OFFLINE_DB_VERSION = 1;
const OFFLINE_MAX_AGE_MS = 7 * 24 * 60 * 60 * 1000;

interface OfflineDbSchema extends DBSchema {
  mutations: {
    key: string;
    value: MutationQueueItem;
  };
  tripCache: {
    key: string;
    value: {
      key: string;
      trip: TripSummary;
      cachedAt: number;
    };
  };
}

let dbPromise: Promise<IDBPDatabase<OfflineDbSchema>> | null = null;

function supportsIndexedDb() {
  return typeof window !== "undefined" && typeof indexedDB !== "undefined";
}

async function getDb() {
  if (!supportsIndexedDb()) {
    throw new Error("IndexedDB is not available");
  }

  if (!dbPromise) {
    dbPromise = openDB<OfflineDbSchema>(OFFLINE_DB_NAME, OFFLINE_DB_VERSION, {
      upgrade(db) {
        if (!db.objectStoreNames.contains("mutations")) {
          db.createObjectStore("mutations", { keyPath: "id" });
        }
        if (!db.objectStoreNames.contains("tripCache")) {
          db.createObjectStore("tripCache", { keyPath: "key" });
        }
      }
    });
  }

  return dbPromise;
}

async function readwriteTxn() {
  const db = await getDb();
  return db.transaction(["mutations", "tripCache"], "readwrite");
}

export async function clearOfflineExpiredData(now = Date.now(), maxAgeMs = OFFLINE_MAX_AGE_MS) {
  if (!supportsIndexedDb()) {
    return { purgedMutations: 0, purgedTrips: 0 };
  }

  const expireBefore = now - maxAgeMs;
  const tx = await readwriteTxn();
  const mutationsStore = tx.objectStore("mutations");
  const tripsStore = tx.objectStore("tripCache");
  let purgedMutations = 0;
  let purgedTrips = 0;

  for (let cursor = await mutationsStore.openCursor(); cursor; cursor = await cursor.continue()) {
    if (cursor.value.enqueuedAt < expireBefore) {
      await cursor.delete();
      purgedMutations += 1;
    }
  }

  for (let cursor = await tripsStore.openCursor(); cursor; cursor = await cursor.continue()) {
    if (cursor.value.cachedAt < expireBefore) {
      await cursor.delete();
      purgedTrips += 1;
    }
  }

  await tx.done;
  return { purgedMutations, purgedTrips };
}

export async function listPersistedMutations(): Promise<MutationQueueItem[]> {
  if (!supportsIndexedDb()) {
    return [];
  }

  const db = await getDb();
  const rows = await db.getAll("mutations");
  return rows.sort((a, b) => a.enqueuedAt - b.enqueuedAt);
}

export async function savePersistedMutation(item: MutationQueueItem) {
  if (!supportsIndexedDb()) {
    return;
  }

  const db = await getDb();
  await db.put("mutations", item);
}

export async function deletePersistedMutation(id: string) {
  if (!supportsIndexedDb()) {
    return;
  }

  const db = await getDb();
  await db.delete("mutations", id);
}

export async function saveTripToCache(trip: TripSummary, cachedAt = Date.now()) {
  if (!supportsIndexedDb()) {
    return;
  }

  const db = await getDb();
  await db.put("tripCache", { key: trip.id, trip, cachedAt });
}

export async function saveTripsToCache(trips: TripSummary[], cachedAt = Date.now()) {
  if (!supportsIndexedDb()) {
    return;
  }

  const db = await getDb();
  const tx = db.transaction("tripCache", "readwrite");
  for (const trip of trips) {
    await tx.store.put({ key: trip.id, trip, cachedAt });
  }
  await tx.done;
}

export async function getCachedTrip(tripId: string): Promise<TripSummary | null> {
  if (!supportsIndexedDb()) {
    return null;
  }

  const db = await getDb();
  const item = await db.get("tripCache", tripId);
  return item?.trip ?? null;
}

export async function listCachedTrips(): Promise<TripSummary[]> {
  if (!supportsIndexedDb()) {
    return [];
  }

  const db = await getDb();
  const items = await db.getAll("tripCache");
  return items
    .sort((a, b) => b.cachedAt - a.cachedAt)
    .map((row) => row.trip);
}
