import { relayFetch, type RelayEnvelope, type RelaySuccessEnvelope } from "./api";

export type ProjectExplorer = {
  project: {
    projectId: string;
    name: string;
    status: string;
  };
  counts: {
    notes: number;
    artifacts: number;
    decisions: number;
    openQuestions: number;
    judgmentTraces: number;
    pendingProposals: number;
    approvedHeuristics: number;
    rejectedProposals: number;
    packetSnapshots: number;
  };
  latestSnapshot?: {
    snapshotId: string;
    packetKind: string;
    target: string;
    taskSummary?: string;
    createdAt: string;
    publicReadable: boolean;
  };
  styleMemory: {
    nextProposalId?: string;
    nextProposalText?: string;
  };
  recentActivity: Array<{
    kind: string;
    id: string;
    title: string;
    createdAt: string;
  }>;
};

type ServerProjectExplorer = {
  project: {
    project_id: string;
    name: string;
    status: string;
  };
  counts: {
    notes: number;
    artifacts: number;
    decisions: number;
    open_questions: number;
    judgment_traces: number;
    pending_proposals: number;
    approved_heuristics: number;
    rejected_proposals: number;
    packet_snapshots: number;
  };
  latest_snapshot?: {
    snapshot_id: string;
    packet_kind: string;
    target: string;
    task_summary?: string;
    created_at: string;
    public_readable: boolean;
  };
  style_memory: {
    next_proposal_id?: string;
    next_proposal_text?: string;
  };
  recent_activity?: Array<{
    kind: string;
    id: string;
    title: string;
    created_at: string;
  }>;
};

type ExplorerOptions = {
  headers?: HeadersInit;
  signal?: AbortSignal;
};

export class ProjectExplorerError extends Error {
  readonly status: number;
  readonly code: string;
  readonly retryable: boolean;

  constructor(status: number, code: string, message: string, retryable = false) {
    super(message);
    this.name = "ProjectExplorerError";
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
    throw new ProjectExplorerError(
      res.status,
      "INVALID_RESPONSE",
      `non-JSON response (status ${res.status})`,
      res.status >= 500,
    );
  }

  if (!body || body.ok === false) {
    const err = (body as { ok: false; error?: { code?: string; message?: string; retryable?: boolean } } | null)?.error;
    throw new ProjectExplorerError(
      res.status,
      err?.code ?? "UNKNOWN",
      err?.message ?? `relay api ${res.status}`,
      err?.retryable ?? res.status >= 500,
    );
  }

  return (body as RelaySuccessEnvelope<T>).data;
}

function mapExplorer(data: ServerProjectExplorer): ProjectExplorer {
  return {
    project: {
      projectId: data.project.project_id,
      name: data.project.name,
      status: data.project.status,
    },
    counts: {
      notes: data.counts.notes,
      artifacts: data.counts.artifacts,
      decisions: data.counts.decisions,
      openQuestions: data.counts.open_questions,
      judgmentTraces: data.counts.judgment_traces,
      pendingProposals: data.counts.pending_proposals,
      approvedHeuristics: data.counts.approved_heuristics,
      rejectedProposals: data.counts.rejected_proposals,
      packetSnapshots: data.counts.packet_snapshots,
    },
    latestSnapshot: data.latest_snapshot
      ? {
          snapshotId: data.latest_snapshot.snapshot_id,
          packetKind: data.latest_snapshot.packet_kind,
          target: data.latest_snapshot.target,
          taskSummary: data.latest_snapshot.task_summary || undefined,
          createdAt: data.latest_snapshot.created_at,
          publicReadable: data.latest_snapshot.public_readable,
        }
      : undefined,
    styleMemory: {
      nextProposalId: data.style_memory.next_proposal_id || undefined,
      nextProposalText: data.style_memory.next_proposal_text || undefined,
    },
    recentActivity: (data.recent_activity ?? []).map((item) => ({
      kind: item.kind,
      id: item.id,
      title: item.title,
      createdAt: item.created_at,
    })),
  };
}

export async function getProjectExplorer(
  projectId: string,
  opts?: ExplorerOptions,
): Promise<ProjectExplorer> {
  const res = await relayFetch(`/v1/projects/${encodeURIComponent(projectId)}/explorer`, {
    method: "GET",
    headers: opts?.headers,
    signal: opts?.signal,
  });
  const data = await unwrap<ServerProjectExplorer>(res);
  return mapExplorer(data);
}
