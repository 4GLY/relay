import { relayFetch, type RelayEnvelope, type RelayErrorPayload } from "@/lib/api";

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

export class ProviderCredentialAPIError extends Error {
  code: string;
  retryable: boolean;

  constructor(payload: RelayErrorPayload) {
    super(payload.message || payload.code);
    this.name = "ProviderCredentialAPIError";
    this.code = payload.code;
    this.retryable = payload.retryable;
  }
}

function throwRelayError(payload: RelayErrorPayload): never {
  throw new ProviderCredentialAPIError(payload);
}

export async function listProviderCredentials(headers?: HeadersInit): Promise<ProviderCredentialListData> {
  const res = await relayFetch("/v1/settings/provider-credentials", {
    method: "GET",
    headers,
    cache: "no-store",
  });
  const body = (await res.json()) as RelayEnvelope<ProviderCredentialListData>;
  if (!body.ok) {
    throwRelayError(body.error);
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
    throwRelayError(body.error);
  }
  return body.data;
}

export async function disconnectProviderCredential(): Promise<void> {
  const res = await relayFetch("/v1/settings/provider-credentials/anthropic", {
    method: "DELETE",
  });
  const body = (await res.json()) as RelayEnvelope<{ status: "ok" }>;
  if (!body.ok) {
    throwRelayError(body.error);
  }
}
