import { useParams } from "react-router-dom";
import { SurfaceCard } from "../../components/surface-card";
import { StatusPill } from "../../components/status-pill";
import { useState } from "react";
import { useBudgetProfileQuery, useCreateExpenseMutation, useDeleteExpenseMutation, useExpensesQuery, usePatchExpenseMutation, useUpsertBudgetMutation } from "../../lib/queries";
import { useUiStore } from "../../store/ui-store";

export function BudgetPage() {
  const { tripId = "" } = useParams();
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
    pushToast("Budget profile saved");
  };

  const addSampleExpense = async () => {
    await createExpense.mutateAsync({
      category: "food",
      amount: 4800,
      currency: profile?.currency ?? "JPY",
      note: "Pontocho dinner"
    });
    pushToast("Expense created");
  };

  const removeExpense = async (expenseId: string) => {
    await deleteExpense.mutateAsync(expenseId);
    pushToast("Expense removed");
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
      pushToast("Amount must be a non-negative number");
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
    pushToast("Expense updated");
  };

  return (
    <div className="grid gap-6 lg:grid-cols-[0.9fr_1.1fr]">
      <SurfaceCard eyebrow="Budget Module" title="Cost guardrails">
        {loadingProfile ? <div className="mb-4 rounded-[20px] bg-sand p-3 text-sm text-ink/65">Loading budget profile...</div> : null}
        <div className="grid gap-4 sm:grid-cols-3">
          <div className="rounded-[24px] bg-sand p-4">
            <p className="text-sm text-ink/60">Total budget</p>
            <p className="mt-2 font-display text-3xl font-bold text-ink">{profile?.currency ?? "JPY"} {estimated.toLocaleString()}</p>
          </div>
          <div className="rounded-[24px] bg-sand p-4">
            <p className="text-sm text-ink/60">Actual spend</p>
            <p className="mt-2 font-display text-3xl font-bold text-ink">{profile?.currency ?? "JPY"} {actual.toLocaleString()}</p>
          </div>
          <div className="rounded-[24px] bg-sand p-4">
            <p className="text-sm text-ink/60">Per day target</p>
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
            {upsertBudget.isPending ? "Saving..." : "Save budget profile"}
          </button>
          <button
            className="rounded-full bg-coral px-5 py-3 text-sm font-medium text-white"
            disabled={createExpense.isPending}
            onClick={() => {
              void addSampleExpense();
            }}
            type="button"
          >
            {createExpense.isPending ? "Adding..." : "Add sample expense"}
          </button>
        </div>
        {profile?.overBudget ? <p className="mt-4 text-sm font-medium text-coral">Over budget alert: actual spend is above 110% threshold.</p> : null}
      </SurfaceCard>
      <SurfaceCard eyebrow="Breakdown" title="Estimated vs actual">
        {loadingExpenses ? <div className="rounded-[20px] bg-sand p-3 text-sm text-ink/65">Loading expenses...</div> : null}
        <div className="space-y-4">
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
                      Save
                    </button>
                    <button
                      className="rounded-full border border-ink/15 px-3 py-1 text-xs font-medium text-ink"
                      onClick={() => setEditingExpenseId(null)}
                      type="button"
                    >
                      Cancel
                    </button>
                  </div>
                  <input
                    className="md:col-span-3 rounded-xl border border-ink/10 bg-white px-3 py-2 text-sm text-ink"
                    placeholder="Note"
                    value={editingNote}
                    onChange={(event) => setEditingNote(event.target.value)}
                  />
                </div>
              ) : (
                <div className="flex flex-wrap items-center justify-between gap-3">
                  <div>
                    <p className="font-medium text-ink">{expense.category}</p>
                    <p className="text-sm text-ink/60">{expense.currency} {expense.amount.toLocaleString()} {expense.note ? `· ${expense.note}` : ""}</p>
                  </div>
                  <div className="flex items-center gap-2">
                    <button
                      className="rounded-full border border-ink/15 px-3 py-1 text-xs font-medium text-ink"
                      disabled={deleteExpense.isPending}
                      onClick={() => beginEditExpense(expense.id, expense.category, expense.amount, expense.note)}
                      type="button"
                    >
                      Edit
                    </button>
                    <button
                      className="rounded-full border border-ink/15 px-3 py-1 text-xs font-medium text-ink"
                      disabled={deleteExpense.isPending}
                      onClick={() => {
                        void removeExpense(expense.id);
                      }}
                      type="button"
                    >
                      Remove
                    </button>
                  </div>
                </div>
              )}
            </div>
          ))}

          {(profile?.categories ?? []).map((category) => {
            const categoryActual = expenses.filter((item) => item.category === category.category).reduce((sum, item) => sum + item.amount, 0);
            const variance = categoryActual - category.plannedAmount;
            return (
              <div key={category.category} className="rounded-[24px] bg-white p-4">
                <div className="flex flex-wrap items-center justify-between gap-3">
                  <div>
                    <p className="font-medium text-ink">{category.category}</p>
                    <p className="text-sm text-ink/55">Est. {category.plannedAmount.toLocaleString()} / Actual {categoryActual.toLocaleString()}</p>
                  </div>
                  <StatusPill tone={variance > 0 ? "danger" : "success"}>
                    {variance > 0 ? `+${variance.toLocaleString()}` : `${variance.toLocaleString()}`}
                  </StatusPill>
                </div>
              </div>
            );
          })}
          {!loadingProfile && (profile?.categories?.length ?? 0) === 0 ? (
            <div className="rounded-[24px] bg-sand p-4 text-sm text-ink/65">No category plan yet. Save a budget profile first.</div>
          ) : null}
        </div>
      </SurfaceCard>
    </div>
  );
}
