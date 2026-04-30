import { cookies } from "next/headers";
import Link from "next/link";
import { redirect } from "next/navigation";
import type { CSSProperties } from "react";

import { RelayAppShell } from "@/components/relay-app-shell";
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
    .filter(
      (
        item,
      ): item is {
        edge: ProjectGraph["edges"][number];
        from: GraphNodeView;
        to: GraphNodeView;
      } => Boolean(item.from && item.to),
    );
  const selectedPath = selectActivePath(edges);
  const counts = countKinds(graph.nodes);

  return (
    <RelayAppShell
      activeStep="Dissect"
      userLabel={userDisplayName}
      projectHref={`/projects/${encodeURIComponent(graph.projectId)}`}
      railItems={projectRailItems(graph.projectId, "graph")}
    >
      <section style={pageHeadStyle} aria-labelledby="graph-title">
        <div>
          <p style={eyebrowStyle}>{graph.projectId} · Decision Graph</p>
          <h1 id="graph-title" style={titleStyle}>
            Decision Graph
          </h1>
          <p style={subtitleStyle}>
            {nodes.length > 1
              ? "How decisions reached swan."
              : "Capture evidence to reveal the path to swan."}
          </p>
        </div>
        <nav style={actionsStyle} aria-label="Graph actions">
          <button type="button" style={secondaryButtonStyle}>
            Re-layout
          </button>
          <Link href={`/projects/${encodeURIComponent(graph.projectId)}/packet-builder`} style={primaryLinkStyle}>
            Compose handoff
          </Link>
        </nav>
      </section>

      <section style={summaryGridStyle} aria-label="Graph summary">
        <Metric label="Decisions" value={counts.decision ?? 0} />
        <Metric label="Traces" value={counts.judgment_trace ?? 0} />
        <Metric label="Proposals" value={counts.heuristic_proposal ?? 0} />
        <Metric label="Heuristics" value={counts.approved_heuristic ?? 0} />
        <Metric label="Snapshots" value={counts.packet_snapshot ?? 0} />
      </section>

      {nodes.length > 1 ? (
        <section className="relay-card" style={graphShellStyle} aria-label="Decision Graph map">
          <svg viewBox="0 0 960 540" role="img" aria-label="Decision evidence graph" style={svgStyle}>
            <defs>
              <pattern id="relay-graph-dot" width="20" height="20" patternUnits="userSpaceOnUse">
                <circle cx="1" cy="1" r="0.75" fill="var(--border-strong)" />
              </pattern>
            </defs>
            <rect width="960" height="540" fill="url(#relay-graph-dot)" opacity="0.45" />
            <g fill="none">
              {edges.map(({ edge, from, to }, index) => {
                const active = selectedPath.has(edgeKey(edge.from, edge.to));
                return (
                  <path
                    key={`${edge.type}:${edge.from}:${edge.to}:${index}`}
                    d={edgePath(from, to)}
                    stroke={active ? "var(--magic-primary-strong)" : "var(--border-strong)"}
                    strokeWidth={active ? 2.4 : 1.5}
                    strokeLinecap="round"
                    opacity={edge.status === "candidate" ? 0.45 : 0.9}
                  />
                );
              })}
            </g>
            {nodes.map((node) => {
              const active = selectedPathNode(selectedPath, node.id);
              const swatch = nodeSwatch(node.kind, active);
              return (
                <g key={node.id}>
                  <rect
                    x={node.x - 76}
                    y={node.y - 21}
                    width="152"
                    height="42"
                    rx="6"
                    fill={swatch.fill}
                    stroke={swatch.stroke}
                    strokeWidth={active ? 1.8 : 1}
                  />
                  <text x={node.x} y={node.y - 2} textAnchor="middle" style={{ ...nodeKindTextStyle, fill: swatch.text }}>
                    {nodeGlyph(node.kind)} {formatKind(node.kind)}
                  </text>
                  <text x={node.x} y={node.y + 13} textAnchor="middle" style={{ ...nodeLabelTextStyle, fill: swatch.text }}>
                    {compactNodeLabel(node)}
                  </text>
                </g>
              );
            })}
          </svg>
          <div style={legendStyle}>
            <span>● Project</span>
            <span>◇ Trace</span>
            <span>◈ Swan</span>
            <span>△ Duckling</span>
            <span>■ Snapshot</span>
          </div>
        </section>
      ) : (
        <section style={emptyPanelStyle}>
          <h2 style={emptyTitleStyle}>No graph evidence yet.</h2>
          <p style={quietCopyStyle}>Capture notes, traces, decisions, heuristics, or snapshots to fill the map.</p>
        </section>
      )}

      <section style={nodeListStyle} aria-label="Graph nodes">
        {nodes.map((node) => (
          <article key={node.id} className="relay-card" style={nodeCardStyle}>
            <span style={kindStyle}>{formatKind(node.kind)}</span>
            <h2 style={nodeTitleStyle}>{node.title || node.id}</h2>
            <p style={nodeMetaStyle}>{node.workflow || node.packetKind || node.state || node.sourcePath || node.id}</p>
          </article>
        ))}
      </section>
    </RelayAppShell>
  );
}

function projectRailItems(projectId: string, active: "traces" | "graph") {
  const encoded = encodeURIComponent(projectId);
  return [
    { href: `/projects/${encoded}`, label: "Project Explorer", glyph: "empty" as const },
    {
      href: `/projects/${encoded}/traces`,
      label: "Trace Browser",
      glyph: "active" as const,
      active: active === "traces",
    },
    {
      href: `/projects/${encoded}/graph`,
      label: "Decision Graph",
      glyph: "snapshot" as const,
      active: active === "graph",
    },
    { href: `/style-memory?project=${encoded}`, label: "Style Memory", glyph: "pending" as const },
    { href: `/projects/${encoded}/packet-builder`, label: "Packet Builder", glyph: "snapshot" as const },
    { href: "/settings/providers", label: "Settings", glyph: "empty" as const },
  ];
}

function Metric({ label, value }: { label: string; value: number }) {
  return (
    <div className="relay-metric">
      <span className="relay-metric-value">{value}</span>
      <span className="relay-metric-label">{label}</span>
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
    const lane = lanes.indexOf(node.kind);
    const index = lane === -1 ? fallbackLane : lane;
    grouped.set(index, [...(grouped.get(index) ?? []), node]);
  }
  return nodes.map((node) => {
    const lane = lanes.indexOf(node.kind);
    const index = lane === -1 ? fallbackLane : lane;
    const peers = grouped.get(index) ?? [];
    const peerIndex = peers.findIndex((item) => item.id === node.id);
    return {
      ...node,
      x: 86 + index * 158,
      y: 76 + (peerIndex + 1) * (388 / (peers.length + 1)),
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

function edgePath(from: GraphNodeView, to: GraphNodeView) {
  const mid = Math.max(32, Math.abs(to.x - from.x) * 0.46);
  return `M ${from.x} ${from.y} C ${from.x + mid} ${from.y}, ${to.x - mid} ${to.y}, ${to.x} ${to.y}`;
}

function edgeKey(from: string, to: string) {
  return `${from}->${to}`;
}

function nodeGlyph(kind: string) {
  const glyphs: Record<string, string> = {
    project: "●",
    judgment_trace: "◇",
    heuristic_proposal: "△",
    approved_heuristic: "◈",
    decision: "◈",
    packet_snapshot: "■",
  };
  return glyphs[kind] ?? "○";
}

function nodeSwatch(kind: string, active: boolean) {
  if (kind === "packet_snapshot") {
    return { fill: "var(--ink)", stroke: "var(--ink)", text: "var(--canvas)" };
  }
  if (kind === "heuristic_proposal") {
    return { fill: "var(--problem)", stroke: "var(--problem)", text: "var(--canvas)" };
  }
  if (kind === "approved_heuristic" || kind === "decision") {
    return {
      fill: "color-mix(in srgb, var(--magic-primary) 65%, var(--canvas-raised))",
      stroke: active ? "var(--magic-primary-strong)" : "var(--border-strong)",
      text: "var(--ink)",
    };
  }
  return {
    fill: "var(--canvas-raised)",
    stroke: active ? "var(--magic-primary-strong)" : "var(--border-strong)",
    text: "var(--ink)",
  };
}

function compactNodeLabel(node: ProjectGraphNode) {
  const label = node.title || node.id;
  if (label.length <= 22) return label;
  return `${label.slice(0, 20)}…`;
}

function formatKind(kind: string) {
  return kind.replaceAll("_", " ");
}

const pageHeadStyle: CSSProperties = {
  display: "flex",
  gap: "28px",
  alignItems: "flex-end",
  justifyContent: "space-between",
  flexWrap: "wrap",
  marginBottom: "24px",
};

const eyebrowStyle: CSSProperties = {
  margin: "0 0 14px",
  color: "var(--muted)",
  fontFamily: "var(--font-mono)",
  fontSize: "12px",
  letterSpacing: "0.16em",
  textTransform: "uppercase",
};

const titleStyle: CSSProperties = {
  margin: 0,
  color: "var(--ink)",
  fontFamily: "var(--font-display)",
  fontSize: "clamp(44px, 5.5vw, 72px)",
  fontWeight: 500,
  lineHeight: 1,
  fontVariationSettings: '"opsz" 144, "SOFT" 50',
};

const subtitleStyle: CSSProperties = {
  margin: "12px 0 0",
  color: "var(--ink-muted)",
  fontFamily: "var(--font-display)",
  fontSize: "24px",
  fontStyle: "italic",
  fontVariationSettings: '"opsz" 48',
};

const actionsStyle: CSSProperties = {
  display: "flex",
  gap: "12px",
  flexWrap: "wrap",
};

const primaryLinkStyle: CSSProperties = {
  display: "inline-flex",
  alignItems: "center",
  justifyContent: "center",
  minHeight: "46px",
  padding: "0 18px",
  borderRadius: "8px",
  background: "var(--ink)",
  color: "var(--canvas)",
  fontFamily: "var(--font-sans)",
  fontWeight: 800,
  textDecoration: "none",
};

const secondaryButtonStyle: CSSProperties = {
  ...primaryLinkStyle,
  background: "transparent",
  color: "var(--ink)",
  border: "1px solid var(--border-strong)",
  cursor: "pointer",
};

const summaryGridStyle: CSSProperties = {
  display: "grid",
  gridTemplateColumns: "repeat(auto-fit, minmax(140px, 1fr))",
  gap: "12px",
  marginBottom: "18px",
};

const graphShellStyle: CSSProperties = {
  position: "relative",
  overflow: "hidden",
  minHeight: "480px",
  background: "var(--canvas-raised)",
};

const svgStyle: CSSProperties = {
  display: "block",
  width: "100%",
  minHeight: "460px",
};

const nodeKindTextStyle: CSSProperties = {
  fontFamily: "var(--font-mono)",
  fontSize: "8.5px",
  letterSpacing: "0.08em",
  textTransform: "uppercase",
};

const nodeLabelTextStyle: CSSProperties = {
  fontFamily: "var(--font-mono)",
  fontSize: "10px",
  fontWeight: 700,
};

const legendStyle: CSSProperties = {
  position: "absolute",
  left: "16px",
  bottom: "14px",
  display: "flex",
  gap: "14px",
  flexWrap: "wrap",
  color: "var(--muted)",
  fontFamily: "var(--font-mono)",
  fontSize: "10px",
  letterSpacing: "0.14em",
  textTransform: "uppercase",
};

const nodeListStyle: CSSProperties = {
  display: "grid",
  gridTemplateColumns: "repeat(auto-fit, minmax(240px, 1fr))",
  gap: "12px",
  marginTop: "18px",
};

const nodeCardStyle: CSSProperties = {
  padding: "18px",
};

const kindStyle: CSSProperties = {
  color: "var(--magic-primary-strong)",
  fontFamily: "var(--font-mono)",
  fontSize: "11px",
  letterSpacing: "0.14em",
  textTransform: "uppercase",
};

const nodeTitleStyle: CSSProperties = {
  margin: "10px 0 8px",
  color: "var(--ink)",
  fontFamily: "var(--font-display)",
  fontSize: "24px",
  fontWeight: 500,
  lineHeight: 1.2,
};

const nodeMetaStyle: CSSProperties = {
  margin: 0,
  color: "var(--ink-muted)",
  fontFamily: "var(--font-mono)",
  fontSize: "12px",
  overflowWrap: "anywhere",
};

const quietCopyStyle: CSSProperties = {
  margin: 0,
  color: "var(--ink-muted)",
  fontSize: "16px",
  lineHeight: 1.6,
};

const emptyPanelStyle: CSSProperties = {
  padding: "48px",
  border: "1px dashed var(--border-strong)",
  borderRadius: "8px",
  textAlign: "center",
};

const emptyPageStyle: CSSProperties = {
  maxWidth: "620px",
  margin: "0 auto",
  padding: "120px 32px",
};

const emptyTitleStyle: CSSProperties = {
  margin: "0 0 18px",
  fontFamily: "var(--font-display)",
  fontSize: "46px",
  fontWeight: 500,
};

const errorBoxStyle: CSSProperties = {
  margin: "0 0 22px",
  padding: "14px",
  border: "1px solid var(--danger)",
  borderRadius: "8px",
  color: "var(--danger)",
  background: "color-mix(in srgb, var(--danger) 8%, transparent)",
  fontFamily: "var(--font-mono)",
  fontSize: "13px",
};
