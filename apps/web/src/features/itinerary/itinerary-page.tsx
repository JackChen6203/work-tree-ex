import {
  DndContext,
  DragOverlay,
  PointerSensor,
  TouchSensor,
  closestCenter,
  useDroppable,
  useSensor,
  useSensors,
  type DragOverEvent,
  type DragEndEvent,
  type DragStartEvent
} from "@dnd-kit/core";
import { SortableContext, useSortable, verticalListSortingStrategy } from "@dnd-kit/sortable";
import { CSS } from "@dnd-kit/utilities";
import { useEffect, useMemo, useRef, useState } from "react";
import type { ReactNode } from "react";
import { useParams } from "react-router-dom";
import { SurfaceCard } from "../../components/surface-card";
import { StatusPill } from "../../components/status-pill";
import type { ItineraryDayApi, ItineraryItemApi } from "../../lib/itinerary-api";
import {
  useCreateItineraryItemMutation,
  useDeleteItineraryItemMutation,
  useItineraryDaysQuery,
  usePatchItineraryItemMutation,
  useReorderItineraryItemsMutation
} from "../../lib/queries";
import { useI18n } from "../../lib/i18n";
import { useUiStore } from "../../store/ui-store";

const DAY_CONTAINER_PREFIX = "day-container:";

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
  children
}: {
  item: ItineraryItemApi;
  editing: boolean;
  dragEnabled: boolean;
  dragLabel: string;
  children: ReactNode;
}) {
  const { attributes, listeners, setNodeRef, transform, transition, isDragging } = useSortable({
    id: item.id,
    disabled: !dragEnabled || editing
  });

  return (
    <div
      ref={setNodeRef}
      className={`rounded-[24px] bg-white p-4 transition ${isDragging ? "opacity-40" : ""}`}
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
  const [localDays, setLocalDays] = useState<ItineraryDayApi[]>(days);
  const [activeItemId, setActiveItemId] = useState<string | null>(null);
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
    attraction: t("itinerary.attraction"),
    restaurant: t("itinerary.restaurant"),
    transit: t("itinerary.transit"),
    lodging: t("itinerary.lodging"),
    activity: t("itinerary.activity"),
    free: t("itinerary.free"),
    custom: t("itinerary.activity")
  };

  const activeDragItem = useMemo(() => {
    if (!activeItemId) {
      return null;
    }
    return findItemPosition(localDays, activeItemId)?.item ?? null;
  }, [activeItemId, localDays]);

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

  const commitReorder = async (
    itemId: string,
    targetDayId: string,
    targetSortOrder: number,
    fallbackDays: ItineraryDayApi[]
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
    await commitReorder(itemId, finalPos.dayId, finalPos.itemIndex + 1, before);
  };

  const startEdit = (itemId: string, title: string, note?: string) => {
    setEditingItemId(itemId);
    setEditingTitle(title);
    setEditingNote(note ?? "");
  };

  const cancelEdit = () => {
    setEditingItemId(null);
    setEditingTitle("");
    setEditingNote("");
  };

  const saveEdit = async (itemId: string, version: number) => {
    const nextTitle = editingTitle.trim();
    if (!nextTitle) {
      return;
    }
    await patchItem.mutateAsync({
      itemId,
      version,
      input: {
        title: nextTitle,
        note: editingNote.trim()
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

    await commitReorder(activeId, afterPos.dayId, afterPos.itemIndex + 1, before);
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
      <SurfaceCard
        eyebrow={t("nav.itinerary")}
        title={t("itinerary.title")}
        action={
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
                </div>
                <SortableContext items={day.items.map((item) => item.id)} strategy={verticalListSortingStrategy}>
                  <div className="mt-5 grid gap-4">
                    {day.items.map((item, itemIndex) => (
                      <SortableItemCard
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
                              </div>
                            ) : (
                              <>
                                <p className="text-sm font-semibold text-ink">{item.title}</p>
                                <p className="mt-1 text-sm text-ink/60">
                                  {typeLabels[item.itemType] ?? item.itemType}
                                </p>
                              </>
                            )}
                          </div>
                          <div className="flex flex-wrap items-center gap-2">
                            <StatusPill tone="accent">{item.allDay ? t("itinerary.allDay") : t("itinerary.startTime")}</StatusPill>
                            {editingItemId === item.id ? (
                              <>
                                <button
                                  className="rounded-full border border-ink/15 px-3 py-1 text-xs font-medium text-ink disabled:opacity-40"
                                  disabled={patchItem.isPending}
                                  onClick={() => {
                                    void saveEdit(item.id, item.version);
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
                                  startEdit(item.id, item.title, item.note);
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
                      </SortableItemCard>
                    ))}
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
