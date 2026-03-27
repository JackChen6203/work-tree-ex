import { useEffect, useState } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { SurfaceCard } from "../../components/surface-card";
import {
  useCreateMyLlmProviderMutation,
  useDeleteMyAccountMutation,
  useDeleteMyLlmProviderMutation,
  useMyLlmProvidersQuery,
  useMyNotificationPreferencesQuery,
  useMyPreferencesQuery,
  useMyProfileQuery,
  usePutMyNotificationPreferencesMutation,
  usePatchMyProfileMutation,
  usePutMyPreferencesMutation
} from "../../lib/queries";
import { useUiStore } from "../../store/ui-store";
import { useI18n } from "../../lib/i18n";
import { llmProviderSchema, validationMessages } from "../../lib/schemas";
import { isPushConfigured, setupPushMessaging } from "../../lib/fcm-messaging";
import { oauthProviders } from "../../lib/oauth-providers";
import type { Locale } from "../../lib/translations";
import { useSessionStore } from "../../store/session-store";

interface ProfileFormValues {
  displayName: string;
  locale: string;
  timezone: string;
  currency: string;
}

interface PreferenceFormValues {
  tripPace: string;
  wakePattern: string;
  transportPreference: string;
  foodPreference: string;
  avoidTags: string;
}

interface NotificationPreferenceFormValues {
  pushEnabled: boolean;
  emailEnabled: boolean;
  digestFrequency: "instant" | "daily" | "weekly";
  quietHoursStart: string;
  quietHoursEnd: string;
  tripUpdates: boolean;
  budgetAlerts: boolean;
  aiPlanReadyAlerts: boolean;
}

interface LlmProviderFormValues {
  provider: string;
  label: string;
  model: string;
  encryptedApiKeyEnvelope: string;
}

export function SettingsPage() {
  const { t } = useI18n();
  const pushToast = useUiStore((state) => state.pushToast);
  const openConfirmModal = useUiStore((state) => state.openConfirmModal);
  const [testingConnection, setTestingConnection] = useState(false);
  const [linkedProviders, setLinkedProviders] = useState<string[]>([]);

  const { data: profile, isLoading: profileLoading } = useMyProfileQuery();
  const { data: preferences, isLoading: preferencesLoading } = useMyPreferencesQuery();
  const { data: notificationPreferences, isLoading: notificationPreferencesLoading } = useMyNotificationPreferencesQuery();
  const { data: providers = [], isLoading: providersLoading } = useMyLlmProvidersQuery();

  const patchProfile = usePatchMyProfileMutation();
  const putPreferences = usePutMyPreferencesMutation();
  const putNotificationPreferences = usePutMyNotificationPreferencesMutation();
  const createProvider = useCreateMyLlmProviderMutation();
  const deleteProvider = useDeleteMyLlmProviderMutation();
  const deleteMyAccount = useDeleteMyAccountMutation();
  const clearUser = useSessionStore((state) => state.clearUser);

  const profileForm = useForm<ProfileFormValues>({
    values: profile
      ? {
          displayName: profile.displayName,
          locale: profile.locale,
          timezone: profile.timezone,
          currency: profile.currency
        }
      : undefined
  });

  const preferenceForm = useForm<PreferenceFormValues>({
    values: preferences
      ? {
          tripPace: preferences.tripPace,
          wakePattern: preferences.wakePattern,
          transportPreference: preferences.transportPreference,
          foodPreference: preferences.foodPreference.join(","),
          avoidTags: preferences.avoidTags.join(",")
        }
      : undefined
  });

  const providerForm = useForm({
    resolver: zodResolver(llmProviderSchema),
    defaultValues: {
      provider: "openai",
      label: "",
      model: "gpt-4.1-mini",
      encryptedApiKeyEnvelope: ""
    }
  });
  const { formState: { errors: providerErrors } } = providerForm;
  const { locale } = useI18n();
  const msgs = validationMessages[locale as Locale] ?? validationMessages.en;

  const notificationPreferenceForm = useForm<NotificationPreferenceFormValues>({
    values: notificationPreferences
      ? {
          pushEnabled: notificationPreferences.pushEnabled,
          emailEnabled: notificationPreferences.emailEnabled,
          digestFrequency: notificationPreferences.digestFrequency,
          quietHoursStart: notificationPreferences.quietHoursStart,
          quietHoursEnd: notificationPreferences.quietHoursEnd,
          tripUpdates: notificationPreferences.tripUpdates,
          budgetAlerts: notificationPreferences.budgetAlerts,
          aiPlanReadyAlerts: notificationPreferences.aiPlanReadyAlerts
        }
      : undefined
  });

  const onSaveProfile = profileForm.handleSubmit(async (values) => {
    await patchProfile.mutateAsync(values);
    pushToast(t("settings.profileSaved"));
  });

  const onSavePreferences = preferenceForm.handleSubmit(async (values) => {
    await putPreferences.mutateAsync({
      tripPace: values.tripPace,
      wakePattern: values.wakePattern,
      transportPreference: values.transportPreference,
      foodPreference: values.foodPreference
        .split(",")
        .map((item) => item.trim())
        .filter(Boolean),
      avoidTags: values.avoidTags
        .split(",")
        .map((item) => item.trim())
        .filter(Boolean)
    });
    pushToast(t("settings.preferencesSaved"));
  });

  const onAddProvider = providerForm.handleSubmit(async (values) => {
    await createProvider.mutateAsync(values);
    providerForm.reset({
      provider: values.provider,
      label: "",
      model: values.model,
      encryptedApiKeyEnvelope: ""
    });
    pushToast(t("settings.providerAdded"));
  });

  const onSaveNotificationPreferences = notificationPreferenceForm.handleSubmit(async (values) => {
    if (!/^\d{2}:\d{2}$/.test(values.quietHoursStart) || !/^\d{2}:\d{2}$/.test(values.quietHoursEnd)) {
      pushToast(t("settings.quietHoursFormat"));
      return;
    }
    if (values.emailEnabled && values.digestFrequency === "instant") {
      pushToast(t("settings.emailNoInstant"));
      return;
    }

    let pushEnabled = values.pushEnabled;
    if (values.pushEnabled) {
      if (!isPushConfigured()) {
        pushEnabled = false;
        pushToast(t("notifications.pushNotConfigured"));
      } else {
        const pushResult = await setupPushMessaging({
          promptForPermission: true,
          forceUpload: true
        });

        if (pushResult.status === "denied") {
          pushEnabled = false;
          pushToast(t("notifications.pushDenied"));
        } else if (pushResult.status === "unsupported") {
          pushEnabled = false;
          pushToast(t("notifications.pushUnsupported"));
        } else if (pushResult.status === "not_configured" || pushResult.status === "error" || pushResult.status === "permission_required") {
          pushEnabled = false;
          pushToast(t("notifications.pushNotConfigured"));
        }
      }
    }

    await putNotificationPreferences.mutateAsync({
      ...values,
      pushEnabled
    });
    pushToast(t("settings.notificationsSaved"));
  });

  const onDeleteProvider = async (providerId: string) => {
    await deleteProvider.mutateAsync(providerId);
    pushToast(t("settings.providerRemoved"));
  };

  const onDeleteAccount = async () => {
    await deleteMyAccount.mutateAsync();
    clearUser();
    pushToast(t("settings.accountDeleted"));
  };

  const onSendPasswordReset = () => {
    pushToast(t("settings.resetLinkSent"));
  };

  const toggleProviderBinding = (providerId: string) => {
    setLinkedProviders((current) => {
      const next = current.includes(providerId)
        ? current.filter((item) => item !== providerId)
        : [...current, providerId];

      if (typeof window !== "undefined") {
        window.localStorage.setItem("linked_oauth_providers", JSON.stringify(next));
      }
      pushToast(t("settings.bindingUpdated"));
      return next;
    });
  };

  const onTestConnection = async () => {
    setTestingConnection(true);
    await new Promise((resolve) => setTimeout(resolve, 1200));
    setTestingConnection(false);
    pushToast(t("settings.testSuccess"));
  };

  useEffect(() => {
    if (typeof window === "undefined") {
      return;
    }

    try {
      const raw = window.localStorage.getItem("linked_oauth_providers");
      if (!raw) {
        return;
      }
      const parsed = JSON.parse(raw) as string[];
      if (Array.isArray(parsed)) {
        setLinkedProviders(parsed.filter((item) => typeof item === "string"));
      }
    } catch {
      // Ignore invalid persisted values
    }
  }, []);

  if (profileLoading || preferencesLoading || notificationPreferencesLoading || providersLoading) {
    return <div className="rounded-[28px] bg-white/80 p-6 text-sm text-ink/65">{t("settings.loadingSettings")}</div>;
  }

  return (
    <div className="grid gap-6 lg:grid-cols-[1.1fr_0.9fr]">
      <SurfaceCard eyebrow={t("settings.profile")} title={t("settings.title")}>
        <form className="grid gap-4" onSubmit={onSaveProfile}>
          <label className="block">
            <span className="mb-2 block text-sm font-medium text-ink">{t("settings.displayName")}</span>
            <input className="w-full rounded-2xl border border-ink/10 bg-sand px-4 py-3" {...profileForm.register("displayName", { required: true })} />
          </label>
          <div className="grid gap-4 sm:grid-cols-2">
            <label className="block">
              <span className="mb-2 block text-sm font-medium text-ink">{t("settings.locale")}</span>
              <input className="w-full rounded-2xl border border-ink/10 bg-sand px-4 py-3" {...profileForm.register("locale", { required: true })} />
            </label>
            <label className="block">
              <span className="mb-2 block text-sm font-medium text-ink">{t("settings.timezone")}</span>
              <input className="w-full rounded-2xl border border-ink/10 bg-sand px-4 py-3" {...profileForm.register("timezone", { required: true })} />
            </label>
          </div>
          <label className="block">
            <span className="mb-2 block text-sm font-medium text-ink">{t("settings.currency")}</span>
            <input className="w-full rounded-2xl border border-ink/10 bg-sand px-4 py-3" {...profileForm.register("currency", { required: true })} />
          </label>
          <button className="rounded-full bg-pine px-5 py-3 text-sm font-medium text-white" disabled={patchProfile.isPending} type="submit">
            {patchProfile.isPending ? t("common.saving") : t("settings.saveProfile")}
          </button>
        </form>

        <form className="mt-6 grid gap-4 border-t border-ink/10 pt-6" onSubmit={onSavePreferences}>
          <p className="text-sm font-semibold text-ink">{t("settings.preferences")}</p>
          <div className="grid gap-4 sm:grid-cols-2">
            <label className="block">
              <span className="mb-2 block text-sm font-medium text-ink">{t("settings.tripPace")}</span>
              <select className="w-full rounded-2xl border border-ink/10 bg-sand px-4 py-3" {...preferenceForm.register("tripPace")}> 
                <option value="relaxed">{t("settings.relaxed")}</option>
                <option value="balanced">{t("settings.balanced")}</option>
                <option value="packed">{t("settings.packed")}</option>
              </select>
            </label>
            <label className="block">
              <span className="mb-2 block text-sm font-medium text-ink">{t("settings.wakePattern")}</span>
              <select className="w-full rounded-2xl border border-ink/10 bg-sand px-4 py-3" {...preferenceForm.register("wakePattern")}> 
                <option value="early">{t("settings.early")}</option>
                <option value="normal">{t("settings.normal")}</option>
                <option value="late">{t("settings.late")}</option>
              </select>
            </label>
          </div>
          <label className="block">
            <span className="mb-2 block text-sm font-medium text-ink">{t("settings.transportPreference")}</span>
            <select className="w-full rounded-2xl border border-ink/10 bg-sand px-4 py-3" {...preferenceForm.register("transportPreference")}> 
              <option value="walk">{t("settings.walkTransport")}</option>
              <option value="transit">{t("settings.transitTransport")}</option>
              <option value="taxi">{t("settings.taxiTransport")}</option>
              <option value="mixed">{t("settings.mixedTransport")}</option>
            </select>
          </label>
          <label className="block">
            <span className="mb-2 block text-sm font-medium text-ink">{t("settings.foodPreference")}（{t("settings.foodPreferenceHint")}）</span>
            <input className="w-full rounded-2xl border border-ink/10 bg-sand px-4 py-3" {...preferenceForm.register("foodPreference")} />
          </label>
          <label className="block">
            <span className="mb-2 block text-sm font-medium text-ink">{t("settings.avoidTags")}（{t("settings.avoidTagsHint")}）</span>
            <input className="w-full rounded-2xl border border-ink/10 bg-sand px-4 py-3" {...preferenceForm.register("avoidTags")} />
          </label>
          <button className="rounded-full bg-ink px-5 py-3 text-sm font-medium text-white" disabled={putPreferences.isPending} type="submit">
            {putPreferences.isPending ? t("common.saving") : t("settings.savePreferences")}
          </button>
        </form>

        <form className="mt-6 grid gap-4 border-t border-ink/10 pt-6" onSubmit={onSaveNotificationPreferences}>
          <p className="text-sm font-semibold text-ink">{t("settings.notificationPreferences")}</p>
          <div className="grid gap-4 sm:grid-cols-2">
            <label className="flex items-center gap-2 text-sm text-ink">
              <input type="checkbox" {...notificationPreferenceForm.register("pushEnabled")} />
              {t("settings.pushEnabled")}
            </label>
            <label className="flex items-center gap-2 text-sm text-ink">
              <input type="checkbox" {...notificationPreferenceForm.register("emailEnabled")} />
              {t("settings.emailEnabled")}
            </label>
          </div>
          <label className="block">
            <span className="mb-2 block text-sm font-medium text-ink">{t("settings.digestFrequency")}</span>
            <select className="w-full rounded-2xl border border-ink/10 bg-sand px-4 py-3" {...notificationPreferenceForm.register("digestFrequency")}>
              <option value="instant">{t("settings.instant")}</option>
              <option value="daily">{t("settings.daily")}</option>
              <option value="weekly">{t("settings.weekly")}</option>
            </select>
          </label>
          <div className="grid gap-4 sm:grid-cols-2">
            <label className="block">
              <span className="mb-2 block text-sm font-medium text-ink">{t("settings.quietStart")}</span>
              <input className="w-full rounded-2xl border border-ink/10 bg-sand px-4 py-3" {...notificationPreferenceForm.register("quietHoursStart")} />
            </label>
            <label className="block">
              <span className="mb-2 block text-sm font-medium text-ink">{t("settings.quietEnd")}</span>
              <input className="w-full rounded-2xl border border-ink/10 bg-sand px-4 py-3" {...notificationPreferenceForm.register("quietHoursEnd")} />
            </label>
          </div>
          <div className="grid gap-4 sm:grid-cols-3">
            <label className="flex items-center gap-2 text-sm text-ink">
              <input type="checkbox" {...notificationPreferenceForm.register("tripUpdates")} />
              {t("settings.tripUpdates")}
            </label>
            <label className="flex items-center gap-2 text-sm text-ink">
              <input type="checkbox" {...notificationPreferenceForm.register("budgetAlerts")} />
              {t("settings.budgetAlerts")}
            </label>
            <label className="flex items-center gap-2 text-sm text-ink">
              <input type="checkbox" {...notificationPreferenceForm.register("aiPlanReadyAlerts")} />
              {t("settings.aiPlanReady")}
            </label>
          </div>
          <button className="rounded-full bg-ink px-5 py-3 text-sm font-medium text-white" disabled={putNotificationPreferences.isPending} type="submit">
            {putNotificationPreferences.isPending ? t("common.saving") : t("settings.saveNotifications")}
          </button>
        </form>

        <div className="mt-6 grid gap-4 border-t border-ink/10 pt-6">
          <p className="text-sm font-semibold text-ink">{t("settings.accountSecurity")}</p>
          <div className="rounded-2xl border border-ink/10 bg-white p-4">
            <p className="text-sm font-medium text-ink">{t("settings.passwordAuth")}</p>
            <p className="mt-1 text-xs text-ink/60">{t("settings.passwordAuthUnavailable")}</p>
            <button
              className="mt-3 rounded-full border border-ink/15 px-4 py-2 text-xs font-medium text-ink"
              onClick={onSendPasswordReset}
              type="button"
            >
              {t("settings.sendResetLink")}
            </button>
          </div>
          <div className="rounded-2xl border border-ink/10 bg-white p-4">
            <p className="text-sm font-medium text-ink">{t("settings.socialBindings")}</p>
            <div className="mt-3 grid gap-2">
              {oauthProviders.map((provider) => {
                const linked = linkedProviders.includes(provider.id);
                return (
                  <div className="flex items-center justify-between rounded-xl border border-ink/10 bg-sand/70 px-3 py-2" key={provider.id}>
                    <div>
                      <p className="text-sm font-medium text-ink">{provider.label}</p>
                      <p className="text-xs text-ink/55">{provider.category}</p>
                    </div>
                    <button
                      className={`rounded-full px-3 py-1 text-xs font-medium ${
                        linked ? "border border-coral/30 text-coral" : "border border-ink/15 text-ink"
                      }`}
                      onClick={() => toggleProviderBinding(provider.id)}
                      type="button"
                    >
                      {linked ? t("settings.unlinkAccount") : t("settings.linkAccount")}
                    </button>
                  </div>
                );
              })}
            </div>
          </div>
        </div>

        <div className="mt-6 border-t border-ink/10 pt-6">
          <p className="text-sm font-semibold text-coral">{t("settings.deleteAccount")}</p>
          <p className="mt-2 text-xs text-ink/60">{t("settings.deleteAccountConfirmDescription")}</p>
          <button
            className="mt-3 rounded-full border border-coral/30 px-5 py-2 text-sm font-medium text-coral transition hover:bg-coral/10"
            onClick={() => {
              openConfirmModal({
                title: t("settings.deleteAccountConfirmTitle"),
                description: t("settings.deleteAccountConfirmDescription"),
                confirmLabel: t("settings.deleteAccount"),
                cancelLabel: t("common.cancel"),
                tone: "danger",
                onConfirm: onDeleteAccount
              });
            }}
            type="button"
          >
            {t("settings.deleteAccount")}
          </button>
        </div>
      </SurfaceCard>

      <SurfaceCard eyebrow={t("settings.llmProviders")} title={t("settings.llmProvidersTitle")}>
        <form className="grid gap-4" onSubmit={onAddProvider}>
          <label className="block">
            <span className="mb-2 block text-sm font-medium text-ink">{t("settings.provider")}</span>
            <select className="w-full rounded-2xl border border-ink/10 bg-sand px-4 py-3" {...providerForm.register("provider")}> 
              <option value="openai">OpenAI</option>
              <option value="anthropic">Anthropic</option>
              <option value="google">Google</option>
              <option value="xai">xAI</option>
            </select>
          </label>
          <label className="block">
            <span className="mb-2 block text-sm font-medium text-ink">{t("settings.label")}</span>
            <input className="w-full rounded-2xl border border-ink/10 bg-sand px-4 py-3" {...providerForm.register("label")} />
          </label>
          <label className="block">
            <span className="mb-2 block text-sm font-medium text-ink">{t("settings.model")}</span>
            <input className="w-full rounded-2xl border border-ink/10 bg-sand px-4 py-3" {...providerForm.register("model")} />
            {providerErrors.model ? <p className="mt-1 text-xs text-coral">{msgs[providerErrors.model.message ?? ""] ?? providerErrors.model.message}</p> : null}
          </label>
          <label className="block">
            <span className="mb-2 block text-sm font-medium text-ink">{t("settings.apiKey")}</span>
            <input className="w-full rounded-2xl border border-ink/10 bg-sand px-4 py-3" type="password" {...providerForm.register("encryptedApiKeyEnvelope")} />
            {providerErrors.encryptedApiKeyEnvelope ? <p className="mt-1 text-xs text-coral">{msgs[providerErrors.encryptedApiKeyEnvelope.message ?? ""] ?? providerErrors.encryptedApiKeyEnvelope.message}</p> : null}
          </label>
          <div className="flex flex-wrap gap-3">
            <button className="rounded-full bg-coral px-5 py-3 text-sm font-medium text-white" disabled={createProvider.isPending} type="submit">
              {createProvider.isPending ? t("settings.addingProvider") : t("settings.addProvider")}
            </button>
            <button
              className="rounded-full border border-ink/20 px-5 py-3 text-sm font-medium text-ink transition hover:bg-sand"
              disabled={testingConnection}
              onClick={() => { void onTestConnection(); }}
              type="button"
            >
              {testingConnection ? t("settings.testing") : t("settings.testConnection")}
            </button>
          </div>
        </form>

        <div className="mt-6 space-y-3 border-t border-ink/10 pt-6">
          {providers.length === 0 ? <p className="text-sm text-ink/60">{t("settings.noProviders")}</p> : null}
          {providers.map((provider) => (
            <div className="rounded-2xl border border-ink/10 bg-white p-4" key={provider.id}>
              <div className="flex items-start justify-between gap-3">
                <div>
                  <p className="text-sm font-semibold text-ink">{provider.label || provider.provider}</p>
                  <p className="text-xs text-ink/65">{t("settings.model")}: {provider.model}</p>
                  <p className="text-xs text-ink/65">{t("settings.apiKey")}: {provider.maskedKey}</p>
                </div>
                <button
                  className="rounded-full border border-ink/15 px-3 py-1 text-xs font-medium text-ink"
                  disabled={deleteProvider.isPending}
                  onClick={() => {
                    void onDeleteProvider(provider.id);
                  }}
                  type="button"
                >
                  {t("settings.removeProvider")}
                </button>
              </div>
            </div>
          ))}
        </div>
      </SurfaceCard>
    </div>
  );
}
