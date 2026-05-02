import { cookies } from "next/headers";
import { getLocale, getTranslations } from "next-intl/server";
import Link from "next/link";
import { redirect } from "next/navigation";

import {
  RelayAppShell,
  RelayCard,
  RelayEmptyState,
  RelayFeedback,
  RelayLinkButton,
  RelayPageHead,
  RelayStatusBadge,
  RelayTabs,
} from "@/components/relay";
import { RELAY_API_URL, relayFetch, type RelayEnvelope } from "@/lib/api";
import type { AuthMe } from "@/lib/onboarding";
import {
  getProjectTraces,
  ProjectTracesError,
  type JudgmentTrace,
} from "@/lib/project-traces";

export const dynamic = "force-dynamic";

type PageParams = {
  projectId: string;
};

type PageSearchParams = {
  trace?: string;
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
  url.searchParams.set("redirect_to", `/projects/${projectId}/traces`);
  return url.toString();
}

export default async function TraceBrowserPage({
  params,
  searchParams,
}: {
  params: Promise<PageParams>;
  searchParams?: Promise<PageSearchParams>;
}) {
  const { projectId } = await params;
  const query = await searchParams;
  const selectedTraceId = query?.trace;
  const cookieStore = await cookies();
  const cookieHeader = cookieStore.toString();
  const locale = await getLocale();
  const t = await getTranslations("Traces");
  const me = await resolveSession(cookieHeader);

  if (!me) {
    return <SignInRequired projectId={projectId} t={t} />;
  }

  if (!me.onboarding_complete) {
    redirect("/onboarding");
  }

  let traces: JudgmentTrace[];
  try {
    const result = await getProjectTraces(projectId, {
      headers: { cookie: cookieHeader },
      limit: 50,
    });
    traces = result.items;
  } catch (error) {
    return <TraceBrowserError projectId={projectId} error={error} userDisplayName={me.display_name} t={t} />;
  }

  return (
    <TraceBrowser
      projectId={projectId}
      traces={traces}
      selectedTraceId={selectedTraceId}
      userDisplayName={me.display_name}
      t={t}
      locale={locale}
    />
  );
}

function TraceBrowser({
  projectId,
  traces,
  selectedTraceId,
  userDisplayName,
  t,
  locale,
}: {
  projectId: string;
  traces: JudgmentTrace[];
  selectedTraceId?: string;
  userDisplayName?: string;
  t: ProductT;
  locale: string;
}) {
  const selectedTrace =
    traces.find((trace) => trace.traceId === selectedTraceId) ?? traces[0];
  const tabs = [
    { label: t("tabs.all"), count: traces.length, active: true },
    { label: t("tabs.decisions"), count: countByArtifact(traces, ["decision", "style_memory", "heuristic"]) },
    { label: t("tabs.openQuestions"), count: countByArtifact(traces, ["question", "open_question"]) },
    { label: t("tabs.notes"), count: countByArtifact(traces, ["note", "design_doc", "doc"]) },
  ];

  return (
    <RelayAppShell
      activeStep="Dissect"
      userLabel={userDisplayName}
      projectHref={`/projects/${encodeURIComponent(projectId)}`}
      railItems={projectRailItems(projectId, "traces", t, traces.length)}
    >
      <RelayPageHead
        title={t(traces.length === 1 ? "titleOne" : "titleMany", { count: traces.length })}
        titleId="trace-title"
        eyebrow={t("eyebrow", { projectId })}
        actions={
          <>
            <RelayLinkButton href={`/projects/${encodeURIComponent(projectId)}`} variant="secondary">
              {t("actions.projectExplorer")}
            </RelayLinkButton>
            <RelayLinkButton href={`/style-memory?project=${encodeURIComponent(projectId)}`} variant="primary">
              {t("actions.styleMemory")}
            </RelayLinkButton>
          </>
        }
      />

      <RelayTabs aria-label={t("filtersLabel")} items={tabs} />

      {selectedTrace ? (
        <RelayCard className="relay-trace-table" aria-label={t("listLabel")}>
          <div className="relay-trace-header-row" aria-hidden="true">
            <span>{t("columns.trace")}</span>
            <span>{t("columns.decision")}</span>
            <span>{t("columns.state")}</span>
            <span>{t("columns.captured")}</span>
            <span>{t("columns.refs")}</span>
          </div>
          {traces.map((trace) => {
            const selected = trace.traceId === selectedTrace.traceId;
            return (
              <article
                key={trace.traceId}
                id={trace.traceId}
                className="relay-trace-row"
                data-selected={selected || undefined}
              >
                <Link
                  href={`/projects/${encodeURIComponent(projectId)}/traces?trace=${encodeURIComponent(trace.traceId)}`}
                  className="relay-trace-id"
                >
                  {truncateId(trace.traceId)}
                </Link>
                <div className="relay-trace-decision-cell">
                  <h2 className="relay-trace-decision">{trace.decision}</h2>
                  {trace.rationale ? <p className="relay-trace-rationale">{trace.rationale}</p> : null}
                  <div className="relay-trace-meta-line">
                    <span>{firstNonEmpty(t, trace.workflow, t("labels.workflow"))}</span>
                    <span>·</span>
                    <span>{firstNonEmpty(t, trace.artifactType, t("labels.artifact"))}</span>
                    {trace.taskId ? (
                      <>
                        <span>·</span>
                        <span>{trace.taskId}</span>
                      </>
                    ) : null}
                    {trace.agentId ? (
                      <>
                        <span>·</span>
                        <span>{trace.agentId}</span>
                      </>
                    ) : null}
                  </div>
                  {trace.sourceRefs.length > 0 ? (
                    <ul aria-label={t("sourceRefsLabel", { traceId: trace.traceId })} className="relay-source-ref-list">
                      {trace.sourceRefs.slice(0, 3).map((sourceRef) => (
                        <li key={sourceRef} className="relay-source-ref">
                          <span className="relay-source-ref-dot" aria-hidden="true" />
                          {sourceRef}
                        </li>
                      ))}
                    </ul>
                  ) : null}
                </div>
                <RelayStatusBadge variant={traceStateVariant(trace)}>{traceState(trace, t)}</RelayStatusBadge>
                <time dateTime={trace.createdAt} className="relay-trace-time">
                  {formatDate(trace.createdAt, locale)}
                </time>
                <span className="relay-trace-refs">
                  {trace.sourceRefs.length} {trace.sourceRefs.length === 1 ? t("labels.refSingular") : t("labels.refPlural")}
                </span>
              </article>
            );
          })}
        </RelayCard>
      ) : (
        <RelayEmptyState
          title={t("empty.title")}
          copy={t("empty.copy")}
        />
      )}
    </RelayAppShell>
  );
}

function projectRailItems(projectId: string, active: "traces" | "graph", t: ProductT, traceCount = 0) {
  const encoded = encodeURIComponent(projectId);
  return [
    { href: `/projects/${encoded}`, label: t("nav.projectExplorer"), glyph: "empty" as const },
    {
      href: `/projects/${encoded}/traces`,
      label: t("nav.traceBrowser"),
      glyph: "active" as const,
      active: active === "traces",
      ducklings: active === "traces" ? traceCount : undefined,
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

function countByArtifact(traces: JudgmentTrace[], needles: string[]) {
  return traces.filter((trace) => {
    const haystack = `${trace.artifactType ?? ""} ${trace.workflow ?? ""}`.toLowerCase();
    return needles.some((needle) => haystack.includes(needle));
  }).length;
}

function traceState(trace: JudgmentTrace, t: ProductT) {
  if (`${trace.decision} ${trace.rationale ?? ""}`.toLowerCase().includes("reject")) {
    return t("state.rejected");
  }
  if (trace.rationale && trace.sourceRefs.length > 0) {
    return t("state.swan");
  }
  return t("state.duckling");
}

function traceStateVariant(trace: JudgmentTrace): "magic" | "danger" | "pending" {
  const rejected = `${trace.decision} ${trace.rationale ?? ""}`.toLowerCase().includes("reject");
  if (trace.rationale && trace.sourceRefs.length > 0) return "magic";
  if (rejected) return "danger";
  return "pending";
}

function truncateId(value: string) {
  if (value.length <= 16) return value;
  return `${value.slice(0, 14)}…`;
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

function TraceBrowserError({
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
  const code = error instanceof ProjectTracesError ? error.code : "UNKNOWN";
  const message = error instanceof Error ? error.message : t("error.fallbackMessage");
  return (
    <main className="relay-empty-page">
      <RelayPageHead
        eyebrow={t("error.eyebrow", { user: userDisplayName ?? t("error.signedInFallback") })}
        title={t("error.title")}
        actions={
          <RelayLinkButton href={`/projects/${encodeURIComponent(projectId)}/traces`} variant="primary">
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

function firstNonEmpty(t: ProductT, ...values: Array<string | undefined>) {
  for (const value of values) {
    if (value?.trim()) return value;
  }
  return t("labels.untitled");
}

function formatDate(value: string, locale: string) {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return new Intl.DateTimeFormat(locale, {
    month: "short",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  }).format(date);
}
