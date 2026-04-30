import { relayFetch, type RelayEnvelope, type RelaySuccessEnvelope } from "./api";

export type PacketBuilderSnapshot = {
  snapshotId: string;
  projectId: string;
  schemaVersion?: string;
  type: string;
  target: string;
  taskSummary?: string;
  renderedBody: string;
  styleCues: Array<{
    heuristicId: string;
    heuristicKey?: string;
    canonicalText?: string;
    whySelected: string;
    whyIncluded?: string;
    sourceSummary: string;
    sourceRefs: string[];
  }>;
  supportingNotes: Array<{ noteId: string; source?: string; excerpt: string; evidence?: string }>;
  supportingDecisions: Array<{ decisionId: string; summary: string; why?: string }>;
  supportingQuestions: Array<{ questionId: string; summary: string }>;
  supportingArtifacts: Array<{ artifactId: string; type?: string; sourcePath?: string; trustLevel?: string }>;
  whyIncluded: string[];
  approvedHeuristicIds: string[];
  decisionIds: string[];
  openQuestionIds: string[];
  sourceArtifactIds: string[];
  missingContext: string[];
  publicReadable: boolean;
  publicToken?: string;
  createdAt: string;
};

type ServerPacketBuilderSnapshot = {
  snapshot_id: string;
  project_id: string;
  schema_version?: string;
  type: string;
  target: string;
  task_summary?: string;
  rendered_body?: string;
  style_cues?: Array<{
    heuristic_id: string;
    heuristic_key?: string;
    canonical_text?: string;
    why_selected: string;
    why_included?: string;
    source_summary: string;
    source_refs?: string[];
  }>;
  supporting_notes?: Array<{ note_id: string; source?: string; excerpt: string; evidence?: string }>;
  supporting_decisions?: Array<{ decision_id: string; summary: string; why?: string }>;
  supporting_questions?: Array<{ question_id: string; summary: string }>;
  supporting_artifacts?: Array<{ artifact_id: string; type?: string; source_path?: string; trust_level?: string }>;
  why_included?: string[];
  approved_heuristic_ids?: string[];
  decision_ids?: string[];
  open_question_ids?: string[];
  source_artifact_ids?: string[];
  missing_context?: string[];
  public_readable?: boolean;
  public_token?: string;
  created_at: string;
};

type PacketBuilderOptions = {
  headers?: HeadersInit;
  signal?: AbortSignal;
};

export class PacketBuilderError extends Error {
  readonly status: number;
  readonly code: string;
  readonly retryable: boolean;

  constructor(status: number, code: string, message: string, retryable = false) {
    super(message);
    this.name = "PacketBuilderError";
    this.status = status;
    this.code = code;
    this.retryable = retryable;
  }
}

async function unwrap<T>(res: Response): Promise<T> {
  let body: RelayEnvelope<T> | null = null;
  try {
    body = (await res.json()) as RelayEnvelope<T>;
  } catch {
    throw new PacketBuilderError(
      res.status,
      "INVALID_RESPONSE",
      `non-JSON response (status ${res.status})`,
      res.status >= 500,
    );
  }

  if (!body || body.ok === false) {
    const err = (body as { ok: false; error?: { code?: string; message?: string; retryable?: boolean } } | null)?.error;
    throw new PacketBuilderError(
      res.status,
      err?.code ?? "UNKNOWN",
      err?.message ?? `relay api ${res.status}`,
      err?.retryable ?? res.status >= 500,
    );
  }

  return (body as RelaySuccessEnvelope<T>).data;
}

function mapSnapshot(data: ServerPacketBuilderSnapshot): PacketBuilderSnapshot {
  return {
    snapshotId: data.snapshot_id,
    projectId: data.project_id,
    schemaVersion: data.schema_version || undefined,
    type: data.type,
    target: data.target,
    taskSummary: data.task_summary || undefined,
    renderedBody: data.rendered_body ?? "",
    styleCues: (data.style_cues ?? []).map((cue) => ({
      heuristicId: cue.heuristic_id,
      heuristicKey: cue.heuristic_key || undefined,
      canonicalText: cue.canonical_text || undefined,
      whySelected: cue.why_selected,
      whyIncluded: cue.why_included || undefined,
      sourceSummary: cue.source_summary,
      sourceRefs: cue.source_refs ?? [],
    })),
    supportingNotes: (data.supporting_notes ?? []).map((note) => ({
      noteId: note.note_id,
      source: note.source || undefined,
      excerpt: note.excerpt,
      evidence: note.evidence || undefined,
    })),
    supportingDecisions: (data.supporting_decisions ?? []).map((decision) => ({
      decisionId: decision.decision_id,
      summary: decision.summary,
      why: decision.why || undefined,
    })),
    supportingQuestions: (data.supporting_questions ?? []).map((question) => ({
      questionId: question.question_id,
      summary: question.summary,
    })),
    supportingArtifacts: (data.supporting_artifacts ?? []).map((artifact) => ({
      artifactId: artifact.artifact_id,
      type: artifact.type || undefined,
      sourcePath: artifact.source_path || undefined,
      trustLevel: artifact.trust_level || undefined,
    })),
    whyIncluded: data.why_included ?? [],
    approvedHeuristicIds: data.approved_heuristic_ids ?? [],
    decisionIds: data.decision_ids ?? [],
    openQuestionIds: data.open_question_ids ?? [],
    sourceArtifactIds: data.source_artifact_ids ?? [],
    missingContext: data.missing_context ?? [],
    publicReadable: data.public_readable ?? false,
    publicToken: data.public_token || undefined,
    createdAt: data.created_at,
  };
}

export async function getLatestPacketSnapshot(
  projectId: string,
  opts?: PacketBuilderOptions,
): Promise<PacketBuilderSnapshot> {
  const res = await relayFetch(
    `/v1/projects/${encodeURIComponent(projectId)}/packet-snapshots/latest`,
    {
      method: "GET",
      headers: opts?.headers,
      signal: opts?.signal,
    },
  );
  const data = await unwrap<ServerPacketBuilderSnapshot>(res);
  return mapSnapshot(data);
}
