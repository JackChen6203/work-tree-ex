import type { ReactNode } from "react";

interface SurfaceCardProps {
  title?: string;
  eyebrow?: string;
  action?: ReactNode;
  children: ReactNode;
  className?: string;
}

export function SurfaceCard({ title, eyebrow, action, children, className = "" }: SurfaceCardProps) {
  return (
    <section className={`rounded-[28px] border border-white/70 bg-white/80 p-6 shadow-card backdrop-blur ${className}`}>
      {(title || eyebrow || action) && (
        <header className="mb-5 flex items-start justify-between gap-4">
          <div>
            {eyebrow ? <p className="text-xs uppercase tracking-[0.22em] text-ink/45">{eyebrow}</p> : null}
            {title ? <h2 className="mt-2 font-display text-2xl font-bold text-ink">{title}</h2> : null}
          </div>
          {action}
        </header>
      )}
      {children}
    </section>
  );
}
