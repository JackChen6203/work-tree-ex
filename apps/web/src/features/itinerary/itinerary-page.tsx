import { SurfaceCard } from "../../components/surface-card";
import { StatusPill } from "../../components/status-pill";
import { itineraryDays } from "../../lib/mock-data";

export function ItineraryPage() {
  return (
    <div className="grid gap-6">
      <SurfaceCard
        eyebrow="Itinerary Module"
        title="Daily timeline"
        action={<button className="rounded-full bg-ink px-4 py-2 text-sm font-medium text-sand">Add item</button>}
      >
        <div className="grid gap-5">
          {itineraryDays.map((day) => (
            <div key={day.id} className="rounded-[28px] border border-ink/10 bg-sand/70 p-5">
              <div className="flex flex-wrap items-center justify-between gap-4">
                <div>
                  <p className="text-xs uppercase tracking-[0.22em] text-ink/45">
                    {day.label} . {day.date}
                  </p>
                  <h3 className="mt-2 font-display text-2xl font-bold text-ink">{day.summary}</h3>
                </div>
                <StatusPill tone="neutral">Versioned reorder</StatusPill>
              </div>
              <div className="mt-5 grid gap-4">
                {day.items.map((item) => (
                  <div key={item.id} className="rounded-[24px] bg-white p-4">
                    <div className="flex flex-wrap items-start justify-between gap-3">
                      <div>
                        <p className="text-sm font-semibold text-ink">{item.title}</p>
                        <p className="mt-1 text-sm text-ink/60">
                          {item.time} . {item.location}
                        </p>
                      </div>
                      <div className="flex flex-wrap gap-2">
                        <StatusPill tone="neutral">{item.transit}</StatusPill>
                        <StatusPill tone="accent">{item.cost}</StatusPill>
                      </div>
                    </div>
                    {item.warning ? <p className="mt-3 text-sm text-coral">{item.warning}</p> : null}
                    {item.draftDiff ? <p className="mt-2 text-sm text-pine">{item.draftDiff}</p> : null}
                  </div>
                ))}
              </div>
            </div>
          ))}
        </div>
      </SurfaceCard>
    </div>
  );
}
