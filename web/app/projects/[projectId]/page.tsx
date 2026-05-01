import { cookies } from "next/headers";
import { redirect } from "next/navigation";

import { RELAY_API_URL, relayFetch, type RelayEnvelope } from "@/lib/api";
import type { AuthMe } from "@/lib/onboarding";
import {
  getProjectExplorer,
  ProjectExplorerError,
  type ProjectExplorer,
} from "@/lib/project-explorer";
import {
  RelayAppShell,
  RelayCard,
  RelayFeedback,
  RelayLinkButton,
  RelayMetaGrid,
  RelayMetricTile,
  RelayPageHead,
  RelayTopRail,
} from "@/components/relay";

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
      <RelayPageHead
        eyebrow={`Project Explorer · ${explorer.project.status}`}
        title={explorer.project.name}
        titleId="project-title"
        copy={memoryReady}
        actions={
          <>
            <RelayLinkButton href={`/style-memory?project=${encodeURIComponent(explorer.project.projectId)}`} variant="primary">
              Style Memory
            </RelayLinkButton>
            <RelayLinkButton href={`/projects/${encodeURIComponent(explorer.project.projectId)}/traces`} variant="secondary">
              Trace Browser
            </RelayLinkButton>
            <RelayLinkButton href={`/projects/${encodeURIComponent(explorer.project.projectId)}/graph`} variant="secondary">
              Decision Graph
            </RelayLinkButton>
            <RelayLinkButton href={`/projects/${encodeURIComponent(explorer.project.projectId)}/packet-builder`} variant="secondary">
              Packet Builder
            </RelayLinkButton>
            <RelayLinkButton href="/settings/providers" variant="secondary">
              Provider Settings
            </RelayLinkButton>
            <RelayLinkButton href="/settings/api-keys" variant="secondary">
              API Key Settings
            </RelayLinkButton>
          </>
        }
      />

      <section className="relay-summary-grid" aria-label="Project summary">
        <RelayMetricTile label="Notes" value={c.notes} />
        <RelayMetricTile label="Decisions" value={c.decisions} />
        <RelayMetricTile label="Snapshots" value={c.packetSnapshots} />
      </section>

      <details className="relay-workspace-inspector">
        <summary className="relay-details-summary">Workspace inspector — detailed counts</summary>
        <RelayMetaGrid className="relay-workspace-inspector-grid" aria-label="Workspace inspector counts">
          <InspectorMetric label="Artifacts" value={c.artifacts} />
          <InspectorMetric label="Questions" value={c.openQuestions} />
          <InspectorMetric label="Traces" value={c.judgmentTraces} />
          <InspectorMetric label="Pending proposals" value={c.pendingProposals} />
          <InspectorMetric label="Approved heuristics" value={c.approvedHeuristics} />
          <InspectorMetric label="Rejected proposals" value={c.rejectedProposals} />
        </RelayMetaGrid>
      </details>

      <section className="relay-work-grid">
        <Panel title="Style Memory" kicker={`${c.pendingProposals} pending`}>
          {explorer.styleMemory.nextProposalText ? (
            <>
              <p className="relay-proposal-text">{explorer.styleMemory.nextProposalText}</p>
              <RelayLinkButton
                href={`/style-memory?project=${encodeURIComponent(explorer.project.projectId)}`}
                variant="ghost"
              >
                Review proposal
              </RelayLinkButton>
            </>
          ) : (
            <p className="relay-quiet-copy">No pending proposals.</p>
          )}
        </Panel>

        <Panel title="Latest Snapshot" kicker={`${c.packetSnapshots} total`}>
          {explorer.latestSnapshot ? (
            <>
              <Snapshot snapshot={explorer.latestSnapshot} />
              <RelayLinkButton
                href={`/projects/${encodeURIComponent(explorer.project.projectId)}/packet-builder`}
                variant="ghost"
              >
                Open Packet Builder
              </RelayLinkButton>
            </>
          ) : (
            <p className="relay-quiet-copy">No packet snapshots yet.</p>
          )}
        </Panel>

        <Panel title="Recent Activity" kicker={`${explorer.recentActivity.length} latest`}>
          {explorer.recentActivity.length > 0 ? (
            <ol className="relay-activity-list">
              {explorer.recentActivity.map((item) => (
                <li key={`${item.kind}:${item.id}`} className="relay-activity-item">
                  <span className="relay-activity-kind">{formatKind(item.kind)}</span>
                  {item.kind === "judgment_trace" ? (
                    <a
                      href={traceURL(explorer.project.projectId, item.id)}
                      className="relay-activity-title"
                    >
                      {item.title}
                    </a>
                  ) : (
                    <span className="relay-activity-title">{item.title}</span>
                  )}
                  <time dateTime={item.createdAt} className="relay-activity-time">
                    {formatDate(item.createdAt)}
                  </time>
                </li>
              ))}
            </ol>
          ) : (
            <p className="relay-quiet-copy">No recent activity.</p>
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
      <p className="relay-page-kicker">Current Transform</p>
      <h2 className="relay-inspector-title">Face → Project workspace</h2>
      <RelayCard className="relay-inspector-card">
        <p className="relay-meta-label relay-meta-heading">Scope Matrix</p>
        <RelayMetaGrid className="relay-inspector-rows">
          <InspectorMetric label="Status" value={explorer.project.status} />
          <InspectorMetric label="Artifacts" value={c.artifacts} />
          <InspectorMetric label="Questions" value={c.openQuestions} />
          <InspectorMetric label="Traces" value={c.judgmentTraces} />
          <InspectorMetric label="Pending proposals" value={c.pendingProposals} />
          <InspectorMetric label="Approved heuristics" value={c.approvedHeuristics} />
          <InspectorMetric label="Rejected proposals" value={c.rejectedProposals} />
        </RelayMetaGrid>
      </RelayCard>
      <section className="relay-inspector-metrics">
        <RelayMetricTile label="Notes" value={c.notes} />
        <RelayMetricTile label="Decisions" value={c.decisions} />
        <RelayMetricTile label="Snapshots" value={c.packetSnapshots} />
        <RelayMetricTile label="Traces" value={c.judgmentTraces} />
      </section>
    </div>
  );
}

function InspectorMetric({ label, value }: { label: string; value: number | string }) {
  return (
    <div className="relay-inspector-metric">
      <dt className="relay-meta-label">{label}</dt>
      <dd className="relay-meta-value">{value}</dd>
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
    <RelayCard className="relay-work-panel">
      <div className="relay-panel-header">
        <h2 className="relay-panel-title">{title}</h2>
        <span className="relay-panel-kicker">{kicker}</span>
      </div>
      {children}
    </RelayCard>
  );
}

function Snapshot({ snapshot }: { snapshot: NonNullable<ProjectExplorer["latestSnapshot"]> }) {
  return (
    <div>
      <p className="relay-snapshot-title">{snapshot.taskSummary || snapshot.target}</p>
      <RelayMetaGrid className="relay-snapshot-meta">
        <div>
          <dt className="relay-meta-label">Kind</dt>
          <dd className="relay-meta-value">{snapshot.packetKind}</dd>
        </div>
        <div>
          <dt className="relay-meta-label">Visibility</dt>
          <dd className="relay-meta-value">{snapshot.publicReadable ? "public" : "private"}</dd>
        </div>
        <div>
          <dt className="relay-meta-label">Created</dt>
          <dd className="relay-meta-value">{formatDate(snapshot.createdAt)}</dd>
        </div>
      </RelayMetaGrid>
      {snapshot.publicReadable && snapshot.publicToken ? (
        <RelayLinkButton href={`/p/${encodeURIComponent(snapshot.publicToken)}`} variant="ghost">
          Open public snapshot
        </RelayLinkButton>
      ) : null}
    </div>
  );
}

function SignInRequired({ projectId }: { projectId: string }) {
  return (
    <>
      <RelayTopRail activeStep="Face" userLabel="signed out" />
      <main className="relay-empty-page">
        <RelayPageHead
          eyebrow="Project Explorer"
          title="Sign in first"
          copy="Project workspaces are private."
          actions={
            <RelayLinkButton href={signInURL(projectId)} variant="primary">
              Continue with GitHub
            </RelayLinkButton>
          }
        />
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
      <main className="relay-empty-page">
        <RelayPageHead
          eyebrow={`Project Explorer · ${userDisplayName ?? "signed in"}`}
          title="Couldn’t open this project"
          actions={
            <RelayLinkButton href={`/projects/${encodeURIComponent(projectId)}`} variant="primary">
              Retry
            </RelayLinkButton>
          }
        />
        <RelayFeedback role="alert" variant="error">
          {code}: {message}
        </RelayFeedback>
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
