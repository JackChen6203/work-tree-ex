import { apiRequest } from "./api";

export interface AiPlanDraft {
  id: string;
  tripId: string;
  title: string;
  status: "valid" | "warning" | "invalid";
  summary: string;
  warnings: string[];
  totalEstimated: number;
  budget: number;
  currency: string;
  createdAt: string;
}

interface CreateAiPlanInput {
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
}

interface CreateAiPlanResponse {
  jobId: string;
  status: string;
  acceptedAt: string;
}

interface AdoptAiPlanResponse {
  tripId: string;
  planId: string;
  adopted: boolean;
  status: "valid" | "warning" | "invalid";
  warnings: string[];
}

export function createAiPlan(tripId: string, input: CreateAiPlanInput) {
  return apiRequest<CreateAiPlanResponse>(`/api/v1/trips/${tripId}/ai/plans`, {
    method: "POST",
    headers: {
      "Idempotency-Key": crypto.randomUUID()
    },
    body: JSON.stringify(input)
  });
}

export function listAiPlans(tripId: string) {
  return apiRequest<AiPlanDraft[]>(`/api/v1/trips/${tripId}/ai/plans`);
}

export function adoptAiPlan(tripId: string, planId: string) {
  return apiRequest<AdoptAiPlanResponse>(`/api/v1/trips/${tripId}/ai/plans/${planId}/adopt`, {
    method: "POST",
    headers: {
      "Idempotency-Key": crypto.randomUUID()
    }
  });
}
