import { apiRequest } from "./api";

export interface ItineraryItemApi {
  id: string;
  dayId: string;
  title: string;
  itemType: string;
  startAt?: string;
  endAt?: string;
  allDay: boolean;
  sortOrder: number;
  note?: string;
  placeId?: string;
  lat?: number;
  lng?: number;
  estimatedCostAmount?: number;
  estimatedCostCurrency?: string;
  version: number;
}

export interface ItineraryDayApi {
  dayId: string;
  date: string;
  sortOrder: number;
  items: ItineraryItemApi[];
}

export function listItineraryDays(tripId: string) {
  return apiRequest<ItineraryDayApi[]>(`/api/v1/trips/${tripId}/days`);
}

export function createItineraryItem(
  tripId: string,
  input: { dayId: string; title: string; itemType: string; allDay: boolean; note?: string }
) {
  return apiRequest<ItineraryItemApi>(`/api/v1/trips/${tripId}/items`, {
    method: "POST",
    headers: {
      "Idempotency-Key": crypto.randomUUID()
    },
    body: JSON.stringify(input)
  });
}

export function deleteItineraryItem(tripId: string, itemId: string) {
  return apiRequest<void>(`/api/v1/trips/${tripId}/items/${itemId}`, {
    method: "DELETE"
  });
}

export function reorderItineraryItems(
  tripId: string,
  input: {
    operations: Array<{ itemId: string; targetDayId: string; targetSortOrder: number }>;
  }
) {
  return apiRequest<ItineraryDayApi[]>(`/api/v1/trips/${tripId}/items/reorder`, {
    method: "POST",
    headers: {
      "Idempotency-Key": crypto.randomUUID()
    },
    body: JSON.stringify(input)
  });
}
