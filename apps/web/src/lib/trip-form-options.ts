export interface CurrencyOption {
  code: string;
  label: string;
}

export const currencyOptions: CurrencyOption[] = [
  { code: "TWD", label: "TWD - New Taiwan Dollar" },
  { code: "USD", label: "USD - US Dollar" },
  { code: "JPY", label: "JPY - Japanese Yen" },
  { code: "KRW", label: "KRW - South Korean Won" },
  { code: "CNY", label: "CNY - Chinese Yuan" },
  { code: "HKD", label: "HKD - Hong Kong Dollar" },
  { code: "SGD", label: "SGD - Singapore Dollar" },
  { code: "EUR", label: "EUR - Euro" },
  { code: "GBP", label: "GBP - British Pound" },
  { code: "AUD", label: "AUD - Australian Dollar" },
  { code: "CAD", label: "CAD - Canadian Dollar" }
];

const defaultTimezone = Intl.DateTimeFormat().resolvedOptions().timeZone || "UTC";
const intlWithSupportedValues = Intl as typeof Intl & { supportedValuesOf?: (key: "timeZone") => string[] };

export const timezoneOptions = (() => {
  if (typeof intlWithSupportedValues.supportedValuesOf === "function") {
    try {
      const zones = intlWithSupportedValues.supportedValuesOf("timeZone");
      if (zones.length > 0) {
        return zones;
      }
    } catch {
      // Fallback to static options when this runtime cannot enumerate time zones.
    }
  }

  return [
    defaultTimezone,
    "UTC",
    "Asia/Taipei",
    "Asia/Tokyo",
    "Asia/Seoul",
    "Asia/Singapore",
    "Europe/London",
    "Europe/Paris",
    "America/Los_Angeles",
    "America/New_York"
  ];
})();

export const budgetSeedCategories = ["lodging", "transit", "food", "attraction", "shopping", "misc"] as const;
