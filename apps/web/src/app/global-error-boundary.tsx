import { Component, type ErrorInfo, type ReactNode } from "react";
import { dictionaries, type Locale } from "../lib/translations";

interface GlobalErrorBoundaryProps {
  children: ReactNode;
}

interface GlobalErrorBoundaryState {
  error: Error | null;
}

function resolveLocale(): Locale {
  if (typeof window === "undefined") {
    return "en";
  }

  const persisted = window.localStorage.getItem("tt.locale");
  if (persisted === "zh-TW" || persisted === "en") {
    return persisted;
  }

  return window.navigator.language.toLowerCase().startsWith("zh") ? "zh-TW" : "en";
}

function getErrorDescription(error: Error | null, locale: Locale) {
  const generic = dictionaries[locale]["errorBoundary.description"];
  if (!error?.message) {
    return generic;
  }

  return `${generic} (${error.message})`;
}

export class GlobalErrorBoundary extends Component<GlobalErrorBoundaryProps, GlobalErrorBoundaryState> {
  override state: GlobalErrorBoundaryState = {
    error: null
  };

  static getDerivedStateFromError(error: Error) {
    return { error };
  }

  override componentDidCatch(error: Error, errorInfo: ErrorInfo) {
    console.error("Global error boundary caught a render failure.", error, errorInfo);
  }

  private resetBoundary = () => {
    this.setState({ error: null });
  };

  private reloadApplication = () => {
    window.location.reload();
  };

  override render() {
    if (!this.state.error) {
      return this.props.children;
    }

    const locale = resolveLocale();
    const title = dictionaries[locale]["errorBoundary.title"];
    const label = dictionaries[locale]["errorBoundary.label"];
    const retry = dictionaries[locale]["errorBoundary.retry"];
    const reload = dictionaries[locale]["errorBoundary.reload"];
    const description = getErrorDescription(this.state.error, locale);

    return (
      <div className="flex min-h-screen items-center justify-center px-6 py-10">
        <div className="w-full max-w-2xl rounded-[36px] border border-white/70 bg-white/85 p-8 shadow-card backdrop-blur">
          <p className="text-xs uppercase tracking-[0.26em] text-coral">{label}</p>
          <h1 className="mt-4 font-display text-4xl font-bold text-ink">{title}</h1>
          <p className="mt-4 max-w-xl text-sm leading-7 text-ink/72">{description}</p>
          <div className="mt-8 flex flex-wrap gap-3">
            <button
              className="rounded-full bg-ink px-5 py-3 text-sm font-medium text-sand transition hover:opacity-90"
              onClick={this.resetBoundary}
              type="button"
            >
              {retry}
            </button>
            <button
              className="rounded-full border border-ink/12 bg-sand px-5 py-3 text-sm font-medium text-ink transition hover:bg-white"
              onClick={this.reloadApplication}
              type="button"
            >
              {reload}
            </button>
          </div>
        </div>
      </div>
    );
  }
}
