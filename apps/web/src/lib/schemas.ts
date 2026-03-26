import { z } from "zod";

// ── Trip ──────────────────────────────────────────────

export const createTripSchema = z.object({
  name: z.string().min(1, "required").max(200, "max200"),
  departureText: z.string().min(1, "required"),
  destinationText: z.string().min(1, "required"),
  startDate: z.string().min(1, "required"),
  endDate: z.string().min(1, "required"),
  timezone: z.string().min(1, "required"),
  currency: z.string().length(3, "currency3"),
  totalBudget: z.number({ coerce: true }).min(0, "amountNonNegative").optional(),
  pace: z.enum(["relaxed", "balanced", "packed"]).optional(),
  travelersCount: z.number({ coerce: true }).int().min(1, "min1").max(50, "max50")
}).refine((data) => data.endDate >= data.startDate, {
  message: "endDateBeforeStart",
  path: ["endDate"]
});

export type CreateTripFormValues = z.infer<typeof createTripSchema>;

// ── Expense ──────────────────────────────────────────

export const addExpenseSchema = z.object({
  category: z.string().min(1, "required"),
  amount: z.number({ coerce: true }).min(0, "amountNonNegative"),
  currency: z.string().min(1, "required"),
  note: z.string().max(1000).optional(),
  expenseAt: z.string().optional()
});

export type AddExpenseFormValues = z.infer<typeof addExpenseSchema>;

export const budgetProfileSchema = z.object({
  totalBudget: z.number({ coerce: true }).min(0, "amountNonNegative").optional(),
  perPersonBudget: z.number({ coerce: true }).min(0, "amountNonNegative").optional(),
  perDayBudget: z.number({ coerce: true }).min(0, "amountNonNegative").optional(),
  currency: z.string().length(3, "currency3"),
  categories: z.array(
    z.object({
      category: z.string().min(1, "required"),
      plannedAmount: z.number({ coerce: true }).min(0, "amountNonNegative")
    })
  )
});

export type BudgetProfileFormValues = z.infer<typeof budgetProfileSchema>;

// ── Member ──────────────────────────────────────────

export const addMemberSchema = z.object({
  email: z.string().email("invalidEmail"),
  role: z.enum(["editor", "commenter", "viewer"])
});

export type AddMemberFormValues = z.infer<typeof addMemberSchema>;

// ── Itinerary Item ──────────────────────────────────

export const addItineraryItemSchema = z.object({
  dayId: z.string().min(1),
  title: z.string().min(1, "required").max(200, "max200"),
  itemType: z.string().min(1, "required"),
  allDay: z.boolean(),
  startAt: z.string().optional(),
  endAt: z.string().optional(),
  note: z.string().max(5000, "max5000").optional()
});

export type AddItineraryItemFormValues = z.infer<typeof addItineraryItemSchema>;

// ── LLM Provider ────────────────────────────────────

export const llmProviderSchema = z.object({
  provider: z.string().min(1, "required"),
  label: z.string().min(1, "required"),
  model: z.string().min(1, "required"),
  encryptedApiKeyEnvelope: z.string().min(16, "apiKeyMin16")
});

export type LlmProviderFormValues = z.infer<typeof llmProviderSchema>;

// ── Validation Error i18n Map ───────────────────────

export const validationMessages: Record<string, Record<string, string>> = {
  "zh-TW": {
    required: "此欄位為必填",
    max200: "最多 200 字",
    max5000: "最多 5000 字",
    currency3: "幣別需為 3 碼（如 TWD）",
    min1: "最少 1 人",
    max50: "最多 50 人",
    endDateBeforeStart: "結束日期不可早於開始日期",
    amountPositive: "金額需大於 0",
    amountNonNegative: "金額不可為負數",
    requiredDestinationChoice: "請先選擇目的地",
    invalidEmail: "Email 格式不正確",
    apiKeyMin16: "API 金鑰至少 16 字元"
  },
  en: {
    required: "This field is required",
    max200: "Max 200 characters",
    max5000: "Max 5000 characters",
    currency3: "Currency must be 3 characters (e.g. USD)",
    min1: "Min 1 traveler",
    max50: "Max 50 travelers",
    endDateBeforeStart: "End date must be on or after start date",
    amountPositive: "Amount must be positive",
    amountNonNegative: "Amount cannot be negative",
    requiredDestinationChoice: "Select a destination first",
    invalidEmail: "Invalid email format",
    apiKeyMin16: "API key must be at least 16 characters"
  }
};
