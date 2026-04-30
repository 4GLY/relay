import { cookies } from "next/headers";
import Link from "next/link";
import { redirect } from "next/navigation";

import { RELAY_API_URL, relayFetch, type RelayEnvelope } from "@/lib/api";
import type { AuthMe } from "@/lib/onboarding";
import {
  getProjectGraph,
  ProjectGraphError,
  type ProjectGraph,
  type ProjectGraphNode,
} from "@/lib/project-graph";

export const dynamic = "force-dynamic";

type PageParams = {
  projectId: string;
};

type GraphNodeView = ProjectGraphNode & {
  x: number;
  y: number;
};

async function resolveSession(cookieHeader: string): Promise<AuthMe | null> {
  const res = await relayFetch("/v1/auth/me", {
    method: "GET",
    headers: { cookie: cookieHeader },
    cache: "no-store",
  });
  if (res.status === 401) return null;
  if (!res.ok) return null;
  const body = (await res.json()) as RelayEnvelope<AuthMe>;
  if (!body.ok) return null;
  return body.data;
}

function signInURL(projectId: string) {
  const url = new URL("/v1/auth/github/start", RELAY_API_URL);
  url.searchParams.set("redirect_to", `/projects/${projectId}/graph`);
  return url.toString();
}

export default async function DecisionGraphPage({
  params,
}: {
  params: Promise<PageParams>;
}) {
  const { projectId } = await params;
  const cookieStore = await cookies();
  const cookieHeader = cookieStore.toString();
  const me = await resolveSession(cookieHeader);

  if (!me) {
    return <SignInRequired projectId={projectId} />;
  }

  if (!me.onboarding_complete) {
    redirect("/onboarding");
  }

  let graph: ProjectGraph;
  try {
    graph = await getProjectGraph(projectId, {
      headers: { cookie: cookieHeader },
    });
  } catch (error) {
    return <DecisionGraphError projectId={projectId} error={error} userDisplayName={me.display_name} />;
  }

  return <DecisionGraph graph={graph} userDisplayName={me.display_name} />;
}

function DecisionGraph({
  graph,
  userDisplayName,
}: {
  graph: ProjectGraph;
  userDisplayName?: string;
}) {
  const nodes = layoutNodes(graph.nodes);
  const byId = new Map(nodes.map((node) => [node.id, node]));
  const edges = graph.edges
    .map((edge) => ({ edge, from: byId.get(edge.from), to: byId.get(edge.to) }))
    .filter((item): item is { edge: ProjectGraph["edges"][number]; from: GraphNodeView; to: GraphNodeView } => Boolean(item.from && item.to));
  const selectedPath = selectActivePath(edges);
  const counts = countKinds(graph.nodes);

  return (
    <main style={pageStyle}>
      <header style={topbarStyle}>
        <Link href="/" style={brandStyle}>
          Relay<span style={{ color: "var(--magic-primary-strong)" }}>.</span>
        </Link>
        <span style={userStyle}>{userDisplayName ?? "signed in"}</span>
      </header>

      <section style={heroStyle} aria-labelledby="graph-title">
        <p style={eyebrowStyle}>Project Explorer · Decision Graph</p>
        <div style={heroRowStyle}>
          <div>
            <h1 id="graph-title" style={titleStyle}>
              Decision Graph
            </h1>
            <p style={subtitleStyle}>
              {graph.nodes.length} nodes · {graph.edges.length} edges
            </p>
          </div>
          <nav style={actionsStyle} aria-label="Graph actions">
            <a href={`/projects/${encodeURIComponent(graph.projectId)}`} style={secondaryLinkStyle}>
              Project Explorer
            </a>
            <a href={`/projects/${encodeURIComponent(graph.projectId)}/traces`} style={primaryLinkStyle}>
              Trace Browser
            </a>
          </nav>
        </div>
      </section>

      <section style={summaryGridStyle} aria-label="Graph summary">
        <Metric label="Decisions" value={counts.decision ?? 0} />
        <Metric label="Traces" value={counts.judgment_trace ?? 0} />
        <Metric label="Proposals" value={counts.heuristic_proposal ?? 0} />
        <Metric label="Heuristics" value={counts.approved_heuristic ?? 0} />
        <Metric label="Snapshots" value={counts.packet_snapshot ?? 0} />
      </section>

      {nodes.length > 1 ? (
        <section style={graphShellStyle} aria-label="Decision Graph map">
          <svg viewBox="0 0 960 560" role="img" aria-label="Decision evidence graph" style={svgStyle}>
            {edges.map(({ edge, from, to }) => {
              const active = selectedPath.has(edgeKey(edge.from, edge.to));
              return (
                <line
                  key={`${edge.type}:${edge.from}:${edge.to}`}
                  x1={from.x}
                  y1={from.y}
                  x2={to.x}
                  y2={to.y}
                  stroke={active ? "var(--magic-primary-strong)" : "var(--border-strong)"}
                  strokeWidth={active ? 4 : 2}
                  strokeLinecap="round"
                  opacity={edge.status === "candidate" ? 0.48 : 0.9}
                />
              );
            })}
            {nodes.map((node) => (
              <g key={node.id}>
                <circle
                  cx={node.x}
                  cy={node.y}
                  r={node.kind === "project" ? 34 : 26}
                  fill={nodeFill(node.kind)}
                  stroke={selectedPathNode(selectedPath, node.id) ? "var(--magic-primary-strong)" : "var(--border-strong)"}
                  strokeWidth={selectedPathNode(selectedPath, node.id) ? 4 : 2}
                />
                <text x={node.x} y={node.y + 5} textAnchor="middle" style={nodeGlyphStyle}>
                  {nodeGlyph(node.kind)}
                </text>
              </g>
            ))}
          </svg>
        </section>
      ) : (
        <section style={emptyPanelStyle}>
          <h2 style={emptyTitleStyle}>No graph evidence yet.</h2>
          <p style={quietCopyStyle}>Capture notes, traces, decisions, heuristics, or snapshots to fill the map.</p>
        </section>
      )}

      <section style={nodeListStyle} aria-label="Graph nodes">
        {nodes.map((node) => (
          <article key={node.id} style={nodeCardStyle}>
            <span style={kindStyle}>{formatKind(node.kind)}</span>
            <h2 style={nodeTitleStyle}>{node.title || node.id}</h2>
            <p style={nodeMetaStyle}>{node.workflow || node.packetKind || node.state || node.sourcePath || node.id}</p>
          </article>
        ))}
      </section>
    </main>
  );
}

function Metric({ label, value }: { label: string; value: number }) {
  return (
    <div style={metricStyle}>
      <span style={metricValueStyle}>{value}</span>
      <span style={metricLabelStyle}>{label}</span>
    </div>
  );
}

function SignInRequired({ projectId }: { projectId: string }) {
  return (
    <main style={emptyPageStyle}>
      <p style={eyebrowStyle}>Decision Graph</p>
      <h1 style={emptyTitleStyle}>Sign in first</h1>
      <p style={quietCopyStyle}>Decision evidence is private to the project workspace.</p>
      <a href={signInURL(projectId)} style={primaryLinkStyle}>
        Continue with GitHub
      </a>
    </main>
  );
}

function DecisionGraphError({
  projectId,
  error,
  userDisplayName,
}: {
  projectId: string;
  error: unknown;
  userDisplayName?: string;
}) {
  const code = error instanceof ProjectGraphError ? error.code : "UNKNOWN";
  const message = error instanceof Error ? error.message : "Decision Graph failed to load.";
  return (
    <main style={emptyPageStyle}>
      <p style={eyebrowStyle}>Decision Graph · {userDisplayName ?? "signed in"}</p>
      <h1 style={emptyTitleStyle}>Couldn’t open graph</h1>
      <p style={errorBoxStyle}>
        {code}: {message}
      </p>
      <a href={`/projects/${encodeURIComponent(projectId)}/graph`} style={primaryLinkStyle}>
        Retry
      </a>
    </main>
  );
}

function layoutNodes(nodes: ProjectGraphNode[]): GraphNodeView[] {
  const lanes = ["project", "judgment_trace", "heuristic_proposal", "approved_heuristic", "decision", "packet_snapshot"];
  const fallbackLane = lanes.length - 1;
  const grouped = new Map<number, ProjectGraphNode[]>();
  for (const node of nodes) {
    const lane = Math.max(0, lanes.indexOf(node.kind));
    const index = lane === -1 ? fallbackLane : lane;
    grouped.set(index, [...(grouped.get(index) ?? []), node]);
  }
  return nodes.map((node) => {
    const lane = Math.max(0, lanes.indexOf(node.kind));
    const index = lane === -1 ? fallbackLane : lane;
    const peers = grouped.get(index) ?? [];
    const peerIndex = peers.findIndex((item) => item.id === node.id);
    return {
      ...node,
      x: 90 + index * 155,
      y: 90 + (peerIndex + 1) * (390 / (peers.length + 1)),
    };
  });
}

function countKinds(nodes: ProjectGraphNode[]) {
  return nodes.reduce<Record<string, number>>((acc, node) => {
    acc[node.kind] = (acc[node.kind] ?? 0) + 1;
    return acc;
  }, {});
}

function selectActivePath(edges: Array<{ edge: ProjectGraph["edges"][number]; from: GraphNodeView; to: GraphNodeView }>) {
  const selected = new Set<string>();
  for (const { edge } of edges) {
    if (edge.type === "derived_from" && (edge.from.startsWith("psnap_") || edge.from.startsWith("heur_"))) {
      selected.add(edgeKey(edge.from, edge.to));
    }
  }
  return selected;
}

function selectedPathNode(selectedPath: Set<string>, nodeId: string) {
  for (const key of selectedPath) {
    const [from, to] = key.split("->");
    if (from === nodeId || to === nodeId) return true;
  }
  return false;
}

function edgeKey(from: string, to: string) {
  return `${from}->${to}`;
}

function nodeGlyph(kind: string) {
  const glyphs: Record<string, string> = {
    project: "P",
    judgment_trace: "T",
    heuristic_proposal: "Q",
    approved_heuristic: "H",
    decision: "D",
    packet_snapshot: "S",
  };
  return glyphs[kind] ?? "N";
}

function nodeFill(kind: string) {
  if (kind === "project") return "var(--ink)";
  if (kind === "packet_snapshot") return "var(--canvas)";
  if (kind === "approved_heuristic") return "color-mix(in srgb, var(--magic-primary-strong) 14%, var(--canvas-raised))";
  return "var(--canvas-raised)";
}

function formatKind(kind: string) {
  return kind.replaceAll("_", " ");
}

const pageStyle: React.CSSProperties = {
  maxWidth: "1180px",
  margin: "0 auto",
  padding: "0 28px 96px",
};

const topbarStyle: React.CSSProperties = {
  minHeight: "74px",
  display: "flex",
  alignItems: "center",
  justifyContent: "space-between",
  borderBottom: "1px solid var(--border)",
};

const brandStyle: React.CSSProperties = {
  color: "var(--ink)",
  fontFamily: "var(--font-display)",
  fontSize: "30px",
  fontWeight: 700,
  textDecoration: "none",
  fontVariationSettings: '"opsz" 96',
};

const userStyle: React.CSSProperties = {
  color: "var(--muted)",
  fontFamily: "var(--font-mono)",
  fontSize: "13px",
  letterSpacing: "0.16em",
};

const heroStyle: React.CSSProperties = {
  padding: "72px 0 36px",
};

const heroRowStyle: React.CSSProperties = {
  display: "flex",
  gap: "28px",
  alignItems: "flex-end",
  justifyContent: "space-between",
  flexWrap: "wrap",
};

const eyebrowStyle: React.CSSProperties = {
  margin: "0 0 14px",
  color: "var(--muted)",
  fontFamily: "var(--font-mono)",
  fontSize: "12px",
  letterSpacing: "0.16em",
  textTransform: "uppercase",
};

const titleStyle: React.CSSProperties = {
  margin: 0,
  color: "var(--ink)",
  fontFamily: "var(--font-display)",
  fontSize: "clamp(48px, 7vw, 84px)",
  fontWeight: 500,
  lineHeight: 1,
  fontVariationSettings: '"opsz" 144, "SOFT" 50',
};

const subtitleStyle: React.CSSProperties = {
  margin: "14px 0 0",
  color: "var(--ink-muted)",
  fontFamily: "var(--font-display)",
  fontSize: "24px",
  fontStyle: "italic",
  fontVariationSettings: '"opsz" 48',
};

const actionsStyle: React.CSSProperties = {
  display: "flex",
  gap: "12px",
  flexWrap: "wrap",
};

const primaryLinkStyle: React.CSSProperties = {
  display: "inline-flex",
  alignItems: "center",
  justifyContent: "center",
  minHeight: "46px",
  padding: "0 18px",
  borderRadius: "8px",
  background: "var(--ink)",
  color: "var(--canvas)",
  fontWeight: 800,
  textDecoration: "none",
};

const secondaryLinkStyle: React.CSSProperties = {
  ...primaryLinkStyle,
  background: "transparent",
  color: "var(--ink)",
  border: "1px solid var(--border-strong)",
};

const summaryGridStyle: React.CSSProperties = {
  display: "grid",
  gridTemplateColumns: "repeat(auto-fit, minmax(140px, 1fr))",
  gap: "12px",
  marginBottom: "18px",
};

const metricStyle: React.CSSProperties = {
  padding: "16px",
  border: "1px solid var(--border)",
  borderRadius: "8px",
  background: "var(--canvas-raised)",
};

const metricValueStyle: React.CSSProperties = {
  display: "block",
  color: "var(--ink)",
  fontFamily: "var(--font-display)",
  fontSize: "32px",
  lineHeight: 1,
  fontWeight: 500,
};

const metricLabelStyle: React.CSSProperties = {
  display: "block",
  marginTop: "8px",
  color: "var(--muted)",
  fontFamily: "var(--font-mono)",
  fontSize: "11px",
  letterSpacing: "0.14em",
  textTransform: "uppercase",
};

const graphShellStyle: React.CSSProperties = {
  border: "1px solid var(--border)",
  borderRadius: "8px",
  background: "var(--canvas-raised)",
  overflow: "hidden",
};

const svgStyle: React.CSSProperties = {
  display: "block",
  width: "100%",
  minHeight: "420px",
};

const nodeGlyphStyle: React.CSSProperties = {
  fill: "var(--ink)",
  fontFamily: "var(--font-mono)",
  fontSize: "14px",
  fontWeight: 800,
};

const nodeListStyle: React.CSSProperties = {
  display: "grid",
  gridTemplateColumns: "repeat(auto-fit, minmax(240px, 1fr))",
  gap: "12px",
  marginTop: "18px",
};

const nodeCardStyle: React.CSSProperties = {
  padding: "18px",
  border: "1px solid var(--border)",
  borderRadius: "8px",
  background: "var(--canvas-raised)",
};

const kindStyle: React.CSSProperties = {
  color: "var(--magic-primary-strong)",
  fontFamily: "var(--font-mono)",
  fontSize: "11px",
  letterSpacing: "0.14em",
  textTransform: "uppercase",
};

const nodeTitleStyle: React.CSSProperties = {
  margin: "10px 0 8px",
  color: "var(--ink)",
  fontFamily: "var(--font-display)",
  fontSize: "24px",
  fontWeight: 500,
  lineHeight: 1.2,
};

const nodeMetaStyle: React.CSSProperties = {
  margin: 0,
  color: "var(--ink-muted)",
  fontFamily: "var(--font-mono)",
  fontSize: "12px",
  overflowWrap: "anywhere",
};

const quietCopyStyle: React.CSSProperties = {
  margin: 0,
  color: "var(--ink-muted)",
  fontSize: "16px",
  lineHeight: 1.6,
};

const emptyPanelStyle: React.CSSProperties = {
  padding: "48px",
  border: "1px dashed var(--border-strong)",
  borderRadius: "8px",
  textAlign: "center",
};

const emptyPageStyle: React.CSSProperties = {
  maxWidth: "620px",
  margin: "0 auto",
  padding: "120px 32px",
};

const emptyTitleStyle: React.CSSProperties = {
  margin: "0 0 18px",
  fontFamily: "var(--font-display)",
  fontSize: "46px",
  fontWeight: 500,
};

const errorBoxStyle: React.CSSProperties = {
  margin: "0 0 22px",
  padding: "14px",
  border: "1px solid var(--danger)",
  borderRadius: "8px",
  color: "var(--danger)",
  background: "color-mix(in srgb, var(--danger) 8%, transparent)",
  fontFamily: "var(--font-mono)",
  fontSize: "13px",
};
