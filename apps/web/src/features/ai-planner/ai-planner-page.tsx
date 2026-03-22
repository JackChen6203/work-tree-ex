import { useState } from "react";
import { useParams } from "react-router-dom";
import { SurfaceCard } from "../../components/surface-card";
import { StatusPill } from "../../components/status-pill";
import { useCreateAiPlanMutation, useAdoptAiPlanMutation, useAiPlanQuery, useAiPlansQuery } from "../../lib/queries";
import { useUiStore } from "../../store/ui-store";

export function AiPlannerPage() {
  const { tripId = "" } = useParams();
  const pushToast = useUiStore((state) => state.pushToast);
  const [selectedPlanId, setSelectedPlanId] = useState("");
  const { data: drafts = [], isLoading } = useAiPlansQuery(tripId);
  const { data: selectedPlan, isLoading: detailLoading } = useAiPlanQuery(tripId, selectedPlanId);
  const createPlan = useCreateAiPlanMutation(tripId);
  const adoptPlan = useAdoptAiPlanMutation(tripId);

  const runPlan = async () => {
    await createPlan.mutateAsync({
      providerConfigId: "default-provider",
      title: "Kyoto Budget-aware Draft",
      constraints: {
        totalBudget: 70500,
        currency: "JPY",
        pace: "balanced",
        transportPreference: "transit",
        mustVisit: ["清水寺", "嵐山"],
        avoid: ["過度換乘"]
      }
    });
    pushToast("AI draft generated");
  };

  const onAdopt = async (planId: string, status: "valid" | "warning" | "invalid") => {
    const confirmWarnings =
      status === "warning"
        ? window.confirm("This draft has warnings. Confirm to adopt it anyway?")
        : false;

    if (status === "warning" && !confirmWarnings) {
      pushToast("Adoption cancelled. Review warnings before confirming.");
      return;
    }

    const result = await adoptPlan.mutateAsync({ planId, confirmWarnings });
    pushToast(result.adopted ? "Draft adopted" : "Draft not adopted");
  };

  return (
    <div className="grid gap-6 xl:grid-cols-[0.95fr_1.05fr]">
      <SurfaceCard
        eyebrow="AI Planner"
        title="Planning constraints"
        action={
          <button
            className="rounded-full bg-ink px-4 py-2 text-sm font-medium text-sand"
            disabled={createPlan.isPending}
            onClick={() => {
              void runPlan();
            }}
            type="button"
          >
            {createPlan.isPending ? "Planning..." : "Run planning"}
          </button>
        }
      >
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
        {isLoading ? <div className="rounded-[24px] bg-sand p-4 text-sm text-ink/65">Loading AI drafts...</div> : null}
        {!isLoading && drafts.length === 0 ? <div className="rounded-[24px] bg-sand p-4 text-sm text-ink/65">No drafts yet. Run planning to generate one.</div> : null}
        <div className="grid gap-4">
          {drafts.map((draft) => (
            <div key={draft.id} className="rounded-[28px] border border-ink/10 bg-white p-5">
              <div className="flex flex-wrap items-center justify-between gap-3">
                <div>
                  <p className="font-display text-2xl font-bold text-ink">{draft.title}</p>
                  <p className="mt-2 text-sm text-ink/65">{draft.summary}</p>
                </div>
                <StatusPill tone={draft.status === "invalid" ? "danger" : draft.status === "warning" ? "neutral" : "success"}>{draft.status}</StatusPill>
              </div>
              <p className="mt-3 text-sm text-ink/60">
                Est. {Math.round(draft.totalEstimated).toLocaleString()} {draft.currency} / Budget {Math.round(draft.budget).toLocaleString()} {draft.currency}
              </p>
              <div className="mt-4 flex flex-wrap gap-2">
                {draft.warnings.map((warning) => (
                  <StatusPill key={warning} tone="danger">
                    {warning}
                  </StatusPill>
                ))}
              </div>
              <button
                className="mt-5 rounded-full bg-pine px-4 py-2 text-sm font-medium text-white disabled:cursor-not-allowed disabled:bg-ink/35"
                disabled={draft.status === "invalid" || adoptPlan.isPending}
                onClick={() => {
                  void onAdopt(draft.id, draft.status);
                }}
                type="button"
              >
                {adoptPlan.isPending ? "Adopting..." : "Adopt via server transaction"}
              </button>
              <button
                className="mt-3 rounded-full border border-ink/20 px-4 py-2 text-sm font-medium text-ink"
                onClick={() => {
                  setSelectedPlanId(draft.id);
                }}
                type="button"
              >
                Inspect detail
              </button>
            </div>
          ))}
        </div>
        {selectedPlanId ? (
          <div className="mt-6 rounded-[24px] border border-ink/10 bg-sand p-4">
            <p className="text-xs uppercase tracking-[0.22em] text-ink/45">Draft detail</p>
            {detailLoading ? <p className="mt-2 text-sm text-ink/65">Loading selected draft...</p> : null}
            {selectedPlan ? (
              <>
                <p className="mt-2 text-sm font-semibold text-ink">{selectedPlan.title}</p>
                <p className="mt-1 text-sm text-ink/65">Created at: {new Date(selectedPlan.createdAt).toLocaleString()}</p>
                <p className="mt-1 text-sm text-ink/65">Summary: {selectedPlan.summary}</p>
                <p className="mt-1 text-sm text-ink/65">
                  Budget fit: {Math.round(selectedPlan.totalEstimated).toLocaleString()} / {Math.round(selectedPlan.budget).toLocaleString()} {selectedPlan.currency}
                </p>
              </>
            ) : null}
          </div>
        ) : null}
      </SurfaceCard>
    </div>
  );
}
