import { useState } from "react";
import { useForm } from "react-hook-form";
import { useParams } from "react-router-dom";
import { SurfaceCard } from "../../components/surface-card";
import { StatusPill } from "../../components/status-pill";
import { useTripPermission } from "../auth/use-trip-permission";
import { useAddTripMemberMutation, usePatchTripMutation, useRemoveTripMemberMutation, useTripMembersQuery, useTripQuery, useUpdateTripMemberRoleMutation } from "../../lib/queries";
import { useUiStore } from "../../store/ui-store";
import { useI18n } from "../../lib/i18n";

interface TripPatchValues {
  name: string;
  destinationText: string;
  startDate: string;
  endDate: string;
  timezone: string;
  currency: string;
  travelersCount: number;
  status: "draft" | "active" | "archived";
}

interface AddMemberValues {
  email: string;
  displayName: string;
  role: "editor" | "commenter" | "viewer";
}

export function TripOverviewPage() {
  const { tripId } = useParams();
  const { t } = useI18n();
  const pushToast = useUiStore((state) => state.pushToast);
  const [memberRoleFilter, setMemberRoleFilter] = useState<"all" | "owner" | "editor" | "commenter" | "viewer">("all");
  const { data: trip, isLoading, error } = useTripQuery(tripId ?? "");
  const { data: members = [] } = useTripMembersQuery(tripId ?? "", memberRoleFilter === "all" ? undefined : memberRoleFilter);
  const patchTrip = usePatchTripMutation(tripId ?? "");
  const addTripMember = useAddTripMemberMutation(tripId ?? "");
  const removeTripMember = useRemoveTripMemberMutation(tripId ?? "");
  const updateMemberRole = useUpdateTripMemberRoleMutation(tripId ?? "");
  const form = useForm<TripPatchValues>({
    values: trip
      ? {
          name: trip.name,
          destinationText: trip.destination,
          startDate: trip.startDate,
          endDate: trip.endDate,
          timezone: trip.timezone,
          currency: trip.currency,
          travelersCount: trip.travelersCount,
          status: trip.status
        }
      : undefined
  });
  const memberForm = useForm<AddMemberValues>({
    defaultValues: {
      email: "",
      displayName: "",
      role: "viewer"
    }
  });

  if (isLoading) {
    return <div className="rounded-[28px] bg-white/80 p-6 text-sm text-ink/65">{t("common.loading")}</div>;
  }

  if (error || !trip) {
    return <div className="rounded-[28px] bg-coral/10 p-6 text-sm text-coral">{t("dashboard.tripLoadError")}</div>;
  }

  const permission = useTripPermission(trip.role);

  const onSubmit = form.handleSubmit(async (values) => {
    const updated = await patchTrip.mutateAsync({
      version: trip.version,
      input: values
    });
    pushToast(t("trip.updated"));
  });

  const onAddMember = memberForm.handleSubmit(async (values) => {
    await addTripMember.mutateAsync({
      email: values.email,
      displayName: values.displayName,
      role: values.role
    });
    memberForm.reset({
      email: "",
      displayName: "",
      role: values.role
    });
    pushToast(t("members.addMember"));
  });

  const onRemoveMember = async (memberId: string) => {
    await removeTripMember.mutateAsync(memberId);
    pushToast(t("common.remove"));
  };

  const onUpdateRole = async (memberId: string, role: "owner" | "editor" | "commenter" | "viewer") => {
    await updateMemberRole.mutateAsync({ memberId, role });
    pushToast(t("trip.updated"));
  };

  return (
    <div className="grid gap-6 lg:grid-cols-[1.1fr_0.9fr]">
      <SurfaceCard eyebrow={t("trip.overview")} title={trip.name}>
        <div className={`rounded-[28px] bg-gradient-to-br ${trip.coverGradient} p-6 text-white`}>
          <p className="text-xs uppercase tracking-[0.24em] text-white/70">{trip.destination}</p>
          <div className="mt-4 flex flex-wrap items-center gap-3">
            <StatusPill tone="accent">{trip.status}</StatusPill>
            <StatusPill tone="accent">{trip.role}</StatusPill>
          </div>
          <div className="mt-6 grid gap-4 sm:grid-cols-2">
            <div>
              <p className="text-sm text-white/70">{t("trip.startDate")} ~ {t("trip.endDate")}</p>
              <p className="mt-1 text-lg font-medium">{trip.dateRange}</p>
            </div>
            <div>
              <p className="text-sm text-white/70">{t("trip.timezone")}</p>
              <p className="mt-1 text-lg font-medium">{trip.timezone}</p>
            </div>
            <div>
              <p className="text-sm text-white/70">{t("common.members")}</p>
              <p className="mt-1 text-lg font-medium">{members.length || trip.travelersCount}</p>
            </div>
            <div>
              <p className="text-sm text-white/70">{t("trip.version")}</p>
              <p className="mt-1 text-lg font-medium">v{trip.version}</p>
            </div>
          </div>
          <div className="mt-6 rounded-2xl border border-white/25 bg-white/10 p-4">
            <div className="flex items-center justify-between gap-3">
              <p className="text-xs uppercase tracking-[0.24em] text-white/70">{t("members.title")}</p>
              <select
                className="rounded-full border border-white/30 bg-white/10 px-3 py-1 text-xs font-medium text-white"
                onChange={(event) => {
                  setMemberRoleFilter(event.target.value as "all" | "owner" | "editor" | "commenter" | "viewer");
                }}
                value={memberRoleFilter}
              >
                <option value="all">{t("members.role")}</option>
                <option value="owner">{t("members.owner")}</option>
                <option value="editor">{t("members.editor")}</option>
                <option value="commenter">{t("members.commenter")}</option>
                <option value="viewer">{t("members.viewer")}</option>
              </select>
            </div>
            {!permission.canManageMembers ? null : null}
            {members.length === 0 ? <p className="mt-2 text-sm text-white/75">{t("common.noData")}</p> : null}
            <div className="mt-3 grid gap-2">
              {members.map((member) => (
                <div className="flex items-center justify-between rounded-xl border border-white/20 px-3 py-2" key={member.id}>
                  <div>
                    <p className="text-sm font-medium">{member.displayName || member.email || member.userId || "Unknown"}</p>
                    <p className="text-xs text-white/70">{member.email || member.userId}</p>
                  </div>
                  <div className="flex items-center gap-2">
                    <select
                      className="rounded-full border border-white/30 bg-white/10 px-3 py-1 text-xs font-medium text-white"
                      defaultValue={member.role}
                      disabled={!permission.canManageMembers || updateMemberRole.isPending}
                      onChange={(event) => {
                        void onUpdateRole(member.id, event.target.value as "owner" | "editor" | "commenter" | "viewer");
                      }}
                    >
                      <option value="owner">{t("members.owner")}</option>
                      <option value="editor">{t("members.editor")}</option>
                      <option value="commenter">{t("members.commenter")}</option>
                      <option value="viewer">{t("members.viewer")}</option>
                    </select>
                    <button
                      className="rounded-full border border-white/30 px-3 py-1 text-xs font-medium text-white/90"
                      disabled={!permission.canManageMembers || removeTripMember.isPending || updateMemberRole.isPending}
                      onClick={() => onRemoveMember(member.id)}
                      type="button"
                    >
                      {t("common.remove")}
                    </button>
                  </div>
                </div>
              ))}
            </div>
          </div>
        </div>
      </SurfaceCard>
      <SurfaceCard eyebrow={t("trip.editMetadata")} title={t("trip.editMetadata")}>
        <form className="grid gap-4" onSubmit={onSubmit}>
          {!permission.canEdit ? null : null}
          <fieldset className="grid gap-4" disabled={!permission.canEdit || patchTrip.isPending}>
          <label className="block">
            <span className="mb-2 block text-sm font-medium text-ink">{t("trip.name")}</span>
            <input className="w-full rounded-2xl border border-ink/10 bg-sand px-4 py-3" {...form.register("name")} />
          </label>
          <label className="block">
            <span className="mb-2 block text-sm font-medium text-ink">{t("trip.destination")}</span>
            <input className="w-full rounded-2xl border border-ink/10 bg-sand px-4 py-3" {...form.register("destinationText")} />
          </label>
          <div className="grid gap-4 sm:grid-cols-2">
            <label className="block">
              <span className="mb-2 block text-sm font-medium text-ink">{t("trip.startDate")}</span>
              <input className="w-full rounded-2xl border border-ink/10 bg-sand px-4 py-3" type="date" {...form.register("startDate")} />
            </label>
            <label className="block">
              <span className="mb-2 block text-sm font-medium text-ink">{t("trip.endDate")}</span>
              <input className="w-full rounded-2xl border border-ink/10 bg-sand px-4 py-3" type="date" {...form.register("endDate")} />
            </label>
          </div>
          <div className="grid gap-4 sm:grid-cols-2">
            <label className="block">
              <span className="mb-2 block text-sm font-medium text-ink">{t("trip.timezone")}</span>
              <input className="w-full rounded-2xl border border-ink/10 bg-sand px-4 py-3" {...form.register("timezone")} />
            </label>
            <label className="block">
              <span className="mb-2 block text-sm font-medium text-ink">{t("trip.currency")}</span>
              <input className="w-full rounded-2xl border border-ink/10 bg-sand px-4 py-3" {...form.register("currency")} />
            </label>
          </div>
          <div className="grid gap-4 sm:grid-cols-2">
            <label className="block">
              <span className="mb-2 block text-sm font-medium text-ink">{t("trip.travelers")}</span>
              <input
                className="w-full rounded-2xl border border-ink/10 bg-sand px-4 py-3"
                type="number"
                min={1}
                {...form.register("travelersCount", { valueAsNumber: true })}
              />
            </label>
            <label className="block">
              <span className="mb-2 block text-sm font-medium text-ink">{t("trip.status")}</span>
              <select className="w-full rounded-2xl border border-ink/10 bg-sand px-4 py-3" {...form.register("status")}>
                <option value="draft">draft</option>
                <option value="active">active</option>
                <option value="archived">archived</option>
              </select>
            </label>
          </div>
          </fieldset>
          <button className="rounded-full bg-pine px-5 py-3 text-sm font-medium text-white disabled:opacity-60" disabled={!permission.canEdit || patchTrip.isPending} type="submit">
            {patchTrip.isPending ? t("common.saving") : t("trip.update")}
          </button>
        </form>
        {permission.canManageMembers ? (
          <form className="mt-6 grid gap-4 border-t border-ink/10 pt-6" onSubmit={onAddMember}>
            <p className="text-sm font-semibold text-ink">{t("members.addMember")}</p>
            <label className="block">
              <span className="mb-2 block text-sm font-medium text-ink">{t("members.addMemberEmail")}</span>
              <input className="w-full rounded-2xl border border-ink/10 bg-sand px-4 py-3" type="email" {...memberForm.register("email", { required: true })} />
            </label>
            <label className="block">
              <span className="mb-2 block text-sm font-medium text-ink">{t("settings.displayName")}</span>
              <input className="w-full rounded-2xl border border-ink/10 bg-sand px-4 py-3" {...memberForm.register("displayName")} />
            </label>
            <label className="block">
              <span className="mb-2 block text-sm font-medium text-ink">{t("members.addMemberRole")}</span>
              <select className="w-full rounded-2xl border border-ink/10 bg-sand px-4 py-3" {...memberForm.register("role")}>
                <option value="viewer">{t("members.viewer")}</option>
                <option value="commenter">{t("members.commenter")}</option>
                <option value="editor">{t("members.editor")}</option>
              </select>
            </label>
            <button className="rounded-full bg-ink px-5 py-3 text-sm font-medium text-white" disabled={addTripMember.isPending} type="submit">
              {addTripMember.isPending ? t("members.adding") : t("members.addMember")}
            </button>
          </form>
        ) : null}
      </SurfaceCard>
    </div>
  );
}
