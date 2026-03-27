import type { ItineraryDayApi } from "../../lib/itinerary-api";

export interface ItineraryMapPoint {
  id: string;
  itemId: string;
  dayId: string;
  dayDate: string;
  daySortOrder: number;
  itemSortOrder: number;
  title: string;
  itemType: string;
  lat: number;
  lng: number;
  note?: string;
}

export function isFiniteCoordinate(lat: number | undefined, lng: number | undefined) {
  return (
    typeof lat === "number" &&
    typeof lng === "number" &&
    Number.isFinite(lat) &&
    Number.isFinite(lng) &&
    lat >= -90 &&
    lat <= 90 &&
    lng >= -180 &&
    lng <= 180 &&
    !(lat === 0 && lng === 0)
  );
}

export function extractItineraryMapPoints(days: ItineraryDayApi[]) {
  const points: ItineraryMapPoint[] = [];

  for (const day of days) {
    for (const item of day.items) {
      if (!isFiniteCoordinate(item.lat, item.lng)) {
        continue;
      }
      points.push({
        id: item.id,
        itemId: item.id,
        dayId: day.dayId,
        dayDate: day.date,
        daySortOrder: day.sortOrder,
        itemSortOrder: item.sortOrder,
        title: item.title,
        itemType: item.itemType,
        lat: item.lat as number,
        lng: item.lng as number,
        note: item.note
      });
    }
  }

  points.sort((a, b) => {
    if (a.daySortOrder !== b.daySortOrder) {
      return a.daySortOrder - b.daySortOrder;
    }
    return a.itemSortOrder - b.itemSortOrder;
  });

  return points;
}

export function toRoutePath(points: ItineraryMapPoint[]) {
  return points.map((point) => ({ lat: point.lat, lng: point.lng }));
}

