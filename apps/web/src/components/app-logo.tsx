import { useI18n } from "../lib/i18n";

export function AppLogo() {
  const { t } = useI18n();

  return (
    <div className="flex items-center gap-3">
      <div className="flex h-10 w-10 items-center justify-center rounded-2xl bg-ink text-sm font-bold text-sand">
        TT
      </div>
      <div>
        <p className="font-display text-lg font-bold text-ink">{t("app.name")}</p>
      </div>
    </div>
  );
}
