import { create } from "zustand";

export interface Toast {
  id: string;
  type: "success" | "error" | "warning" | "info";
  message: string;
  durationMs?: number;
}

interface ConfirmModalPayload {
  title: string;
  description: string;
  confirmLabel: string;
  cancelLabel: string;
  tone?: "neutral" | "danger";
  onConfirm: () => void | Promise<void>;
}

interface InviteModalPayload {
  tripId: string;
  tripName: string;
}

interface AdoptDraftModalPayload {
  draftId: string;
  tripId: string;
  draftTitle: string;
  hasWarnings: boolean;
  onConfirm: (confirmWarnings: boolean) => void | Promise<void>;
}

type ActiveModal =
  | { type: "confirm"; payload: ConfirmModalPayload }
  | { type: "invite"; payload: InviteModalPayload }
  | { type: "adopt_draft"; payload: AdoptDraftModalPayload }
  | null;

interface LoadingOverlayState {
  visible: boolean;
  label: string;
}

type ActiveSheet =
  | { type: "mobile-nav" }
  | { type: "invite-context"; payload: { token: string; redirectTo: string } }
  | null;

type ToastInput =
  | string
  | {
      type?: Toast["type"];
      message: string;
      durationMs?: number;
    };

interface UiState {
  toasts: Toast[];
  activeModal: ActiveModal;
  activeSheet: ActiveSheet;
  loadingOverlay: LoadingOverlayState;
  swUpdateAvailable: boolean;
  pwaInstallable: boolean;
  pushToast: (input: ToastInput) => void;
  dismissToast: (id: string) => void;
  openConfirmModal: (payload: ConfirmModalPayload) => void;
  openInviteModal: (payload: InviteModalPayload) => void;
  openAdoptDraftModal: (payload: AdoptDraftModalPayload) => void;
  closeModal: () => void;
  openSheet: (type: "mobile-nav") => void;
  closeSheet: () => void;
  showLoadingOverlay: (label?: string) => void;
  hideLoadingOverlay: () => void;
  setSwUpdateAvailable: (available: boolean) => void;
  setPwaInstallable: (installable: boolean) => void;
}

const defaultUiState = {
  toasts: [],
  activeModal: null,
  activeSheet: null,
  loadingOverlay: {
    visible: false,
    label: "Loading..."
  },
  swUpdateAvailable: false,
  pwaInstallable: false
} satisfies Pick<UiState, "toasts" | "activeModal" | "activeSheet" | "loadingOverlay" | "swUpdateAvailable" | "pwaInstallable">;

export const useUiStore = create<UiState>((set) => ({
  ...defaultUiState,
  pushToast: (input) =>
    set((state) => {
      const nextToast =
        typeof input === "string"
          ? { id: crypto.randomUUID(), type: "info" as const, message: input, durationMs: 3000 }
          : {
              id: crypto.randomUUID(),
              type: input.type ?? "info",
              message: input.message,
              durationMs: input.durationMs ?? 3000
            };

      return {
        toasts: [...state.toasts, nextToast]
      };
    }),
  dismissToast: (id) =>
    set((state) => ({
      toasts: state.toasts.filter((toast) => toast.id !== id)
    })),
  openConfirmModal: (payload) =>
    set({ activeModal: { type: "confirm", payload } }),
  openInviteModal: (payload) =>
    set({ activeModal: { type: "invite", payload } }),
  openAdoptDraftModal: (payload) =>
    set({ activeModal: { type: "adopt_draft", payload } }),
  closeModal: () =>
    set({ activeModal: null }),
  openSheet: (type) =>
    set({ activeSheet: { type } }),
  closeSheet: () =>
    set({ activeSheet: null }),
  showLoadingOverlay: (label = "Loading...") =>
    set({ loadingOverlay: { visible: true, label } }),
  hideLoadingOverlay: () =>
    set({ loadingOverlay: { visible: false, label: "Loading..." } }),
  setSwUpdateAvailable: (available) =>
    set({ swUpdateAvailable: available }),
  setPwaInstallable: (installable) =>
    set({ pwaInstallable: installable })
}));

export function resetUiStore() {
  useUiStore.setState((state) => ({
    ...state,
    ...defaultUiState
  }));
}
