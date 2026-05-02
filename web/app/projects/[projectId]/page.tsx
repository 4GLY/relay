import { cookies } from "next/headers";
import { getTranslations } from "next-intl/server";
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
  const t = await getTranslations("ProjectExplorer");
  const me = await resolveSession(cookieHeader);

  if (!me) {
    return <SignInRequired projectId={projectId} t={t} />;
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
    return <ExplorerError projectId={projectId} error={error} userDisplayName={me.display_name} t={t} />;
  }

  return <Explorer explorer={explorer} userDisplayName={me.display_name} t={t} />;
}

function Explorer({
  explorer,
  userDisplayName,
  t,
}: {
  explorer: ProjectExplorer;
  userDisplayName?: string;
  t: ProductT;
}) {
  const c = explorer.counts;
  const memoryReady = c.pendingProposals > 0
    ? t(c.pendingProposals === 1 ? "memoryPendingOne" : "memoryPendingMany", { count: c.pendingProposals })
    : t(c.approvedHeuristics === 1 ? "memoryApprovedOne" : "memoryApprovedMany", { count: c.approvedHeuristics });
  const projectHref = `/projects/${encodeURIComponent(explorer.project.projectId)}`;

  return (
    <RelayAppShell
      activeStep="Face"
      userLabel={userDisplayName}
      projectHref={projectHref}
      railItems={projectRailItems(explorer, t)}
      inspector={<ProjectInspector explorer={explorer} t={t} />}
    >
      <RelayPageHead
        eyebrow={t("eyebrow", { status: explorer.project.status })}
        title={explorer.project.name}
        titleId="project-title"
        copy={memoryReady}
        actions={
          <>
            <RelayLinkButton href={`/style-memory?project=${encodeURIComponent(explorer.project.projectId)}`} variant="primary">
              {t("actions.styleMemory")}
            </RelayLinkButton>
            <RelayLinkButton href={`/projects/${encodeURIComponent(explorer.project.projectId)}/traces`} variant="secondary">
              {t("actions.traceBrowser")}
            </RelayLinkButton>
            <RelayLinkButton href={`/projects/${encodeURIComponent(explorer.project.projectId)}/graph`} variant="secondary">
              {t("actions.decisionGraph")}
            </RelayLinkButton>
            <RelayLinkButton href={`/projects/${encodeURIComponent(explorer.project.projectId)}/packet-builder`} variant="secondary">
              {t("actions.packetBuilder")}
            </RelayLinkButton>
            <RelayLinkButton href="/settings/providers" variant="secondary">
              {t("actions.providerSettings")}
            </RelayLinkButton>
            <RelayLinkButton href="/settings/api-keys" variant="secondary">
              {t("actions.apiKeySettings")}
            </RelayLinkButton>
          </>
        }
      />

      <section className="relay-summary-grid" aria-label={t("labels.projectSummary")}>
        <RelayMetricTile label={t("labels.notes")} value={c.notes} />
        <RelayMetricTile label={t("labels.decisions")} value={c.decisions} />
        <RelayMetricTile label={t("labels.snapshots")} value={c.packetSnapshots} />
      </section>

      <details className="relay-workspace-inspector">
        <summary className="relay-details-summary">{t("inspector.summary")}</summary>
        <RelayMetaGrid className="relay-workspace-inspector-grid" aria-label={t("inspector.counts")}>
          <InspectorMetric label={t("labels.artifacts")} value={c.artifacts} />
          <InspectorMetric label={t("labels.questions")} value={c.openQuestions} />
          <InspectorMetric label={t("labels.traces")} value={c.judgmentTraces} />
          <InspectorMetric label={t("labels.pendingProposals")} value={c.pendingProposals} />
          <InspectorMetric label={t("labels.approvedHeuristics")} value={c.approvedHeuristics} />
          <InspectorMetric label={t("labels.rejectedProposals")} value={c.rejectedProposals} />
        </RelayMetaGrid>
      </details>

      <section className="relay-work-grid">
        <Panel title={t("panels.styleMemory")} kicker={t("panels.pending", { count: c.pendingProposals })}>
          {explorer.styleMemory.nextProposalText ? (
            <>
              <p className="relay-proposal-text">{explorer.styleMemory.nextProposalText}</p>
              <RelayLinkButton
                href={`/style-memory?project=${encodeURIComponent(explorer.project.projectId)}`}
                variant="ghost"
              >
                {t("actions.reviewProposal")}
              </RelayLinkButton>
            </>
          ) : (
            <p className="relay-quiet-copy">{t("empty.noPendingProposals")}</p>
          )}
        </Panel>

        <Panel title={t("panels.latestSnapshot")} kicker={t("panels.total", { count: c.packetSnapshots })}>
          {explorer.latestSnapshot ? (
            <>
              <Snapshot snapshot={explorer.latestSnapshot} t={t} />
              <RelayLinkButton
                href={`/projects/${encodeURIComponent(explorer.project.projectId)}/packet-builder`}
                variant="ghost"
              >
                {t("actions.openPacketBuilder")}
              </RelayLinkButton>
            </>
          ) : (
            <p className="relay-quiet-copy">{t("empty.noPacketSnapshots")}</p>
          )}
        </Panel>

        <Panel title={t("panels.recentActivity")} kicker={t("panels.latest", { count: explorer.recentActivity.length })}>
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
            <p className="relay-quiet-copy">{t("empty.noRecentActivity")}</p>
          )}
        </Panel>
      </section>
    </RelayAppShell>
  );
}

function projectRailItems(explorer: ProjectExplorer, t: ProductT) {
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
      label: t("actions.styleMemory"),
      glyph: c.pendingProposals > 0 ? ("pending" as const) : ("snapshot" as const),
      ducklings: c.pendingProposals,
      swans: c.approvedHeuristics,
    },
    {
      href: `/projects/${projectId}/traces`,
      label: t("nav.judgmentTraces"),
      glyph: c.judgmentTraces > 0 ? ("snapshot" as const) : ("empty" as const),
      swans: c.judgmentTraces,
    },
    {
      href: `/projects/${projectId}/graph`,
      label: t("actions.decisionGraph"),
      glyph: c.decisions > 0 ? ("snapshot" as const) : ("empty" as const),
      swans: c.decisions,
    },
    {
      href: `/projects/${projectId}/packet-builder`,
      label: t("actions.packetBuilder"),
      glyph: c.packetSnapshots > 0 ? ("snapshot" as const) : ("empty" as const),
      swans: c.packetSnapshots,
    },
    {
      href: "/settings/providers",
      label: t("actions.providerSettings"),
      glyph: "empty" as const,
    },
    {
      href: "/settings/api-keys",
      label: t("actions.apiKeySettings"),
      glyph: "empty" as const,
    },
  ];
}

function ProjectInspector({ explorer, t }: { explorer: ProjectExplorer; t: ProductT }) {
  const c = explorer.counts;
  return (
    <div>
      <p className="relay-page-kicker">{t("inspector.currentTransform")}</p>
      <h2 className="relay-inspector-title">{t("inspector.title")}</h2>
      <RelayCard className="relay-inspector-card">
        <p className="relay-meta-label relay-meta-heading">{t("inspector.scopeMatrix")}</p>
        <RelayMetaGrid className="relay-inspector-rows">
          <InspectorMetric label={t("labels.status")} value={explorer.project.status} />
          <InspectorMetric label={t("labels.artifacts")} value={c.artifacts} />
          <InspectorMetric label={t("labels.questions")} value={c.openQuestions} />
          <InspectorMetric label={t("labels.traces")} value={c.judgmentTraces} />
          <InspectorMetric label={t("labels.pendingProposals")} value={c.pendingProposals} />
          <InspectorMetric label={t("labels.approvedHeuristics")} value={c.approvedHeuristics} />
          <InspectorMetric label={t("labels.rejectedProposals")} value={c.rejectedProposals} />
        </RelayMetaGrid>
      </RelayCard>
      <section className="relay-inspector-metrics">
        <RelayMetricTile label={t("labels.notes")} value={c.notes} />
        <RelayMetricTile label={t("labels.decisions")} value={c.decisions} />
        <RelayMetricTile label={t("labels.snapshots")} value={c.packetSnapshots} />
        <RelayMetricTile label={t("labels.traces")} value={c.judgmentTraces} />
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

function Snapshot({ snapshot, t }: { snapshot: NonNullable<ProjectExplorer["latestSnapshot"]>; t: ProductT }) {
  return (
    <div>
      <p className="relay-snapshot-title">{snapshot.taskSummary || snapshot.target}</p>
      <RelayMetaGrid className="relay-snapshot-meta">
        <div>
          <dt className="relay-meta-label">{t("labels.kind")}</dt>
          <dd className="relay-meta-value">{snapshot.packetKind}</dd>
        </div>
        <div>
          <dt className="relay-meta-label">{t("labels.visibility")}</dt>
          <dd className="relay-meta-value">{snapshot.publicReadable ? t("labels.public") : t("labels.private")}</dd>
        </div>
        <div>
          <dt className="relay-meta-label">{t("labels.created")}</dt>
          <dd className="relay-meta-value">{formatDate(snapshot.createdAt)}</dd>
        </div>
      </RelayMetaGrid>
      {snapshot.publicReadable && snapshot.publicToken ? (
        <RelayLinkButton href={`/p/${encodeURIComponent(snapshot.publicToken)}`} variant="ghost">
          {t("actions.openPublicSnapshot")}
        </RelayLinkButton>
      ) : null}
    </div>
  );
}

function SignInRequired({ projectId, t }: { projectId: string; t: ProductT }) {
  return (
    <>
      <RelayTopRail activeStep="Face" userLabel={t("signIn.userLabel")} />
      <main className="relay-empty-page">
        <RelayPageHead
          eyebrow={t("title")}
          title={t("signIn.title")}
          copy={t("signIn.copy")}
          actions={
            <RelayLinkButton href={signInURL(projectId)} variant="primary">
              {t("actions.continueWithGitHub")}
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
  t,
}: {
  projectId: string;
  error: unknown;
  userDisplayName?: string;
  t: ProductT;
}) {
  const code = error instanceof ProjectExplorerError ? error.code : "UNKNOWN";
  const message = error instanceof Error ? error.message : t("error.fallbackMessage");
  return (
    <>
      <RelayTopRail activeStep="Face" userLabel={userDisplayName} />
      <main className="relay-empty-page">
        <RelayPageHead
          eyebrow={t("eyebrow", { status: userDisplayName ?? t("error.signedInFallback") })}
          title={t("error.title")}
          actions={
            <RelayLinkButton href={`/projects/${encodeURIComponent(projectId)}`} variant="primary">
              {t("actions.retry")}
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
