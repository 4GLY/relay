import { relayFetch, type RelayEnvelope, type RelaySuccessEnvelope } from "./api";

export type ProjectGraphNode = {
  id: string;
  kind: string;
  title?: string;
  source?: string;
  sourcePath?: string;
  trustLevel?: string;
  workflow?: string;
  artifactType?: string;
  state?: string;
  packetKind?: string;
  target?: string;
  publicReadable?: boolean;
  createdAt?: string;
};

export type ProjectGraphEdge = {
  type: string;
  from: string;
  to: string;
  status?: string;
  score?: number;
  whyIncluded?: string;
};

export type ProjectGraph = {
  projectId: string;
  nodes: ProjectGraphNode[];
  edges: ProjectGraphEdge[];
};

type ServerProjectGraphNode = {
  id: string;
  kind: string;
  title?: string;
  source?: string;
  source_path?: string;
  trust_level?: string;
  workflow?: string;
  artifact_type?: string;
  state?: string;
  packet_kind?: string;
  target?: string;
  public_readable?: boolean;
  created_at?: string;
};

type ServerProjectGraphEdge = {
  type: string;
  from: string;
  to: string;
  status?: string;
  score?: number;
  why_included?: string;
};

type ServerProjectGraph = {
  project_id: string;
  nodes?: ServerProjectGraphNode[];
  edges?: ServerProjectGraphEdge[];
};

type ProjectGraphOptions = {
  headers?: HeadersInit;
  signal?: AbortSignal;
};

export class ProjectGraphError extends Error {
  readonly status: number;
  readonly code: string;
  readonly retryable: boolean;

  constructor(status: number, code: string, message: string, retryable = false) {
    super(message);
    this.name = "ProjectGraphError";
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
    throw new ProjectGraphError(
      res.status,
      "INVALID_RESPONSE",
      `non-JSON response (status ${res.status})`,
      res.status >= 500,
    );
  }

  if (!body || body.ok === false) {
    const err = (body as { ok: false; error?: { code?: string; message?: string; retryable?: boolean } } | null)?.error;
    throw new ProjectGraphError(
      res.status,
      err?.code ?? "UNKNOWN",
      err?.message ?? `relay api ${res.status}`,
      err?.retryable ?? res.status >= 500,
    );
  }

  return (body as RelaySuccessEnvelope<T>).data;
}

function mapGraph(data: ServerProjectGraph): ProjectGraph {
  return {
    projectId: data.project_id,
    nodes: (data.nodes ?? []).map((node) => ({
      id: node.id,
      kind: node.kind,
      title: node.title || undefined,
      source: node.source || undefined,
      sourcePath: node.source_path || undefined,
      trustLevel: node.trust_level || undefined,
      workflow: node.workflow || undefined,
      artifactType: node.artifact_type || undefined,
      state: node.state || undefined,
      packetKind: node.packet_kind || undefined,
      target: node.target || undefined,
      publicReadable: node.public_readable,
      createdAt: node.created_at || undefined,
    })),
    edges: (data.edges ?? []).map((edge) => ({
      type: edge.type,
      from: edge.from,
      to: edge.to,
      status: edge.status || undefined,
      score: edge.score,
      whyIncluded: edge.why_included || undefined,
    })),
  };
}

export async function getProjectGraph(
  projectId: string,
  opts?: ProjectGraphOptions,
): Promise<ProjectGraph> {
  const res = await relayFetch(`/v1/projects/${encodeURIComponent(projectId)}/graph`, {
    method: "GET",
    headers: opts?.headers,
    signal: opts?.signal,
  });
  const data = await unwrap<ServerProjectGraph>(res);
  return mapGraph(data);
}
