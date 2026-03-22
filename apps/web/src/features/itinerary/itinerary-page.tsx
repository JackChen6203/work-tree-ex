import { useState } from "react";
import { useParams } from "react-router-dom";
import { SurfaceCard } from "../../components/surface-card";
import { StatusPill } from "../../components/status-pill";
import { useCreateItineraryItemMutation, useDeleteItineraryItemMutation, useItineraryDaysQuery, usePatchItineraryItemMutation, useReorderItineraryItemsMutation } from "../../lib/queries";
import { useUiStore } from "../../store/ui-store";

export function ItineraryPage() {
  const { tripId = "" } = useParams();
  const pushToast = useUiStore((state) => state.pushToast);
  const { data: days = [], isLoading } = useItineraryDaysQuery(tripId);
  const createItem = useCreateItineraryItemMutation(tripId);
  const deleteItem = useDeleteItineraryItemMutation(tripId);
  const patchItem = usePatchItineraryItemMutation(tripId);
  const reorderItems = useReorderItineraryItemsMutation(tripId);
  const [editingItemId, setEditingItemId] = useState<string | null>(null);
  const [editingTitle, setEditingTitle] = useState("");
  const [editingNote, setEditingNote] = useState("");

  const addItem = async () => {
    const targetDay = days[0]?.dayId ?? "day-1";
    await createItem.mutateAsync({
      dayId: targetDay,
      title: "新行程項目",
      itemType: "custom",
      allDay: false,
      note: "由 itinerary page 建立"
    });
    pushToast("Itinerary item created");
  };

  const removeItem = async (itemId: string) => {
    await deleteItem.mutateAsync(itemId);
    pushToast("Itinerary item removed");
  };

  const moveItem = async (dayId: string, itemId: string, targetSortOrder: number) => {
    await reorderItems.mutateAsync({
      operations: [
        {
          itemId,
          targetDayId: dayId,
          targetSortOrder: targetSortOrder
        }
      ]
    });
    pushToast("Itinerary order updated");
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
      pushToast("Title is required");
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
    pushToast("Itinerary item updated");
  };

  return (
    <div className="grid gap-6">
      <SurfaceCard
        eyebrow="Itinerary Module"
        title="Daily timeline"
        action={
          <button
            className="rounded-full bg-ink px-4 py-2 text-sm font-medium text-sand"
            disabled={createItem.isPending}
            onClick={() => {
              void addItem();
            }}
            type="button"
          >
            {createItem.isPending ? "Adding..." : "Add item"}
          </button>
        }
      >
        {isLoading ? <div className="mb-4 rounded-[24px] bg-sand p-4 text-sm text-ink/65">Loading itinerary...</div> : null}
        <div className="grid gap-5">
          {days.map((day, index) => (
            <div key={day.dayId} className="rounded-[28px] border border-ink/10 bg-sand/70 p-5">
              <div className="flex flex-wrap items-center justify-between gap-4">
                <div>
                  <p className="text-xs uppercase tracking-[0.22em] text-ink/45">
                    Day {index + 1} . {day.date}
                  </p>
                  <h3 className="mt-2 font-display text-2xl font-bold text-ink">{day.items.length} items planned</h3>
                </div>
                <StatusPill tone="neutral">Versioned reorder</StatusPill>
              </div>
              <div className="mt-5 grid gap-4">
                {day.items.map((item, itemIndex) => (
                  <div key={item.id} className="rounded-[24px] bg-white p-4">
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
                              {item.itemType} . sort #{item.sortOrder}
                            </p>
                          </>
                        )}
                      </div>
                      <div className="flex flex-wrap items-center gap-2">
                        <StatusPill tone="neutral">v{item.version}</StatusPill>
                        <StatusPill tone="accent">{item.allDay ? "all-day" : "timed"}</StatusPill>
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
                              Save
                            </button>
                            <button
                              className="rounded-full border border-ink/15 px-3 py-1 text-xs font-medium text-ink"
                              onClick={cancelEdit}
                              type="button"
                            >
                              Cancel
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
                            Edit
                          </button>
                        )}
                        <button
                          className="rounded-full border border-ink/15 px-3 py-1 text-xs font-medium text-ink disabled:opacity-40"
                          disabled={reorderItems.isPending || editingItemId === item.id || itemIndex === 0}
                          onClick={() => {
                            void moveItem(day.dayId, item.id, Math.max(item.sortOrder-1, 1));
                          }}
                          type="button"
                        >
                          Up
                        </button>
                        <button
                          className="rounded-full border border-ink/15 px-3 py-1 text-xs font-medium text-ink disabled:opacity-40"
                          disabled={reorderItems.isPending || editingItemId === item.id || itemIndex === day.items.length - 1}
                          onClick={() => {
                            void moveItem(day.dayId, item.id, item.sortOrder + 1);
                          }}
                          type="button"
                        >
                          Down
                        </button>
                        <button
                          className="rounded-full border border-ink/15 px-3 py-1 text-xs font-medium text-ink"
                          disabled={deleteItem.isPending || reorderItems.isPending || editingItemId === item.id}
                          onClick={() => {
                            void removeItem(item.id);
                          }}
                          type="button"
                        >
                          Remove
                        </button>
                      </div>
                    </div>
                    {editingItemId === item.id ? null : item.note ? <p className="mt-3 text-sm text-ink/70">{item.note}</p> : null}
                  </div>
                ))}
              </div>
            </div>
          ))}
          {!isLoading && days.length === 0 ? <div className="rounded-[24px] bg-sand p-4 text-sm text-ink/65">No itinerary days found.</div> : null}
        </div>
      </SurfaceCard>
    </div>
  );
}
