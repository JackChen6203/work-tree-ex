import { useSessionStore } from "../store/session-store";
import { StatusPill } from "./status-pill";

export function SyncStatusBar() {
  const { isOnline, pendingMutations } = useSessionStore();

  return (
    <div className="flex flex-wrap items-center gap-3 rounded-full border border-ink/10 bg-white/75 px-4 py-3 text-sm text-ink/70">
      <StatusPill tone={isOnline ? "success" : "danger"}>{isOnline ? "Online" : "Offline"}</StatusPill>
      <span>Mutation queue: {pendingMutations}</span>
      <span>Server authoritative sync enabled</span>
    </div>
  );
}
