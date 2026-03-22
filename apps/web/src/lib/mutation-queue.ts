import { useSessionStore } from "../store/session-store";

export async function trackQueuedMutation<T>(scope: string, run: () => Promise<T>) {
  const state = useSessionStore.getState();
  const mutationId = state.enqueuePendingMutation(scope);

  try {
    return await run();
  } finally {
    useSessionStore.getState().resolvePendingMutation(mutationId);
  }
}
