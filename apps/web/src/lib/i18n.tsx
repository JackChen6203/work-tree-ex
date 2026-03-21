import { createContext, useContext, useMemo, useState, type ReactNode } from "react";
import { dictionaries, type Locale, type TranslationKey } from "./translations";

interface I18nContextValue {
  locale: Locale;
  setLocale: (locale: Locale) => void;
  t: (key: TranslationKey) => string;
}

const LOCALE_STORAGE_KEY = "tt.locale";

const I18nContext = createContext<I18nContextValue | null>(null);

function resolveInitialLocale(): Locale {
  const persisted = localStorage.getItem(LOCALE_STORAGE_KEY);
  if (persisted === "zh-TW" || persisted === "en") {
    return persisted;
  }

  const browserLocale = navigator.language.toLowerCase();
  return browserLocale.startsWith("zh") ? "zh-TW" : "en";
}

export function I18nProvider({ children }: { children: ReactNode }) {
  const [locale, setLocaleState] = useState<Locale>(() => resolveInitialLocale());

  const setLocale = (nextLocale: Locale) => {
    setLocaleState(nextLocale);
    localStorage.setItem(LOCALE_STORAGE_KEY, nextLocale);
  };

  const value = useMemo<I18nContextValue>(() => {
    return {
      locale,
      setLocale,
      t: (key) => dictionaries[locale][key]
    };
  }, [locale]);

  return <I18nContext.Provider value={value}>{children}</I18nContext.Provider>;
}

export function useI18n() {
  const context = useContext(I18nContext);
  if (!context) {
    throw new Error("useI18n must be used within I18nProvider");
  }

  return context;
}
