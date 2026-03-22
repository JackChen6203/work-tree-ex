import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { addTripMember, createTrip, getTrip, listTripMembers, listTrips, patchTrip, removeTripMember, updateTripMemberRole } from "./trips-api";
import { requestMagicLink, verifyMagicLink } from "./auth-api";
import { adoptAiPlan, createAiPlan, getAiPlan, listAiPlans } from "./ai-planner-api";
import { createExpense, deleteExpense, getBudgetProfile, listExpenses, patchExpense, upsertBudgetProfile } from "./budget-api";
import { deleteNotification, listNotifications, markAllNotificationsRead, markNotificationRead } from "./notifications-api";
import { createItineraryItem, deleteItineraryItem, listItineraryDays, patchItineraryItem, reorderItineraryItems } from "./itinerary-api";
import { estimateRoute, searchPlaces } from "./maps-api";
import {
  createMyLlmProvider,
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
    mutationFn: (input: CreateTripInput) => createTrip(input),
    onSuccess: (trip) => {
      queryClient.invalidateQueries({ queryKey: ["trips"] });
      queryClient.setQueryData(["trips", trip.id], trip);
    }
  });
}

export function usePatchTripMutation(tripId: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ version, input }: { version: number; input: PatchTripInput }) => patchTrip(tripId, version, input),
    onSuccess: (trip) => {
      queryClient.invalidateQueries({ queryKey: ["trips"] });
      queryClient.setQueryData(["trips", trip.id], trip);
    }
  });
}

export function useTripMembersQuery(tripId: string) {
  return useQuery({
    queryKey: ["trip-members", tripId],
    queryFn: () => listTripMembers(tripId),
    enabled: Boolean(tripId)
  });
}

export function useAddTripMemberMutation(tripId: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (input: AddTripMemberInput) => addTripMember(tripId, input),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["trip-members", tripId] });
    }
  });
}

export function useRemoveTripMemberMutation(tripId: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (memberId: string) => removeTripMember(tripId, memberId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["trip-members", tripId] });
    }
  });
}

export function useUpdateTripMemberRoleMutation(tripId: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ memberId, role }: { memberId: string; role: "owner" | "editor" | "commenter" | "viewer" }) =>
      updateTripMemberRole(tripId, memberId, role),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["trip-members", tripId] });
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
        mustVisit: string[];
        avoid: string[];
      };
    }) => createAiPlan(tripId, input),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["ai-plans", tripId] });
    }
  });
}

export function useAdoptAiPlanMutation(tripId: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (planId: string) => adoptAiPlan(tripId, planId),
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
    }) => upsertBudgetProfile(tripId, input),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["budget", tripId] });
    }
  });
}

export function useCreateExpenseMutation(tripId: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (input: { category: string; amount: number; currency: string; expenseAt?: string; note?: string }) => createExpense(tripId, input),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["expenses", tripId] });
      queryClient.invalidateQueries({ queryKey: ["budget", tripId] });
    }
  });
}

export function useDeleteExpenseMutation(tripId: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (expenseId: string) => deleteExpense(tripId, expenseId),
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
      input: { category?: string; amount?: number; currency?: string; expenseAt?: string; note?: string };
    }) => patchExpense(tripId, expenseId, input),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["expenses", tripId] });
      queryClient.invalidateQueries({ queryKey: ["budget", tripId] });
    }
  });
}

export function useNotificationsQuery(unreadOnly = false) {
  return useQuery({
    queryKey: ["notifications", unreadOnly],
    queryFn: () => listNotifications(unreadOnly)
  });
}

export function useMarkNotificationReadMutation() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (notificationId: string) => markNotificationRead(notificationId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["notifications"] });
    }
  });
}

export function useMarkAllNotificationsReadMutation() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: markAllNotificationsRead,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["notifications"] });
    }
  });
}

export function useDeleteNotificationMutation() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (notificationId: string) => deleteNotification(notificationId),
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
    mutationFn: (input: { dayId: string; title: string; itemType: string; allDay: boolean; note?: string }) => createItineraryItem(tripId, input),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["itinerary-days", tripId] });
    }
  });
}

export function useDeleteItineraryItemMutation(tripId: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (itemId: string) => deleteItineraryItem(tripId, itemId),
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
    }) => patchItineraryItem(tripId, itemId, version, input),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["itinerary-days", tripId] });
    }
  });
}

export function useReorderItineraryItemsMutation(tripId: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (input: { operations: Array<{ itemId: string; targetDayId: string; targetSortOrder: number }> }) => reorderItineraryItems(tripId, input),
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
    queryFn: listMyLlmProviders
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
