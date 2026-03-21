import { apiRequest } from "./api";

export interface BudgetCategoryPlan {
  category: string;
  plannedAmount: number;
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
  input: { category: string; amount: number; currency: string; expenseAt?: string; note?: string }
) {
  return apiRequest<ExpenseItem>(`/api/v1/trips/${tripId}/expenses`, {
    method: "POST",
    headers: {
      "Idempotency-Key": crypto.randomUUID()
    },
    body: JSON.stringify(input)
  });
}
