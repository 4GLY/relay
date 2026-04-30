import { cookies } from "next/headers";
import { redirect } from "next/navigation";

import { RELAY_API_URL, relayFetch, type RelayEnvelope } from "@/lib/api";
import type { AuthMe } from "@/lib/onboarding";
import {
  getProjectExplorer,
  ProjectExplorerError,
  type ProjectExplorer,
} from "@/lib/project-explorer";
import { RelayAppShell, RelayTopRail } from "@/components/relay-app-shell";

export const dynamic = "force-dynamic";

type PageParams = {
  projectId: string;
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
  url.searchParams.set("redirect_to", `/projects/${projectId}`);
  return url.toString();
}

export default async function ProjectExplorerPage({
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

  let explorer: ProjectExplorer;
  try {
    explorer = await getProjectExplorer(projectId, {
      headers: { cookie: cookieHeader },
    });
  } catch (error) {
    return <ExplorerError projectId={projectId} error={error} userDisplayName={me.display_name} />;
  }

  return <Explorer explorer={explorer} userDisplayName={me.display_name} />;
}

function Explorer({
  explorer,
  userDisplayName,
}: {
  explorer: ProjectExplorer;
  userDisplayName?: string;
}) {
  const c = explorer.counts;
  const memoryReady = c.pendingProposals > 0
    ? `${c.pendingProposals} proposal${c.pendingProposals === 1 ? "" : "s"} waiting`
    : `${c.approvedHeuristics} swan${c.approvedHeuristics === 1 ? "" : "s"} minted`;
  const projectHref = `/projects/${encodeURIComponent(explorer.project.projectId)}`;

  return (
    <RelayAppShell
      activeStep="Face"
      userLabel={userDisplayName}
      projectHref={projectHref}
      railItems={projectRailItems(explorer)}
      inspector={<ProjectInspector explorer={explorer} />}
    >
      <section style={heroStyle} aria-labelledby="project-title">
        <p style={eyebrowStyle}>Project Explorer · {explorer.project.status}</p>
        <div style={heroRowStyle}>
          <div>
            <h1 id="project-title" style={titleStyle}>
              {explorer.project.name}
            </h1>
            <p style={subtitleStyle}>{memoryReady}</p>
          </div>
          <nav style={actionsStyle} aria-label="Project actions">
            <a href={`/style-memory?project=${encodeURIComponent(explorer.project.projectId)}`} style={primaryLinkStyle}>
              Style Memory
            </a>
            <a href={`/projects/${encodeURIComponent(explorer.project.projectId)}/traces`} style={secondaryLinkStyle}>
              Trace Browser
            </a>
            <a href={`/projects/${encodeURIComponent(explorer.project.projectId)}/graph`} style={secondaryLinkStyle}>
              Decision Graph
            </a>
            <a href={`/projects/${encodeURIComponent(explorer.project.projectId)}/packet-builder`} style={secondaryLinkStyle}>
              Packet Builder
            </a>
            <a href="/settings/providers" style={secondaryLinkStyle}>
              Provider Settings
            </a>
            <a href="/settings/api-keys" style={secondaryLinkStyle}>
              API Key Settings
            </a>
          </nav>
        </div>
      </section>

      <section style={metricsGridStyle} aria-label="Project summary">
        <Metric label="Notes" value={c.notes} />
        <Metric label="Decisions" value={c.decisions} />
        <Metric label="Snapshots" value={c.packetSnapshots} />
      </section>

      <details style={inspectorStyle}>
        <summary style={inspectorSummaryStyle}>Workspace inspector — detailed counts</summary>
        <dl style={inspectorGridStyle} aria-label="Workspace inspector counts">
          <InspectorMetric label="Artifacts" value={c.artifacts} />
          <InspectorMetric label="Questions" value={c.openQuestions} />
          <InspectorMetric label="Traces" value={c.judgmentTraces} />
          <InspectorMetric label="Pending proposals" value={c.pendingProposals} />
          <InspectorMetric label="Approved heuristics" value={c.approvedHeuristics} />
          <InspectorMetric label="Rejected proposals" value={c.rejectedProposals} />
        </dl>
      </details>

      <section style={workGridStyle}>
        <Panel title="Style Memory" kicker={`${c.pendingProposals} pending`}>
          {explorer.styleMemory.nextProposalText ? (
            <>
              <p style={proposalTextStyle}>{explorer.styleMemory.nextProposalText}</p>
              <a
                href={`/style-memory?project=${encodeURIComponent(explorer.project.projectId)}`}
                style={inlineActionStyle}
              >
                Review proposal
              </a>
            </>
          ) : (
            <p style={quietCopyStyle}>No pending proposals.</p>
          )}
        </Panel>

        <Panel title="Latest Snapshot" kicker={`${c.packetSnapshots} total`}>
          {explorer.latestSnapshot ? (
            <>
              <Snapshot snapshot={explorer.latestSnapshot} />
              <a
                href={`/projects/${encodeURIComponent(explorer.project.projectId)}/packet-builder`}
                style={inlineActionStyle}
              >
                Open Packet Builder
              </a>
            </>
          ) : (
            <p style={quietCopyStyle}>No packet snapshots yet.</p>
          )}
        </Panel>

        <Panel title="Recent Activity" kicker={`${explorer.recentActivity.length} latest`}>
          {explorer.recentActivity.length > 0 ? (
            <ol style={activityListStyle}>
              {explorer.recentActivity.map((item) => (
                <li key={`${item.kind}:${item.id}`} style={activityItemStyle}>
                  <span style={activityKindStyle}>{formatKind(item.kind)}</span>
                  {item.kind === "judgment_trace" ? (
                    <a
                      href={traceURL(explorer.project.projectId, item.id)}
                      style={activityLinkStyle}
                    >
                      {item.title}
                    </a>
                  ) : (
                    <span style={activityTitleStyle}>{item.title}</span>
                  )}
                  <time dateTime={item.createdAt} style={activityTimeStyle}>
                    {formatDate(item.createdAt)}
                  </time>
                </li>
              ))}
            </ol>
          ) : (
            <p style={quietCopyStyle}>No recent activity.</p>
          )}
        </Panel>
      </section>
    </RelayAppShell>
  );
}

function projectRailItems(explorer: ProjectExplorer) {
  const projectId = encodeURIComponent(explorer.project.projectId);
  const c = explorer.counts;
  return [
    {
      href: `/projects/${projectId}`,
      label: explorer.project.name,
      glyph: "active" as const,
      active: true,
      ducklings: c.pendingProposals,
      swans: c.approvedHeuristics,
    },
    {
      href: `/style-memory?project=${projectId}`,
      label: "Style Memory",
      glyph: c.pendingProposals > 0 ? ("pending" as const) : ("snapshot" as const),
      ducklings: c.pendingProposals,
      swans: c.approvedHeuristics,
    },
    {
      href: `/projects/${projectId}/traces`,
      label: "Judgment Traces",
      glyph: c.judgmentTraces > 0 ? ("snapshot" as const) : ("empty" as const),
      swans: c.judgmentTraces,
    },
    {
      href: `/projects/${projectId}/graph`,
      label: "Decision Graph",
      glyph: c.decisions > 0 ? ("snapshot" as const) : ("empty" as const),
      swans: c.decisions,
    },
    {
      href: `/projects/${projectId}/packet-builder`,
      label: "Packet Builder",
      glyph: c.packetSnapshots > 0 ? ("snapshot" as const) : ("empty" as const),
      swans: c.packetSnapshots,
    },
    {
      href: "/settings/providers",
      label: "Provider settings",
      glyph: "empty" as const,
    },
    {
      href: "/settings/api-keys",
      label: "API key settings",
      glyph: "empty" as const,
    },
  ];
}

function ProjectInspector({ explorer }: { explorer: ProjectExplorer }) {
  const c = explorer.counts;
  return (
    <div>
      <p style={eyebrowStyle}>Current Transform</p>
      <h2 style={inspectorTitleStyle}>Face → Project workspace</h2>
      <section className="relay-card" style={{ padding: "16px", marginBottom: "22px" }}>
        <p style={{ ...metaLabelStyle, margin: "0 0 12px" }}>Scope Matrix</p>
        <dl style={inspectorRowsStyle}>
          <InspectorMetric label="Status" value={explorer.project.status} />
          <InspectorMetric label="Artifacts" value={c.artifacts} />
          <InspectorMetric label="Questions" value={c.openQuestions} />
          <InspectorMetric label="Traces" value={c.judgmentTraces} />
          <InspectorMetric label="Pending proposals" value={c.pendingProposals} />
          <InspectorMetric label="Approved heuristics" value={c.approvedHeuristics} />
          <InspectorMetric label="Rejected proposals" value={c.rejectedProposals} />
        </dl>
      </section>
      <section style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: "10px" }}>
        <Metric label="Notes" value={c.notes} />
        <Metric label="Decisions" value={c.decisions} />
        <Metric label="Snapshots" value={c.packetSnapshots} />
        <Metric label="Traces" value={c.judgmentTraces} />
      </section>
    </div>
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

function InspectorMetric({ label, value }: { label: string; value: number | string }) {
  return (
    <div style={inspectorMetricStyle}>
      <dt style={metaLabelStyle}>{label}</dt>
      <dd style={metaValueStyle}>{value}</dd>
    </div>
  );
}

function Panel({
  title,
  kicker,
  children,
}: {
  title: string;
  kicker: string;
  children: React.ReactNode;
}) {
  return (
    <section style={panelStyle}>
      <div style={panelHeaderStyle}>
        <h2 style={panelTitleStyle}>{title}</h2>
        <span style={panelKickerStyle}>{kicker}</span>
      </div>
      {children}
    </section>
  );
}

function Snapshot({ snapshot }: { snapshot: NonNullable<ProjectExplorer["latestSnapshot"]> }) {
  return (
    <div>
      <p style={snapshotTitleStyle}>{snapshot.taskSummary || snapshot.target}</p>
      <dl style={snapshotMetaStyle}>
        <div>
          <dt style={metaLabelStyle}>Kind</dt>
          <dd style={metaValueStyle}>{snapshot.packetKind}</dd>
        </div>
        <div>
          <dt style={metaLabelStyle}>Visibility</dt>
          <dd style={metaValueStyle}>{snapshot.publicReadable ? "public" : "private"}</dd>
        </div>
        <div>
          <dt style={metaLabelStyle}>Created</dt>
          <dd style={metaValueStyle}>{formatDate(snapshot.createdAt)}</dd>
        </div>
      </dl>
      {snapshot.publicReadable && snapshot.publicToken ? (
        <a href={`/p/${encodeURIComponent(snapshot.publicToken)}`} style={inlineActionStyle}>
          Open public snapshot
        </a>
      ) : null}
    </div>
  );
}

function SignInRequired({ projectId }: { projectId: string }) {
  return (
    <>
      <RelayTopRail activeStep="Face" userLabel="signed out" />
      <main style={emptyPageStyle}>
        <p style={eyebrowStyle}>Project Explorer</p>
        <h1 style={emptyTitleStyle}>Sign in first</h1>
        <p style={quietCopyStyle}>Project workspaces are private.</p>
        <a href={signInURL(projectId)} style={primaryLinkStyle}>
          Continue with GitHub
        </a>
      </main>
    </>
  );
}

function ExplorerError({
  projectId,
  error,
  userDisplayName,
}: {
  projectId: string;
  error: unknown;
  userDisplayName?: string;
}) {
  const code = error instanceof ProjectExplorerError ? error.code : "UNKNOWN";
  const message = error instanceof Error ? error.message : "Project Explorer failed to load.";
  return (
    <>
      <RelayTopRail activeStep="Face" userLabel={userDisplayName} />
      <main style={emptyPageStyle}>
        <p style={eyebrowStyle}>Project Explorer · {userDisplayName ?? "signed in"}</p>
        <h1 style={emptyTitleStyle}>Couldn’t open this project</h1>
        <p style={errorBoxStyle}>
          {code}: {message}
        </p>
        <a href={`/projects/${encodeURIComponent(projectId)}`} style={primaryLinkStyle}>
          Retry
        </a>
      </main>
    </>
  );
}

function formatDate(value: string) {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return new Intl.DateTimeFormat("en", {
    month: "short",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  }).format(date);
}

function formatKind(kind: string) {
  return kind.replaceAll("_", " ");
}

function traceURL(projectId: string, traceId: string) {
  return `/projects/${encodeURIComponent(projectId)}/traces?trace=${encodeURIComponent(traceId)}`;
}

const heroStyle: React.CSSProperties = {
  padding: "24px 0 36px",
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
  overflowWrap: "anywhere",
  wordBreak: "break-word",
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

const metricsGridStyle: React.CSSProperties = {
  display: "grid",
  gridTemplateColumns: "repeat(auto-fit, minmax(180px, 1fr))",
  gap: "12px",
  marginBottom: "12px",
};

const metricStyle: React.CSSProperties = {
  padding: "18px",
  border: "1px solid var(--border)",
  borderRadius: "8px",
  background: "var(--canvas-raised)",
};

const metricValueStyle: React.CSSProperties = {
  display: "block",
  color: "var(--ink)",
  fontFamily: "var(--font-display)",
  fontSize: "36px",
  lineHeight: 1,
  fontWeight: 500,
};

const metricLabelStyle: React.CSSProperties = {
  display: "block",
  marginTop: "10px",
  color: "var(--muted)",
  fontFamily: "var(--font-mono)",
  fontSize: "11px",
  letterSpacing: "0.14em",
  textTransform: "uppercase",
};

const workGridStyle: React.CSSProperties = {
  display: "grid",
  gridTemplateColumns: "repeat(auto-fit, minmax(280px, 1fr))",
  gap: "16px",
  alignItems: "stretch",
  marginTop: "24px",
};

const inspectorStyle: React.CSSProperties = {
  marginBottom: "28px",
  border: "1px solid var(--border)",
  borderRadius: "8px",
  background: "var(--canvas-raised)",
};

const inspectorSummaryStyle: React.CSSProperties = {
  cursor: "pointer",
  padding: "14px 16px",
  color: "var(--ink)",
  fontFamily: "var(--font-mono)",
  fontSize: "12px",
  letterSpacing: "0.14em",
  textTransform: "uppercase",
};

const inspectorGridStyle: React.CSSProperties = {
  display: "grid",
  gridTemplateColumns: "repeat(auto-fit, minmax(160px, 1fr))",
  gap: "12px",
  margin: 0,
  padding: "0 16px 16px",
};

const inspectorMetricStyle: React.CSSProperties = {
  minWidth: 0,
  padding: "12px",
  border: "1px solid var(--border)",
  borderRadius: "8px",
};

const inspectorRowsStyle: React.CSSProperties = {
  display: "grid",
  gap: "10px",
  margin: 0,
};

const inspectorTitleStyle: React.CSSProperties = {
  margin: "0 0 18px",
  fontFamily: "var(--font-display)",
  fontSize: "22px",
  fontWeight: 500,
  lineHeight: 1.15,
};

const panelStyle: React.CSSProperties = {
  minHeight: "260px",
  padding: "24px",
  border: "1px solid var(--border)",
  borderRadius: "8px",
  background: "var(--canvas-raised)",
};

const panelHeaderStyle: React.CSSProperties = {
  display: "flex",
  justifyContent: "space-between",
  gap: "16px",
  marginBottom: "22px",
};

const panelTitleStyle: React.CSSProperties = {
  margin: 0,
  fontFamily: "var(--font-display)",
  fontSize: "28px",
  fontWeight: 500,
};

const panelKickerStyle: React.CSSProperties = {
  color: "var(--muted)",
  fontFamily: "var(--font-mono)",
  fontSize: "11px",
  letterSpacing: "0.14em",
  textTransform: "uppercase",
  whiteSpace: "nowrap",
};

const proposalTextStyle: React.CSSProperties = {
  margin: "0 0 22px",
  color: "var(--ink)",
  fontFamily: "var(--font-mono)",
  fontSize: "17px",
  lineHeight: 1.6,
};

const inlineActionStyle: React.CSSProperties = {
  color: "var(--magic-primary-strong)",
  fontWeight: 800,
  textDecoration: "none",
};

const quietCopyStyle: React.CSSProperties = {
  margin: 0,
  color: "var(--ink-muted)",
  fontSize: "16px",
  lineHeight: 1.6,
};

const snapshotTitleStyle: React.CSSProperties = {
  margin: "0 0 22px",
  color: "var(--ink)",
  fontFamily: "var(--font-display)",
  fontSize: "24px",
  fontStyle: "italic",
  lineHeight: 1.35,
};

const snapshotMetaStyle: React.CSSProperties = {
  display: "grid",
  gap: "12px",
  margin: 0,
};

const metaLabelStyle: React.CSSProperties = {
  color: "var(--muted)",
  fontFamily: "var(--font-mono)",
  fontSize: "11px",
  letterSpacing: "0.14em",
  textTransform: "uppercase",
};

const metaValueStyle: React.CSSProperties = {
  margin: "2px 0 0",
  color: "var(--ink)",
  fontWeight: 800,
};

const activityListStyle: React.CSSProperties = {
  display: "grid",
  gap: "14px",
  margin: 0,
  padding: 0,
  listStyle: "none",
};

const activityItemStyle: React.CSSProperties = {
  display: "grid",
  gridTemplateColumns: "minmax(0, 1fr)",
  gap: "4px",
  paddingBottom: "14px",
  borderBottom: "1px solid var(--border)",
};

const activityKindStyle: React.CSSProperties = {
  color: "var(--magic-primary-strong)",
  fontFamily: "var(--font-mono)",
  fontSize: "11px",
  letterSpacing: "0.14em",
  textTransform: "uppercase",
};

const activityTitleStyle: React.CSSProperties = {
  color: "var(--ink)",
  fontSize: "15px",
  fontWeight: 800,
};

const activityLinkStyle: React.CSSProperties = {
  ...activityTitleStyle,
  textDecoration: "none",
};

const activityTimeStyle: React.CSSProperties = {
  color: "var(--muted)",
  fontSize: "13px",
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
