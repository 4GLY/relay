import { relayFetch, type RelayEnvelope, type RelayErrorPayload } from "@/lib/api";

export type UserAPIKeySummary = {
  key_id: string;
  name: string;
  token_prefix: string;
  scope: "global" | "project";
  project_id?: string;
  revoked: boolean;
};

export type ListUserAPIKeysResult = {
  items: UserAPIKeySummary[];
};

export type IssueUserAPIKeyResult = {
  key_id: string;
  name: string;
  token: string;
  token_prefix: string;
  scope: "global" | "project";
  project_id?: string;
};

export type RevokeUserAPIKeyResult = {
  key_id: string;
  name: string;
  token_prefix: string;
  scope: "global" | "project";
  project_id?: string;
  revoked: boolean;
};

export class RelayAPIError extends Error {
  code: string;
  retryable: boolean;
  missingFields?: string[];

  constructor(payload: RelayErrorPayload) {
    super(payload.message || payload.code);
    this.name = "RelayAPIError";
    this.code = payload.code;
    this.retryable = payload.retryable;
    this.missingFields = payload.missing_fields;
  }
}

async function parseRelayResponse<T>(res: Response): Promise<T> {
  const body = (await res.json()) as RelayEnvelope<T>;
  if (!body.ok) {
    throw new RelayAPIError(body.error);
  }
  return body.data;
}

export async function listUserAPIKeys(headers?: HeadersInit): Promise<ListUserAPIKeysResult> {
  const res = await relayFetch("/v1/settings/api-keys", {
    method: "GET",
    headers,
    cache: "no-store",
  });
  return parseRelayResponse<ListUserAPIKeysResult>(res);
}

export async function issueUserAPIKey(name: string): Promise<IssueUserAPIKeyResult> {
  const res = await relayFetch("/v1/settings/api-keys", {
    method: "POST",
    headers: { "content-type": "application/json" },
    body: JSON.stringify({ name }),
  });
  return parseRelayResponse<IssueUserAPIKeyResult>(res);
}

export async function revokeUserAPIKey(keyID: string): Promise<RevokeUserAPIKeyResult> {
  const res = await relayFetch("/v1/settings/api-keys/revoke", {
    method: "POST",
    headers: { "content-type": "application/json" },
    body: JSON.stringify({ key_id: keyID }),
  });
  return parseRelayResponse<RevokeUserAPIKeyResult>(res);
}
