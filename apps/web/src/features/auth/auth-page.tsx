import { Link } from "react-router-dom";
import { SurfaceCard } from "../../components/surface-card";

export function AuthPage() {
  return (
    <div className="flex min-h-screen items-center justify-center px-4 py-10">
      <SurfaceCard className="w-full max-w-xl" eyebrow="Auth Module" title="Email magic link sign-in">
        <div className="grid gap-6 md:grid-cols-[1.2fr_0.8fr]">
          <div>
            <p className="text-sm leading-7 text-ink/70">
              依規格先保留 email magic link / OTP 流程入口，後續可接 OAuth provider 與 invite pre-auth/post-auth。
            </p>
            <form className="mt-6 space-y-4">
              <label className="block">
                <span className="mb-2 block text-sm font-medium text-ink">Email</span>
                <input
                  className="w-full rounded-2xl border border-ink/10 bg-sand px-4 py-3 outline-none transition focus:border-pine"
                  placeholder="you@example.com"
                />
              </label>
              <button
                type="button"
                className="rounded-full bg-ink px-5 py-3 text-sm font-medium text-sand transition hover:bg-pine"
              >
                Send sign-in link
              </button>
            </form>
          </div>
          <div className="rounded-[24px] bg-gradient-to-br from-[#f5e4d9] to-[#d7e1dd] p-5">
            <p className="text-xs uppercase tracking-[0.2em] text-ink/45">Session policy</p>
            <ul className="mt-4 space-y-3 text-sm text-ink/75">
              <li>Token refresh failure triggers secure sign-out.</li>
              <li>Invite acceptance flow remains consistent across auth states.</li>
              <li>Provider secrets never enter persistent browser storage.</li>
            </ul>
            <Link className="mt-6 inline-block text-sm font-medium text-pine" to="/">
              Enter demo workspace
            </Link>
          </div>
        </div>
      </SurfaceCard>
    </div>
  );
}
