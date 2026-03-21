import { SurfaceCard } from "../../components/surface-card";
import { StatusPill } from "../../components/status-pill";
import { aiDrafts } from "../../lib/mock-data";

export function AiPlannerPage() {
  return (
    <div className="grid gap-6 xl:grid-cols-[0.95fr_1.05fr]">
      <SurfaceCard eyebrow="AI Planner" title="Planning constraints">
        <div className="grid gap-4 sm:grid-cols-2">
          {[
            "總預算 JPY 70,500",
            "偏好慢節奏與咖啡",
            "避免連續早起",
            "大眾運輸優先",
            "必去：清水寺、嵐山",
            "禁忌：過度換乘"
          ].map((rule) => (
            <div key={rule} className="rounded-[24px] bg-sand p-4 text-sm text-ink/75">
              {rule}
            </div>
          ))}
        </div>
        <div className="mt-5 rounded-[24px] bg-ink p-5 text-sand">
          <p className="text-xs uppercase tracking-[0.22em] text-sand/55">Planning job</p>
          <h3 className="mt-2 font-display text-2xl font-bold">Validation pipeline running</h3>
          <p className="mt-2 text-sm text-sand/70">Structured output parsing, temporal checks, budget checks, and draft warnings remain explicit before adoption.</p>
        </div>
      </SurfaceCard>
      <SurfaceCard eyebrow="Draft Compare" title="Candidate plans">
        <div className="grid gap-4">
          {aiDrafts.map((draft) => (
            <div key={draft.id} className="rounded-[28px] border border-ink/10 bg-white p-5">
              <div className="flex flex-wrap items-center justify-between gap-3">
                <div>
                  <p className="font-display text-2xl font-bold text-ink">{draft.name}</p>
                  <p className="mt-2 text-sm text-ink/65">{draft.summary}</p>
                </div>
                <StatusPill tone="accent">Score {draft.score}</StatusPill>
              </div>
              <div className="mt-4 flex flex-wrap gap-2">
                {draft.warnings.map((warning) => (
                  <StatusPill key={warning} tone="danger">
                    {warning}
                  </StatusPill>
                ))}
              </div>
              <button className="mt-5 rounded-full bg-pine px-4 py-2 text-sm font-medium text-white">Adopt via server transaction</button>
            </div>
          ))}
        </div>
      </SurfaceCard>
    </div>
  );
}
