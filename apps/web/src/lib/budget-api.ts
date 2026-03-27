import { apiRequest } from "./api";

export interface BudgetCategoryPlan {
  category: string;
  plannedAmount: number;
}

export interface BudgetRateSnapshot {
  from: string;
  to: string;
  rate: number;
  source: string;
  fetchedAt: string;
  staleAt?: string;
}

export interface BudgetProfile {
  tripId: string;
  totalBudget?: number;
  perPersonBudget?: number;
  perDayBudget?: number;
  currency: string;
  categories: BudgetCategoryPlan[];
  version: number;
  actualSpend: number;
  overBudget: boolean;
  createdAt?: string;
  updatedAt?: string;
}

export interface ExpenseItem {
  id: string;
  tripId: string;
  category: string;
  amount: number;
  currency: string;
  expenseAt?: string;
  note?: string;
  linkedItemId?: string;
  createdAt: string;
}

export function getBudgetProfile(tripId: string) {
  return apiRequest<BudgetProfile>(`/api/v1/trips/${tripId}/budget`);
}

export function upsertBudgetProfile(
  tripId: string,
  input: {
    totalBudget?: number;
    perPersonBudget?: number;
    perDayBudget?: number;
    currency: string;
    categories: BudgetCategoryPlan[];
  }
) {
  return apiRequest<BudgetProfile>(`/api/v1/trips/${tripId}/budget`, {
    method: "PUT",
    headers: {
      "Idempotency-Key": crypto.randomUUID()
    },
    body: JSON.stringify(input)
  });
}

export function listExpenses(tripId: string) {
  return apiRequest<ExpenseItem[]>(`/api/v1/trips/${tripId}/expenses`);
}

export function createExpense(
  tripId: string,
  input: {
    category: string;
    amount: number;
    currency: string;
    expenseAt?: string;
    note?: string;
    linkedItemId?: string;
  }
) {
  return apiRequest<ExpenseItem>(`/api/v1/trips/${tripId}/expenses`, {
    method: "POST",
    headers: {
      "Idempotency-Key": crypto.randomUUID()
    },
    body: JSON.stringify(input)
  });
}

export function deleteExpense(tripId: string, expenseId: string) {
  return apiRequest<void>(`/api/v1/trips/${tripId}/expenses/${expenseId}`, {
    method: "DELETE"
  });
}

export function patchExpense(
  tripId: string,
  expenseId: string,
  input: {
    category?: string;
    amount?: number;
    currency?: string;
    expenseAt?: string;
    note?: string;
    linkedItemId?: string;
  }
) {
  return apiRequest<ExpenseItem>(`/api/v1/trips/${tripId}/expenses/${expenseId}`, {
    method: "PATCH",
    body: JSON.stringify(input)
  });
}

export function getBudgetRates(tripId: string, options?: { from?: string; to?: string }) {
  const params = new URLSearchParams();
  if (options?.from) {
    params.set("from", options.from.toUpperCase());
  }
  if (options?.to) {
    params.set("to", options.to.toUpperCase());
  }

  const query = params.toString();
  return apiRequest<BudgetRateSnapshot | BudgetRateSnapshot[]>(
    `/api/v1/trips/${tripId}/budget/rates${query ? `?${query}` : ""}`
  );
}

export function refreshBudgetRate(tripId: string, from: string, to: string) {
  const params = new URLSearchParams({
    from: from.toUpperCase(),
    to: to.toUpperCase()
  });
  return apiRequest<BudgetRateSnapshot>(`/api/v1/trips/${tripId}/budget/rates/refresh?${params.toString()}`, {
    method: "POST"
  });
}
