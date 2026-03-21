import { Link } from "react-router-dom";
import { SurfaceCard } from "../../components/surface-card";
import { LocaleSwitcher } from "../../components/locale-switcher";
import { oauthStartUrl } from "../../lib/auth-api";
import { useI18n } from "../../lib/i18n";
import { oauthProviders } from "../../lib/oauth-providers";

export function WelcomePage() {
  const { t } = useI18n();

  return (
    <div className="relative min-h-screen overflow-hidden px-4 py-10">
      <div className="absolute inset-0 -z-10 bg-[radial-gradient(circle_at_15%_20%,rgba(218,106,78,0.24),transparent_24%),radial-gradient(circle_at_78%_12%,rgba(45,90,74,0.22),transparent_22%),linear-gradient(180deg,#f9f4ea_0%,#efe5d5_100%)]" />
      <div className="mx-auto flex w-full max-w-5xl justify-end pb-4">
        <LocaleSwitcher />
      </div>
      <SurfaceCard className="mx-auto w-full max-w-5xl" eyebrow={t("welcome.badge")} title={t("welcome.title")}>
        <div className="grid gap-8 md:grid-cols-[1.15fr_0.85fr]">
          <div>
            <p className="text-sm leading-7 text-ink/75">{t("welcome.subtitle")}</p>
            <div className="mt-6 flex flex-wrap gap-3">
              <Link className="rounded-full bg-ink px-5 py-3 text-sm font-medium text-sand transition hover:bg-pine" to="/login">
                {t("welcome.ctaLogin")}
              </Link>
              <Link className="rounded-full bg-white px-5 py-3 text-sm font-medium text-ink transition hover:bg-sand" to="/">
                {t("welcome.ctaWorkspace")}
              </Link>
            </div>
            <div className="mt-6 grid gap-2 sm:grid-cols-2">
              {oauthProviders.slice(0, 6).map((provider) => (
                <a
                  key={provider.id}
                  href={oauthStartUrl(provider.id)}
                  className="rounded-full border border-ink/15 bg-white px-4 py-2 text-center text-sm font-medium text-ink transition hover:bg-sand"
                >
                  {provider.label}
                </a>
              ))}
            </div>
          </div>
          <div className="rounded-[24px] bg-white p-5">
            <ul className="space-y-3 text-sm text-ink/75">
              <li>{t("welcome.feature.1")}</li>
              <li>{t("welcome.feature.2")}</li>
              <li>{t("welcome.feature.3")}</li>
              <li>{t("welcome.feature.4")}</li>
            </ul>
          </div>
        </div>
      </SurfaceCard>
    </div>
  );
}
