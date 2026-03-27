import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { addTripMember, createTrip, getTrip, listTripMembers, listTrips, patchTrip, removeTripMember, updateTripMemberRole } from "./trips-api";
import {
  createTripInvitation,
  createTripShareLink,
  listTripInvitations,
  listTripShareLinks,
  revokeTripInvitation,
  revokeTripShareLink
} from "./trips-collaboration-api";
import { requestMagicLink, verifyMagicLink } from "./auth-api";
import { adoptAiPlan, createAiPlan, getAiPlan, listAiPlans } from "./ai-planner-api";
import { createExpense, deleteExpense, getBudgetProfile, getBudgetRates, listExpenses, patchExpense, refreshBudgetRate, upsertBudgetProfile } from "./budget-api";
import { cleanupReadNotifications, deleteNotification, listNotifications, markAllNotificationsRead, markNotificationRead, markNotificationUnread } from "./notifications-api";
import { createItineraryItem, deleteItineraryItem, listItineraryDays, patchItineraryItem, reorderItineraryItems } from "./itinerary-api";
import { estimateRoute, searchPlaces } from "./maps-api";
import {
  createMyLlmProvider,
  deleteMyAccount,
  deleteMyLlmProvider,
  getMyNotificationPreferences,
  getMyPreferences,
  getMyProfile,
  listMyLlmProviders,
  patchMyProfile,
  putMyNotificationPreferences,
  putMyPreferences
} from "./users-api";
import { flushSyncMutations, getSyncBootstrap } from "./sync-api";
import type { AddTripMemberInput, CreateTripInput, PatchTripInput } from "./trips-api";
import { trackQueuedMutation } from "./mutation-queue";

export function useTripsQuery() {
  return useQuery({
    queryKey: ["trips"],
    queryFn: listTrips
  });
}

export function useTripQuery(tripId: string) {
  return useQuery({
    queryKey: ["trips", tripId],
    queryFn: () => getTrip(tripId),
    enabled: Boolean(tripId)
  });
}

export function useCreateTripMutation() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (input: CreateTripInput) => trackQueuedMutation("trips.create", () => createTrip(input)),
    onSuccess: (trip) => {
      queryClient.invalidateQueries({ queryKey: ["trips"] });
      queryClient.setQueryData(["trips", trip.id], trip);
    }
  });
}

export function usePatchTripMutation(tripId: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ version, input }: { version: number; input: PatchTripInput }) => trackQueuedMutation("trips.patch", () => patchTrip(tripId, version, input)),
    onSuccess: (trip) => {
      queryClient.invalidateQueries({ queryKey: ["trips"] });
      queryClient.setQueryData(["trips", trip.id], trip);
    }
  });
}

export function useTripMembersQuery(tripId: string, role?: "owner" | "editor" | "commenter" | "viewer") {
  return useQuery({
    queryKey: ["trip-members", tripId, role ?? "all"],
    queryFn: () => listTripMembers(tripId, role),
    enabled: Boolean(tripId)
  });
}

export function useAddTripMemberMutation(tripId: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (input: AddTripMemberInput) => trackQueuedMutation("trip-members.add", () => addTripMember(tripId, input)),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["trip-members", tripId] });
    }
  });
}

export function useRemoveTripMemberMutation(tripId: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (memberId: string) => trackQueuedMutation("trip-members.remove", () => removeTripMember(tripId, memberId)),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["trip-members", tripId] });
    }
  });
}

export function useUpdateTripMemberRoleMutation(tripId: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ memberId, role }: { memberId: string; role: "owner" | "editor" | "commenter" | "viewer" }) =>
      trackQueuedMutation("trip-members.role", () => updateTripMemberRole(tripId, memberId, role)),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["trip-members", tripId] });
    }
  });
}

export function useTripShareLinksQuery(tripId: string) {
  return useQuery({
    queryKey: ["trip-share-links", tripId],
    queryFn: () => listTripShareLinks(tripId),
    enabled: Boolean(tripId)
  });
}

export function useCreateTripShareLinkMutation(tripId: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: () => trackQueuedMutation("trip-share-links.create", () => createTripShareLink(tripId)),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["trip-share-links", tripId] });
    }
  });
}

export function useRevokeTripShareLinkMutation(tripId: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (linkId: string) => trackQueuedMutation("trip-share-links.revoke", () => revokeTripShareLink(tripId, linkId)),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["trip-share-links", tripId] });
    }
  });
}

export function useTripInvitationsQuery(tripId: string) {
  return useQuery({
    queryKey: ["trip-invitations", tripId],
    queryFn: () => listTripInvitations(tripId),
    enabled: Boolean(tripId)
  });
}

export function useCreateTripInvitationMutation(tripId: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (input: { inviteeEmail: string; role: "editor" | "commenter" | "viewer" }) =>
      trackQueuedMutation("trip-invitations.create", () => createTripInvitation(tripId, input)),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["trip-invitations", tripId] });
    }
  });
}

export function useRevokeTripInvitationMutation(tripId: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (invitationId: string) =>
      trackQueuedMutation("trip-invitations.revoke", () => revokeTripInvitation(tripId, invitationId)),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["trip-invitations", tripId] });
    }
  });
}

export function useRequestMagicLinkMutation() {
  return useMutation({
    mutationFn: (email: string) => requestMagicLink(email)
  });
}

export function useVerifyMagicLinkMutation() {
  return useMutation({
    mutationFn: ({ email, code }: { email: string; code: string }) => verifyMagicLink(email, code)
  });
}

export function useAiPlansQuery(tripId: string) {
  return useQuery({
    queryKey: ["ai-plans", tripId],
    queryFn: () => listAiPlans(tripId),
    enabled: Boolean(tripId)
  });
}

export function useAiPlanQuery(tripId: string, planId: string) {
  return useQuery({
    queryKey: ["ai-plan", tripId, planId],
    queryFn: () => getAiPlan(tripId, planId),
    enabled: Boolean(tripId) && Boolean(planId)
  });
}

export function useCreateAiPlanMutation(tripId: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (input: {
      providerConfigId: string;
      title: string;
      constraints: {
        totalBudget: number;
        currency: string;
        pace: "relaxed" | "balanced" | "packed";
        transportPreference: "walk" | "transit" | "taxi" | "mixed";
        wakePattern?: "early" | "normal" | "late";
        poiDensity?: "sparse" | "medium" | "dense";
        mustVisit: string[];
        avoid: string[];
      };
    }) => trackQueuedMutation("ai-plans.create", () => createAiPlan(tripId, input)),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["ai-plans", tripId] });
    }
  });
}

export function useAdoptAiPlanMutation(tripId: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ planId, confirmWarnings = false }: { planId: string; confirmWarnings?: boolean }) =>
      trackQueuedMutation("ai-plans.adopt", () => adoptAiPlan(tripId, planId, { confirmWarnings })),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["ai-plans", tripId] });
    }
  });
}

export function useBudgetProfileQuery(tripId: string) {
  return useQuery({
    queryKey: ["budget", tripId],
    queryFn: () => getBudgetProfile(tripId),
    enabled: Boolean(tripId)
  });
}

export function useExpensesQuery(tripId: string) {
  return useQuery({
    queryKey: ["expenses", tripId],
    queryFn: () => listExpenses(tripId),
    enabled: Boolean(tripId)
  });
}

export function useUpsertBudgetMutation(tripId: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (input: {
      totalBudget?: number;
      perPersonBudget?: number;
      perDayBudget?: number;
      currency: string;
      categories: Array<{ category: string; plannedAmount: number }>;
    }) => trackQueuedMutation("budget.upsert", () => upsertBudgetProfile(tripId, input)),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["budget", tripId] });
    }
  });
}

export function useCreateExpenseMutation(tripId: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (input: {
      category: string;
      amount: number;
      currency: string;
      expenseAt?: string;
      note?: string;
      linkedItemId?: string;
    }) => trackQueuedMutation("expenses.create", () => createExpense(tripId, input)),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["expenses", tripId] });
      queryClient.invalidateQueries({ queryKey: ["budget", tripId] });
    }
  });
}

export function useDeleteExpenseMutation(tripId: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (expenseId: string) => trackQueuedMutation("expenses.delete", () => deleteExpense(tripId, expenseId)),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["expenses", tripId] });
      queryClient.invalidateQueries({ queryKey: ["budget", tripId] });
    }
  });
}

export function usePatchExpenseMutation(tripId: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({
      expenseId,
      input
    }: {
      expenseId: string;
      input: {
        category?: string;
        amount?: number;
        currency?: string;
        expenseAt?: string;
        note?: string;
        linkedItemId?: string;
      };
    }) => trackQueuedMutation("expenses.patch", () => patchExpense(tripId, expenseId, input)),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["expenses", tripId] });
      queryClient.invalidateQueries({ queryKey: ["budget", tripId] });
    }
  });
}

export function useBudgetRatesQuery(tripId: string, options?: { from?: string; to?: string }) {
  return useQuery({
    queryKey: ["budget-rates", tripId, options?.from ?? "*", options?.to ?? "*"],
    queryFn: () => getBudgetRates(tripId, options),
    enabled: Boolean(tripId)
  });
}

export function useRefreshBudgetRateMutation(tripId: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ from, to }: { from: string; to: string }) => refreshBudgetRate(tripId, from, to),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["budget-rates", tripId] });
    }
  });
}

export function useNotificationsQuery(unreadOnly = false) {
  return useQuery({
    queryKey: ["notifications", unreadOnly],
    queryFn: () => listNotifications({ unreadOnly })
  });
}

export function useMarkNotificationReadMutation() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (notificationId: string) => trackQueuedMutation("notifications.read", () => markNotificationRead(notificationId)),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["notifications"] });
    }
  });
}

export function useMarkNotificationUnreadMutation() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (notificationId: string) => trackQueuedMutation("notifications.unread", () => markNotificationUnread(notificationId)),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["notifications"] });
    }
  });
}

export function useMarkAllNotificationsReadMutation() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: () => trackQueuedMutation("notifications.read-all", () => markAllNotificationsRead()),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["notifications"] });
    }
  });
}

export function useDeleteNotificationMutation() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (notificationId: string) => trackQueuedMutation("notifications.delete", () => deleteNotification(notificationId)),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["notifications"] });
    }
  });
}

export function useCleanupReadNotificationsMutation() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: () => trackQueuedMutation("notifications.cleanup-read", () => cleanupReadNotifications()),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["notifications"] });
    }
  });
}

export function useItineraryDaysQuery(tripId: string) {
  return useQuery({
    queryKey: ["itinerary-days", tripId],
    queryFn: () => listItineraryDays(tripId),
    enabled: Boolean(tripId)
  });
}

export function useCreateItineraryItemMutation(tripId: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (input: {
      dayId: string;
      title: string;
      itemType: string;
      allDay: boolean;
      note?: string;
      startAt?: string;
      endAt?: string;
      placeId?: string;
      lat?: number;
      lng?: number;
      placeSnapshotId?: string;
      routeSnapshotId?: string;
      estimatedCostAmount?: number;
      estimatedCostCurrency?: string;
    }) => trackQueuedMutation("itinerary-items.create", () => createItineraryItem(tripId, input)),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["itinerary-days", tripId] });
    }
  });
}

export function useDeleteItineraryItemMutation(tripId: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (itemId: string) => trackQueuedMutation("itinerary-items.delete", () => deleteItineraryItem(tripId, itemId)),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["itinerary-days", tripId] });
    }
  });
}

export function usePatchItineraryItemMutation(tripId: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({
      itemId,
      version,
      input
    }: {
      itemId: string;
      version: number;
      input: { title?: string; startAt?: string; endAt?: string; allDay?: boolean; note?: string; sortOrder?: number; placeId?: string; lat?: number; lng?: number };
    }) => trackQueuedMutation("itinerary-items.patch", () => patchItineraryItem(tripId, itemId, version, input)),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["itinerary-days", tripId] });
    }
  });
}

export function useReorderItineraryItemsMutation(tripId: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (input: { operations: Array<{ itemId: string; targetDayId: string; targetSortOrder: number }> }) => trackQueuedMutation("itinerary-items.reorder", () => reorderItineraryItems(tripId, input)),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["itinerary-days", tripId] });
    }
  });
}

export function useMapPlacesQuery(query: string) {
  return useQuery({
    queryKey: ["map-places", query],
    queryFn: () => searchPlaces(query),
    enabled: query.trim().length > 0
  });
}

export function useEstimateRouteMutation() {
  return useMutation({
    mutationFn: (input: { origin: { lat: number; lng: number }; destination: { lat: number; lng: number }; mode: "walk" | "transit" | "drive" | "taxi" }) =>
      estimateRoute(input)
  });
}

export function useSyncBootstrapQuery(tripId: string, sinceVersion = 0) {
  return useQuery({
    queryKey: ["sync-bootstrap", tripId, sinceVersion],
    queryFn: () => getSyncBootstrap(tripId, sinceVersion),
    enabled: true,
    refetchInterval: 30000
  });
}

export function useFlushSyncMutationsMutation() {
  return useMutation({
    mutationFn: ({
      tripId,
      mutations
    }: {
      tripId: string;
      mutations: Array<{ id: string; entityType: string; entityId: string; baseVersion: number }>;
    }) => flushSyncMutations(tripId, mutations)
  });
}

export function useMyProfileQuery() {
  return useQuery({
    queryKey: ["users", "me"],
    queryFn: getMyProfile
  });
}

export function usePatchMyProfileMutation() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: patchMyProfile,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["users", "me"] });
    }
  });
}

export function useDeleteMyAccountMutation() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: deleteMyAccount,
    onSuccess: () => {
      queryClient.clear();
    }
  });
}

export function useMyPreferencesQuery() {
  return useQuery({
    queryKey: ["users", "preferences"],
    queryFn: getMyPreferences
  });
}

export function usePutMyPreferencesMutation() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: putMyPreferences,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["users", "preferences"] });
    }
  });
}

export function useMyNotificationPreferencesQuery() {
  return useQuery({
    queryKey: ["users", "notification-preferences"],
    queryFn: getMyNotificationPreferences
  });
}

export function usePutMyNotificationPreferencesMutation() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: putMyNotificationPreferences,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["users", "notification-preferences"] });
    }
  });
}

export function useMyLlmProvidersQuery() {
  return useQuery({
    queryKey: ["users", "llm-providers"],
    queryFn: () => listMyLlmProviders()
  });
}

export function useCreateMyLlmProviderMutation() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: createMyLlmProvider,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["users", "llm-providers"] });
    }
  });
}

export function useDeleteMyLlmProviderMutation() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (providerId: string) => deleteMyLlmProvider(providerId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["users", "llm-providers"] });
    }
  });
}
