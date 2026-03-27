const TRIP_COVERS_STORAGE_KEY = "tt.trip-covers.v1";

export type TripCoverMap = Record<string, string>;

function canUseStorage() {
  return typeof window !== "undefined" && typeof window.localStorage !== "undefined";
}

function parseCoverMap(raw: string | null): TripCoverMap {
  if (!raw) {
    return {};
  }

  try {
    const parsed = JSON.parse(raw);
    if (!parsed || typeof parsed !== "object") {
      return {};
    }

    const map: TripCoverMap = {};
    for (const [key, value] of Object.entries(parsed as Record<string, unknown>)) {
      if (typeof value === "string" && value.startsWith("data:image/")) {
        map[key] = value;
      }
    }
    return map;
  } catch {
    return {};
  }
}

function readCoverMap() {
  if (!canUseStorage()) {
    return {};
  }
  return parseCoverMap(window.localStorage.getItem(TRIP_COVERS_STORAGE_KEY));
}

function writeCoverMap(map: TripCoverMap) {
  if (!canUseStorage()) {
    return;
  }
  window.localStorage.setItem(TRIP_COVERS_STORAGE_KEY, JSON.stringify(map));
}

export function listTripCoverImages() {
  return readCoverMap();
}

export function getTripCoverImage(tripId: string) {
  return readCoverMap()[tripId];
}

export function saveTripCoverImage(tripId: string, dataUrl: string) {
  const current = readCoverMap();
  current[tripId] = dataUrl;
  writeCoverMap(current);
}

export function removeTripCoverImage(tripId: string) {
  const current = readCoverMap();
  if (!current[tripId]) {
    return;
  }
  delete current[tripId];
  writeCoverMap(current);
}
