import { cookies } from "next/headers";
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
  const me = await resolveSession(cookieHeader);

  if (!me) {
    return <SignInRequired projectId={projectId} />;
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
    return <TraceBrowserError projectId={projectId} error={error} userDisplayName={me.display_name} />;
  }

  return (
    <TraceBrowser
      projectId={projectId}
      traces={traces}
      selectedTraceId={selectedTraceId}
      userDisplayName={me.display_name}
    />
  );
}

function TraceBrowser({
  projectId,
  traces,
  selectedTraceId,
  userDisplayName,
}: {
  projectId: string;
  traces: JudgmentTrace[];
  selectedTraceId?: string;
  userDisplayName?: string;
}) {
  const selectedTrace =
    traces.find((trace) => trace.traceId === selectedTraceId) ?? traces[0];
  const tabs = [
    { label: "All", count: traces.length, active: true },
    { label: "Decisions", count: countByArtifact(traces, ["decision", "style_memory", "heuristic"]) },
    { label: "Open questions", count: countByArtifact(traces, ["question", "open_question"]) },
    { label: "Notes", count: countByArtifact(traces, ["note", "design_doc", "doc"]) },
  ];

  return (
    <RelayAppShell
      activeStep="Dissect"
      userLabel={userDisplayName}
      projectHref={`/projects/${encodeURIComponent(projectId)}`}
      railItems={projectRailItems(projectId, "traces", traces.length)}
    >
      <RelayPageHead
        title={`${traces.length} trace${traces.length === 1 ? "" : "s"} captured.`}
        titleId="trace-title"
        eyebrow={`${projectId} · Judgment Traces`}
        actions={
          <>
            <RelayLinkButton href={`/projects/${encodeURIComponent(projectId)}`} variant="secondary">
              Project Explorer
            </RelayLinkButton>
            <RelayLinkButton href={`/style-memory?project=${encodeURIComponent(projectId)}`} variant="primary">
              Style Memory
            </RelayLinkButton>
          </>
        }
      />

      <RelayTabs aria-label="Trace filters" items={tabs} />

      {selectedTrace ? (
        <RelayCard className="relay-trace-table" aria-label="Judgment trace list">
          <div className="relay-trace-header-row" aria-hidden="true">
            <span>Trace</span>
            <span>Decision</span>
            <span>State</span>
            <span>Captured</span>
            <span>Refs</span>
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
                    <span>{firstNonEmpty(trace.workflow, "workflow")}</span>
                    <span>·</span>
                    <span>{firstNonEmpty(trace.artifactType, "artifact")}</span>
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
                    <ul aria-label={`Source refs for ${trace.traceId}`} className="relay-source-ref-list">
                      {trace.sourceRefs.slice(0, 3).map((sourceRef) => (
                        <li key={sourceRef} className="relay-source-ref">
                          <span className="relay-source-ref-dot" aria-hidden="true" />
                          {sourceRef}
                        </li>
                      ))}
                    </ul>
                  ) : null}
                </div>
                <RelayStatusBadge variant={traceStateVariant(trace)}>{traceState(trace)}</RelayStatusBadge>
                <time dateTime={trace.createdAt} className="relay-trace-time">
                  {formatDate(trace.createdAt)}
                </time>
                <span className="relay-trace-refs">
                  {trace.sourceRefs.length} ref{trace.sourceRefs.length === 1 ? "" : "s"}
                </span>
              </article>
            );
          })}
        </RelayCard>
      ) : (
        <RelayEmptyState
          title="No traces captured yet."
          copy="Capture judgment traces to give Style Memory reviewable evidence."
        />
      )}
    </RelayAppShell>
  );
}

function projectRailItems(projectId: string, active: "traces" | "graph", traceCount = 0) {
  const encoded = encodeURIComponent(projectId);
  return [
    { href: `/projects/${encoded}`, label: "Project Explorer", glyph: "empty" as const },
    {
      href: `/projects/${encoded}/traces`,
      label: "Trace Browser",
      glyph: "active" as const,
      active: active === "traces",
      ducklings: active === "traces" ? traceCount : undefined,
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

function countByArtifact(traces: JudgmentTrace[], needles: string[]) {
  return traces.filter((trace) => {
    const haystack = `${trace.artifactType ?? ""} ${trace.workflow ?? ""}`.toLowerCase();
    return needles.some((needle) => haystack.includes(needle));
  }).length;
}

function traceState(trace: JudgmentTrace) {
  if (`${trace.decision} ${trace.rationale ?? ""}`.toLowerCase().includes("reject")) {
    return "Rejected";
  }
  if (trace.rationale && trace.sourceRefs.length > 0) {
    return "Swan";
  }
  return "Duckling";
}

function traceStateVariant(trace: JudgmentTrace): "magic" | "danger" | "pending" {
  const state = traceState(trace);
  if (state === "Swan") return "magic";
  if (state === "Rejected") return "danger";
  return "pending";
}

function truncateId(value: string) {
  if (value.length <= 16) return value;
  return `${value.slice(0, 14)}…`;
}

function SignInRequired({ projectId }: { projectId: string }) {
  return (
    <main className="relay-empty-page">
      <RelayPageHead
        eyebrow="Trace Browser"
        title="Sign in first"
        copy="Trace history is private to the project workspace."
        actions={
          <RelayLinkButton href={signInURL(projectId)} variant="primary">
            Continue with GitHub
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
}: {
  projectId: string;
  error: unknown;
  userDisplayName?: string;
}) {
  const code = error instanceof ProjectTracesError ? error.code : "UNKNOWN";
  const message = error instanceof Error ? error.message : "Trace Browser failed to load.";
  return (
    <main className="relay-empty-page">
      <RelayPageHead
        eyebrow={`Trace Browser · ${userDisplayName ?? "signed in"}`}
        title="Couldn’t open traces"
        actions={
          <RelayLinkButton href={`/projects/${encodeURIComponent(projectId)}/traces`} variant="primary">
            Retry
          </RelayLinkButton>
        }
      />
      <RelayFeedback role="alert" variant="error">
        {code}: {message}
      </RelayFeedback>
    </main>
  );
}

function firstNonEmpty(...values: Array<string | undefined>) {
  for (const value of values) {
    if (value?.trim()) return value;
  }
  return "untitled";
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
