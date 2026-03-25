import { useUiStore } from "../store/ui-store";

export function LoadingOverlay() {
  const loadingOverlay = useUiStore((state) => state.loadingOverlay);

  if (!loadingOverlay.visible) {
    return null;
  }

  return (
    <div
      aria-busy="true"
      aria-live="assertive"
      className="fixed inset-0 z-[80] flex items-center justify-center bg-ink/35 px-4 py-6 backdrop-blur-sm"
      role="status"
    >
      <div className="w-full max-w-sm rounded-[28px] border border-white/70 bg-white/92 p-6 text-center shadow-card">
        <div className="mx-auto h-12 w-12 animate-spin rounded-full border-4 border-sand border-t-pine" />
        <p className="mt-5 text-xs uppercase tracking-[0.24em] text-ink/45">Workspace busy</p>
        <p className="mt-3 text-sm font-medium text-ink">{loadingOverlay.label}</p>
      </div>
    </div>
  );
}
