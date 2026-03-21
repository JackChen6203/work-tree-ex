import type { ReactNode } from "react";
import clsx from "clsx";

interface StatusPillProps {
  tone?: "neutral" | "danger" | "success" | "accent";
  children: ReactNode;
}

export function StatusPill({ tone = "neutral", children }: StatusPillProps) {
  return (
    <span
      className={clsx(
        "inline-flex rounded-full px-3 py-1 text-xs font-medium",
        tone === "neutral" && "bg-ink/5 text-ink/70",
        tone === "danger" && "bg-coral/15 text-coral",
        tone === "success" && "bg-pine/15 text-pine",
        tone === "accent" && "bg-[#e8d9b5] text-[#785d1f]"
      )}
    >
      {children}
    </span>
  );
}
