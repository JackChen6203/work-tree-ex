import { useEffect, useState } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { useParams } from "react-router-dom";
import { SurfaceCard } from "../../components/surface-card";
import { StatusPill } from "../../components/status-pill";
import { useTripPermission } from "../auth/use-trip-permission";
import {
  useAddTripMemberMutation,
  useCreateTripInvitationMutation,
  useCreateTripShareLinkMutation,
  usePatchTripMutation,
  useRemoveTripMemberMutation,
  useRevokeTripInvitationMutation,
  useRevokeTripShareLinkMutation,
  useTripInvitationsQuery,
  useTripMembersQuery,
  useTripQuery,
  useTripShareLinksQuery,
  useUpdateTripMemberRoleMutation
} from "../../lib/queries";
import { useUiStore } from "../../store/ui-store";
import { useI18n } from "../../lib/i18n";
import { addMemberSchema, validationMessages } from "../../lib/schemas";
import type { AddMemberFormValues } from "../../lib/schemas";
import type { Locale } from "../../lib/translations";
import { currencyOptions, timezoneOptions } from "../../lib/trip-form-options";
import { getTripCoverImage } from "../../lib/trip-cover-storage";

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

type InvitationDisplayStatus = "pending" | "accepted" | "revoked" | "expired";

function resolveInvitationStatus(status: string, expiresAt: string): InvitationDisplayStatus {
  if (status === "accepted" || status === "revoked" || status === "expired") {
    return status;
  }
  const expiresMs = new Date(expiresAt).getTime();
  if (Number.isFinite(expiresMs) && expiresMs < Date.now()) {
    return "expired";
  }
  return "pending";
}

export function TripOverviewPage() {
  const { tripId } = useParams();
  const { t, locale } = useI18n();
  const msgs = validationMessages[locale as Locale] ?? validationMessages.en;
  const pushToast = useUiStore((state) => state.pushToast);
  const openConfirmModal = useUiStore((state) => state.openConfirmModal);
  const [memberRoleFilter, setMemberRoleFilter] = useState<"all" | "owner" | "editor" | "commenter" | "viewer">("all");
  const [coverImageUrl, setCoverImageUrl] = useState<string | null>(null);
  const { data: trip, isLoading, error } = useTripQuery(tripId ?? "");
  const { data: members = [] } = useTripMembersQuery(tripId ?? "", memberRoleFilter === "all" ? undefined : memberRoleFilter);
  const { data: shareLinks = [] } = useTripShareLinksQuery(tripId ?? "");
  const { data: invitations = [] } = useTripInvitationsQuery(tripId ?? "");
  const patchTrip = usePatchTripMutation(tripId ?? "");
  const addTripMember = useAddTripMemberMutation(tripId ?? "");
  const removeTripMember = useRemoveTripMemberMutation(tripId ?? "");
  const updateMemberRole = useUpdateTripMemberRoleMutation(tripId ?? "");
  const createShareLink = useCreateTripShareLinkMutation(tripId ?? "");
  const revokeShareLink = useRevokeTripShareLinkMutation(tripId ?? "");
  const createInvitation = useCreateTripInvitationMutation(tripId ?? "");
  const revokeInvitation = useRevokeTripInvitationMutation(tripId ?? "");
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
  const memberForm = useForm<AddMemberFormValues>({
    resolver: zodResolver(addMemberSchema),
    defaultValues: {
      email: "",
      role: "viewer"
    }
  });
  const invitationForm = useForm<AddMemberFormValues>({
    resolver: zodResolver(addMemberSchema),
    defaultValues: {
      email: "",
      role: "viewer"
    }
  });
  const { formState: { errors: memberErrors } } = memberForm;
  const { formState: { errors: invitationErrors } } = invitationForm;
  const timezoneSelectOptions = Array.from(new Set([trip?.timezone, ...timezoneOptions].filter(Boolean) as string[]));
  const currencySelectOptions = Array.from(
    new Set([trip?.currency, ...currencyOptions.map((item) => item.code)].filter(Boolean) as string[])
  );
  const invitationPreviewEmail = invitationForm.watch("email");
  const invitationPreviewRole = invitationForm.watch("role");
  const invitationRows = invitations.map((item) => ({
    ...item,
    displayStatus: resolveInvitationStatus(item.status, item.expiresAt)
  }));
  const baseAppUrl = typeof window !== "undefined" ? window.location.origin : "";

  useEffect(() => {
    if (!trip?.id) {
      setCoverImageUrl(null);
      return;
    }
    setCoverImageUrl(getTripCoverImage(trip.id) ?? null);
  }, [trip?.id]);

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
      displayName: values.email.split("@")[0],
      role: values.role
    });
    memberForm.reset({
      email: "",
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

  const onCreateShareLink = async () => {
    const created = await createShareLink.mutateAsync();
    if (created.token) {
      const shareUrl = `${baseAppUrl}/trips/${trip.id}/share/${created.token}`;
      await navigator.clipboard?.writeText(shareUrl).catch(() => undefined);
      pushToast(t("members.shareLinkCopied"));
      return;
    }
    pushToast(t("members.shareLinkCreated"));
  };

  const onRevokeShareLink = (linkId: string) => {
    openConfirmModal({
      title: t("members.revokeShareLinkTitle"),
      description: t("members.revokeShareLinkDescription"),
      confirmLabel: t("common.confirm"),
      cancelLabel: t("common.cancel"),
      tone: "danger",
      onConfirm: async () => {
        await revokeShareLink.mutateAsync(linkId);
        pushToast(t("members.shareLinkRevoked"));
      }
    });
  };

  const onCreateInvitation = invitationForm.handleSubmit(async (values) => {
    await createInvitation.mutateAsync({
      inviteeEmail: values.email,
      role: values.role
    });
    invitationForm.reset({
      email: "",
      role: values.role
    });
    pushToast(t("members.invitationSent"));
  });

  const onRevokeInvitation = (invitationId: string) => {
    openConfirmModal({
      title: t("members.revokeInviteTitle"),
      description: t("members.revokeInviteDescription"),
      confirmLabel: t("common.confirm"),
      cancelLabel: t("common.cancel"),
      tone: "danger",
      onConfirm: async () => {
        await revokeInvitation.mutateAsync(invitationId);
        pushToast(t("members.invitationRevoked"));
      }
    });
  };

  const onReinvite = async (email: string, role: "editor" | "commenter" | "viewer") => {
    await createInvitation.mutateAsync({ inviteeEmail: email, role });
    pushToast(t("members.invitationSent"));
  };

  return (
    <div className="grid gap-6 lg:grid-cols-[1.1fr_0.9fr]">
      <SurfaceCard eyebrow={t("trip.overview")} title={trip.name}>
        <div className={`relative overflow-hidden rounded-[28px] bg-gradient-to-br ${trip.coverGradient} p-6 text-white`}>
          {coverImageUrl ? (
            <>
              <img
                alt={t("trip.coverImageAlt")}
                className="absolute inset-0 h-full w-full object-cover"
                decoding="async"
                loading="lazy"
                src={coverImageUrl}
              />
              <div className="absolute inset-0 bg-ink/35" />
            </>
          ) : null}
          <div className="relative z-10">
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
              <select className="w-full rounded-2xl border border-ink/10 bg-sand px-4 py-3" {...form.register("timezone")}>
                {timezoneSelectOptions.map((timezone) => (
                  <option key={timezone} value={timezone}>
                    {timezone}
                  </option>
                ))}
              </select>
            </label>
            <label className="block">
              <span className="mb-2 block text-sm font-medium text-ink">{t("trip.currency")}</span>
              <select className="w-full rounded-2xl border border-ink/10 bg-sand px-4 py-3" {...form.register("currency")}>
                {currencySelectOptions.map((currency) => (
                  <option key={currency} value={currency}>
                    {currency}
                  </option>
                ))}
              </select>
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
          <>
            <form className="mt-6 grid gap-4 border-t border-ink/10 pt-6" onSubmit={onAddMember}>
              <p className="text-sm font-semibold text-ink">{t("members.addMember")}</p>
              <label className="block">
                <span className="mb-2 block text-sm font-medium text-ink">{t("members.addMemberEmail")}</span>
                <input className="w-full rounded-2xl border border-ink/10 bg-sand px-4 py-3" type="email" {...memberForm.register("email")} />
                {memberErrors.email ? <p className="mt-1 text-xs text-coral">{msgs[memberErrors.email.message ?? ""] ?? memberErrors.email.message}</p> : null}

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

            <section className="mt-6 grid gap-4 border-t border-ink/10 pt-6">
              <div className="flex flex-wrap items-center justify-between gap-3">
                <p className="text-sm font-semibold text-ink">{t("members.shareLinks")}</p>
                <button
                  className="rounded-full border border-ink/15 px-4 py-2 text-xs font-medium text-ink"
                  disabled={createShareLink.isPending}
                  onClick={() => {
                    void onCreateShareLink();
                  }}
                  type="button"
                >
                  {createShareLink.isPending ? t("members.creatingShareLink") : t("members.createShareLink")}
                </button>
              </div>
              {shareLinks.length === 0 ? <p className="text-sm text-ink/60">{t("members.noShareLinks")}</p> : null}
              <div className="grid gap-3">
                {shareLinks.map((shareLink) => (
                  <div className="rounded-2xl border border-ink/10 bg-sand/60 p-3" key={shareLink.id}>
                    <div className="flex flex-wrap items-center justify-between gap-2">
                      <p className="text-sm font-medium text-ink">{shareLink.accessScope}</p>
                      <StatusPill tone={shareLink.revokedAt ? "danger" : "success"}>
                        {shareLink.revokedAt ? t("members.revoked") : t("members.active")}
                      </StatusPill>
                    </div>
                    <p className="mt-1 text-xs text-ink/60">
                      {t("members.createdAt")}: {new Date(shareLink.createdAt).toLocaleString()}
                    </p>
                    {shareLink.token ? (
                      <input
                        className="mt-2 w-full rounded-xl border border-ink/10 bg-white px-3 py-2 text-xs text-ink/70"
                        readOnly
                        value={`${baseAppUrl}/trips/${trip.id}/share/${shareLink.token}`}
                      />
                    ) : null}
                    {shareLink.revokedAt ? null : (
                      <button
                        className="mt-2 rounded-full border border-ink/15 px-3 py-1 text-xs font-medium text-ink"
                        disabled={revokeShareLink.isPending}
                        onClick={() => {
                          onRevokeShareLink(shareLink.id);
                        }}
                        type="button"
                      >
                        {t("members.revokeShareLink")}
                      </button>
                    )}
                  </div>
                ))}
              </div>
            </section>

            <section className="mt-6 grid gap-4 border-t border-ink/10 pt-6">
              <form className="grid gap-4" onSubmit={onCreateInvitation}>
                <p className="text-sm font-semibold text-ink">{t("members.invitations")}</p>
                <label className="block">
                  <span className="mb-2 block text-sm font-medium text-ink">{t("members.addMemberEmail")}</span>
                  <input className="w-full rounded-2xl border border-ink/10 bg-sand px-4 py-3" type="email" {...invitationForm.register("email")} />
                  {invitationErrors.email ? <p className="mt-1 text-xs text-coral">{msgs[invitationErrors.email.message ?? ""] ?? invitationErrors.email.message}</p> : null}
                </label>
                <label className="block">
                  <span className="mb-2 block text-sm font-medium text-ink">{t("members.addMemberRole")}</span>
                  <select className="w-full rounded-2xl border border-ink/10 bg-sand px-4 py-3" {...invitationForm.register("role")}>
                    <option value="viewer">{t("members.viewer")}</option>
                    <option value="commenter">{t("members.commenter")}</option>
                    <option value="editor">{t("members.editor")}</option>
                  </select>
                </label>
                <div className="rounded-2xl border border-ink/10 bg-sand/60 p-3 text-sm text-ink/70">
                  <p className="font-medium text-ink">{t("members.invitePreviewTitle")}</p>
                  <p className="mt-1 text-xs">
                    {t("members.invitePreviewSubject").replace("{trip}", trip.name)}
                  </p>
                  <p className="mt-1 text-xs">
                    {t("members.invitePreviewBody")
                      .replace("{email}", invitationPreviewEmail || "example@domain.com")
                      .replace("{role}", invitationPreviewRole || "viewer")}
                  </p>
                </div>
                <button className="rounded-full bg-ink px-5 py-3 text-sm font-medium text-white" disabled={createInvitation.isPending} type="submit">
                  {createInvitation.isPending ? t("members.sendingInvite") : t("members.sendInvite")}
                </button>
              </form>

              {invitationRows.length === 0 ? <p className="text-sm text-ink/60">{t("members.noInvitations")}</p> : null}
              <div className="grid gap-3">
                {invitationRows.map((invitation) => (
                  <div className="rounded-2xl border border-ink/10 bg-white p-3" key={invitation.id}>
                    <div className="flex flex-wrap items-center justify-between gap-2">
                      <p className="text-sm font-medium text-ink">{invitation.inviteeEmail}</p>
                      <StatusPill tone={invitation.displayStatus === "accepted" ? "success" : invitation.displayStatus === "pending" ? "accent" : "danger"}>
                        {invitation.displayStatus}
                      </StatusPill>
                    </div>
                    <p className="mt-1 text-xs text-ink/60">
                      {invitation.role} · {t("members.expiresAt")}: {new Date(invitation.expiresAt).toLocaleDateString()}
                    </p>
                    <div className="mt-2 flex items-center gap-2">
                      {invitation.displayStatus === "pending" ? (
                        <button
                          className="rounded-full border border-ink/15 px-3 py-1 text-xs font-medium text-ink"
                          disabled={revokeInvitation.isPending}
                          onClick={() => {
                            onRevokeInvitation(invitation.id);
                          }}
                          type="button"
                        >
                          {t("members.revokeInvite")}
                        </button>
                      ) : null}
                      {invitation.displayStatus === "expired" || invitation.displayStatus === "revoked" ? (
                        <button
                          className="rounded-full border border-ink/15 px-3 py-1 text-xs font-medium text-ink"
                          disabled={createInvitation.isPending}
                          onClick={() => {
                            void onReinvite(invitation.inviteeEmail, invitation.role);
                          }}
                          type="button"
                        >
                          {t("members.reinvite")}
                        </button>
                      ) : null}
                    </div>
                  </div>
                ))}
              </div>
            </section>
          </>
        ) : null}
      </SurfaceCard>
    </div>
  );
}
