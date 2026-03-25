import clsx from "clsx";
import { useEffect } from "react";
import { useUiStore } from "../store/ui-store";

export function ToastRegion() {
  const { toasts, dismissToast } = useUiStore();

  useEffect(() => {
    const timers = toasts.map((toast) =>
      window.setTimeout(() => {
        dismissToast(toast.id);
      }, toast.durationMs ?? 3000)
    );

    return () => {
      timers.forEach((timer) => window.clearTimeout(timer));
    };
  }, [dismissToast, toasts]);

  return (
    <div
      aria-atomic="true"
      aria-live="polite"
      className="fixed right-4 top-4 z-50 flex w-[min(360px,calc(100vw-2rem))] flex-col gap-3"
      role="status"
    >
      {toasts.map((toast) => (
        <div
          key={toast.id}
          className={clsx(
            "rounded-[22px] border px-4 py-3 text-sm shadow-card backdrop-blur",
            toast.type === "info" && "border-ink/10 bg-ink text-sand",
            toast.type === "success" && "border-pine/20 bg-pine text-white",
            toast.type === "warning" && "border-[#d7ae57]/25 bg-[#f7e2a8] text-[#5e4612]",
            toast.type === "error" && "border-coral/20 bg-coral text-white"
          )}
        >
          <p className="font-medium">{toast.message}</p>
        </div>
      ))}
    </div>
  );
}
