import type { BudgetRateSnapshot, ExpenseItem } from "../../lib/budget-api";

export interface ConvertedAmount {
  value: number;
  rate: BudgetRateSnapshot | null;
  direction: "direct" | "inverse" | "same";
}

export function buildRateLookup(rates: BudgetRateSnapshot[]) {
  const map = new Map<string, BudgetRateSnapshot>();
  for (const rate of rates) {
    map.set(`${rate.from}:${rate.to}`, rate);
  }
  return map;
}

export function convertAmount(
  amount: number,
  fromCurrency: string,
  toCurrency: string,
  lookup: Map<string, BudgetRateSnapshot>
): ConvertedAmount | null {
  const from = fromCurrency.toUpperCase();
  const to = toCurrency.toUpperCase();

  if (from === to) {
    return {
      value: amount,
      rate: null,
      direction: "same"
    };
  }

  const direct = lookup.get(`${from}:${to}`);
  if (direct) {
    return {
      value: amount * direct.rate,
      rate: direct,
      direction: "direct"
    };
  }

  const inverse = lookup.get(`${to}:${from}`);
  if (inverse && inverse.rate > 0) {
    return {
      value: amount / inverse.rate,
      rate: inverse,
      direction: "inverse"
    };
  }

  return null;
}

export function isRateStale(rate: BudgetRateSnapshot | null, nowMs = Date.now(), thresholdHours = 24) {
  if (!rate) {
    return false;
  }
  if (rate.staleAt) {
    return true;
  }
  const fetchedMs = new Date(rate.fetchedAt).getTime();
  if (Number.isNaN(fetchedMs)) {
    return true;
  }
  return nowMs - fetchedMs > thresholdHours * 60 * 60 * 1000;
}

export function toTripCurrencyAmount(
  expense: ExpenseItem,
  tripCurrency: string,
  lookup: Map<string, BudgetRateSnapshot>
) {
  const converted = convertAmount(expense.amount, expense.currency, tripCurrency, lookup);
  if (!converted) {
    return null;
  }
  return {
    amount: converted.value,
    rate: converted.rate,
    direction: converted.direction
  };
}

