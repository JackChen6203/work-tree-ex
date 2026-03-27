import { useState } from "react";
import { useEffect } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";
import { Link } from "react-router-dom";
import { SurfaceCard } from "../../components/surface-card";
import { LocaleSwitcher } from "../../components/locale-switcher";
import { analyticsEventNames, trackEvent } from "../../lib/analytics";
import { useI18n } from "../../lib/i18n";
import { getSession, oauthStartUrl } from "../../lib/auth-api";
import { magicLinkAuthEnabled, oauthProviders } from "../../lib/oauth-providers";
import { useRequestMagicLinkMutation, useVerifyMagicLinkMutation } from "../../lib/queries";
import { broadcastSessionSignedIn } from "../../lib/session-sync";
import { useSessionStore } from "../../store/session-store";
import { useUiStore } from "../../store/ui-store";

const oauthCategories = ["social", "travel"] as const;

export function AuthPage() {
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const { t } = useI18n();
  const setUser = useSessionStore((state) => state.setUser);
  const pushToast = useUiStore((state) => state.pushToast);
  const requestMagicLink = useRequestMagicLinkMutation();
  const verifyMagicLink = useVerifyMagicLinkMutation();
  const [email, setEmail] = useState("");
  const [code, setCode] = useState("");
  const [requestedEmail, setRequestedEmail] = useState("");
  const [previewCode, setPreviewCode] = useState<string | null>(null);
  const oauthProviderGroups = oauthCategories
    .map((category) => ({
      category,
      items: oauthProviders.filter((provider) => provider.category === category)
    }))
    .filter((group) => group.items.length > 0);

  const onRequest = async () => {
    trackEvent({ name: analyticsEventNames.authLoginRequested, context: { method: "email_magic_link" } });
    try {
      const response = await requestMagicLink.mutateAsync(email);
      setRequestedEmail(email);
      setPreviewCode(response.previewCode ?? null);
      pushToast(t("auth.linkSent"));
    } catch {
      trackEvent({ name: analyticsEventNames.authLoginFailed, context: { method: "email_magic_link", reason: "request_failed" } });
    }
  };

  const onVerify = async () => {
    try {
      const response = await verifyMagicLink.mutateAsync({ email, code });
      setUser(response.user, response.roles);
      broadcastSessionSignedIn(response.user, response.roles);
      pushToast(t("auth.loginSuccess"));
      trackEvent({ name: analyticsEventNames.authLoginSucceeded, context: { method: "email_magic_link" } });
      navigate("/");
    } catch {
      trackEvent({ name: analyticsEventNames.authLoginFailed, context: { method: "email_magic_link", reason: "verify_failed" } });
    }
  };

  const resetMagicLinkFlow = () => {
    setRequestedEmail("");
    setPreviewCode(null);
    setCode("");
  };

  useEffect(() => {
    if (searchParams.get("oauth") === "error") {
      pushToast(t("auth.oauthError"));
      trackEvent({
        name: analyticsEventNames.authLoginFailed,
        context: {
          method: "oauth",
          reason: searchParams.get("reason") ?? "oauth_error",
          provider: searchParams.get("provider") ?? "unknown"
        }
      });
      return;
    }

    if (searchParams.get("oauth") !== "success") {
      return;
    }

    void (async () => {
      try {
        const session = await getSession();
        if (session.user) {
          setUser(session.user, session.roles);
          broadcastSessionSignedIn(session.user, session.roles);
          pushToast(t("auth.oauthSuccess"));
          trackEvent({
            name: analyticsEventNames.authLoginSucceeded,
            context: {
              method: "oauth",
              provider: searchParams.get("provider") ?? "unknown"
            }
          });
          navigate("/");
        }
      } catch {
        trackEvent({ name: analyticsEventNames.authLoginFailed, context: { method: "oauth", reason: "session_hydration_failed" } });
      }
    })();
  }, [navigate, pushToast, searchParams, setUser, t]);

  return (
    <div className="relative min-h-screen overflow-hidden px-4 py-10">
      <div className="absolute inset-0 -z-10 bg-[radial-gradient(circle_at_15%_20%,rgba(218,106,78,0.24),transparent_24%),radial-gradient(circle_at_78%_12%,rgba(45,90,74,0.22),transparent_22%),linear-gradient(180deg,#f9f4ea_0%,#efe5d5_100%)]" />
      <header className="mx-auto flex w-full max-w-5xl justify-end pb-4">
        <LocaleSwitcher />
      </header>
      <main className="mx-auto w-full max-w-5xl" id="main-content">
        <SurfaceCard className="mx-auto w-full max-w-5xl" eyebrow={t("auth.module")} title={t("auth.title")} titleAs="h1">
          <div className="grid gap-6 md:grid-cols-[1.2fr_0.8fr]">
            <div>
              <p className="text-sm leading-7 text-ink/70">{t("auth.description")}</p>
              {magicLinkAuthEnabled ? (
                requestedEmail ? (
                  <div className="mt-6 space-y-4 rounded-[26px] border border-ink/10 bg-sand/70 p-5">
                    <div>
                      <p className="text-xs uppercase tracking-[0.24em] text-pine">{t("auth.waitingEyebrow")}</p>
                      <h3 className="mt-2 font-display text-2xl font-bold text-ink">{t("auth.waitingTitle")}</h3>
                      <p className="mt-2 text-sm leading-7 text-ink/70">{t("auth.waitingDescription")}</p>
                      <p className="mt-3 rounded-2xl bg-white px-4 py-3 text-sm font-medium text-ink">{requestedEmail}</p>
                    </div>

                    <form
                      className="space-y-4"
                      onSubmit={(event) => {
                        event.preventDefault();
                        void onVerify();
                      }}
                    >
                      <label className="block">
                        <span className="mb-2 block text-sm font-medium text-ink">{t("auth.code")}</span>
                        <input
                          className="w-full rounded-2xl border border-ink/10 bg-white px-4 py-3 outline-none transition focus:border-pine"
                          placeholder="123456"
                          value={code}
                          onChange={(event) => setCode(event.target.value)}
                          required
                        />
                      </label>
                      {previewCode ? <p className="text-xs text-pine">{t("auth.previewCode")}: {previewCode}</p> : null}
                      {verifyMagicLink.error ? <p className="text-xs text-coral">{verifyMagicLink.error.message}</p> : null}
                      <div className="flex flex-wrap gap-3">
                        <button
                          type="submit"
                          className="rounded-full bg-ink px-5 py-3 text-sm font-medium text-sand transition hover:bg-pine"
                          disabled={verifyMagicLink.isPending}
                        >
                          {verifyMagicLink.isPending ? t("auth.verifying") : t("auth.verifyCode")}
                        </button>
                        <button
                          type="button"
                          className="rounded-full border border-ink/20 bg-white px-5 py-3 text-sm font-medium text-ink transition hover:bg-sand"
                          disabled={requestMagicLink.isPending}
                          onClick={() => {
                            void onRequest();
                          }}
                        >
                          {requestMagicLink.isPending ? t("auth.resending") : t("auth.requestAnother")}
                        </button>
                        <button
                          type="button"
                          className="rounded-full border border-ink/12 px-5 py-3 text-sm font-medium text-ink/75 transition hover:bg-white"
                          onClick={resetMagicLinkFlow}
                        >
                          {t("auth.changeEmail")}
                        </button>
                      </div>
                    </form>
                  </div>
                ) : (
                  <form
                    className="mt-6 space-y-4"
                    onSubmit={(event) => {
                      event.preventDefault();
                      void onRequest();
                    }}
                  >
                    <label className="block">
                      <span className="mb-2 block text-sm font-medium text-ink">{t("auth.email")}</span>
                      <input
                        className="w-full rounded-2xl border border-ink/10 bg-sand px-4 py-3 outline-none transition focus:border-pine"
                        placeholder="you@example.com"
                        type="email"
                        value={email}
                        onChange={(event) => setEmail(event.target.value)}
                        required
                      />
                    </label>
                    <p className="text-xs leading-6 text-ink/60">{t("auth.magicLinkHint")}</p>
                    {requestMagicLink.error ? <p className="text-xs text-coral">{requestMagicLink.error.message}</p> : null}
                    <button
                      type="submit"
                      className="rounded-full bg-ink px-5 py-3 text-sm font-medium text-sand transition hover:bg-pine"
                      disabled={requestMagicLink.isPending}
                    >
                      {requestMagicLink.isPending ? t("auth.sendingLink") : t("auth.sendLink")}
                    </button>
                  </form>
                )
              ) : null}

              <div className={magicLinkAuthEnabled ? "mt-7 border-t border-ink/10 pt-5" : "mt-6"}>
                <p className="text-sm font-medium text-ink">{t("auth.oauthTitle")}</p>
                <p className="mt-1 text-xs text-ink/70">{t("auth.oauthDescription")}</p>
                {oauthProviderGroups.length > 0 ? (
                  <div className="mt-4 space-y-4">
                    {oauthProviderGroups.map((group) => (
                      <section key={group.category}>
                        <p className="mb-2 text-xs uppercase tracking-[0.22em] text-ink/70">
                          {group.category === "social" ? t("auth.oauthCategorySocial") : t("auth.oauthCategoryTravel")}
                        </p>
                        <div className="grid gap-2 sm:grid-cols-2">
                          {group.items.map((provider) => (
                            <a
                              key={provider.id}
                              href={oauthStartUrl(provider.id)}
                              onClick={() => {
                                trackEvent({ name: analyticsEventNames.authLoginRequested, context: { method: "oauth", provider: provider.id } });
                              }}
                              className="rounded-[20px] border border-ink/15 bg-white px-4 py-3 text-center text-sm font-medium text-ink transition hover:bg-sand"
                            >
                              {provider.label}
                            </a>
                          ))}
                        </div>
                      </section>
                    ))}
                  </div>
                ) : (
                  <p className="mt-4 rounded-[20px] border border-ink/10 bg-sand/60 px-4 py-3 text-sm text-ink/65">{t("auth.noOAuthProviders")}</p>
                )}
              </div>
            </div>
            <div className="rounded-[24px] bg-gradient-to-br from-[#f5e4d9] to-[#d7e1dd] p-5">
              <p className="text-xs uppercase tracking-[0.2em] text-ink/70">{t("auth.sessionPolicy")}</p>
              <ul className="mt-4 space-y-3 text-sm text-ink/75">
                <li>{t("auth.policy.1")}</li>
                <li>{t("auth.policy.2")}</li>
                <li>{t("auth.policy.3")}</li>
              </ul>
              <Link className="mt-6 inline-block text-sm font-medium text-pine" to="/">
                {t("auth.enterDemo")}
              </Link>
              <Link className="mt-3 block text-sm font-medium text-ink/75" to="/welcome">
                {t("auth.backWelcome")}
              </Link>
            </div>
          </div>
        </SurfaceCard>
      </main>
    </div>
  );
}
