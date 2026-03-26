import { useEffect, useMemo, useState } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { useParams } from "react-router-dom";
import { SurfaceCard } from "../../components/surface-card";
import { StatusPill } from "../../components/status-pill";
import {
  useBudgetProfileQuery,
  useCreateExpenseMutation,
  useDeleteExpenseMutation,
  useExpensesQuery,
  usePatchExpenseMutation,
  useUpsertBudgetMutation
} from "../../lib/queries";
import {
  addExpenseSchema,
  budgetProfileSchema,
  validationMessages,
  type AddExpenseFormValues,
  type BudgetProfileFormValues
} from "../../lib/schemas";
import type { Locale } from "../../lib/translations";
import { useUiStore } from "../../store/ui-store";
import { useI18n } from "../../lib/i18n";

const BUDGET_CATEGORIES = ["lodging", "transit", "food", "attraction", "shopping", "misc"] as const;

export function BudgetPage() {
  const { tripId = "" } = useParams();
  const { t, locale } = useI18n();
  const msgs = validationMessages[locale as Locale] ?? validationMessages.en;
  const pushToast = useUiStore((state) => state.pushToast);
  const { data: profile, isLoading: loadingProfile } = useBudgetProfileQuery(tripId);
  const { data: expenses = [], isLoading: loadingExpenses } = useExpensesQuery(tripId);
  const upsertBudget = useUpsertBudgetMutation(tripId);
  const createExpense = useCreateExpenseMutation(tripId);
  const deleteExpense = useDeleteExpenseMutation(tripId);
  const patchExpense = usePatchExpenseMutation(tripId);
  const [editingExpenseId, setEditingExpenseId] = useState<string | null>(null);
  const [editingAmount, setEditingAmount] = useState<string>("");
  const [editingNote, setEditingNote] = useState<string>("");
  const [editingCategory, setEditingCategory] = useState<string>("food");

  const catLabels: Record<string, string> = {
    lodging: t("budget.lodging"),
    transit: t("budget.transit"),
    food: t("budget.food"),
    attraction: t("budget.attraction"),
    shopping: t("budget.shopping"),
    misc: t("budget.misc")
  };

  const budgetForm = useForm<BudgetProfileFormValues>({
    resolver: zodResolver(budgetProfileSchema),
    defaultValues: {
      totalBudget: 0,
      perPersonBudget: 0,
      perDayBudget: 0,
      currency: "JPY",
      categories: BUDGET_CATEGORIES.map((category) => ({ category, plannedAmount: 0 }))
    }
  });

  const expenseForm = useForm<AddExpenseFormValues>({
    resolver: zodResolver(addExpenseSchema),
    defaultValues: {
      category: "food",
      amount: 0,
      currency: "JPY",
      note: "",
      expenseAt: ""
    }
  });

  useEffect(() => {
    if (!profile) {
      return;
    }

    budgetForm.reset({
      totalBudget: profile.totalBudget ?? 0,
      perPersonBudget: profile.perPersonBudget ?? 0,
      perDayBudget: profile.perDayBudget ?? 0,
      currency: profile.currency,
      categories: BUDGET_CATEGORIES.map((category) => ({
        category,
        plannedAmount: profile.categories.find((item) => item.category === category)?.plannedAmount ?? 0
      }))
    });

    expenseForm.setValue("currency", profile.currency);
  }, [budgetForm, expenseForm, profile]);

  const estimated = profile?.totalBudget ?? 0;
  const actual = profile?.actualSpend ?? 0;
  const overBudget = actual > estimated && estimated > 0;

  const perDaySeries = useMemo(() => {
    const totalsByDay = new Map<string, number>();
    for (const expense of expenses) {
      const day = (expense.expenseAt ?? expense.createdAt).slice(0, 10);
      totalsByDay.set(day, (totalsByDay.get(day) ?? 0) + expense.amount);
    }

    return Array.from(totalsByDay.entries())
      .sort(([a], [b]) => a.localeCompare(b))
      .map(([date, total]) => ({ date, total }));
  }, [expenses]);

  const trendPoints = useMemo(() => {
    if (perDaySeries.length === 0) {
      return [];
    }

    const maxValue = Math.max(...perDaySeries.map((item) => item.total), 1);
    const width = 320;
    const height = 120;
    const leftPad = 14;
    const rightPad = 12;
    const topPad = 12;
    const bottomPad = 18;
    const usableWidth = width - leftPad - rightPad;
    const usableHeight = height - topPad - bottomPad;

    if (perDaySeries.length === 1) {
      return [{ x: leftPad + usableWidth / 2, y: topPad + usableHeight / 2 }];
    }

    return perDaySeries.map((item, index) => ({
      x: leftPad + (index / (perDaySeries.length - 1)) * usableWidth,
      y: topPad + usableHeight - (item.total / maxValue) * usableHeight
    }));
  }, [perDaySeries]);

  const onSaveBudget = budgetForm.handleSubmit(async (values) => {
    await upsertBudget.mutateAsync({
      totalBudget: values.totalBudget,
      perPersonBudget: values.perPersonBudget,
      perDayBudget: values.perDayBudget,
      currency: values.currency,
      categories: values.categories
    });
    pushToast(t("budget.saved"));
  });

  const onAddExpense = expenseForm.handleSubmit(async (values) => {
    await createExpense.mutateAsync({
      category: values.category,
      amount: values.amount,
      currency: values.currency,
      note: values.note,
      expenseAt: values.expenseAt || undefined
    });
    expenseForm.reset({
      category: values.category,
      amount: 0,
      currency: values.currency,
      note: "",
      expenseAt: ""
    });
    pushToast(t("budget.addExpense"));
  });

  const removeExpense = async (expenseId: string) => {
    await deleteExpense.mutateAsync(expenseId);
    pushToast(t("common.delete"));
  };

  const beginEditExpense = (expenseId: string, category: string, amount: number, note?: string) => {
    setEditingExpenseId(expenseId);
    setEditingCategory(category);
    setEditingAmount(String(amount));
    setEditingNote(note ?? "");
  };

  const saveExpense = async () => {
    if (!editingExpenseId) {
      return;
    }

    const parsed = addExpenseSchema.safeParse({
      category: editingCategory,
      amount: editingAmount,
      currency: profile?.currency ?? "JPY",
      note: editingNote
    });
    if (!parsed.success) {
      const issue = parsed.error.issues[0];
      pushToast(msgs[issue.message] ?? issue.message);
      return;
    }

    await patchExpense.mutateAsync({
      expenseId: editingExpenseId,
      input: {
        category: parsed.data.category,
        amount: parsed.data.amount,
        note: parsed.data.note,
        currency: parsed.data.currency
      }
    });

    setEditingExpenseId(null);
    pushToast(t("common.save"));
  };

  const budgetErrors = budgetForm.formState.errors;
  const expenseErrors = expenseForm.formState.errors;

  return (
    <div className="grid gap-6 lg:grid-cols-[0.9fr_1.1fr]">
      <SurfaceCard eyebrow={t("nav.budget")} title={t("budget.title")}>
        {loadingProfile ? <div className="mb-4 rounded-[20px] bg-sand p-3 text-sm text-ink/65">{t("common.loading")}</div> : null}

        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
          <div className="rounded-[24px] bg-sand p-4">
            <p className="text-sm text-ink/60">{t("budget.totalBudget")}</p>
            <p className="mt-2 font-display text-3xl font-bold text-ink">
              {profile?.currency ?? "JPY"} {estimated.toLocaleString()}
            </p>
          </div>
          <div className="rounded-[24px] bg-sand p-4">
            <p className="text-sm text-ink/60">{t("budget.actual")}</p>
            <p className="mt-2 font-display text-3xl font-bold text-ink">
              {profile?.currency ?? "JPY"} {actual.toLocaleString()}
            </p>
          </div>
          <div className="rounded-[24px] bg-sand p-4">
            <p className="text-sm text-ink/60">{t("budget.perDay")}</p>
            <p className="mt-2 font-display text-3xl font-bold text-ink">
              {profile?.currency ?? "JPY"} {(profile?.perDayBudget ?? 0).toLocaleString()}
            </p>
          </div>
          <div className="rounded-[24px] bg-sand p-4">
            <p className="text-sm text-ink/60">{t("budget.perPerson")}</p>
            <p className="mt-2 font-display text-3xl font-bold text-ink">
              {profile?.currency ?? "JPY"} {(profile?.perPersonBudget ?? Math.round(estimated / 2)).toLocaleString()}
            </p>
          </div>
        </div>

        <form className="mt-5 grid gap-4 rounded-[24px] border border-ink/10 bg-white p-4" onSubmit={onSaveBudget}>
          <div className="grid gap-4 md:grid-cols-2">
            <label>
              <span className="mb-1 block text-xs uppercase tracking-[0.18em] text-ink/55">{t("budget.totalBudget")}</span>
              <input
                className="w-full rounded-xl border border-ink/10 bg-sand px-3 py-2 text-sm text-ink"
                min={0}
                type="number"
                {...budgetForm.register("totalBudget")}
              />
              {budgetErrors.totalBudget ? <p className="mt-1 text-xs text-coral">{msgs[budgetErrors.totalBudget.message ?? ""] ?? budgetErrors.totalBudget.message}</p> : null}
            </label>
            <label>
              <span className="mb-1 block text-xs uppercase tracking-[0.18em] text-ink/55">{t("trip.currency")}</span>
              <input
                className="w-full rounded-xl border border-ink/10 bg-sand px-3 py-2 text-sm text-ink"
                {...budgetForm.register("currency")}
              />
              {budgetErrors.currency ? <p className="mt-1 text-xs text-coral">{msgs[budgetErrors.currency.message ?? ""] ?? budgetErrors.currency.message}</p> : null}
            </label>
          </div>
          <div className="grid gap-4 md:grid-cols-2">
            <label>
              <span className="mb-1 block text-xs uppercase tracking-[0.18em] text-ink/55">{t("budget.perDay")}</span>
              <input
                className="w-full rounded-xl border border-ink/10 bg-sand px-3 py-2 text-sm text-ink"
                min={0}
                type="number"
                {...budgetForm.register("perDayBudget")}
              />
            </label>
            <label>
              <span className="mb-1 block text-xs uppercase tracking-[0.18em] text-ink/55">{t("budget.perPerson")}</span>
              <input
                className="w-full rounded-xl border border-ink/10 bg-sand px-3 py-2 text-sm text-ink"
                min={0}
                type="number"
                {...budgetForm.register("perPersonBudget")}
              />
            </label>
          </div>

          <div className="grid gap-3 md:grid-cols-2">
            {BUDGET_CATEGORIES.map((category, index) => (
              <label key={category}>
                <span className="mb-1 block text-xs uppercase tracking-[0.18em] text-ink/55">{catLabels[category]}</span>
                <input
                  className="w-full rounded-xl border border-ink/10 bg-sand px-3 py-2 text-sm text-ink"
                  min={0}
                  type="number"
                  {...budgetForm.register(`categories.${index}.plannedAmount`)}
                />
              </label>
            ))}
          </div>

          <button className="w-fit rounded-full bg-ink px-5 py-3 text-sm font-medium text-white" disabled={upsertBudget.isPending} type="submit">
            {upsertBudget.isPending ? t("budget.savingBudget") : t("budget.saveBudget")}
          </button>
        </form>

        <form className="mt-4 grid gap-4 rounded-[24px] border border-ink/10 bg-white p-4" onSubmit={onAddExpense}>
          <p className="text-sm font-semibold text-ink">{t("budget.addExpense")}</p>
          <div className="grid gap-4 md:grid-cols-3">
            <label>
              <span className="mb-1 block text-xs uppercase tracking-[0.18em] text-ink/55">{t("budget.expenseCategory")}</span>
              <select className="w-full rounded-xl border border-ink/10 bg-sand px-3 py-2 text-sm text-ink" {...expenseForm.register("category")}>
                {BUDGET_CATEGORIES.map((category) => (
                  <option key={category} value={category}>
                    {catLabels[category]}
                  </option>
                ))}
              </select>
            </label>
            <label>
              <span className="mb-1 block text-xs uppercase tracking-[0.18em] text-ink/55">{t("budget.expenseAmount")}</span>
              <input
                className="w-full rounded-xl border border-ink/10 bg-sand px-3 py-2 text-sm text-ink"
                min={0}
                type="number"
                {...expenseForm.register("amount")}
              />
              {expenseErrors.amount ? <p className="mt-1 text-xs text-coral">{msgs[expenseErrors.amount.message ?? ""] ?? expenseErrors.amount.message}</p> : null}
            </label>
            <label>
              <span className="mb-1 block text-xs uppercase tracking-[0.18em] text-ink/55">{t("budget.expenseDate")}</span>
              <input className="w-full rounded-xl border border-ink/10 bg-sand px-3 py-2 text-sm text-ink" type="date" {...expenseForm.register("expenseAt")} />
            </label>
          </div>
          <label>
            <span className="mb-1 block text-xs uppercase tracking-[0.18em] text-ink/55">{t("budget.expenseNote")}</span>
            <input className="w-full rounded-xl border border-ink/10 bg-sand px-3 py-2 text-sm text-ink" {...expenseForm.register("note")} />
          </label>
          <button className="w-fit rounded-full bg-coral px-5 py-3 text-sm font-medium text-white" disabled={createExpense.isPending} type="submit">
            {createExpense.isPending ? t("budget.addingExpense") : t("budget.addExpense")}
          </button>
        </form>

        {profile?.overBudget || overBudget ? <p className="mt-4 text-sm font-medium text-coral">{t("budget.overBudgetWarning")}</p> : null}
      </SurfaceCard>

      <SurfaceCard eyebrow={t("budget.breakdown")} title={t("budget.estimatedVsActual")}>
        {loadingExpenses ? <div className="rounded-[20px] bg-sand p-3 text-sm text-ink/65">{t("common.loading")}</div> : null}

        {estimated > 0 ? (() => {
          const pct = Math.min((actual / estimated) * 100, 100);
          const gaugeOverBudget = actual > estimated;
          const r = 54;
          const circ = 2 * Math.PI * r;
          const offset = circ - (pct / 100) * circ;
          return (
            <div className="mb-6 flex flex-col items-center">
              <svg className="h-32 w-32" viewBox="0 0 120 120">
                <circle cx="60" cy="60" fill="none" r={r} stroke="#e8e0d8" strokeWidth="10" />
                <circle
                  className={gaugeOverBudget ? "animate-pulse" : ""}
                  cx="60"
                  cy="60"
                  fill="none"
                  r={r}
                  stroke={gaugeOverBudget ? "#da6a4e" : "#2d5a4a"}
                  strokeDasharray={circ}
                  strokeDashoffset={offset}
                  strokeLinecap="round"
                  strokeWidth="10"
                  transform="rotate(-90 60 60)"
                />
                <text className="fill-ink text-lg font-bold" fontSize="18" textAnchor="middle" x="60" y="56">
                  {Math.round(pct)}%
                </text>
                <text className="fill-ink/60" fontSize="10" textAnchor="middle" x="60" y="74">
                  {t("budget.actual")}
                </text>
              </svg>
              <p className="mt-2 text-sm text-ink/65">
                {profile?.currency ?? "JPY"} {actual.toLocaleString()} / {estimated.toLocaleString()}
              </p>
              {gaugeOverBudget ? <p className="mt-1 animate-pulse text-xs font-medium text-coral">{t("budget.overBudgetWarning")}</p> : null}
            </div>
          );
        })() : null}

        {(profile?.categories ?? []).map((category) => {
          const categoryActual = expenses.filter((item) => item.category === category.category).reduce((sum, item) => sum + item.amount, 0);
          const maxVal = Math.max(category.plannedAmount, categoryActual, 1);
          const plannedPct = (category.plannedAmount / maxVal) * 100;
          const actualPct = (categoryActual / maxVal) * 100;
          const overspent = categoryActual > category.plannedAmount;
          return (
            <div key={category.category} className="mb-4 rounded-[20px] bg-white p-4">
              <div className="flex items-center justify-between text-sm">
                <span className="font-medium text-ink">{catLabels[category.category] ?? category.category}</span>
                <StatusPill tone={overspent ? "danger" : "success"}>
                  {overspent ? "+" : ""}
                  {(categoryActual - category.plannedAmount).toLocaleString()}
                </StatusPill>
              </div>
              <div className="mt-3 space-y-2">
                <div className="flex items-center gap-3">
                  <span className="w-12 text-xs text-ink/50">{t("budget.planned")}</span>
                  <div className="relative h-3 flex-1 overflow-hidden rounded-full bg-sand">
                    <div className="h-full rounded-full bg-pine/60 transition-all duration-500" style={{ width: `${plannedPct}%` }} />
                  </div>
                  <span className="w-16 text-right text-xs text-ink/65">{category.plannedAmount.toLocaleString()}</span>
                </div>
                <div className="flex items-center gap-3">
                  <span className="w-12 text-xs text-ink/50">{t("budget.actual")}</span>
                  <div className="relative h-3 flex-1 overflow-hidden rounded-full bg-sand">
                    <div className={`h-full rounded-full transition-all duration-500 ${overspent ? "animate-pulse bg-coral/70" : "bg-coral/40"}`} style={{ width: `${actualPct}%` }} />
                  </div>
                  <span className="w-16 text-right text-xs text-ink/65">{categoryActual.toLocaleString()}</span>
                </div>
              </div>
            </div>
          );
        })}

        <div className="mb-5 rounded-[20px] border border-ink/10 bg-white p-4">
          <div className="mb-2 flex items-center justify-between gap-3">
            <p className="text-sm font-semibold text-ink">{t("budget.perDay")} Trend</p>
            <span className="text-xs text-ink/55">
              FX source: backend_snapshot · {profile?.updatedAt ? new Date(profile.updatedAt).toLocaleDateString() : "-"}
            </span>
          </div>
          {trendPoints.length === 0 ? <p className="text-sm text-ink/60">{t("common.noData")}</p> : null}
          {trendPoints.length > 0 ? (
            <>
              <svg className="h-32 w-full" viewBox="0 0 320 120">
                <polyline
                  fill="none"
                  points={trendPoints.map((point) => `${point.x},${point.y}`).join(" ")}
                  stroke="#2d5a4a"
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth="3"
                />
                {trendPoints.map((point, index) => (
                  <circle cx={point.x} cy={point.y} fill="#da6a4e" key={`${perDaySeries[index].date}-${point.x}`} r="3" />
                ))}
              </svg>
              <div className="mt-2 grid gap-1 text-xs text-ink/60 sm:grid-cols-2">
                {perDaySeries.map((item) => (
                  <p key={item.date}>
                    {item.date}: {profile?.currency ?? "JPY"} {Math.round(item.total).toLocaleString()}
                  </p>
                ))}
              </div>
            </>
          ) : null}
        </div>

        <div className="space-y-3">
          {expenses.map((expense) => (
            <div className="rounded-[24px] border border-ink/10 bg-sand/60 p-4" key={expense.id}>
              {editingExpenseId === expense.id ? (
                <div className="grid gap-3 md:grid-cols-[1fr_1fr_auto]">
                  <select
                    className="rounded-xl border border-ink/10 bg-white px-3 py-2 text-sm text-ink"
                    value={editingCategory}
                    onChange={(event) => setEditingCategory(event.target.value)}
                  >
                    {BUDGET_CATEGORIES.map((category) => (
                      <option key={category} value={category}>
                        {catLabels[category]}
                      </option>
                    ))}
                  </select>
                  <input
                    className="rounded-xl border border-ink/10 bg-white px-3 py-2 text-sm text-ink"
                    min={0}
                    type="number"
                    value={editingAmount}
                    onChange={(event) => setEditingAmount(event.target.value)}
                  />
                  <div className="flex items-center gap-2">
                    <button
                      className="rounded-full border border-ink/15 px-3 py-1 text-xs font-medium text-ink"
                      disabled={patchExpense.isPending}
                      onClick={() => {
                        void saveExpense();
                      }}
                      type="button"
                    >
                      {t("common.save")}
                    </button>
                    <button
                      className="rounded-full border border-ink/15 px-3 py-1 text-xs font-medium text-ink"
                      onClick={() => setEditingExpenseId(null)}
                      type="button"
                    >
                      {t("common.cancel")}
                    </button>
                  </div>
                  <input
                    className="rounded-xl border border-ink/10 bg-white px-3 py-2 text-sm text-ink md:col-span-3"
                    placeholder={t("budget.expenseNote")}
                    value={editingNote}
                    onChange={(event) => setEditingNote(event.target.value)}
                  />
                </div>
              ) : (
                <div className="flex flex-wrap items-center justify-between gap-3">
                  <div>
                    <p className="font-medium text-ink">{catLabels[expense.category] ?? expense.category}</p>
                    <p className="text-sm text-ink/60">
                      {expense.currency} {expense.amount.toLocaleString()} {expense.note ? `· ${expense.note}` : ""}
                    </p>
                  </div>
                  <div className="flex items-center gap-2">
                    <button
                      className="rounded-full border border-ink/15 px-3 py-1 text-xs font-medium text-ink"
                      disabled={deleteExpense.isPending}
                      onClick={() => beginEditExpense(expense.id, expense.category, expense.amount, expense.note)}
                      type="button"
                    >
                      {t("common.edit")}
                    </button>
                    <button
                      className="rounded-full border border-ink/15 px-3 py-1 text-xs font-medium text-ink"
                      disabled={deleteExpense.isPending}
                      onClick={() => {
                        void removeExpense(expense.id);
                      }}
                      type="button"
                    >
                      {t("common.delete")}
                    </button>
                  </div>
                </div>
              )}
            </div>
          ))}
        </div>

        {!loadingProfile && (profile?.categories?.length ?? 0) === 0 ? (
          <div className="rounded-[24px] bg-sand p-4 text-sm text-ink/65">{t("budget.noBudget")}</div>
        ) : null}
      </SurfaceCard>
    </div>
  );
}
