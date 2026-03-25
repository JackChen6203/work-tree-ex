import { useParams } from "react-router-dom";
import { SurfaceCard } from "../../components/surface-card";
import { StatusPill } from "../../components/status-pill";
import { useState } from "react";
import { useBudgetProfileQuery, useCreateExpenseMutation, useDeleteExpenseMutation, useExpensesQuery, usePatchExpenseMutation, useUpsertBudgetMutation } from "../../lib/queries";
import { useUiStore } from "../../store/ui-store";
import { useI18n } from "../../lib/i18n";

export function BudgetPage() {
  const { tripId = "" } = useParams();
  const { t } = useI18n();
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
  const [editingCategory, setEditingCategory] = useState<string>("");

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

  const seedBudget = async () => {
    await upsertBudget.mutateAsync({
      totalBudget: 70500,
      perDayBudget: 14100,
      currency: "JPY",
      categories: [
        { category: "lodging", plannedAmount: 28500 },
        { category: "transit", plannedAmount: 12000 },
        { category: "food", plannedAmount: 15000 },
        { category: "attraction", plannedAmount: 7000 },
        { category: "shopping", plannedAmount: 8000 }
      ]
    });
    pushToast(t("budget.saved"));
  };

  const addSampleExpense = async () => {
    await createExpense.mutateAsync({
      category: "food",
      amount: 4800,
      currency: profile?.currency ?? "JPY",
      note: ""
    });
    pushToast(t("budget.addExpense"));
  };

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

  return (
    <div className="grid gap-6 lg:grid-cols-[0.9fr_1.1fr]">
      <SurfaceCard eyebrow={t("nav.budget")} title={t("budget.title")}>
        {loadingProfile ? <div className="mb-4 rounded-[20px] bg-sand p-3 text-sm text-ink/65">{t("common.loading")}</div> : null}
        <div className="grid gap-4 sm:grid-cols-3">
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
        </div>
        <div className="mt-5 flex flex-wrap gap-3">
          <button
            className="rounded-full bg-ink px-5 py-3 text-sm font-medium text-white"
            disabled={upsertBudget.isPending}
            onClick={() => {
              void seedBudget();
            }}
            type="button"
          >
            {upsertBudget.isPending ? t("budget.savingBudget") : t("budget.saveBudget")}
          </button>
          <button
            className="rounded-full bg-coral px-5 py-3 text-sm font-medium text-white"
            disabled={createExpense.isPending}
            onClick={() => {
              void addSampleExpense();
            }}
            type="button"
          >
            {createExpense.isPending ? t("budget.addingExpense") : t("budget.addExpense")}
          </button>
        </div>
        {profile?.overBudget ? <p className="mt-4 text-sm font-medium text-coral">{t("budget.overBudgetWarning")}</p> : null}
      </SurfaceCard>
      <SurfaceCard eyebrow={t("budget.breakdown")} title={t("budget.estimatedVsActual")}>
        {loadingExpenses ? <div className="rounded-[20px] bg-sand p-3 text-sm text-ink/65">{t("common.loading")}</div> : null}

        {/* SVG Gauge — total spend / total budget */}
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
                  cx="60" cy="60" r={r} fill="none"
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
              {overBudget ? <p className="mt-1 text-xs font-medium text-coral animate-pulse">{t("budget.overBudgetWarning")}</p> : null}
            </div>
          );
        })() : null}

        {/* CSS Bar Chart — category planned vs actual */}
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

        {/* Expense list */}
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
