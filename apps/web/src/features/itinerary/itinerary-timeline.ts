import type { ItineraryDayApi, ItineraryItemApi } from "../../lib/itinerary-api";

interface ConflictBucket {
  ids: Set<string>;
  titles: Set<string>;
}

function pad2(value: number) {
  return String(value).padStart(2, "0");
}

function parseIso(iso?: string) {
  if (!iso) {
    return null;
  }
  const date = new Date(iso);
  if (Number.isNaN(date.getTime())) {
    return null;
  }
  return date;
}

export function toTimeInputValue(iso?: string) {
  const date = parseIso(iso);
  if (!date) {
    return "";
  }
  return `${pad2(date.getHours())}:${pad2(date.getMinutes())}`;
}

export function toIsoFromDayAndTime(dayDate: string, timeValue: string) {
  const value = timeValue.trim();
  if (!value) {
    return undefined;
  }
  const localDate = new Date(`${dayDate}T${value}:00`);
  if (Number.isNaN(localDate.getTime())) {
    return undefined;
  }
  return localDate.toISOString();
}

export function getDurationMinutes(item: ItineraryItemApi) {
  const start = parseIso(item.startAt);
  const end = parseIso(item.endAt);
  if (!start || !end) {
    return null;
  }
  const diff = end.getTime() - start.getTime();
  if (diff <= 0) {
    return null;
  }
  return Math.round(diff / (60 * 1000));
}

export function formatDurationLabel(minutes: number) {
  if (minutes < 60) {
    return `${minutes}m`;
  }
  const hours = Math.floor(minutes / 60);
  const rest = minutes % 60;
  if (rest === 0) {
    return `${hours}h`;
  }
  return `${hours}h ${rest}m`;
}

export function formatItemTimeLabel(item: ItineraryItemApi) {
  if (item.allDay) {
    return "all-day";
  }
  const start = parseIso(item.startAt);
  const end = parseIso(item.endAt);
  if (!start || !end) {
    return "";
  }
  return `${pad2(start.getHours())}:${pad2(start.getMinutes())} - ${pad2(end.getHours())}:${pad2(end.getMinutes())}`;
}

export function buildConflictIndex(days: ItineraryDayApi[]) {
  const conflictByItemId: Record<string, { ids: string[]; titles: string[] }> = {};

  for (const day of days) {
    const timed = day.items
      .filter((item) => !item.allDay && item.startAt && item.endAt)
      .map((item) => ({
        item,
        start: parseIso(item.startAt)?.getTime() ?? NaN,
        end: parseIso(item.endAt)?.getTime() ?? NaN
      }))
      .filter((entry) => Number.isFinite(entry.start) && Number.isFinite(entry.end) && entry.end > entry.start);

    const buckets = new Map<string, ConflictBucket>();

    const ensureBucket = (itemId: string) => {
      const existing = buckets.get(itemId);
      if (existing) {
        return existing;
      }
      const next = { ids: new Set<string>(), titles: new Set<string>() };
      buckets.set(itemId, next);
      return next;
    };

    for (let i = 0; i < timed.length; i += 1) {
      for (let j = i + 1; j < timed.length; j += 1) {
        const a = timed[i];
        const b = timed[j];
        const overlap = a.start < b.end && b.start < a.end;
        if (!overlap) {
          continue;
        }

        const aBucket = ensureBucket(a.item.id);
        aBucket.ids.add(b.item.id);
        aBucket.titles.add(b.item.title);

        const bBucket = ensureBucket(b.item.id);
        bBucket.ids.add(a.item.id);
        bBucket.titles.add(a.item.title);
      }
    }

    for (const [itemId, bucket] of buckets.entries()) {
      conflictByItemId[itemId] = {
        ids: Array.from(bucket.ids),
        titles: Array.from(bucket.titles)
      };
    }
  }

  return conflictByItemId;
}

