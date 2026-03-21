import { useForm } from "react-hook-form";
import { useParams } from "react-router-dom";
import { SurfaceCard } from "../../components/surface-card";
import { StatusPill } from "../../components/status-pill";
import { useAddTripMemberMutation, usePatchTripMutation, useRemoveTripMemberMutation, useTripMembersQuery, useTripQuery } from "../../lib/queries";
import { useUiStore } from "../../store/ui-store";

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
  const pushToast = useUiStore((state) => state.pushToast);
  const { data: trip, isLoading, error } = useTripQuery(tripId ?? "");
  const { data: members = [] } = useTripMembersQuery(tripId ?? "");
  const patchTrip = usePatchTripMutation(tripId ?? "");
  const addTripMember = useAddTripMemberMutation(tripId ?? "");
  const removeTripMember = useRemoveTripMemberMutation(tripId ?? "");
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
    return <div className="rounded-[28px] bg-white/80 p-6 text-sm text-ink/65">Loading trip detail from API...</div>;
  }

  if (error || !trip) {
    return <div className="rounded-[28px] bg-coral/10 p-6 text-sm text-coral">Trip detail could not be loaded from backend.</div>;
  }

  const onSubmit = form.handleSubmit(async (values) => {
    const updated = await patchTrip.mutateAsync({
      version: trip.version,
      input: values
    });
    pushToast(`Trip updated: ${updated.name}`);
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
    pushToast("Member added");
  });

  const onRemoveMember = async (memberId: string) => {
    await removeTripMember.mutateAsync(memberId);
    pushToast("Member removed");
  };

  return (
    <div className="grid gap-6 lg:grid-cols-[1.1fr_0.9fr]">
      <SurfaceCard eyebrow="Trip Module" title={trip.name}>
        <div className={`rounded-[28px] bg-gradient-to-br ${trip.coverGradient} p-6 text-white`}>
          <p className="text-xs uppercase tracking-[0.24em] text-white/70">{trip.destination}</p>
          <div className="mt-4 flex flex-wrap items-center gap-3">
            <StatusPill tone="accent">{trip.status}</StatusPill>
            <StatusPill tone="accent">{trip.role}</StatusPill>
          </div>
          <div className="mt-6 grid gap-4 sm:grid-cols-2">
            <div>
              <p className="text-sm text-white/70">Date range</p>
              <p className="mt-1 text-lg font-medium">{trip.dateRange}</p>
            </div>
            <div>
              <p className="text-sm text-white/70">Timezone</p>
              <p className="mt-1 text-lg font-medium">{trip.timezone}</p>
            </div>
            <div>
              <p className="text-sm text-white/70">Members</p>
              <p className="mt-1 text-lg font-medium">{members.length || trip.travelersCount}</p>
            </div>
            <div>
              <p className="text-sm text-white/70">Version</p>
              <p className="mt-1 text-lg font-medium">v{trip.version}</p>
            </div>
          </div>
          <div className="mt-6 rounded-2xl border border-white/25 bg-white/10 p-4">
            <p className="text-xs uppercase tracking-[0.24em] text-white/70">Collaboration members</p>
            {members.length === 0 ? <p className="mt-2 text-sm text-white/75">No members added yet.</p> : null}
            <div className="mt-3 grid gap-2">
              {members.map((member) => (
                <div className="flex items-center justify-between rounded-xl border border-white/20 px-3 py-2" key={member.id}>
                  <div>
                    <p className="text-sm font-medium">{member.displayName || member.email || member.userId || "Unknown"}</p>
                    <p className="text-xs text-white/70">{member.email || member.userId}</p>
                  </div>
                  <div className="flex items-center gap-2">
                    <StatusPill tone="accent">{member.role}</StatusPill>
                    <button
                      className="rounded-full border border-white/30 px-3 py-1 text-xs font-medium text-white/90"
                      disabled={removeTripMember.isPending}
                      onClick={() => onRemoveMember(member.id)}
                      type="button"
                    >
                      Remove
                    </button>
                  </div>
                </div>
              ))}
            </div>
          </div>
        </div>
      </SurfaceCard>
      <SurfaceCard eyebrow="Server Data" title="Patch trip metadata">
        <form className="grid gap-4" onSubmit={onSubmit}>
          <label className="block">
            <span className="mb-2 block text-sm font-medium text-ink">Trip name</span>
            <input className="w-full rounded-2xl border border-ink/10 bg-sand px-4 py-3" {...form.register("name")} />
          </label>
          <label className="block">
            <span className="mb-2 block text-sm font-medium text-ink">Destination</span>
            <input className="w-full rounded-2xl border border-ink/10 bg-sand px-4 py-3" {...form.register("destinationText")} />
          </label>
          <div className="grid gap-4 sm:grid-cols-2">
            <label className="block">
              <span className="mb-2 block text-sm font-medium text-ink">Start date</span>
              <input className="w-full rounded-2xl border border-ink/10 bg-sand px-4 py-3" type="date" {...form.register("startDate")} />
            </label>
            <label className="block">
              <span className="mb-2 block text-sm font-medium text-ink">End date</span>
              <input className="w-full rounded-2xl border border-ink/10 bg-sand px-4 py-3" type="date" {...form.register("endDate")} />
            </label>
          </div>
          <div className="grid gap-4 sm:grid-cols-2">
            <label className="block">
              <span className="mb-2 block text-sm font-medium text-ink">Timezone</span>
              <input className="w-full rounded-2xl border border-ink/10 bg-sand px-4 py-3" {...form.register("timezone")} />
            </label>
            <label className="block">
              <span className="mb-2 block text-sm font-medium text-ink">Currency</span>
              <input className="w-full rounded-2xl border border-ink/10 bg-sand px-4 py-3" {...form.register("currency")} />
            </label>
          </div>
          <div className="grid gap-4 sm:grid-cols-2">
            <label className="block">
              <span className="mb-2 block text-sm font-medium text-ink">Travelers</span>
              <input
                className="w-full rounded-2xl border border-ink/10 bg-sand px-4 py-3"
                type="number"
                min={1}
                {...form.register("travelersCount", { valueAsNumber: true })}
              />
            </label>
            <label className="block">
              <span className="mb-2 block text-sm font-medium text-ink">Status</span>
              <select className="w-full rounded-2xl border border-ink/10 bg-sand px-4 py-3" {...form.register("status")}>
                <option value="draft">draft</option>
                <option value="active">active</option>
                <option value="archived">archived</option>
              </select>
            </label>
          </div>
          <button className="rounded-full bg-pine px-5 py-3 text-sm font-medium text-white" disabled={patchTrip.isPending} type="submit">
            {patchTrip.isPending ? "Saving..." : "Save trip"}
          </button>
        </form>
        <form className="mt-6 grid gap-4 border-t border-ink/10 pt-6" onSubmit={onAddMember}>
          <p className="text-sm font-semibold text-ink">Add member</p>
          <label className="block">
            <span className="mb-2 block text-sm font-medium text-ink">Email</span>
            <input className="w-full rounded-2xl border border-ink/10 bg-sand px-4 py-3" type="email" {...memberForm.register("email", { required: true })} />
          </label>
          <label className="block">
            <span className="mb-2 block text-sm font-medium text-ink">Display name</span>
            <input className="w-full rounded-2xl border border-ink/10 bg-sand px-4 py-3" {...memberForm.register("displayName")} />
          </label>
          <label className="block">
            <span className="mb-2 block text-sm font-medium text-ink">Role</span>
            <select className="w-full rounded-2xl border border-ink/10 bg-sand px-4 py-3" {...memberForm.register("role")}> 
              <option value="viewer">viewer</option>
              <option value="commenter">commenter</option>
              <option value="editor">editor</option>
            </select>
          </label>
          <button className="rounded-full bg-ink px-5 py-3 text-sm font-medium text-white" disabled={addTripMember.isPending} type="submit">
            {addTripMember.isPending ? "Adding..." : "Add member"}
          </button>
        </form>
      </SurfaceCard>
    </div>
  );
}
