import { relayFetch, type RelayEnvelope } from "@/lib/api";

export type ProviderCredentialStatus = {
  provider: "anthropic";
  connected: boolean;
  key_prefix?: string;
  key_last4?: string;
  updated_at?: string;
};

export type ProviderCredentialListData = {
  credentials: ProviderCredentialStatus[];
};

export async function listProviderCredentials(headers?: HeadersInit): Promise<ProviderCredentialListData> {
  const res = await relayFetch("/v1/settings/provider-credentials", {
    method: "GET",
    headers,
    cache: "no-store",
  });
  const body = (await res.json()) as RelayEnvelope<ProviderCredentialListData>;
  if (!body.ok) {
    throw new Error(body.error.message || body.error.code);
  }
  return body.data;
}

export async function connectProviderCredential(apiKey: string): Promise<ProviderCredentialStatus> {
  const res = await relayFetch("/v1/settings/provider-credentials", {
    method: "POST",
    headers: { "content-type": "application/json" },
    body: JSON.stringify({ provider: "anthropic", api_key: apiKey }),
  });
  const body = (await res.json()) as RelayEnvelope<ProviderCredentialStatus>;
  if (!body.ok) {
    throw new Error(body.error.message || body.error.code);
  }
  return body.data;
}

export async function disconnectProviderCredential(): Promise<void> {
  const res = await relayFetch("/v1/settings/provider-credentials/anthropic", {
    method: "DELETE",
  });
  const body = (await res.json()) as RelayEnvelope<{ status: "ok" }>;
  if (!body.ok) {
    throw new Error(body.error.message || body.error.code);
  }
}
