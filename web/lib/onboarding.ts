import { relayFetch, type RelayEnvelope } from "@/lib/api";

export type AuthMe = {
  user_id: string;
  email?: string;
  display_name?: string;
  avatar_url?: string;
  onboarding_complete: boolean;
  default_project_id?: string;
  last_validated_at?: string;
};

export type OnboardingCompleteResult = {
  onboarding_complete: boolean;
  default_project_id: string;
};

export async function completeOnboarding(): Promise<OnboardingCompleteResult> {
  const res = await relayFetch("/v1/onboarding", {
    method: "POST",
    headers: { "content-type": "application/json" },
    body: JSON.stringify({}),
  });
  const body = (await res.json()) as RelayEnvelope<OnboardingCompleteResult>;
  if (!body.ok) {
    throw new Error(body.error.message || body.error.code);
  }
  return body.data;
}
