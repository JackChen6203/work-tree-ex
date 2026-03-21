export function AppLogo() {
  return (
    <div className="flex items-center gap-3">
      <div className="flex h-10 w-10 items-center justify-center rounded-2xl bg-ink text-sm font-bold text-sand">
        TT
      </div>
      <div>
        <p className="font-display text-lg font-bold text-ink">Travel Planner</p>
        <p className="text-xs uppercase tracking-[0.24em] text-ink/50">Offline-first PWA</p>
      </div>
    </div>
  );
}
