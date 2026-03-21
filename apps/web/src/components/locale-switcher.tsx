import clsx from "clsx";
import { useI18n } from "../lib/i18n";

export function LocaleSwitcher() {
  const { locale, setLocale, t } = useI18n();

  return (
    <div className="inline-flex items-center gap-1 rounded-full border border-ink/10 bg-white/85 p-1">
      <button
        className={clsx(
          "rounded-full px-3 py-1.5 text-xs font-medium transition",
          locale === "zh-TW" ? "bg-ink text-sand" : "text-ink/70 hover:bg-sand"
        )}
        onClick={() => setLocale("zh-TW")}
        type="button"
      >
        {t("locale.zh-TW")}
      </button>
      <button
        className={clsx(
          "rounded-full px-3 py-1.5 text-xs font-medium transition",
          locale === "en" ? "bg-ink text-sand" : "text-ink/70 hover:bg-sand"
        )}
        onClick={() => setLocale("en")}
        type="button"
      >
        {t("locale.en")}
      </button>
    </div>
  );
}
