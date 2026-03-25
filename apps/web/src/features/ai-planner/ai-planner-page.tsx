import { useState } from "react";
import { useForm } from "react-hook-form";
import { useParams } from "react-router-dom";
import { SurfaceCard } from "../../components/surface-card";
import { StatusPill } from "../../components/status-pill";
import { useCreateAiPlanMutation, useAdoptAiPlanMutation, useAiPlanQuery, useAiPlansQuery, useMyPreferencesQuery, useBudgetProfileQuery } from "../../lib/queries";
import { useUiStore } from "../../store/ui-store";
import { useI18n } from "../../lib/i18n";

interface AiConstraintFormValues {
  totalBudget: number;
  currency: string;
  pace: "relaxed" | "balanced" | "packed";
  transportPreference: "walk" | "transit" | "taxi" | "mixed";
  wakePattern: "early" | "normal" | "late";
  poiDensity: "sparse" | "medium" | "dense";
  mustVisit: string;
  avoid: string;
}

export function AiPlannerPage() {
  const { tripId = "" } = useParams();
  const { t } = useI18n();
  const pushToast = useUiStore((state) => state.pushToast);
  const openConfirmModal = useUiStore((state) => state.openConfirmModal);
  const [selectedPlanId, setSelectedPlanId] = useState("");
  const { data: drafts = [], isLoading } = useAiPlansQuery(tripId);
  const { data: selectedPlan, isLoading: detailLoading } = useAiPlanQuery(tripId, selectedPlanId);
  const createPlan = useCreateAiPlanMutation(tripId);
  const adoptPlan = useAdoptAiPlanMutation(tripId);
  const { data: preferences } = useMyPreferencesQuery();
  const { data: budgetProfile } = useBudgetProfileQuery(tripId);

  const form = useForm<AiConstraintFormValues>({
    defaultValues: {
      totalBudget: budgetProfile?.totalBudget ?? 70500,
      currency: budgetProfile?.currency ?? "JPY",
      pace: (preferences?.tripPace as "relaxed" | "balanced" | "packed") ?? "balanced",
      transportPreference: (preferences?.transportPreference as "walk" | "transit" | "taxi" | "mixed") ?? "transit",
      wakePattern: (preferences?.wakePattern as "early" | "normal" | "late") ?? "normal",
      poiDensity: "medium",
      mustVisit: "",
      avoid: ""
    }
  });

  const runPlan = form.handleSubmit(async (values) => {
    await createPlan.mutateAsync({
      providerConfigId: "default-provider",
      title: t("ai.title"),
      constraints: {
        totalBudget: values.totalBudget,
        currency: values.currency,
        pace: values.pace,
        transportPreference: values.transportPreference,
        mustVisit: values.mustVisit.split(",").map((s) => s.trim()).filter(Boolean),
        avoid: values.avoid.split(",").map((s) => s.trim()).filter(Boolean)
      }
    });
    pushToast(t("ai.generated"));
  });

  const onAdopt = (planId: string, status: "valid" | "warning" | "invalid") => {
    const confirmWarnings = status === "warning";
    openConfirmModal({
      title: t("ai.adopt"),
      description: t("ai.adoptConfirmDescription"),
      confirmLabel: t("ai.adopt"),
      cancelLabel: t("common.cancel"),
      tone: confirmWarnings ? "danger" : "neutral",
      onConfirm: async () => {
        const result = await adoptPlan.mutateAsync({ planId, confirmWarnings });
        pushToast(result.adopted ? t("ai.adopted") : t("common.cancel"));
      }
    });
  };

  // Compare two drafts
  const [compareIds, setCompareIds] = useState<[string, string] | null>(null);
  const compareDrafts = compareIds
    ? [drafts.find((d) => d.id === compareIds[0]), drafts.find((d) => d.id === compareIds[1])]
    : null;

  return (
    <div className="grid gap-6 xl:grid-cols-[0.95fr_1.05fr]">
      <SurfaceCard
        eyebrow={t("nav.aiPlanner")}
        title={t("ai.constraints")}
        action={
          <button
            className="rounded-full bg-ink px-4 py-2 text-sm font-medium text-sand"
            disabled={createPlan.isPending}
            onClick={() => {
              void runPlan();
            }}
            type="button"
          >
            {createPlan.isPending ? t("ai.generating") : t("ai.generate")}
          </button>
        }
      >
        <form className="grid gap-4 sm:grid-cols-2" onSubmit={runPlan}>
          <label className="block">
            <span className="mb-2 block text-sm font-medium text-ink">{t("ai.totalBudget")}</span>
            <input className="w-full rounded-2xl border border-ink/10 bg-sand px-4 py-3" type="number" min={0} {...form.register("totalBudget", { valueAsNumber: true })} />
          </label>
          <label className="block">
            <span className="mb-2 block text-sm font-medium text-ink">{t("trip.currency")}</span>
            <input className="w-full rounded-2xl border border-ink/10 bg-sand px-4 py-3" {...form.register("currency")} />
          </label>
          <label className="block">
            <span className="mb-2 block text-sm font-medium text-ink">{t("ai.pace")}</span>
            <select className="w-full rounded-2xl border border-ink/10 bg-sand px-4 py-3" {...form.register("pace")}>
              <option value="relaxed">{t("settings.relaxed")}</option>
              <option value="balanced">{t("settings.balanced")}</option>
              <option value="packed">{t("settings.packed")}</option>
            </select>
          </label>
          <label className="block">
            <span className="mb-2 block text-sm font-medium text-ink">{t("ai.transport")}</span>
            <select className="w-full rounded-2xl border border-ink/10 bg-sand px-4 py-3" {...form.register("transportPreference")}>
              <option value="walk">{t("settings.walkTransport")}</option>
              <option value="transit">{t("settings.transitTransport")}</option>
              <option value="taxi">{t("settings.taxiTransport")}</option>
              <option value="mixed">{t("settings.mixedTransport")}</option>
            </select>
          </label>
          <label className="block">
            <span className="mb-2 block text-sm font-medium text-ink">{t("settings.wakePattern")}</span>
            <select className="w-full rounded-2xl border border-ink/10 bg-sand px-4 py-3" {...form.register("wakePattern")}>
              <option value="early">{t("settings.early")}</option>
              <option value="normal">{t("settings.normal")}</option>
              <option value="late">{t("settings.late")}</option>
            </select>
          </label>
          <label className="block">
            <span className="mb-2 block text-sm font-medium text-ink">POI {t("ai.pace")}</span>
            <select className="w-full rounded-2xl border border-ink/10 bg-sand px-4 py-3" {...form.register("poiDensity")}>
              <option value="sparse">{t("settings.relaxed")}</option>
              <option value="medium">{t("settings.balanced")}</option>
              <option value="dense">{t("settings.packed")}</option>
            </select>
          </label>
          <label className="block sm:col-span-2">
            <span className="mb-2 block text-sm font-medium text-ink">{t("ai.mustVisit")}</span>
            <input className="w-full rounded-2xl border border-ink/10 bg-sand px-4 py-3" placeholder={t("ai.mustVisit")} {...form.register("mustVisit")} />
            <p className="mt-1 text-xs text-ink/50">{t("settings.foodPreferenceHint")}</p>
          </label>
          <label className="block sm:col-span-2">
            <span className="mb-2 block text-sm font-medium text-ink">{t("ai.avoid")}</span>
            <input className="w-full rounded-2xl border border-ink/10 bg-sand px-4 py-3" placeholder={t("ai.avoid")} {...form.register("avoid")} />
            <p className="mt-1 text-xs text-ink/50">{t("settings.avoidTagsHint")}</p>
          </label>
        </form>
        <div className="mt-5 rounded-[24px] bg-ink p-5 text-sand">
          <p className="text-xs uppercase tracking-[0.22em] text-sand/55">{t("ai.status")}</p>
          <h3 className="mt-2 font-display text-2xl font-bold">{createPlan.isPending ? t("ai.generating") : t("ai.generate")}</h3>
          <p className="mt-2 text-sm text-sand/70">{t("ai.adoptConfirmDescription")}</p>
        </div>
      </SurfaceCard>
      <SurfaceCard eyebrow={t("ai.drafts")} title={t("ai.drafts")}>
        {isLoading ? <div className="rounded-[24px] bg-sand p-4 text-sm text-ink/65">{t("common.loading")}</div> : null}
        {!isLoading && drafts.length === 0 ? <div className="rounded-[24px] bg-sand p-4 text-sm text-ink/65">{t("ai.noDrafts")}</div> : null}

        {/* Compare toggle */}
        {drafts.length >= 2 ? (
          <div className="mb-4 flex items-center gap-3">
            <button
              className={`rounded-full px-4 py-2 text-xs font-medium transition ${compareIds ? "bg-pine text-white" : "border border-ink/15 bg-sand text-ink"}`}
              onClick={() => {
                if (compareIds) {
                  setCompareIds(null);
                } else if (drafts.length >= 2) {
                  setCompareIds([drafts[0].id, drafts[1].id]);
                }
              }}
              type="button"
            >
              {compareIds ? t("common.cancel") : t("ai.compareDrafts")}
            </button>
          </div>
        ) : null}

        {/* Side-by-side compare */}
        {compareDrafts && compareDrafts[0] && compareDrafts[1] ? (
          <div className="mb-6 grid gap-4 md:grid-cols-2">
            {compareDrafts.map((draft) => draft ? (
              <div key={draft.id} className="rounded-[24px] border border-pine/20 bg-pine/5 p-4">
                <p className="font-display text-lg font-bold text-ink">{draft.title}</p>
                <p className="mt-1 text-sm text-ink/65">{draft.summary}</p>
                <div className="mt-3 space-y-1 text-xs text-ink/60">
                  <p>{t("budget.planned")} {Math.round(draft.totalEstimated).toLocaleString()} {draft.currency}</p>
                </div>
                <StatusPill tone={draft.status === "invalid" ? "danger" : draft.status === "warning" ? "neutral" : "success"}>{draft.status}</StatusPill>
              </div>
            ) : null)}
          </div>
        ) : null}

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
                {t("budget.planned")} {Math.round(draft.totalEstimated).toLocaleString()} {draft.currency} / {t("budget.totalBudget")} {Math.round(draft.budget).toLocaleString()} {draft.currency}
              </p>
              <div className="mt-4 flex flex-wrap gap-2">
                {draft.warnings.map((warning) => (
                  <StatusPill key={warning} tone="danger">
                    {warning}
                  </StatusPill>
                ))}
              </div>
              <div className="mt-5 flex flex-wrap gap-3">
                <button
                  className="rounded-full bg-pine px-4 py-2 text-sm font-medium text-white disabled:cursor-not-allowed disabled:bg-ink/35"
                  disabled={draft.status === "invalid" || adoptPlan.isPending}
                  onClick={() => {
                    onAdopt(draft.id, draft.status);
                  }}
                  type="button"
                >
                  {adoptPlan.isPending ? t("ai.adopting") : t("ai.adopt")}
                </button>
                <button
                  className="rounded-full border border-ink/20 px-4 py-2 text-sm font-medium text-ink"
                  onClick={() => {
                    setSelectedPlanId(draft.id);
                  }}
                  type="button"
                >
                  {t("ai.viewDetails")}
                </button>
              </div>
            </div>
          ))}
        </div>
        {selectedPlanId ? (
          <div className="mt-6 rounded-[24px] border border-ink/10 bg-sand p-4">
            <p className="text-xs uppercase tracking-[0.22em] text-ink/45">{t("ai.viewDetails")}</p>
            {detailLoading ? <p className="mt-2 text-sm text-ink/65">{t("common.loading")}</p> : null}
            {selectedPlan ? (
              <>
                <p className="mt-2 text-sm font-semibold text-ink">{selectedPlan.title}</p>
                <p className="mt-1 text-sm text-ink/65">{new Date(selectedPlan.createdAt).toLocaleString()}</p>
                <p className="mt-1 text-sm text-ink/65">{selectedPlan.summary}</p>
                <p className="mt-1 text-sm text-ink/65">
                  {t("budget.planned")} {Math.round(selectedPlan.totalEstimated).toLocaleString()} / {t("budget.totalBudget")} {Math.round(selectedPlan.budget).toLocaleString()} {selectedPlan.currency}
                </p>
              </>
            ) : null}
          </div>
        ) : null}
      </SurfaceCard>
    </div>
  );
}
