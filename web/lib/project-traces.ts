import { relayFetch, type RelayEnvelope, type RelaySuccessEnvelope } from "./api";

export type JudgmentTrace = {
  traceId: string;
  projectId: string;
  taskId?: string;
  agentId?: string;
  workflow?: string;
  artifactType?: string;
  decision: string;
  rationale?: string;
  sourceRefs: string[];
  createdAt: string;
};

export type ProjectTraces = {
  items: JudgmentTrace[];
  nextCursor?: string;
};

type ServerJudgmentTrace = {
  trace_id: string;
  project_id: string;
  task_id?: string;
  agent_id?: string;
  workflow?: string;
  artifact_type?: string;
  decision: string;
  rationale?: string;
  source_refs?: string[];
  created_at: string;
};

type ServerProjectTraces = {
  items?: ServerJudgmentTrace[];
  next_cursor?: string;
};

type ProjectTracesOptions = {
  headers?: HeadersInit;
  signal?: AbortSignal;
  limit?: number;
  cursor?: string;
};

export class ProjectTracesError extends Error {
  readonly status: number;
  readonly code: string;
  readonly retryable: boolean;

  constructor(status: number, code: string, message: string, retryable = false) {
    super(message);
    this.name = "ProjectTracesError";
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
    throw new ProjectTracesError(
      res.status,
      "INVALID_RESPONSE",
      `non-JSON response (status ${res.status})`,
      res.status >= 500,
    );
  }

  if (!body || body.ok === false) {
    const err = (body as { ok: false; error?: { code?: string; message?: string; retryable?: boolean } } | null)?.error;
    throw new ProjectTracesError(
      res.status,
      err?.code ?? "UNKNOWN",
      err?.message ?? `relay api ${res.status}`,
      err?.retryable ?? res.status >= 500,
    );
  }

  return (body as RelaySuccessEnvelope<T>).data;
}

function mapTraces(data: ServerProjectTraces): ProjectTraces {
  return {
    items: (data.items ?? []).map((trace) => ({
      traceId: trace.trace_id,
      projectId: trace.project_id,
      taskId: trace.task_id || undefined,
      agentId: trace.agent_id || undefined,
      workflow: trace.workflow || undefined,
      artifactType: trace.artifact_type || undefined,
      decision: trace.decision,
      rationale: trace.rationale || undefined,
      sourceRefs: trace.source_refs ?? [],
      createdAt: trace.created_at,
    })),
    nextCursor: data.next_cursor || undefined,
  };
}

export async function getProjectTraces(
  projectId: string,
  opts?: ProjectTracesOptions,
): Promise<ProjectTraces> {
  const params = new URLSearchParams();
  if (opts?.limit) params.set("limit", String(opts.limit));
  if (opts?.cursor) params.set("cursor", opts.cursor);
  const query = params.toString();
  const res = await relayFetch(
    `/v1/projects/${encodeURIComponent(projectId)}/judgment-traces${query ? `?${query}` : ""}`,
    {
      method: "GET",
      headers: opts?.headers,
      signal: opts?.signal,
    },
  );
  const data = await unwrap<ServerProjectTraces>(res);
  return mapTraces(data);
}
