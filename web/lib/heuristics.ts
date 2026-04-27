/**
 * Typed wrappers around the Style Memory endpoints.
 *
 * Mirrors `internal/services/types.go` `PendingProposalSummary` +
 * `ApprovedHeuristicSummary` shapes. Server speaks snake_case; React surface is
 * camelCase, so this module owns the mapping.
 */

import { relayFetch, type RelayEnvelope, type RelaySuccessEnvelope } from "./api";

export type ProposalState = "pending" | "approved" | "rejected" | "archived";
export type ApprovedState = "active" | "disabled" | "archived";
export type ReviewAction = "approve" | "reject" | "archive";
export type ApprovedAction = "disable" | "archive" | "approve" | "enable";

export type PendingProposal = {
  proposalId: string;
  projectId: string;
  originTraceId?: string;
  workflow?: string;
  artifactType?: string;
  heuristicKey: string;
  canonicalText: string;
  normalizedText?: string;
  state: string;
  sourceTraceIds: string[];
  sourceRefs: string[];
  proposedBy?: string;
  reviewNotes?: string;
  createdAt: string;
  updatedAt: string;
};

export type ApprovedHeuristic = {
  heuristicId: string;
  projectId: string;
  originProposalId?: string;
  workflow?: string;
  artifactType?: string;
  heuristicKey: string;
  canonicalText: string;
  state: string;
  sourceTraceIds: string[];
  sourceRefs: string[];
  createdAt: string;
  updatedAt: string;
};

export type ListResult<T> = {
  items: T[];
  nextCursor?: string;
};

type ServerPendingProposal = {
  proposal_id: string;
  project_id: string;
  origin_trace_id?: string;
  workflow?: string;
  artifact_type?: string;
  heuristic_key: string;
  canonical_text: string;
  normalized_text?: string;
  state: string;
  source_trace_ids?: string[];
  source_refs?: string[];
  proposed_by?: string;
  review_notes?: string;
  created_at: string;
  updated_at: string;
};

type ServerApprovedHeuristic = {
  heuristic_id: string;
  project_id: string;
  origin_proposal_id?: string;
  workflow?: string;
  artifact_type?: string;
  heuristic_key: string;
  canonical_text: string;
  state: string;
  source_trace_ids?: string[];
  source_refs?: string[];
  created_at: string;
  updated_at: string;
};

type ServerList<T> = { items?: T[]; next_cursor?: string };

export type ListOptions = {
  limit?: number;
  cursor?: string;
  /** Forward HTTP headers (RSC server fetches need explicit `Cookie`). */
  headers?: HeadersInit;
  /** AbortSignal hook for client retries. */
  signal?: AbortSignal;
};

export class HeuristicsError extends Error {
  readonly status: number;
  readonly code: string;
  readonly retryable: boolean;

  constructor(status: number, code: string, message: string, retryable = false) {
    super(message);
    this.name = "HeuristicsError";
    this.status = status;
    this.code = code;
    this.retryable = retryable;
  }
}

export class ProposalAlreadyResolvedError extends HeuristicsError {
  constructor(message = "Already resolved by another session") {
    super(409, "PROPOSAL_ALREADY_RESOLVED", message, false);
    this.name = "ProposalAlreadyResolvedError";
  }
}

function mapPending(s: ServerPendingProposal): PendingProposal {
  return {
    proposalId: s.proposal_id,
    projectId: s.project_id,
    originTraceId: s.origin_trace_id || undefined,
    workflow: s.workflow || undefined,
    artifactType: s.artifact_type || undefined,
    heuristicKey: s.heuristic_key,
    canonicalText: s.canonical_text,
    normalizedText: s.normalized_text || undefined,
    state: s.state,
    sourceTraceIds: s.source_trace_ids ?? [],
    sourceRefs: s.source_refs ?? [],
    proposedBy: s.proposed_by || undefined,
    reviewNotes: s.review_notes || undefined,
    createdAt: s.created_at,
    updatedAt: s.updated_at,
  };
}

function mapApproved(s: ServerApprovedHeuristic): ApprovedHeuristic {
  return {
    heuristicId: s.heuristic_id,
    projectId: s.project_id,
    originProposalId: s.origin_proposal_id || undefined,
    workflow: s.workflow || undefined,
    artifactType: s.artifact_type || undefined,
    heuristicKey: s.heuristic_key,
    canonicalText: s.canonical_text,
    state: s.state,
    sourceTraceIds: s.source_trace_ids ?? [],
    sourceRefs: s.source_refs ?? [],
    createdAt: s.created_at,
    updatedAt: s.updated_at,
  };
}

async function unwrap<T>(res: Response): Promise<T> {
  let body: RelayEnvelope<T> | null = null;
  try {
    body = (await res.json()) as RelayEnvelope<T>;
  } catch {
    throw new HeuristicsError(res.status, "INVALID_RESPONSE", `non-JSON response (status ${res.status})`, res.status >= 500);
  }
  if (!body || (body as RelayEnvelope<T>).ok === false) {
    const err = (body as { ok: false; error?: { code?: string; message?: string; retryable?: boolean } } | null)?.error;
    const code = err?.code ?? "UNKNOWN";
    const message = err?.message ?? `relay api ${res.status}`;
    if (res.status === 409 && code === "PROPOSAL_ALREADY_RESOLVED") {
      throw new ProposalAlreadyResolvedError(message);
    }
    throw new HeuristicsError(res.status, code, message, err?.retryable ?? res.status >= 500);
  }
  return (body as RelaySuccessEnvelope<T>).data;
}

function buildListQuery(projectId: string, opts?: ListOptions, extra: Record<string, string> = {}): string {
  const params = new URLSearchParams({ project_id: projectId, ...extra });
  if (opts?.limit !== undefined) params.set("limit", String(opts.limit));
  if (opts?.cursor) params.set("cursor", opts.cursor);
  return params.toString();
}

export async function listPendingProposals(
  projectId: string,
  opts?: ListOptions,
): Promise<ListResult<PendingProposal>> {
  const qs = buildListQuery(projectId, opts, { state: "pending" });
  const res = await relayFetch(`/v1/heuristic-proposals?${qs}`, {
    method: "GET",
    headers: opts?.headers,
    signal: opts?.signal,
  });
  const data = await unwrap<ServerList<ServerPendingProposal>>(res);
  return { items: (data.items ?? []).map(mapPending), nextCursor: data.next_cursor };
}

export async function listApprovedHeuristics(
  projectId: string,
  opts?: ListOptions,
): Promise<ListResult<ApprovedHeuristic>> {
  const qs = buildListQuery(projectId, opts);
  const res = await relayFetch(`/v1/approved-heuristics?${qs}`, {
    method: "GET",
    headers: opts?.headers,
    signal: opts?.signal,
  });
  const data = await unwrap<ServerList<ServerApprovedHeuristic>>(res);
  return { items: (data.items ?? []).map(mapApproved), nextCursor: data.next_cursor };
}

export type ReviewProposalArgs = {
  projectId: string;
  proposalId: string;
  action: ReviewAction;
  reviewNotes?: string;
  signal?: AbortSignal;
};

export type ReviewProposalResult = {
  proposalId: string;
  approvedHeuristicId?: string;
  projectId: string;
  state: string;
};

export async function reviewProposal(args: ReviewProposalArgs): Promise<ReviewProposalResult> {
  const res = await relayFetch("/v1/heuristic-proposals/review", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      project_id: args.projectId,
      proposal_id: args.proposalId,
      action: args.action,
      ...(args.reviewNotes !== undefined ? { review_notes: args.reviewNotes } : {}),
    }),
    signal: args.signal,
  });
  const data = await unwrap<{
    proposal_id: string;
    approved_heuristic_id?: string;
    project_id: string;
    state: string;
  }>(res);
  return {
    proposalId: data.proposal_id,
    approvedHeuristicId: data.approved_heuristic_id || undefined,
    projectId: data.project_id,
    state: data.state,
  };
}

export type UpdateApprovedArgs = {
  heuristicId: string;
  action: ApprovedAction;
  signal?: AbortSignal;
};

export type UpdateApprovedResult = {
  heuristicId: string;
  projectId: string;
  state: string;
};

export async function updateApprovedHeuristic(args: UpdateApprovedArgs): Promise<UpdateApprovedResult> {
  const res = await relayFetch("/v1/approved-heuristics/update", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      heuristic_id: args.heuristicId,
      action: args.action,
    }),
    signal: args.signal,
  });
  const data = await unwrap<{ heuristic_id: string; project_id: string; state: string }>(res);
  return {
    heuristicId: data.heuristic_id,
    projectId: data.project_id,
    state: data.state,
  };
}

/** Reject reason taxonomy from the Style Memory plan §"Reject reason taxonomy". */
export const REJECT_REASON_CODES = [
  "duplicate",
  "wrong",
  "too_narrow",
  "too_broad",
  "stale",
  "other",
] as const;

export type RejectReasonCode = (typeof REJECT_REASON_CODES)[number];

/** Serializes the reject taxonomy into the existing `review_notes` field per Decision #13. */
export function serializeRejectNotes(code: RejectReasonCode, freeText?: string): string {
  const trimmed = (freeText ?? "").trim();
  return trimmed ? `reason:${code}; ${trimmed}` : `reason:${code}`;
}
