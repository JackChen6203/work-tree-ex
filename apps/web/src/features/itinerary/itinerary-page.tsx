import {
  DndContext,
  DragOverlay,
  PointerSensor,
  TouchSensor,
  closestCenter,
  useDroppable,
  useSensor,
  useSensors,
  type DragEndEvent,
  type DragOverEvent,
  type DragStartEvent
} from "@dnd-kit/core";
import { SortableContext, useSortable, verticalListSortingStrategy } from "@dnd-kit/sortable";
import { CSS } from "@dnd-kit/utilities";
import { useEffect, useMemo, useRef, useState, type ReactNode } from "react";
import { Link, useParams } from "react-router-dom";
import { SurfaceCard } from "../../components/surface-card";
import { StatusPill } from "../../components/status-pill";
import type { ItineraryDayApi, ItineraryItemApi } from "../../lib/itinerary-api";
import { estimateRoute, getPlaceDetail, type PlaceDetail } from "../../lib/maps-api";
import {
  useCreateItineraryItemMutation,
  useDeleteItineraryItemMutation,
  useItineraryDaysQuery,
  usePatchItineraryItemMutation,
  useReorderItineraryItemsMutation
} from "../../lib/queries";
import { useI18n } from "../../lib/i18n";
import { isValidCoordinate } from "../../lib/map-provider-adapter";
import { useUiStore } from "../../store/ui-store";
import {
  buildConflictIndex,
  formatDurationLabel,
  formatItemTimeLabel,
  getDurationMinutes,
  toIsoFromDayAndTime,
  toTimeInputValue
} from "./itinerary-timeline";

const DAY_CONTAINER_PREFIX = "day-container:";

interface RoutePreviewCard {
  distanceKm: number;
  durationMin: number;
  provider: string;
  estimatedCostAmount?: number;
  estimatedCostCurrency?: string;
}

function getRoutePairKey(fromItemId: string, toItemId: string) {
  return `${fromItemId}__${toItemId}`;
}

function getDayContainerId(dayId: string) {
  return `${DAY_CONTAINER_PREFIX}${dayId}`;
}

function getDayIdFromContainerId(containerId: string) {
  return containerId.replace(DAY_CONTAINER_PREFIX, "");
}

function isDayContainerId(id: string) {
  return id.startsWith(DAY_CONTAINER_PREFIX);
}

function cloneDays(days: ItineraryDayApi[]) {
  return days.map((day) => ({ ...day, items: [...day.items] }));
}

function findDayIndex(days: ItineraryDayApi[], dayId: string) {
  return days.findIndex((day) => day.dayId === dayId);
}

function findItemPosition(days: ItineraryDayApi[], itemId: string) {
  for (let dayIndex = 0; dayIndex < days.length; dayIndex += 1) {
    const itemIndex = days[dayIndex].items.findIndex((item) => item.id === itemId);
    if (itemIndex >= 0) {
      return {
        dayIndex,
        dayId: days[dayIndex].dayId,
        itemIndex,
        item: days[dayIndex].items[itemIndex]
      };
    }
  }
  return null;
}

function resolveTarget(days: ItineraryDayApi[], overId: string) {
  if (isDayContainerId(overId)) {
    const dayId = getDayIdFromContainerId(overId);
    const dayIndex = findDayIndex(days, dayId);
    if (dayIndex < 0) {
      return null;
    }
    return {
      dayIndex,
      dayId,
      itemIndex: days[dayIndex].items.length
    };
  }

  const itemPos = findItemPosition(days, overId);
  if (!itemPos) {
    return null;
  }

  return {
    dayIndex: itemPos.dayIndex,
    dayId: itemPos.dayId,
    itemIndex: itemPos.itemIndex
  };
}

function normalizeSortOrder(days: ItineraryDayApi[]) {
  return days.map((day) => ({
    ...day,
    items: day.items.map((item, index) => ({
      ...item,
      dayId: day.dayId,
      sortOrder: index + 1
    }))
  }));
}

function reorderLocalDays(days: ItineraryDayApi[], activeItemId: string, overId: string) {
  const source = findItemPosition(days, activeItemId);
  const target = resolveTarget(days, overId);

  if (!source || !target) {
    return days;
  }

  if (source.dayId === target.dayId && source.itemIndex === target.itemIndex) {
    return days;
  }

  const next = cloneDays(days);
  const [moved] = next[source.dayIndex].items.splice(source.itemIndex, 1);
  if (!moved) {
    return days;
  }

  const boundedIndex = Math.max(0, Math.min(target.itemIndex, next[target.dayIndex].items.length));
  next[target.dayIndex].items.splice(boundedIndex, 0, moved);

  return normalizeSortOrder(next);
}

function SortableItemCard({
  item,
  editing,
  dragEnabled,
  dragLabel,
  className,
  children
}: {
  item: ItineraryItemApi;
  editing: boolean;
  dragEnabled: boolean;
  dragLabel: string;
  className?: string;
  children: ReactNode;
}) {
  const { attributes, listeners, setNodeRef, transform, transition, isDragging } = useSortable({
    id: item.id,
    disabled: !dragEnabled || editing
  });

  return (
    <div
      ref={setNodeRef}
      className={`rounded-[24px] bg-white p-4 transition ${isDragging ? "opacity-40" : ""} ${className ?? ""}`}
      style={{ transform: CSS.Transform.toString(transform), transition }}
    >
      <div className="mb-3 flex justify-end">
        {dragEnabled ? (
          <button
            aria-label={dragLabel}
            className="cursor-grab rounded-full border border-ink/15 px-3 py-1 text-xs font-medium text-ink active:cursor-grabbing"
            type="button"
            {...attributes}
            {...listeners}
          >
            ☰
          </button>
        ) : null}
      </div>
      {children}
    </div>
  );
}

function DayDropContainer({
  dayId,
  className,
  children
}: {
  dayId: string;
  className: string;
  children: ReactNode;
}) {
  const { setNodeRef, isOver } = useDroppable({ id: getDayContainerId(dayId) });
  return (
    <div className={`${className} ${isOver ? "ring-2 ring-coral/60 ring-offset-2 ring-offset-sand" : ""}`} ref={setNodeRef}>
      {children}
    </div>
  );
}

function getChangedItemKind(localItem: ItineraryItemApi, baseline: ItineraryItemApi | undefined) {
  if (!baseline) {
    return "added" as const;
  }
  if (
    baseline.dayId !== localItem.dayId ||
    baseline.sortOrder !== localItem.sortOrder ||
    baseline.title !== localItem.title ||
    baseline.note !== localItem.note ||
    baseline.startAt !== localItem.startAt ||
    baseline.endAt !== localItem.endAt ||
    baseline.allDay !== localItem.allDay
  ) {
    return "updated" as const;
  }
  return null;
}

export function ItineraryPage() {
  const { tripId = "" } = useParams();
  const { t } = useI18n();
  const pushToast = useUiStore((state) => state.pushToast);
  const { data: days = [], isLoading } = useItineraryDaysQuery(tripId);
  const createItem = useCreateItineraryItemMutation(tripId);
  const deleteItem = useDeleteItineraryItemMutation(tripId);
  const patchItem = usePatchItineraryItemMutation(tripId);
  const reorderItems = useReorderItineraryItemsMutation(tripId);
  const [editingItemId, setEditingItemId] = useState<string | null>(null);
  const [editingTitle, setEditingTitle] = useState("");
  const [editingNote, setEditingNote] = useState("");
  const [editingAllDay, setEditingAllDay] = useState(false);
  const [editingStartTime, setEditingStartTime] = useState("");
  const [editingEndTime, setEditingEndTime] = useState("");
  const [localDays, setLocalDays] = useState<ItineraryDayApi[]>(days);
  const [activeItemId, setActiveItemId] = useState<string | null>(null);
  const [routePreviewByPair, setRoutePreviewByPair] = useState<Record<string, RoutePreviewCard>>({});
  const [routePreviewLoadingKeys, setRoutePreviewLoadingKeys] = useState<Set<string>>(new Set());
  const [placeDetailById, setPlaceDetailById] = useState<Record<string, PlaceDetail>>({});
  const [reorderAnnouncement, setReorderAnnouncement] = useState("");
  const beforeDragDaysRef = useRef<ItineraryDayApi[] | null>(null);

  const dragSupported =
    typeof window !== "undefined" &&
    ("PointerEvent" in window || "ontouchstart" in window || navigator.maxTouchPoints > 0);

  const sensors = useSensors(
    useSensor(PointerSensor, {
      activationConstraint: { distance: 6 }
    }),
    useSensor(TouchSensor, {
      activationConstraint: { delay: 120, tolerance: 6 }
    })
  );

  useEffect(() => {
    if (!activeItemId) {
      setLocalDays(days);
    }
  }, [days, activeItemId]);

  const typeLabels: Record<string, string> = {
    place_visit: t("itinerary.attraction"),
    meal: t("itinerary.restaurant"),
    transit: t("itinerary.transit"),
    hotel: t("itinerary.lodging"),
    free_time: t("itinerary.free"),
    custom: t("itinerary.activity")
  };

  const activeDragItem = useMemo(() => {
    if (!activeItemId) {
      return null;
    }
    return findItemPosition(localDays, activeItemId)?.item ?? null;
  }, [activeItemId, localDays]);

  const conflictByItemId = useMemo(() => buildConflictIndex(localDays), [localDays]);

  const baselineItemById = useMemo(() => {
    const map = new Map<string, ItineraryItemApi>();
    for (const day of days) {
      for (const item of day.items) {
        map.set(item.id, item);
      }
    }
    return map;
  }, [days]);

  const localItemById = useMemo(() => {
    const map = new Map<string, ItineraryItemApi>();
    for (const day of localDays) {
      for (const item of day.items) {
        map.set(item.id, item);
      }
    }
    return map;
  }, [localDays]);

  const deletedItemsByDay = useMemo(() => {
    const deletedByDay: Record<string, string[]> = {};
    for (const day of days) {
      for (const item of day.items) {
        if (!localItemById.has(item.id)) {
          const bucket = deletedByDay[day.dayId] ?? [];
          bucket.push(item.title);
          deletedByDay[day.dayId] = bucket;
        }
      }
    }
    return deletedByDay;
  }, [days, localItemById]);

  const routePairs = useMemo(() => {
    const pairs: Array<{
      key: string;
      fromItemId: string;
      fromTitle: string;
      toTitle: string;
      origin: { lat: number; lng: number };
      destination: { lat: number; lng: number };
    }> = [];

    for (const day of localDays) {
      for (let index = 0; index < day.items.length - 1; index += 1) {
        const from = day.items[index];
        const to = day.items[index + 1];
        if (!isValidCoordinate(from.lat ?? 0, from.lng ?? 0) || !isValidCoordinate(to.lat ?? 0, to.lng ?? 0)) {
          continue;
        }
        pairs.push({
          key: getRoutePairKey(from.id, to.id),
          fromItemId: from.id,
          fromTitle: from.title,
          toTitle: to.title,
          origin: { lat: from.lat as number, lng: from.lng as number },
          destination: { lat: to.lat as number, lng: to.lng as number }
        });
      }
    }

    return pairs;
  }, [localDays]);

  const placeIds = useMemo(() => {
    return Array.from(
      new Set(
        localDays
          .flatMap((day) => day.items)
          .map((item) => item.placeId)
          .filter((item): item is string => Boolean(item))
      )
    );
  }, [localDays]);

  useEffect(() => {
    const activeKeys = new Set(routePairs.map((pair) => pair.key));
    setRoutePreviewByPair((current) => {
      const next: Record<string, RoutePreviewCard> = {};
      for (const [key, value] of Object.entries(current)) {
        if (activeKeys.has(key)) {
          next[key] = value;
        }
      }
      return next;
    });
    setRoutePreviewLoadingKeys((current) => {
      const next = new Set<string>();
      for (const key of current) {
        if (activeKeys.has(key)) {
          next.add(key);
        }
      }
      return next;
    });
  }, [routePairs]);

  useEffect(() => {
    if (placeIds.length === 0) {
      setPlaceDetailById({});
      return;
    }

    const missing = placeIds.filter((placeId) => !placeDetailById[placeId]);
    if (missing.length === 0) {
      return;
    }

    let cancelled = false;
    void Promise.all(
      missing.map(async (placeId) => {
        try {
          const detail = await getPlaceDetail(placeId);
          return { placeId, detail };
        } catch {
          return null;
        }
      })
    ).then((rows) => {
      if (cancelled) {
        return;
      }
      setPlaceDetailById((current) => {
        const next = { ...current };
        for (const row of rows) {
          if (!row) {
            continue;
          }
          next[row.placeId] = row.detail;
        }
        return next;
      });
    });

    return () => {
      cancelled = true;
    };
  }, [placeDetailById, placeIds]);

  useEffect(() => {
    const missing = routePairs.filter((pair) => !routePreviewByPair[pair.key] && !routePreviewLoadingKeys.has(pair.key));
    if (missing.length === 0) {
      return;
    }

    let cancelled = false;
    setRoutePreviewLoadingKeys((current) => {
      const next = new Set(current);
      for (const pair of missing) {
        next.add(pair.key);
      }
      return next;
    });

    void Promise.all(
      missing.map(async (pair) => {
        try {
          const result = await estimateRoute({
            origin: pair.origin,
            destination: pair.destination,
            mode: "transit"
          });
          return {
            key: pair.key,
            card: {
              distanceKm: Math.round((result.distanceMeters / 1000) * 10) / 10,
              durationMin: Math.max(1, Math.round(result.durationSeconds / 60)),
              provider: result.provider ?? "maps",
              estimatedCostAmount: result.estimatedCostAmount,
              estimatedCostCurrency: result.estimatedCostCurrency
            } satisfies RoutePreviewCard
          };
        } catch {
          return null;
        }
      })
    ).then((rows) => {
      if (cancelled) {
        return;
      }
      setRoutePreviewByPair((current) => {
        const next = { ...current };
        for (const row of rows) {
          if (!row) {
            continue;
          }
          next[row.key] = row.card;
        }
        return next;
      });
      setRoutePreviewLoadingKeys((current) => {
        const next = new Set(current);
        for (const pair of missing) {
          next.delete(pair.key);
        }
        return next;
      });
    });

    return () => {
      cancelled = true;
    };
  }, [routePairs, routePreviewByPair, routePreviewLoadingKeys]);

  const addItem = async () => {
    const targetDay = localDays[0]?.dayId ?? "day-1";
    await createItem.mutateAsync({
      dayId: targetDay,
      title: t("itinerary.addItem"),
      itemType: "custom",
      allDay: false,
      note: ""
    });
    pushToast(t("itinerary.addItem"));
  };

  const removeItem = async (itemId: string) => {
    await deleteItem.mutateAsync(itemId);
    pushToast(t("common.delete"));
  };

  const announceReorder = (message: string) => {
    setReorderAnnouncement("");
    window.setTimeout(() => {
      setReorderAnnouncement(message);
    }, 20);
  };

  const buildReorderAnnouncement = (daysState: ItineraryDayApi[], itemId: string, targetDayId: string, targetSortOrder: number) => {
    const dayIndex = daysState.findIndex((day) => day.dayId === targetDayId);
    const dayLabel = t("itinerary.day").replace("{n}", String(dayIndex >= 0 ? dayIndex + 1 : 1));
    const itemTitle = daysState
      .flatMap((day) => day.items)
      .find((item) => item.id === itemId)?.title ?? t("itinerary.addItem");

    return t("itinerary.reorderAnnouncement")
      .replace("{title}", itemTitle)
      .replace("{day}", dayLabel)
      .replace("{position}", String(targetSortOrder));
  };

  const commitReorder = async (
    itemId: string,
    targetDayId: string,
    targetSortOrder: number,
    fallbackDays: ItineraryDayApi[],
    nextDays: ItineraryDayApi[]
  ) => {
    try {
      await reorderItems.mutateAsync({
        operations: [
          {
            itemId,
            targetDayId,
            targetSortOrder
          }
        ]
      });
      announceReorder(buildReorderAnnouncement(nextDays, itemId, targetDayId, targetSortOrder));
    } catch (error) {
      setLocalDays(fallbackDays);
      pushToast({
        type: "error",
        message: error instanceof Error ? error.message : t("common.actionFailed")
      });
    }
  };

  const moveItemWithFallbackButtons = async (dayId: string, itemId: string, targetIndex: number) => {
    const before = localDays;
    const day = localDays.find((entry) => entry.dayId === dayId);
    if (!day) {
      return;
    }

    const targetItem = day.items[Math.max(0, Math.min(targetIndex, day.items.length - 1))];
    if (!targetItem) {
      return;
    }

    const next = reorderLocalDays(localDays, itemId, targetItem.id);
    const finalPos = findItemPosition(next, itemId);
    if (!finalPos) {
      return;
    }

    setLocalDays(next);
    await commitReorder(itemId, finalPos.dayId, finalPos.itemIndex + 1, before, next);
  };

  const startEdit = (item: ItineraryItemApi) => {
    setEditingItemId(item.id);
    setEditingTitle(item.title);
    setEditingNote(item.note ?? "");
    setEditingAllDay(item.allDay);
    setEditingStartTime(toTimeInputValue(item.startAt));
    setEditingEndTime(toTimeInputValue(item.endAt));

    if (item.allDay) {
      setEditingStartTime("");
      setEditingEndTime("");
    } else if (!item.startAt && !item.endAt) {
      // Default to a valid seed time for quick editing.
      setEditingStartTime("09:00");
      setEditingEndTime("10:00");
    }
  };

  const cancelEdit = () => {
    setEditingItemId(null);
    setEditingTitle("");
    setEditingNote("");
    setEditingAllDay(false);
    setEditingStartTime("");
    setEditingEndTime("");
  };

  const saveEdit = async (dayDate: string, itemId: string, version: number) => {
    const nextTitle = editingTitle.trim();
    if (!nextTitle) {
      return;
    }

    const nextStartAt = editingAllDay ? "" : (toIsoFromDayAndTime(dayDate, editingStartTime) ?? "");
    const nextEndAt = editingAllDay ? "" : (toIsoFromDayAndTime(dayDate, editingEndTime) ?? "");

    if (!editingAllDay && nextStartAt && nextEndAt && new Date(nextEndAt).getTime() < new Date(nextStartAt).getTime()) {
      pushToast({ type: "error", message: t("itinerary.endBeforeStart") });
      return;
    }

    const previewDays = localDays.map((day) => ({
      ...day,
      items: day.items.map((item) =>
        item.id === itemId
          ? {
              ...item,
              title: nextTitle,
              note: editingNote.trim(),
              allDay: editingAllDay,
              startAt: nextStartAt || undefined,
              endAt: nextEndAt || undefined
            }
          : item
      )
    }));
    const previewConflict = buildConflictIndex(previewDays)[itemId];
    if (previewConflict && previewConflict.titles.length > 0) {
      const allow = window.confirm(
        t("itinerary.conflictConfirm").replace("{items}", previewConflict.titles.join(", "))
      );
      if (!allow) {
        return;
      }
    }

    await patchItem.mutateAsync({
      itemId,
      version,
      input: {
        title: nextTitle,
        note: editingNote.trim(),
        allDay: editingAllDay,
        startAt: nextStartAt,
        endAt: nextEndAt
      }
    });
    cancelEdit();
    pushToast(t("common.save"));
  };

  const handleDragStart = (event: DragStartEvent) => {
    beforeDragDaysRef.current = localDays;
    setActiveItemId(String(event.active.id));
  };

  const handleDragOver = (event: DragOverEvent) => {
    if (!event.over) {
      return;
    }
    const activeId = String(event.active.id);
    const overId = String(event.over.id);
    setLocalDays((current) => reorderLocalDays(current, activeId, overId));
  };

  const handleDragEnd = async (event: DragEndEvent) => {
    const activeId = String(event.active.id);
    const overId = event.over ? String(event.over.id) : null;
    const before = beforeDragDaysRef.current;
    beforeDragDaysRef.current = null;
    setActiveItemId(null);

    if (!before || !overId) {
      setLocalDays(before ?? localDays);
      return;
    }

    const next = reorderLocalDays(localDays, activeId, overId);
    setLocalDays(next);

    const beforePos = findItemPosition(before, activeId);
    const afterPos = findItemPosition(next, activeId);
    if (!beforePos || !afterPos) {
      setLocalDays(before);
      return;
    }

    const moved = beforePos.dayId !== afterPos.dayId || beforePos.itemIndex !== afterPos.itemIndex;
    if (!moved) {
      return;
    }

    await commitReorder(activeId, afterPos.dayId, afterPos.itemIndex + 1, before, next);
  };

  const handleDragCancel = () => {
    if (beforeDragDaysRef.current) {
      setLocalDays(beforeDragDaysRef.current);
    }
    beforeDragDaysRef.current = null;
    setActiveItemId(null);
  };

  return (
    <div className="grid gap-6">
      <p aria-live="polite" className="sr-only" role="status">
        {reorderAnnouncement}
      </p>
      <SurfaceCard
        eyebrow={t("nav.itinerary")}
        title={t("itinerary.title")}
        action={
          <div className="flex items-center gap-2">
            <Link
              className="rounded-full border border-ink/15 bg-white px-4 py-2 text-sm font-medium text-ink"
              to={`/trips/${tripId}/ai-planner`}
            >
              {t("budget.aiPlanCta")}
            </Link>
            <button
              className="rounded-full bg-ink px-4 py-2 text-sm font-medium text-sand"
              disabled={createItem.isPending}
              onClick={() => {
                void addItem();
              }}
              type="button"
            >
              {createItem.isPending ? t("itinerary.adding") : t("itinerary.addItem")}
            </button>
          </div>
        }
      >
        {isLoading ? <div className="mb-4 rounded-[24px] bg-sand p-4 text-sm text-ink/65">{t("itinerary.loading")}</div> : null}
        <DndContext
          collisionDetection={closestCenter}
          onDragCancel={handleDragCancel}
          onDragEnd={(event) => {
            void handleDragEnd(event);
          }}
          onDragOver={handleDragOver}
          onDragStart={handleDragStart}
          sensors={sensors}
        >
          <div className="grid gap-5">
            {localDays.map((day, index) => (
              <DayDropContainer className="rounded-[28px] border border-ink/10 bg-sand/70 p-5" dayId={day.dayId} key={day.dayId}>
                <div className="flex flex-wrap items-center justify-between gap-4">
                  <div>
                    <p className="text-xs uppercase tracking-[0.22em] text-ink/45">
                      {t("itinerary.day").replace("{n}", String(index + 1))} · {day.date}
                    </p>
                    <h3 className="mt-2 font-display text-2xl font-bold text-ink">
                      {day.items.length > 0 ? `${day.items.length} ${t("ai.items")}` : t("itinerary.noItems")}
                    </h3>
                  </div>
                  {deletedItemsByDay[day.dayId]?.length ? (
                    <div className="rounded-2xl border border-coral/35 bg-coral/10 px-3 py-2 text-xs text-coral">
                      {t("itinerary.diffRemoved").replace("{items}", deletedItemsByDay[day.dayId].join(", "))}
                    </div>
                  ) : null}
                </div>
                <SortableContext items={day.items.map((item) => item.id)} strategy={verticalListSortingStrategy}>
                  <div className="mt-5 grid gap-4">
                    {day.items.map((item, itemIndex) => {
                      const conflict = conflictByItemId[item.id];
                      const durationMinutes = getDurationMinutes(item);
                      const placeDetail = item.placeId ? placeDetailById[item.placeId] : undefined;
                      const routePairKey =
                        itemIndex < day.items.length - 1 ? getRoutePairKey(item.id, day.items[itemIndex + 1].id) : null;
                      const routePreview = routePairKey ? routePreviewByPair[routePairKey] : null;
                      const routeLoading = routePairKey ? routePreviewLoadingKeys.has(routePairKey) : false;
                      const changeKind = getChangedItemKind(item, baselineItemById.get(item.id));

                      return (
                        <SortableItemCard
                          className={`${conflict ? "border border-coral/45" : "border border-transparent"} ${
                            changeKind === "added"
                              ? "bg-pine/5"
                              : changeKind === "updated"
                                ? "bg-[#e4edf8]"
                                : "bg-white"
                          }`}
                          dragEnabled={dragSupported && !reorderItems.isPending}
                          dragLabel={t("itinerary.dragHandle")}
                          editing={editingItemId === item.id}
                          item={item}
                          key={item.id}
                        >
                          <div className="flex flex-wrap items-start justify-between gap-3">
                            <div className="min-w-[16rem] flex-1">
                              {editingItemId === item.id ? (
                                <div className="space-y-2">
                                  <input
                                    className="w-full rounded-xl border border-ink/15 bg-sand px-3 py-2 text-sm text-ink"
                                    onChange={(event) => {
                                      setEditingTitle(event.target.value);
                                    }}
                                    value={editingTitle}
                                  />
                                  <textarea
                                    className="min-h-20 w-full rounded-xl border border-ink/15 bg-sand px-3 py-2 text-sm text-ink"
                                    onChange={(event) => {
                                      setEditingNote(event.target.value);
                                    }}
                                    value={editingNote}
                                  />
                                  <label className="flex items-center gap-2 text-xs text-ink/70">
                                    <input
                                      checked={editingAllDay}
                                      onChange={(event) => setEditingAllDay(event.target.checked)}
                                      type="checkbox"
                                    />
                                    {t("itinerary.allDay")}
                                  </label>
                                  <div className="grid gap-2 sm:grid-cols-2">
                                    <label>
                                      <span className="mb-1 block text-xs uppercase tracking-[0.14em] text-ink/55">{t("itinerary.startTime")}</span>
                                      <input
                                        className="w-full rounded-xl border border-ink/15 bg-sand px-3 py-2 text-sm text-ink disabled:opacity-45"
                                        disabled={editingAllDay}
                                        onChange={(event) => setEditingStartTime(event.target.value)}
                                        type="time"
                                        value={editingStartTime}
                                      />
                                    </label>
                                    <label>
                                      <span className="mb-1 block text-xs uppercase tracking-[0.14em] text-ink/55">{t("itinerary.endTime")}</span>
                                      <input
                                        className="w-full rounded-xl border border-ink/15 bg-sand px-3 py-2 text-sm text-ink disabled:opacity-45"
                                        disabled={editingAllDay}
                                        onChange={(event) => setEditingEndTime(event.target.value)}
                                        type="time"
                                        value={editingEndTime}
                                      />
                                    </label>
                                  </div>
                                </div>
                              ) : (
                                <>
                                  <div className="flex flex-wrap items-center gap-2">
                                    <p className="text-sm font-semibold text-ink">{item.title}</p>
                                    {changeKind === "added" ? <StatusPill tone="success">{t("itinerary.diffAdded")}</StatusPill> : null}
                                    {changeKind === "updated" ? <StatusPill tone="accent">{t("itinerary.diffUpdated")}</StatusPill> : null}
                                  </div>
                                  <p className="mt-1 text-sm text-ink/60">
                                    {typeLabels[item.itemType] ?? item.itemType}
                                  </p>
                                  {item.allDay ? (
                                    <p className="mt-1 text-xs text-ink/60">{t("itinerary.allDay")}</p>
                                  ) : null}
                                  {!item.allDay && formatItemTimeLabel(item) ? (
                                    <p className="mt-1 text-xs text-ink/60">{formatItemTimeLabel(item)}</p>
                                  ) : null}
                                  {!item.allDay && durationMinutes ? (
                                    <p className="mt-1 text-xs text-ink/55">
                                      {t("itinerary.duration")}: {formatDurationLabel(durationMinutes)}
                                    </p>
                                  ) : null}
                                </>
                              )}
                            </div>
                            <div className="flex flex-wrap items-center gap-2">
                              <StatusPill tone="accent">{item.allDay ? t("itinerary.allDay") : t("itinerary.startTime")}</StatusPill>
                              {conflict ? (
                                <StatusPill tone="danger">
                                  <span className="inline-flex items-center gap-1" title={conflict.titles.join(", ")}>
                                    <span aria-hidden>⚠</span>
                                    {t("itinerary.conflictWarning")}
                                  </span>
                                </StatusPill>
                              ) : null}
                              {editingItemId === item.id ? (
                                <>
                                  <button
                                    className="rounded-full border border-ink/15 px-3 py-1 text-xs font-medium text-ink disabled:opacity-40"
                                    disabled={patchItem.isPending}
                                    onClick={() => {
                                      void saveEdit(day.date, item.id, item.version);
                                    }}
                                    type="button"
                                  >
                                    {t("common.save")}
                                  </button>
                                  <button
                                    className="rounded-full border border-ink/15 px-3 py-1 text-xs font-medium text-ink"
                                    onClick={cancelEdit}
                                    type="button"
                                  >
                                    {t("common.cancel")}
                                  </button>
                                </>
                              ) : (
                                <button
                                  className="rounded-full border border-ink/15 px-3 py-1 text-xs font-medium text-ink"
                                  disabled={patchItem.isPending || deleteItem.isPending || reorderItems.isPending}
                                  onClick={() => {
                                    startEdit(item);
                                  }}
                                  type="button"
                                >
                                  {t("common.edit")}
                                </button>
                              )}

                              {!dragSupported ? (
                                <>
                                  <button
                                    className="rounded-full border border-ink/15 px-3 py-1 text-xs font-medium text-ink disabled:opacity-40"
                                    disabled={reorderItems.isPending || editingItemId === item.id || itemIndex === 0}
                                    onClick={() => {
                                      void moveItemWithFallbackButtons(day.dayId, item.id, Math.max(itemIndex - 1, 0));
                                    }}
                                    type="button"
                                  >
                                    {t("itinerary.sortUp")}
                                  </button>
                                  <button
                                    className="rounded-full border border-ink/15 px-3 py-1 text-xs font-medium text-ink disabled:opacity-40"
                                    disabled={reorderItems.isPending || editingItemId === item.id || itemIndex === day.items.length - 1}
                                    onClick={() => {
                                      void moveItemWithFallbackButtons(day.dayId, item.id, itemIndex + 1);
                                    }}
                                    type="button"
                                  >
                                    {t("itinerary.sortDown")}
                                  </button>
                                </>
                              ) : null}

                              <button
                                className="rounded-full border border-ink/15 px-3 py-1 text-xs font-medium text-ink"
                                disabled={deleteItem.isPending || reorderItems.isPending || editingItemId === item.id}
                                onClick={() => {
                                  void removeItem(item.id);
                                }}
                                type="button"
                              >
                                {t("common.delete")}
                              </button>
                            </div>
                          </div>
                          {editingItemId === item.id ? null : item.note ? <p className="mt-3 text-sm text-ink/70">{item.note}</p> : null}
                          {editingItemId !== item.id && item.placeId && placeDetail?.openingHours ? (
                            <p className="mt-2 text-xs text-ink/60">
                              {t("itinerary.openingHours").replace("{hours}", placeDetail.openingHours)}
                            </p>
                          ) : null}
                          {editingItemId !== item.id && item.placeId && !placeDetail?.openingHours ? (
                            <p className="mt-2 text-xs text-ink/50">{t("itinerary.openingHoursUnknown")}</p>
                          ) : null}

                          {conflict && conflict.titles.length > 0 ? (
                            <p className="mt-3 rounded-xl bg-coral/10 px-3 py-2 text-xs text-coral">
                              {t("itinerary.conflictWith").replace("{items}", conflict.titles.join(", "))}
                            </p>
                          ) : null}

                          {routePairKey ? (
                            <div className="mt-3 rounded-xl border border-ink/10 bg-sand/70 px-3 py-2 text-xs text-ink/70">
                              {routeLoading ? <span>{t("map.estimating")}</span> : null}
                              {!routeLoading && routePreview ? (
                                <div className="flex flex-wrap items-center gap-2">
                                  <span>
                                    {t("itinerary.transitInfo")
                                      .replace("{distance}", String(routePreview.distanceKm))
                                      .replace("{duration}", String(routePreview.durationMin))}
                                  </span>
                                  {typeof routePreview.estimatedCostAmount === "number" ? (
                                    <span>
                                      {t("itinerary.transitCost")
                                        .replace("{currency}", routePreview.estimatedCostCurrency ?? "")
                                        .replace("{amount}", Math.round(routePreview.estimatedCostAmount).toLocaleString())}
                                    </span>
                                  ) : null}
                                </div>
                              ) : null}
                              {!routeLoading && !routePreview && typeof item.estimatedCostAmount === "number" ? (
                                <span>
                                  {t("itinerary.transitCost")
                                    .replace("{currency}", item.estimatedCostCurrency ?? "")
                                    .replace("{amount}", Math.round(item.estimatedCostAmount).toLocaleString())}
                                </span>
                              ) : null}
                              {!routeLoading && !routePreview && typeof item.estimatedCostAmount !== "number" ? (
                                <span>{t("itinerary.noTransitEstimate")}</span>
                              ) : null}
                            </div>
                          ) : null}
                        </SortableItemCard>
                      );
                    })}
                    {day.items.length === 0 ? (
                      <div className="rounded-[24px] border border-dashed border-ink/20 bg-white/70 p-4 text-sm text-ink/60">
                        {t("itinerary.dropHere")}
                      </div>
                    ) : null}
                  </div>
                </SortableContext>
              </DayDropContainer>
            ))}
            {!isLoading && localDays.length === 0 ? <div className="rounded-[24px] bg-sand p-4 text-sm text-ink/65">{t("itinerary.noItems")}</div> : null}
          </div>
          <DragOverlay>
            {activeDragItem ? (
              <div className="w-[min(100vw-3rem,30rem)] rounded-[24px] border border-coral/30 bg-white p-4 shadow-xl">
                <p className="text-sm font-semibold text-ink">{activeDragItem.title}</p>
                <p className="mt-1 text-sm text-ink/60">{typeLabels[activeDragItem.itemType] ?? activeDragItem.itemType}</p>
              </div>
            ) : null}
          </DragOverlay>
        </DndContext>
      </SurfaceCard>
    </div>
  );
}
