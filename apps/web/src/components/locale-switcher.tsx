import clsx from "clsx";
import { useI18n } from "../lib/i18n";

export function LocaleSwitcher() {
  const { locale, setLocale, t } = useI18n();

  return (
    <div
      aria-label={t("locale.switcher")}
      className="inline-flex items-center gap-1 rounded-full border border-ink/10 bg-white/85 p-1"
      role="group"
    >
      <button
        aria-label={t("locale.switchToZhTw")}
        aria-pressed={locale === "zh-TW"}
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
        aria-label={t("locale.switchToEn")}
        aria-pressed={locale === "en"}
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
