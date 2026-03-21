import { SurfaceCard } from "../../components/surface-card";
import { StatusPill } from "../../components/status-pill";
import { budgetCategories } from "../../lib/mock-data";

export function BudgetPage() {
  const estimated = budgetCategories.reduce((sum, item) => sum + item.estimated, 0);
  const actual = budgetCategories.reduce((sum, item) => sum + item.actual, 0);

  return (
    <div className="grid gap-6 lg:grid-cols-[0.9fr_1.1fr]">
      <SurfaceCard eyebrow="Budget Module" title="Cost guardrails">
        <div className="grid gap-4 sm:grid-cols-3">
          <div className="rounded-[24px] bg-sand p-4">
            <p className="text-sm text-ink/60">Total budget</p>
            <p className="mt-2 font-display text-3xl font-bold text-ink">JPY {estimated.toLocaleString()}</p>
          </div>
          <div className="rounded-[24px] bg-sand p-4">
            <p className="text-sm text-ink/60">Actual spend</p>
            <p className="mt-2 font-display text-3xl font-bold text-ink">JPY {actual.toLocaleString()}</p>
          </div>
          <div className="rounded-[24px] bg-sand p-4">
            <p className="text-sm text-ink/60">Per day target</p>
            <p className="mt-2 font-display text-3xl font-bold text-ink">JPY 14,100</p>
          </div>
        </div>
        <button className="mt-5 rounded-full bg-coral px-5 py-3 text-sm font-medium text-white">Generate itinerary from budget</button>
      </SurfaceCard>
      <SurfaceCard eyebrow="Breakdown" title="Estimated vs actual">
        <div className="space-y-4">
          {budgetCategories.map((category) => {
            const variance = category.actual - category.estimated;
            return (
              <div key={category.name} className="rounded-[24px] bg-white p-4">
                <div className="flex flex-wrap items-center justify-between gap-3">
                  <div>
                    <p className="font-medium text-ink">{category.name}</p>
                    <p className="text-sm text-ink/55">Est. {category.estimated.toLocaleString()} / Actual {category.actual.toLocaleString()}</p>
                  </div>
                  <StatusPill tone={variance > 0 ? "danger" : "success"}>
                    {variance > 0 ? `+${variance.toLocaleString()}` : `${variance.toLocaleString()}`}
                  </StatusPill>
                </div>
              </div>
            );
          })}
        </div>
      </SurfaceCard>
    </div>
  );
}
