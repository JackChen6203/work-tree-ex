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
import { useSessionStore } from "../../store/session-store";
import { useUiStore } from "../../store/ui-store";

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
  const [previewCode, setPreviewCode] = useState<string | null>(null);

  const onRequest = async () => {
    trackEvent({ name: analyticsEventNames.authLoginRequested, context: { method: "email_magic_link" } });
    try {
      const response = await requestMagicLink.mutateAsync(email);
      setPreviewCode(response.previewCode ?? null);
      pushToast(t("auth.linkSent"));
    } catch {
      trackEvent({ name: analyticsEventNames.authLoginFailed, context: { method: "email_magic_link", reason: "request_failed" } });
    }
  };

  const onVerify = async () => {
    try {
      const response = await verifyMagicLink.mutateAsync({ email, code });
      setUser(response.user);
      pushToast(t("auth.loginSuccess"));
      trackEvent({ name: analyticsEventNames.authLoginSucceeded, context: { method: "email_magic_link" } });
      navigate("/");
    } catch {
      trackEvent({ name: analyticsEventNames.authLoginFailed, context: { method: "email_magic_link", reason: "verify_failed" } });
    }
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
          setUser(session.user);
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
        // Keep the user on auth page for retry if callback hydration fails.
      }
    })();
  }, [navigate, pushToast, searchParams, setUser, t]);

  return (
    <div className="flex min-h-screen items-center justify-center px-4 py-10">
      <div className="w-full max-w-xl">
        <div className="mb-4 flex justify-end">
          <LocaleSwitcher />
        </div>
        <SurfaceCard className="w-full" eyebrow={t("auth.module")} title={t("auth.title")}>
          <div className="grid gap-6 md:grid-cols-[1.2fr_0.8fr]">
            <div>
              <p className="text-sm leading-7 text-ink/70">{t("auth.description")}</p>
              {magicLinkAuthEnabled ? (
                <form
                  className="mt-6 space-y-4"
                  onSubmit={(event) => {
                    event.preventDefault();
                    if (previewCode) {
                      void onVerify();
                      return;
                    }
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
                  {previewCode ? (
                    <label className="block">
                      <span className="mb-2 block text-sm font-medium text-ink">{t("auth.code")}</span>
                      <input
                        className="w-full rounded-2xl border border-ink/10 bg-sand px-4 py-3 outline-none transition focus:border-pine"
                        placeholder="123456"
                        value={code}
                        onChange={(event) => setCode(event.target.value)}
                        required
                      />
                    </label>
                  ) : null}
                  {previewCode ? <p className="text-xs text-pine">{t("auth.previewCode")}: {previewCode}</p> : null}
                  {requestMagicLink.error ? <p className="text-xs text-coral">{requestMagicLink.error.message}</p> : null}
                  {verifyMagicLink.error ? <p className="text-xs text-coral">{verifyMagicLink.error.message}</p> : null}
                  <button
                    type="submit"
                    className="rounded-full bg-ink px-5 py-3 text-sm font-medium text-sand transition hover:bg-pine"
                    disabled={requestMagicLink.isPending || verifyMagicLink.isPending}
                  >
                    {previewCode ? t("auth.verifyCode") : t("auth.sendLink")}
                  </button>
                  {previewCode ? (
                    <button
                      type="button"
                      className="ml-3 rounded-full border border-ink/20 px-5 py-3 text-sm font-medium text-ink transition hover:bg-sand"
                      onClick={() => {
                        setPreviewCode(null);
                        setCode("");
                      }}
                    >
                      {t("auth.requestAnother")}
                    </button>
                  ) : null}
                </form>
              ) : null}

              <div className={magicLinkAuthEnabled ? "mt-7 border-t border-ink/10 pt-5" : "mt-6"}>
                <p className="text-sm font-medium text-ink">{t("auth.oauthTitle")}</p>
                <p className="mt-1 text-xs text-ink/60">{t("auth.oauthDescription")}</p>
                <div className="mt-4 grid gap-2 sm:grid-cols-2">
                  {oauthProviders.map((provider) => (
                    <a
                      key={provider.id}
                      href={oauthStartUrl(provider.id)}
                      onClick={() => {
                        trackEvent({ name: analyticsEventNames.authLoginRequested, context: { method: "oauth", provider: provider.id } });
                      }}
                      className="rounded-full border border-ink/15 bg-white px-4 py-2 text-center text-sm font-medium text-ink transition hover:bg-sand"
                    >
                      {provider.label}
                    </a>
                  ))}
                </div>
              </div>
            </div>
            <div className="rounded-[24px] bg-gradient-to-br from-[#f5e4d9] to-[#d7e1dd] p-5">
              <p className="text-xs uppercase tracking-[0.2em] text-ink/45">{t("auth.sessionPolicy")}</p>
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
      </div>
    </div>
  );
}
