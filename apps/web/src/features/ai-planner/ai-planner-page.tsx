import { useEffect, useMemo, useRef, useState } from "react";
import { useForm } from "react-hook-form";
import { useParams } from "react-router-dom";
import { useQueryClient } from "@tanstack/react-query";
import { SurfaceCard } from "../../components/surface-card";
import { StatusPill } from "../../components/status-pill";
import {
  useAdoptAiPlanMutation,
  useAiPlanQuery,
  useAiPlansQuery,
  useBudgetProfileQuery,
  useCreateAiPlanMutation,
  useMyPreferencesQuery
} from "../../lib/queries";
import { useUiStore } from "../../store/ui-store";
import { useI18n } from "../../lib/i18n";

interface AiConstraintFormValues {
  totalBudget: number;
  currency: string;
  pace: "relaxed" | "balanced" | "packed";
  transportPreference: "walk" | "transit" | "taxi" | "mixed";
  wakePattern: "early" | "normal" | "late";
  poiDensity: "sparse" | "medium" | "dense";
  mustVisit: string[];
  avoid: string[];
}

interface PlanningJobState {
  jobId: string;
  status: "queued" | "running" | "succeeded" | "failed";
  baselineDraftCount: number;
  pollCount: number;
  acceptedAt: string;
}

const wakePatternScale: Array<AiConstraintFormValues["wakePattern"]> = ["early", "normal", "late"];
const poiDensityScale: Array<AiConstraintFormValues["poiDensity"]> = ["sparse", "medium", "dense"];

export function AiPlannerPage() {
  const { tripId = "" } = useParams();
  const { t } = useI18n();
  const queryClient = useQueryClient();
  const pushToast = useUiStore((state) => state.pushToast);
  const openAdoptDraftModal = useUiStore((state) => state.openAdoptDraftModal);
  const [selectedPlanId, setSelectedPlanId] = useState("");
  const [mustVisitInput, setMustVisitInput] = useState("");
  const [avoidInput, setAvoidInput] = useState("");
  const [planningJob, setPlanningJob] = useState<PlanningJobState | null>(null);
  const defaultsHydratedRef = useRef(false);
  const { data: drafts = [], isLoading } = useAiPlansQuery(tripId);
  const { data: selectedPlan, isLoading: detailLoading } = useAiPlanQuery(tripId, selectedPlanId);
  const createPlan = useCreateAiPlanMutation(tripId);
  const adoptPlan = useAdoptAiPlanMutation(tripId);
  const { data: preferences } = useMyPreferencesQuery();
  const { data: budgetProfile } = useBudgetProfileQuery(tripId);

  const form = useForm<AiConstraintFormValues>({
    defaultValues: {
      totalBudget: 70500,
      currency: "JPY",
      pace: "balanced",
      transportPreference: "transit",
      wakePattern: "normal",
      poiDensity: "medium",
      mustVisit: [],
      avoid: []
    }
  });

  useEffect(() => {
    if (defaultsHydratedRef.current || (!preferences && !budgetProfile)) {
      return;
    }

    form.reset({
      totalBudget: budgetProfile?.totalBudget ?? 70500,
      currency: budgetProfile?.currency ?? "JPY",
      pace: (preferences?.tripPace as "relaxed" | "balanced" | "packed") ?? "balanced",
      transportPreference: (preferences?.transportPreference as "walk" | "transit" | "taxi" | "mixed") ?? "transit",
      wakePattern: (preferences?.wakePattern as "early" | "normal" | "late") ?? "normal",
      poiDensity: "medium",
      mustVisit: [],
      avoid: []
    });
    defaultsHydratedRef.current = true;
  }, [budgetProfile, form, preferences]);

  useEffect(() => {
    if (!planningJob || planningJob.status === "succeeded" || planningJob.status === "failed") {
      return;
    }

    const timer = window.setInterval(() => {
      setPlanningJob((current) => {
        if (!current || current.status === "succeeded" || current.status === "failed") {
          return current;
        }

        if (current.pollCount >= 20) {
          return { ...current, status: "failed" };
        }

        return {
          ...current,
          status: "running",
          pollCount: current.pollCount + 1
        };
      });
      void queryClient.invalidateQueries({ queryKey: ["ai-plans", tripId] });
    }, 3000);

    return () => {
      window.clearInterval(timer);
    };
  }, [planningJob, queryClient, tripId]);

  useEffect(() => {
    if (!planningJob || planningJob.status === "succeeded" || planningJob.status === "failed") {
      return;
    }

    if (drafts.length > planningJob.baselineDraftCount) {
      setPlanningJob((current) => (current ? { ...current, status: "succeeded" } : current));
      pushToast(t("ai.generated"));
    }
  }, [drafts.length, planningJob, pushToast, t]);

  const runPlan = form.handleSubmit(async (values) => {
    const result = await createPlan.mutateAsync({
      providerConfigId: "default-provider",
      title: t("ai.title"),
      constraints: {
        totalBudget: values.totalBudget,
        currency: values.currency,
        pace: values.pace,
        transportPreference: values.transportPreference,
        wakePattern: values.wakePattern,
        poiDensity: values.poiDensity,
        mustVisit: values.mustVisit,
        avoid: values.avoid
      }
    });

    setPlanningJob({
      jobId: result.jobId,
      status: result.status === "queued" ? "queued" : "running",
      baselineDraftCount: drafts.length,
      pollCount: 0,
      acceptedAt: result.acceptedAt
    });
  });

  const onAdopt = (planId: string, status: "valid" | "warning" | "invalid") => {
    if (status === "invalid") {
      return;
    }

    const target = drafts.find((draft) => draft.id === planId);
    if (!target) {
      return;
    }

    openAdoptDraftModal({
      draftId: planId,
      tripId,
      draftTitle: target.title,
      hasWarnings: status === "warning",
      onConfirm: async (confirmWarnings) => {
        const result = await adoptPlan.mutateAsync({ planId, confirmWarnings });
        pushToast(result.adopted ? t("ai.adopted") : t("common.cancel"));
      }
    });
  };

  const addTag = (field: "mustVisit" | "avoid", input: string, clearInput: () => void) => {
    const value = input.trim();
    if (!value) {
      return;
    }

    const values = form.getValues(field);
    if (values.includes(value)) {
      clearInput();
      return;
    }

    form.setValue(field, [...values, value], { shouldDirty: true });
    clearInput();
  };

  const removeTag = (field: "mustVisit" | "avoid", value: string) => {
    const values = form.getValues(field);
    form.setValue(
      field,
      values.filter((item) => item !== value),
      { shouldDirty: true }
    );
  };

  const mustVisitTags = form.watch("mustVisit");
  const avoidTags = form.watch("avoid");
  const wakePattern = form.watch("wakePattern");
  const poiDensity = form.watch("poiDensity");

  const wakeIndex = Math.max(0, wakePatternScale.indexOf(wakePattern));
  const densityIndex = Math.max(0, poiDensityScale.indexOf(poiDensity));

  const compareDrafts = useMemo(() => {
    if (drafts.length < 2) {
      return null;
    }
    return [drafts[0], drafts[1]];
  }, [drafts]);

  const jobProgress = useMemo(() => {
    if (!planningJob) {
      return 0;
    }

    if (planningJob.status === "succeeded" || planningJob.status === "failed") {
      return 100;
    }

    if (planningJob.status === "queued") {
      return 12;
    }

    return Math.min(94, 12 + planningJob.pollCount * 4);
  }, [planningJob]);

  return (
    <div className="grid gap-6 xl:grid-cols-[0.95fr_1.05fr]">
      <SurfaceCard
        eyebrow={t("nav.aiPlanner")}
        title={t("ai.constraints")}
        action={
          <button
            className="rounded-full bg-ink px-4 py-2 text-sm font-medium text-sand"
            disabled={createPlan.isPending || (planningJob?.status === "queued" || planningJob?.status === "running")}
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
            <input className="w-full rounded-2xl border border-ink/10 bg-sand px-4 py-3" min={0} type="number" {...form.register("totalBudget", { valueAsNumber: true })} />
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

          <div className="rounded-2xl border border-ink/10 bg-sand px-4 py-3">
            <div className="mb-2 flex items-center justify-between">
              <span className="text-sm font-medium text-ink">{t("settings.wakePattern")}</span>
              <span className="text-xs text-ink/60">{t(`settings.${wakePattern}` as "settings.early")}</span>
            </div>
            <input
              className="w-full accent-pine"
              max={2}
              min={0}
              type="range"
              value={wakeIndex}
              onChange={(event) => {
                form.setValue("wakePattern", wakePatternScale[Number(event.target.value)]);
              }}
            />
          </div>

          <div className="rounded-2xl border border-ink/10 bg-sand px-4 py-3">
            <div className="mb-2 flex items-center justify-between">
              <span className="text-sm font-medium text-ink">POI {t("ai.pace")}</span>
              <span className="text-xs text-ink/60">
                {densityIndex === 0 ? t("settings.relaxed") : densityIndex === 1 ? t("settings.balanced") : t("settings.packed")}
              </span>
            </div>
            <input
              className="w-full accent-pine"
              max={2}
              min={0}
              type="range"
              value={densityIndex}
              onChange={(event) => {
                form.setValue("poiDensity", poiDensityScale[Number(event.target.value)]);
              }}
            />
          </div>

          <div className="sm:col-span-2">
            <span className="mb-2 block text-sm font-medium text-ink">{t("ai.mustVisit")}</span>
            <div className="flex gap-2">
              <input
                className="flex-1 rounded-2xl border border-ink/10 bg-sand px-4 py-3"
                placeholder={t("ai.mustVisitHint")}
                value={mustVisitInput}
                onChange={(event) => setMustVisitInput(event.target.value)}
                onKeyDown={(event) => {
                  if (event.key === "Enter") {
                    event.preventDefault();
                    addTag("mustVisit", mustVisitInput, () => setMustVisitInput(""));
                  }
                }}
              />
              <button
                className="rounded-full border border-ink/20 px-4 py-2 text-sm font-medium text-ink"
                onClick={() => addTag("mustVisit", mustVisitInput, () => setMustVisitInput(""))}
                type="button"
              >
                {t("common.add")}
              </button>
            </div>
            <div className="mt-2 flex flex-wrap gap-2">
              {mustVisitTags.map((tag) => (
                <button
                  className="rounded-full bg-pine/10 px-3 py-1 text-xs font-medium text-pine"
                  key={tag}
                  onClick={() => removeTag("mustVisit", tag)}
                  type="button"
                >
                  {tag} ×
                </button>
              ))}
            </div>
          </div>

          <div className="sm:col-span-2">
            <span className="mb-2 block text-sm font-medium text-ink">{t("ai.avoid")}</span>
            <div className="flex gap-2">
              <input
                className="flex-1 rounded-2xl border border-ink/10 bg-sand px-4 py-3"
                placeholder={t("ai.avoidHint")}
                value={avoidInput}
                onChange={(event) => setAvoidInput(event.target.value)}
                onKeyDown={(event) => {
                  if (event.key === "Enter") {
                    event.preventDefault();
                    addTag("avoid", avoidInput, () => setAvoidInput(""));
                  }
                }}
              />
              <button
                className="rounded-full border border-ink/20 px-4 py-2 text-sm font-medium text-ink"
                onClick={() => addTag("avoid", avoidInput, () => setAvoidInput(""))}
                type="button"
              >
                {t("common.add")}
              </button>
            </div>
            <div className="mt-2 flex flex-wrap gap-2">
              {avoidTags.map((tag) => (
                <button
                  className="rounded-full bg-coral/10 px-3 py-1 text-xs font-medium text-coral"
                  key={tag}
                  onClick={() => removeTag("avoid", tag)}
                  type="button"
                >
                  {tag} ×
                </button>
              ))}
            </div>
          </div>
        </form>

        <div className="mt-5 rounded-[24px] bg-ink p-5 text-sand">
          <p className="text-xs uppercase tracking-[0.22em] text-sand/55">{t("ai.status")}</p>
          <h3 className="mt-2 font-display text-2xl font-bold">
            {planningJob?.status === "queued" ? "Queued" : null}
            {planningJob?.status === "running" ? t("ai.generating") : null}
            {planningJob?.status === "succeeded" ? t("ai.generated") : null}
            {planningJob?.status === "failed" ? t("common.actionFailed") : null}
            {!planningJob ? t("ai.generate") : null}
          </h3>
          <div className="mt-3 h-2 overflow-hidden rounded-full bg-sand/20">
            <div
              className={`h-full transition-all ${planningJob?.status === "failed" ? "bg-coral" : "bg-pine"}`}
              style={{ width: `${jobProgress}%` }}
            />
          </div>
          <p className="mt-2 text-xs text-sand/70">
            {planningJob ? `${planningJob.jobId} · ${new Date(planningJob.acceptedAt).toLocaleTimeString()}` : t("ai.adoptConfirmDescription")}
          </p>
        </div>
      </SurfaceCard>

      <SurfaceCard eyebrow={t("ai.drafts")} title={t("ai.drafts")}>
        {isLoading ? <div className="rounded-[24px] bg-sand p-4 text-sm text-ink/65">{t("common.loading")}</div> : null}
        {!isLoading && drafts.length === 0 ? <div className="rounded-[24px] bg-sand p-4 text-sm text-ink/65">{t("ai.noDrafts")}</div> : null}

        {compareDrafts ? (
          <div className="mb-6 grid gap-4 md:grid-cols-2">
            {compareDrafts.map((draft) => (
              <div className="rounded-[24px] border border-pine/20 bg-pine/5 p-4" key={draft.id}>
                <p className="font-display text-lg font-bold text-ink">{draft.title}</p>
                <p className="mt-1 text-sm text-ink/65">{draft.summary}</p>
                <div className="mt-3 space-y-1 text-xs text-ink/60">
                  <p>
                    {t("budget.planned")} {Math.round(draft.totalEstimated).toLocaleString()} {draft.currency}
                  </p>
                </div>
                <StatusPill tone={draft.status === "invalid" ? "danger" : draft.status === "warning" ? "neutral" : "success"}>
                  {draft.status}
                </StatusPill>
              </div>
            ))}
          </div>
        ) : null}

        <div className="grid gap-4">
          {drafts.map((draft) => (
            <div className="rounded-[28px] border border-ink/10 bg-white p-5" key={draft.id}>
              <div className="flex flex-wrap items-center justify-between gap-3">
                <div>
                  <p className="font-display text-2xl font-bold text-ink">{draft.title}</p>
                  <p className="mt-2 text-sm text-ink/65">{draft.summary}</p>
                </div>
                <StatusPill tone={draft.status === "invalid" ? "danger" : draft.status === "warning" ? "neutral" : "success"}>
                  {draft.status}
                </StatusPill>
              </div>
              <p className="mt-3 text-sm text-ink/60">
                {t("budget.planned")} {Math.round(draft.totalEstimated).toLocaleString()} {draft.currency} / {t("budget.totalBudget")}{" "}
                {Math.round(draft.budget).toLocaleString()} {draft.currency}
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
                  {t("budget.planned")} {Math.round(selectedPlan.totalEstimated).toLocaleString()} / {t("budget.totalBudget")}{" "}
                  {Math.round(selectedPlan.budget).toLocaleString()} {selectedPlan.currency}
                </p>
              </>
            ) : null}
          </div>
        ) : null}
      </SurfaceCard>
    </div>
  );
}
