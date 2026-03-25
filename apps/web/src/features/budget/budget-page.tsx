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
  useTripQuery,
  useUpsertBudgetMutation
} from "../../lib/queries";
import { useUiStore } from "../../store/ui-store";
import { useI18n } from "../../lib/i18n";
import { addExpenseSchema, upsertBudgetSchema, validationMessages } from "../../lib/schemas";
import type { AddExpenseFormValues, UpsertBudgetFormValues } from "../../lib/schemas";
import type { Locale } from "../../lib/translations";

const DEFAULT_BUDGET_CATEGORIES = [
  { category: "lodging", plannedAmount: 0 },
  { category: "transit", plannedAmount: 0 },
  { category: "food", plannedAmount: 0 },
  { category: "attraction", plannedAmount: 0 },
  { category: "shopping", plannedAmount: 0 },
  { category: "misc", plannedAmount: 0 }
];

export function BudgetPage() {
  const { tripId = "" } = useParams();
  const { t, locale } = useI18n();
  const msgs = validationMessages[locale as Locale] ?? validationMessages.en;
  const pushToast = useUiStore((state) => state.pushToast);
  const { data: profile, isLoading: loadingProfile } = useBudgetProfileQuery(tripId);
  const { data: trip } = useTripQuery(tripId);
  const { data: expenses = [], isLoading: loadingExpenses } = useExpensesQuery(tripId);
  const upsertBudget = useUpsertBudgetMutation(tripId);
  const createExpense = useCreateExpenseMutation(tripId);
  const deleteExpense = useDeleteExpenseMutation(tripId);
  const patchExpense = usePatchExpenseMutation(tripId);
  const [editingExpenseId, setEditingExpenseId] = useState<string | null>(null);
  const [editingAmount, setEditingAmount] = useState<string>("");
  const [editingNote, setEditingNote] = useState<string>("");
  const [editingCategory, setEditingCategory] = useState<string>("");

  const budgetForm = useForm<UpsertBudgetFormValues>({
    resolver: zodResolver(upsertBudgetSchema),
    defaultValues: {
      totalBudget: undefined,
      perPersonBudget: undefined,
      perDayBudget: undefined,
      currency: "JPY"
    }
  });
  const {
    formState: { errors: budgetErrors }
  } = budgetForm;

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
  const {
    formState: { errors: expenseErrors }
  } = expenseForm;

  useEffect(() => {
    if (!profile) {
      return;
    }
    budgetForm.reset({
      totalBudget: profile.totalBudget,
      perPersonBudget: profile.perPersonBudget,
      perDayBudget: profile.perDayBudget,
      currency: profile.currency
    });
    expenseForm.setValue("currency", profile.currency, { shouldValidate: false });
  }, [profile, budgetForm, expenseForm]);

  const catLabels: Record<string, string> = {
    lodging: t("budget.lodging"),
    transit: t("budget.transit"),
    food: t("budget.food"),
    attraction: t("budget.attraction"),
    shopping: t("budget.shopping"),
    misc: t("budget.misc")
  };

  const estimated = profile?.totalBudget ?? 0;
  const actual = profile?.actualSpend ?? 0;
  const travelerCount = Math.max(trip?.travelersCount ?? 1, 1);
  const perPersonActual = actual / travelerCount;

  const dailySpendSeries = useMemo(() => {
    const grouped = new Map<string, number>();
    for (const expense of expenses) {
      const dateKey = (expense.expenseAt ?? expense.createdAt ?? "").slice(0, 10);
      if (!dateKey) {
        continue;
      }
      grouped.set(dateKey, (grouped.get(dateKey) ?? 0) + expense.amount);
    }
    return Array.from(grouped.entries())
      .sort(([left], [right]) => left.localeCompare(right))
      .map(([date, amount]) => ({ date, amount }));
  }, [expenses]);

  const trendCoordinates = useMemo(() => {
    if (dailySpendSeries.length === 0) {
      return [];
    }
    const maxSpend = Math.max(...dailySpendSeries.map((item) => item.amount), 1);
    return dailySpendSeries.map((item, index) => ({
      ...item,
      x: dailySpendSeries.length === 1 ? 160 : (index / (dailySpendSeries.length - 1)) * 320,
      y: 100 - (item.amount / maxSpend) * 80
    }));
  }, [dailySpendSeries]);

  const trendPolyline = trendCoordinates.map((point) => `${point.x},${point.y}`).join(" ");
  const averagePerDay = dailySpendSeries.length > 0 ? actual / dailySpendSeries.length : 0;

  const saveBudget = budgetForm.handleSubmit(async (values) => {
    const categories = (profile?.categories?.length ? profile.categories : DEFAULT_BUDGET_CATEGORIES).map((category) => ({
      category: category.category,
      plannedAmount: category.plannedAmount
    }));

    await upsertBudget.mutateAsync({
      totalBudget: values.totalBudget,
      perPersonBudget: values.perPersonBudget,
      perDayBudget: values.perDayBudget,
      currency: values.currency.toUpperCase(),
      categories
    });
    pushToast(t("budget.saved"));
  });

  const addExpense = expenseForm.handleSubmit(async (values) => {
    const currency = (profile?.currency ?? values.currency).toUpperCase();
    await createExpense.mutateAsync({
      category: values.category,
      amount: values.amount,
      currency,
      note: values.note?.trim() || undefined,
      expenseAt: values.expenseAt || undefined
    });
    expenseForm.reset({
      category: values.category,
      amount: 0,
      currency,
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
    const amountValue = Number(editingAmount);
    if (!Number.isFinite(amountValue) || amountValue < 0) {
      pushToast({
        type: "warning",
        message: msgs.amountNonNegative ?? "Amount cannot be negative"
      });
      return;
    }

    await patchExpense.mutateAsync({
      expenseId: editingExpenseId,
      input: {
        category: editingCategory,
        amount: amountValue,
        note: editingNote,
        currency: profile?.currency ?? "JPY"
      }
    });

    setEditingExpenseId(null);
    pushToast(t("common.save"));
  };

  const resolveValidationMessage = (message?: string) => (message ? msgs[message] ?? message : "");

  return (
    <div className="grid gap-6 lg:grid-cols-[0.9fr_1.1fr]">
      <SurfaceCard eyebrow={t("nav.budget")} title={t("budget.title")}>
        {loadingProfile ? <div className="mb-4 rounded-[20px] bg-sand p-3 text-sm text-ink/65">{t("common.loading")}</div> : null}
        <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
          <div className="rounded-[24px] bg-sand p-4">
            <p className="text-sm text-ink/60">{t("budget.totalBudget")}</p>
            <p className="mt-2 font-display text-3xl font-bold text-ink">{profile?.currency ?? "JPY"} {estimated.toLocaleString()}</p>
          </div>
          <div className="rounded-[24px] bg-sand p-4">
            <p className="text-sm text-ink/60">{t("budget.actual")}</p>
            <p className="mt-2 font-display text-3xl font-bold text-ink">{profile?.currency ?? "JPY"} {actual.toLocaleString()}</p>
          </div>
          <div className="rounded-[24px] bg-sand p-4">
            <p className="text-sm text-ink/60">{t("budget.perDay")}</p>
            <p className="mt-2 font-display text-3xl font-bold text-ink">{profile?.currency ?? "JPY"} {(profile?.perDayBudget ?? 0).toLocaleString()}</p>
          </div>
          <div className="rounded-[24px] bg-sand p-4">
            <p className="text-sm text-ink/60">{t("budget.perPerson")}</p>
            <p className="mt-2 font-display text-3xl font-bold text-ink">{profile?.currency ?? "JPY"} {Math.round(perPersonActual).toLocaleString()}</p>
          </div>
        </div>

        <form className="mt-5 grid gap-4 rounded-[24px] bg-sand/70 p-4 sm:grid-cols-2" onSubmit={saveBudget}>
          <label className="block">
            <span className="mb-2 block text-sm font-medium text-ink">{t("budget.totalBudget")}</span>
            <input
              className="w-full rounded-xl border border-ink/10 bg-white px-3 py-2 text-sm text-ink"
              min={0}
              step="0.01"
              type="number"
              {...budgetForm.register("totalBudget", {
                setValueAs: (value) => (value === "" ? undefined : Number(value))
              })}
            />
            {budgetErrors.totalBudget ? <p className="mt-1 text-xs text-coral">{resolveValidationMessage(budgetErrors.totalBudget.message)}</p> : null}
          </label>
          <label className="block">
            <span className="mb-2 block text-sm font-medium text-ink">{t("budget.perPerson")}</span>
            <input
              className="w-full rounded-xl border border-ink/10 bg-white px-3 py-2 text-sm text-ink"
              min={0}
              step="0.01"
              type="number"
              {...budgetForm.register("perPersonBudget", {
                setValueAs: (value) => (value === "" ? undefined : Number(value))
              })}
            />
            {budgetErrors.perPersonBudget ? <p className="mt-1 text-xs text-coral">{resolveValidationMessage(budgetErrors.perPersonBudget.message)}</p> : null}
          </label>
          <label className="block">
            <span className="mb-2 block text-sm font-medium text-ink">{t("budget.perDay")}</span>
            <input
              className="w-full rounded-xl border border-ink/10 bg-white px-3 py-2 text-sm text-ink"
              min={0}
              step="0.01"
              type="number"
              {...budgetForm.register("perDayBudget", {
                setValueAs: (value) => (value === "" ? undefined : Number(value))
              })}
            />
            {budgetErrors.perDayBudget ? <p className="mt-1 text-xs text-coral">{resolveValidationMessage(budgetErrors.perDayBudget.message)}</p> : null}
          </label>
          <label className="block">
            <span className="mb-2 block text-sm font-medium text-ink">{t("trip.currency")}</span>
            <input className="w-full rounded-xl border border-ink/10 bg-white px-3 py-2 text-sm uppercase text-ink" maxLength={3} {...budgetForm.register("currency")} />
            {budgetErrors.currency ? <p className="mt-1 text-xs text-coral">{resolveValidationMessage(budgetErrors.currency.message)}</p> : null}
          </label>
          <div className="sm:col-span-2 flex justify-end">
            <button className="rounded-full bg-ink px-5 py-3 text-sm font-medium text-white disabled:opacity-60" disabled={upsertBudget.isPending} type="submit">
              {upsertBudget.isPending ? t("budget.savingBudget") : t("budget.saveBudget")}
            </button>
          </div>
        </form>

        <form className="mt-5 grid gap-4 rounded-[24px] bg-white p-4 sm:grid-cols-2" onSubmit={addExpense}>
          <label className="block">
            <span className="mb-2 block text-sm font-medium text-ink">{t("budget.expenseCategory")}</span>
            <select className="w-full rounded-xl border border-ink/10 bg-sand px-3 py-2 text-sm text-ink" {...expenseForm.register("category")}>
              {Object.entries(catLabels).map(([value, label]) => (
                <option key={value} value={value}>
                  {label}
                </option>
              ))}
            </select>
            {expenseErrors.category ? <p className="mt-1 text-xs text-coral">{resolveValidationMessage(expenseErrors.category.message)}</p> : null}
          </label>
          <label className="block">
            <span className="mb-2 block text-sm font-medium text-ink">{t("budget.expenseAmount")}</span>
            <input className="w-full rounded-xl border border-ink/10 bg-sand px-3 py-2 text-sm text-ink" min={0} step="0.01" type="number" {...expenseForm.register("amount")} />
            {expenseErrors.amount ? <p className="mt-1 text-xs text-coral">{resolveValidationMessage(expenseErrors.amount.message)}</p> : null}
          </label>
          <label className="block">
            <span className="mb-2 block text-sm font-medium text-ink">{t("budget.expenseDate")}</span>
            <input className="w-full rounded-xl border border-ink/10 bg-sand px-3 py-2 text-sm text-ink" type="date" {...expenseForm.register("expenseAt")} />
          </label>
          <label className="block">
            <span className="mb-2 block text-sm font-medium text-ink">{t("budget.expenseNote")}</span>
            <input className="w-full rounded-xl border border-ink/10 bg-sand px-3 py-2 text-sm text-ink" {...expenseForm.register("note")} />
          </label>
          <input type="hidden" {...expenseForm.register("currency")} />
          <div className="sm:col-span-2 flex justify-end">
            <button className="rounded-full bg-coral px-5 py-3 text-sm font-medium text-white disabled:opacity-60" disabled={createExpense.isPending} type="submit">
              {createExpense.isPending ? t("budget.addingExpense") : t("budget.addExpense")}
            </button>
          </div>
        </form>

        {profile?.overBudget ? <p className="mt-4 text-sm font-medium text-coral">{t("budget.overBudgetWarning")}</p> : null}
      </SurfaceCard>
      <SurfaceCard eyebrow={t("budget.breakdown")} title={t("budget.estimatedVsActual")}>
        {loadingExpenses ? <div className="rounded-[20px] bg-sand p-3 text-sm text-ink/65">{t("common.loading")}</div> : null}

        {estimated > 0 ? (() => {
          const pct = Math.min((actual / estimated) * 100, 100);
          const overBudget = actual > estimated;
          const r = 54;
          const circ = 2 * Math.PI * r;
          const offset = circ - (pct / 100) * circ;
          return (
            <div className="mb-6 flex flex-col items-center">
              <svg viewBox="0 0 120 120" className="h-32 w-32">
                <circle cx="60" cy="60" r={r} fill="none" stroke="#e8e0d8" strokeWidth="10" />
                <circle
                  cx="60"
                  cy="60"
                  r={r}
                  fill="none"
                  stroke={overBudget ? "#da6a4e" : "#2d5a4a"}
                  strokeWidth="10"
                  strokeDasharray={circ}
                  strokeDashoffset={offset}
                  strokeLinecap="round"
                  transform="rotate(-90 60 60)"
                  className={overBudget ? "animate-pulse" : ""}
                />
                <text x="60" y="56" textAnchor="middle" className="fill-ink text-lg font-bold" fontSize="18">{Math.round(pct)}%</text>
                <text x="60" y="74" textAnchor="middle" className="fill-ink/60" fontSize="10">{t("budget.actual")}</text>
              </svg>
              <p className="mt-2 text-sm text-ink/65">
                {profile?.currency ?? "JPY"} {actual.toLocaleString()} / {estimated.toLocaleString()}
              </p>
              {overBudget ? <p className="mt-1 animate-pulse text-xs font-medium text-coral">{t("budget.overBudgetWarning")}</p> : null}
            </div>
          );
        })() : null}

        <div className="mb-4 rounded-[20px] bg-white p-4">
          <div className="mb-3 flex items-center justify-between">
            <p className="text-sm font-semibold text-ink">{t("budget.dailyTrend")}</p>
            <p className="text-xs text-ink/55">{t("budget.perDay")} {Math.round(averagePerDay).toLocaleString()}</p>
          </div>
          {trendCoordinates.length === 0 ? <p className="text-sm text-ink/60">{t("budget.trendNoData")}</p> : null}
          {trendCoordinates.length > 0 ? (
            <>
              <svg viewBox="0 0 320 120" className="h-36 w-full">
                <polyline fill="none" points={trendPolyline} stroke="#2d5a4a" strokeWidth="3" />
                {trendCoordinates.map((point) => (
                  <circle key={point.date} cx={point.x} cy={point.y} fill="#da6a4e" r="4" />
                ))}
              </svg>
              <div className="mt-2 flex items-center justify-between text-xs text-ink/55">
                <span>{trendCoordinates[0]?.date}</span>
                <span>{trendCoordinates[trendCoordinates.length - 1]?.date}</span>
              </div>
            </>
          ) : null}
        </div>

        {(profile?.categories ?? []).map((category) => {
          const categoryActual = expenses.filter((item) => item.category === category.category).reduce((sum, item) => sum + item.amount, 0);
          const maxVal = Math.max(category.plannedAmount, categoryActual, 1);
          const plannedPct = (category.plannedAmount / maxVal) * 100;
          const actualPct = (categoryActual / maxVal) * 100;
          const overSpent = categoryActual > category.plannedAmount;
          return (
            <div key={category.category} className="mb-4 rounded-[20px] bg-white p-4">
              <div className="flex items-center justify-between text-sm">
                <span className="font-medium text-ink">{catLabels[category.category] ?? category.category}</span>
                <StatusPill tone={overSpent ? "danger" : "success"}>
                  {overSpent ? `+${(categoryActual - category.plannedAmount).toLocaleString()}` : `${(categoryActual - category.plannedAmount).toLocaleString()}`}
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
                    <div className={`h-full rounded-full transition-all duration-500 ${overSpent ? "bg-coral/70 animate-pulse" : "bg-coral/40"}`} style={{ width: `${actualPct}%` }} />
                  </div>
                  <span className="w-16 text-right text-xs text-ink/65">{categoryActual.toLocaleString()}</span>
                </div>
              </div>
            </div>
          );
        })}

        <div className="space-y-3">
          {expenses.map((expense) => (
            <div key={expense.id} className="rounded-[24px] border border-ink/10 bg-sand/60 p-4">
              {editingExpenseId === expense.id ? (
                <div className="grid gap-3 md:grid-cols-[1fr_1fr_auto]">
                  <input
                    className="rounded-xl border border-ink/10 bg-white px-3 py-2 text-sm text-ink"
                    value={editingCategory}
                    onChange={(event) => setEditingCategory(event.target.value)}
                  />
                  <input
                    className="rounded-xl border border-ink/10 bg-white px-3 py-2 text-sm text-ink"
                    type="number"
                    min={0}
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
                    className="md:col-span-3 rounded-xl border border-ink/10 bg-white px-3 py-2 text-sm text-ink"
                    placeholder={t("budget.expenseNote")}
                    value={editingNote}
                    onChange={(event) => setEditingNote(event.target.value)}
                  />
                </div>
              ) : (
                <div className="flex flex-wrap items-center justify-between gap-3">
                  <div>
                    <p className="font-medium text-ink">{catLabels[expense.category] ?? expense.category}</p>
                    <p className="text-sm text-ink/60">{expense.currency} {expense.amount.toLocaleString()} {expense.note ? `· ${expense.note}` : ""}</p>
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
