import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { createTrip, getTrip, listTrips, patchTrip } from "./trips-api";
import { requestMagicLink, verifyMagicLink } from "./auth-api";
import type { CreateTripInput, PatchTripInput } from "./trips-api";

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
