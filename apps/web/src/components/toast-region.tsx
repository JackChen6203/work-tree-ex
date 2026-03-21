import { useEffect } from "react";
import { useUiStore } from "../store/ui-store";

export function ToastRegion() {
  const { toasts, dismissToast } = useUiStore();

  useEffect(() => {
    const timers = toasts.map((toast) =>
      window.setTimeout(() => {
        dismissToast(toast.id);
      }, 2400)
    );

    return () => {
      timers.forEach((timer) => window.clearTimeout(timer));
    };
  }, [dismissToast, toasts]);

  return (
    <div className="fixed bottom-4 right-4 z-50 flex w-[min(360px,calc(100vw-2rem))] flex-col gap-3">
      {toasts.map((toast) => (
        <div key={toast.id} className="rounded-2xl bg-ink px-4 py-3 text-sm text-sand shadow-card">
          {toast.message}
        </div>
      ))}
    </div>
  );
}
