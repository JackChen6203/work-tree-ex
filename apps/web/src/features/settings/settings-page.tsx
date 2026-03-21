import { useForm } from "react-hook-form";
import { SurfaceCard } from "../../components/surface-card";
import {
  useCreateMyLlmProviderMutation,
  useDeleteMyLlmProviderMutation,
  useMyLlmProvidersQuery,
  useMyPreferencesQuery,
  useMyProfileQuery,
  usePatchMyProfileMutation,
  usePutMyPreferencesMutation
} from "../../lib/queries";
import { useUiStore } from "../../store/ui-store";

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

interface LlmProviderFormValues {
  provider: string;
  label: string;
  model: string;
  encryptedApiKeyEnvelope: string;
}

export function SettingsPage() {
  const pushToast = useUiStore((state) => state.pushToast);

  const { data: profile, isLoading: profileLoading } = useMyProfileQuery();
  const { data: preferences, isLoading: preferencesLoading } = useMyPreferencesQuery();
  const { data: providers = [], isLoading: providersLoading } = useMyLlmProvidersQuery();

  const patchProfile = usePatchMyProfileMutation();
  const putPreferences = usePutMyPreferencesMutation();
  const createProvider = useCreateMyLlmProviderMutation();
  const deleteProvider = useDeleteMyLlmProviderMutation();

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

  const providerForm = useForm<LlmProviderFormValues>({
    defaultValues: {
      provider: "openai",
      label: "",
      model: "gpt-4.1-mini",
      encryptedApiKeyEnvelope: ""
    }
  });

  const onSaveProfile = profileForm.handleSubmit(async (values) => {
    await patchProfile.mutateAsync(values);
    pushToast("Profile updated");
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
    pushToast("Preferences saved");
  });

  const onAddProvider = providerForm.handleSubmit(async (values) => {
    await createProvider.mutateAsync(values);
    providerForm.reset({
      provider: values.provider,
      label: "",
      model: values.model,
      encryptedApiKeyEnvelope: ""
    });
    pushToast("LLM provider added");
  });

  const onDeleteProvider = async (providerId: string) => {
    await deleteProvider.mutateAsync(providerId);
    pushToast("LLM provider removed");
  };

  if (profileLoading || preferencesLoading || providersLoading) {
    return <div className="rounded-[28px] bg-white/80 p-6 text-sm text-ink/65">Loading user settings...</div>;
  }

  return (
    <div className="grid gap-6 lg:grid-cols-[1.1fr_0.9fr]">
      <SurfaceCard eyebrow="Users Module" title="Profile & Preferences">
        <form className="grid gap-4" onSubmit={onSaveProfile}>
          <label className="block">
            <span className="mb-2 block text-sm font-medium text-ink">Display name</span>
            <input className="w-full rounded-2xl border border-ink/10 bg-sand px-4 py-3" {...profileForm.register("displayName", { required: true })} />
          </label>
          <div className="grid gap-4 sm:grid-cols-2">
            <label className="block">
              <span className="mb-2 block text-sm font-medium text-ink">Locale</span>
              <input className="w-full rounded-2xl border border-ink/10 bg-sand px-4 py-3" {...profileForm.register("locale", { required: true })} />
            </label>
            <label className="block">
              <span className="mb-2 block text-sm font-medium text-ink">Timezone</span>
              <input className="w-full rounded-2xl border border-ink/10 bg-sand px-4 py-3" {...profileForm.register("timezone", { required: true })} />
            </label>
          </div>
          <label className="block">
            <span className="mb-2 block text-sm font-medium text-ink">Currency</span>
            <input className="w-full rounded-2xl border border-ink/10 bg-sand px-4 py-3" {...profileForm.register("currency", { required: true })} />
          </label>
          <button className="rounded-full bg-pine px-5 py-3 text-sm font-medium text-white" disabled={patchProfile.isPending} type="submit">
            {patchProfile.isPending ? "Saving..." : "Save profile"}
          </button>
        </form>

        <form className="mt-6 grid gap-4 border-t border-ink/10 pt-6" onSubmit={onSavePreferences}>
          <p className="text-sm font-semibold text-ink">Preference profile</p>
          <div className="grid gap-4 sm:grid-cols-2">
            <label className="block">
              <span className="mb-2 block text-sm font-medium text-ink">Trip pace</span>
              <select className="w-full rounded-2xl border border-ink/10 bg-sand px-4 py-3" {...preferenceForm.register("tripPace")}> 
                <option value="relaxed">relaxed</option>
                <option value="balanced">balanced</option>
                <option value="packed">packed</option>
              </select>
            </label>
            <label className="block">
              <span className="mb-2 block text-sm font-medium text-ink">Wake pattern</span>
              <select className="w-full rounded-2xl border border-ink/10 bg-sand px-4 py-3" {...preferenceForm.register("wakePattern")}> 
                <option value="early">early</option>
                <option value="normal">normal</option>
                <option value="late">late</option>
              </select>
            </label>
          </div>
          <label className="block">
            <span className="mb-2 block text-sm font-medium text-ink">Transport preference</span>
            <select className="w-full rounded-2xl border border-ink/10 bg-sand px-4 py-3" {...preferenceForm.register("transportPreference")}> 
              <option value="walk">walk</option>
              <option value="transit">transit</option>
              <option value="taxi">taxi</option>
              <option value="mixed">mixed</option>
            </select>
          </label>
          <label className="block">
            <span className="mb-2 block text-sm font-medium text-ink">Food preference (comma separated)</span>
            <input className="w-full rounded-2xl border border-ink/10 bg-sand px-4 py-3" {...preferenceForm.register("foodPreference")} />
          </label>
          <label className="block">
            <span className="mb-2 block text-sm font-medium text-ink">Avoid tags (comma separated)</span>
            <input className="w-full rounded-2xl border border-ink/10 bg-sand px-4 py-3" {...preferenceForm.register("avoidTags")} />
          </label>
          <button className="rounded-full bg-ink px-5 py-3 text-sm font-medium text-white" disabled={putPreferences.isPending} type="submit">
            {putPreferences.isPending ? "Saving..." : "Save preferences"}
          </button>
        </form>
      </SurfaceCard>

      <SurfaceCard eyebrow="LLM Providers" title="Bring your own model key">
        <form className="grid gap-4" onSubmit={onAddProvider}>
          <label className="block">
            <span className="mb-2 block text-sm font-medium text-ink">Provider</span>
            <select className="w-full rounded-2xl border border-ink/10 bg-sand px-4 py-3" {...providerForm.register("provider")}> 
              <option value="openai">openai</option>
              <option value="anthropic">anthropic</option>
              <option value="google">google</option>
              <option value="xai">xai</option>
            </select>
          </label>
          <label className="block">
            <span className="mb-2 block text-sm font-medium text-ink">Label</span>
            <input className="w-full rounded-2xl border border-ink/10 bg-sand px-4 py-3" {...providerForm.register("label")} />
          </label>
          <label className="block">
            <span className="mb-2 block text-sm font-medium text-ink">Model</span>
            <input className="w-full rounded-2xl border border-ink/10 bg-sand px-4 py-3" {...providerForm.register("model", { required: true })} />
          </label>
          <label className="block">
            <span className="mb-2 block text-sm font-medium text-ink">Encrypted API key envelope</span>
            <input className="w-full rounded-2xl border border-ink/10 bg-sand px-4 py-3" {...providerForm.register("encryptedApiKeyEnvelope", { required: true })} />
          </label>
          <button className="rounded-full bg-coral px-5 py-3 text-sm font-medium text-white" disabled={createProvider.isPending} type="submit">
            {createProvider.isPending ? "Adding..." : "Add provider"}
          </button>
        </form>

        <div className="mt-6 space-y-3 border-t border-ink/10 pt-6">
          {providers.length === 0 ? <p className="text-sm text-ink/60">No provider configs yet.</p> : null}
          {providers.map((provider) => (
            <div className="rounded-2xl border border-ink/10 bg-white p-4" key={provider.id}>
              <div className="flex items-start justify-between gap-3">
                <div>
                  <p className="text-sm font-semibold text-ink">{provider.label || provider.provider}</p>
                  <p className="text-xs text-ink/65">Model: {provider.model}</p>
                  <p className="text-xs text-ink/65">Key: {provider.maskedKey}</p>
                </div>
                <button
                  className="rounded-full border border-ink/15 px-3 py-1 text-xs font-medium text-ink"
                  disabled={deleteProvider.isPending}
                  onClick={() => {
                    void onDeleteProvider(provider.id);
                  }}
                  type="button"
                >
                  Remove
                </button>
              </div>
            </div>
          ))}
        </div>
      </SurfaceCard>
    </div>
  );
}
