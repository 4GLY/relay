import { cookies } from "next/headers";
import { getTranslations } from "next-intl/server";
import { redirect } from "next/navigation";

import {
  RelayAppShell,
  RelayCard,
  RelayEmptyState,
  RelayFeedback,
  RelayLinkButton,
  RelayMetricTile,
  RelayPageHead,
} from "@/components/relay";
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

type ProductT = Awaited<ReturnType<typeof getTranslations>>;

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
  const t = await getTranslations("DecisionGraph");
  const me = await resolveSession(cookieHeader);

  if (!me) {
    return <SignInRequired projectId={projectId} t={t} />;
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
    return <DecisionGraphError projectId={projectId} error={error} userDisplayName={me.display_name} t={t} />;
  }

  return <DecisionGraph graph={graph} userDisplayName={me.display_name} t={t} />;
}

function DecisionGraph({
  graph,
  userDisplayName,
  t,
}: {
  graph: ProjectGraph;
  userDisplayName?: string;
  t: ProductT;
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
      railItems={projectRailItems(graph.projectId, "graph", t)}
    >
      <RelayPageHead
        eyebrow={t("eyebrow", { projectId: graph.projectId })}
        title={t("title")}
        titleId="graph-title"
        copy={
          nodes.length > 1
            ? t("copyReady")
            : t("copyEmpty")
        }
        actions={
          <>
            <button type="button" className="relay-action" data-variant="secondary">
              {t("actions.relayout")}
            </button>
            <RelayLinkButton
              href={`/projects/${encodeURIComponent(graph.projectId)}/packet-builder`}
              variant="primary"
            >
              {t("actions.composeHandoff")}
            </RelayLinkButton>
          </>
        }
      />

      <section className="relay-summary-grid" aria-label={t("labels.graphSummary")}>
        <RelayMetricTile label={t("labels.decisions")} value={counts.decision ?? 0} />
        <RelayMetricTile label={t("labels.traces")} value={counts.judgment_trace ?? 0} />
        <RelayMetricTile label={t("labels.proposals")} value={counts.heuristic_proposal ?? 0} />
        <RelayMetricTile label={t("labels.heuristics")} value={counts.approved_heuristic ?? 0} />
        <RelayMetricTile label={t("labels.snapshots")} value={counts.packet_snapshot ?? 0} />
      </section>

      {nodes.length > 1 ? (
        <RelayCard className="relay-graph-shell" aria-label={t("labels.decisionGraphMap")}>
          <svg viewBox="0 0 960 540" role="img" aria-label={t("labels.decisionEvidenceGraph")} className="relay-graph-svg">
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
                  <text
                    x={node.x}
                    y={node.y - 2}
                    textAnchor="middle"
                    className="relay-graph-node-kind"
                    fill={swatch.text}
                  >
                    {nodeGlyph(node.kind)} {nodeKindLabel(node.kind, t)}
                  </text>
                  <text
                    x={node.x}
                    y={node.y + 13}
                    textAnchor="middle"
                    className="relay-graph-node-label"
                    fill={swatch.text}
                  >
                    {compactNodeLabel(node)}
                  </text>
                </g>
              );
            })}
          </svg>
          <div className="relay-graph-legend">
            <span>● {t("legend.project")}</span>
            <span>◇ {t("legend.trace")}</span>
            <span>◈ {t("legend.swan")}</span>
            <span>△ {t("legend.duckling")}</span>
            <span>■ {t("legend.snapshot")}</span>
          </div>
        </RelayCard>
      ) : (
        <RelayEmptyState
          title={t("empty.title")}
          copy={t("empty.copy")}
        />
      )}

      <section className="relay-node-list" aria-label={t("labels.graphNodes")}>
        {nodes.map((node) => (
          <RelayCard key={node.id} className="relay-node-card">
            <span className="relay-node-kind">{nodeKindLabel(node.kind, t)}</span>
            <h2 className="relay-node-title">{node.title || node.id}</h2>
            <p className="relay-node-meta">{node.workflow || node.packetKind || node.state || node.sourcePath || node.id}</p>
          </RelayCard>
        ))}
      </section>
    </RelayAppShell>
  );
}

function projectRailItems(projectId: string, active: "traces" | "graph", t: ProductT) {
  const encoded = encodeURIComponent(projectId);
  return [
    { href: `/projects/${encoded}`, label: t("nav.projectExplorer"), glyph: "empty" as const },
    {
      href: `/projects/${encoded}/traces`,
      label: t("nav.traceBrowser"),
      glyph: "active" as const,
      active: active === "traces",
    },
    {
      href: `/projects/${encoded}/graph`,
      label: t("nav.decisionGraph"),
      glyph: "snapshot" as const,
      active: active === "graph",
    },
    { href: `/style-memory?project=${encoded}`, label: t("nav.styleMemory"), glyph: "pending" as const },
    { href: `/projects/${encoded}/packet-builder`, label: t("nav.packetBuilder"), glyph: "snapshot" as const },
    { href: "/settings/providers", label: t("nav.settings"), glyph: "empty" as const },
  ];
}

function SignInRequired({ projectId, t }: { projectId: string; t: ProductT }) {
  return (
    <main className="relay-empty-page">
      <RelayPageHead
        eyebrow={t("signIn.eyebrow")}
        title={t("signIn.title")}
        copy={t("signIn.copy")}
        actions={
          <RelayLinkButton href={signInURL(projectId)} variant="primary">
            {t("actions.continueWithGitHub")}
          </RelayLinkButton>
        }
      />
    </main>
  );
}

function DecisionGraphError({
  projectId,
  error,
  userDisplayName,
  t,
}: {
  projectId: string;
  error: unknown;
  userDisplayName?: string;
  t: ProductT;
}) {
  const code = error instanceof ProjectGraphError ? error.code : "UNKNOWN";
  const message = error instanceof Error ? error.message : t("error.fallbackMessage");
  return (
    <main className="relay-empty-page">
      <RelayPageHead
        eyebrow={t("error.eyebrow", { user: userDisplayName ?? t("error.signedInFallback") })}
        title={t("error.title")}
        actions={
          <RelayLinkButton href={`/projects/${encodeURIComponent(projectId)}/graph`} variant="primary">
            {t("actions.retry")}
          </RelayLinkButton>
        }
      />
      <RelayFeedback role="alert" variant="error">
        {code}: {message}
      </RelayFeedback>
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

function nodeKindLabel(kind: string, t: ProductT) {
  const labels: Record<string, string> = {
    project: t("nodeKinds.project"),
    judgment_trace: t("nodeKinds.judgmentTrace"),
    heuristic_proposal: t("nodeKinds.heuristicProposal"),
    approved_heuristic: t("nodeKinds.approvedHeuristic"),
    decision: t("nodeKinds.decision"),
    packet_snapshot: t("nodeKinds.packetSnapshot"),
    note: t("nodeKinds.note"),
    artifact: t("nodeKinds.artifact"),
    open_question: t("nodeKinds.openQuestion"),
  };
  return labels[kind] ?? kind.replaceAll("_", " ");
}
