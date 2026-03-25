import { useState } from "react";
import { useUiStore } from "../store/ui-store";
import { useI18n } from "../lib/i18n";

export function GlobalModalHost() {
  const activeModal = useUiStore((state) => state.activeModal);
  const closeModal = useUiStore((state) => state.closeModal);
  const pushToast = useUiStore((state) => state.pushToast);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const { t } = useI18n();

  if (!activeModal) {
    return null;
  }

  if (activeModal.type !== "confirm") {
    return null;
  }

  const { payload } = activeModal;

  const onConfirm = async () => {
    setIsSubmitting(true);
    try {
      await payload.onConfirm();
      closeModal();
    } catch (error) {
      pushToast({
        type: "error",
        message: error instanceof Error ? error.message : t("common.actionFailed")
      });
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <div className="fixed inset-0 z-[70] flex items-center justify-center bg-ink/45 px-4 py-6 backdrop-blur-sm">
      <div
        aria-describedby="global-confirm-description"
        aria-modal="true"
        className="w-full max-w-lg rounded-[32px] border border-white/70 bg-white/95 p-6 shadow-card"
        role="dialog"
      >
        <p className={`text-xs uppercase tracking-[0.24em] ${payload.tone === "danger" ? "text-coral" : "text-pine"}`}>{t("common.confirm")}</p>
        <h2 className="mt-3 font-display text-3xl font-bold text-ink">{payload.title}</h2>
        <p className="mt-3 text-sm leading-7 text-ink/70" id="global-confirm-description">
          {payload.description}
        </p>
        <div className="mt-8 flex flex-wrap justify-end gap-3">
          <button
            className="rounded-full border border-ink/12 bg-sand px-5 py-3 text-sm font-medium text-ink transition hover:bg-white"
            disabled={isSubmitting}
            onClick={closeModal}
            type="button"
          >
            {payload.cancelLabel}
          </button>
          <button
            className={`rounded-full px-5 py-3 text-sm font-medium text-white transition ${
              payload.tone === "danger" ? "bg-coral hover:opacity-90" : "bg-ink hover:bg-pine"
            }`}
            disabled={isSubmitting}
            onClick={() => {
              void onConfirm();
            }}
            type="button"
          >
            {isSubmitting ? `${payload.confirmLabel}...` : payload.confirmLabel}
          </button>
        </div>
      </div>
    </div>
  );
}
