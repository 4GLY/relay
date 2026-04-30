import { cookies } from "next/headers";
import Link from "next/link";
import { redirect } from "next/navigation";
import type { CSSProperties } from "react";

import { RelayAppShell } from "@/components/relay-app-shell";
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
      <section style={pageHeadStyle} aria-labelledby="trace-title">
        <div>
          <p style={eyebrowStyle}>{projectId} · Judgment Traces</p>
          <h1 id="trace-title" style={titleStyle}>
            {traces.length} trace{traces.length === 1 ? "" : "s"} captured.
          </h1>
        </div>
        <nav aria-label="Trace actions" style={actionsStyle}>
          <Link href={`/projects/${encodeURIComponent(projectId)}`} style={secondaryLinkStyle}>
            Project Explorer
          </Link>
          <Link href={`/style-memory?project=${encodeURIComponent(projectId)}`} style={primaryLinkStyle}>
            Style Memory
          </Link>
        </nav>
      </section>

      <nav aria-label="Trace filters" style={tabsStyle}>
        {tabs.map((tab) => (
          <span
            key={tab.label}
            aria-current={tab.active ? "page" : undefined}
            style={tab.active ? activeTabStyle : tabStyle}
          >
            {tab.label} · {tab.count}
          </span>
        ))}
      </nav>

      {selectedTrace ? (
        <section className="relay-card relay-trace-table" style={traceTableStyle} aria-label="Judgment trace list">
          <div className="relay-trace-header-row" style={traceHeaderRowStyle} aria-hidden="true">
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
                style={selected ? selectedTraceRowStyle : traceRowStyle}
              >
                <Link
                  href={`/projects/${encodeURIComponent(projectId)}/traces?trace=${encodeURIComponent(trace.traceId)}`}
                  style={traceIdStyle}
                >
                  {truncateId(trace.traceId)}
                </Link>
                <div style={traceDecisionCellStyle}>
                  <h2 style={traceDecisionStyle}>{trace.decision}</h2>
                  {trace.rationale ? <p style={traceRationaleStyle}>{trace.rationale}</p> : null}
                  <div style={traceMetaLineStyle}>
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
                    <ul aria-label={`Source refs for ${trace.traceId}`} style={sourceRefListStyle}>
                      {trace.sourceRefs.slice(0, 3).map((sourceRef) => (
                        <li key={sourceRef} style={sourceRefStyle}>
                          <span style={sourceDotStyle} aria-hidden="true" />
                          {sourceRef}
                        </li>
                      ))}
                    </ul>
                  ) : null}
                </div>
                <span style={stateBadgeStyle(traceState(trace))}>{traceState(trace)}</span>
                <time dateTime={trace.createdAt} style={timeStyle}>
                  {formatDate(trace.createdAt)}
                </time>
                <span style={refsStyle}>
                  {trace.sourceRefs.length} ref{trace.sourceRefs.length === 1 ? "" : "s"}
                </span>
              </article>
            );
          })}
        </section>
      ) : (
        <section style={emptyPanelStyle}>
          <h2 style={emptyTitleStyle}>No traces captured yet.</h2>
          <p style={quietCopyStyle}>Capture judgment traces to give Style Memory reviewable evidence.</p>
        </section>
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

function truncateId(value: string) {
  if (value.length <= 16) return value;
  return `${value.slice(0, 14)}…`;
}

function SignInRequired({ projectId }: { projectId: string }) {
  return (
    <main style={emptyPageStyle}>
      <p style={eyebrowStyle}>Trace Browser</p>
      <h1 style={emptyTitleStyle}>Sign in first</h1>
      <p style={quietCopyStyle}>Trace history is private to the project workspace.</p>
      <a href={signInURL(projectId)} style={primaryLinkStyle}>
        Continue with GitHub
      </a>
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
    <main style={emptyPageStyle}>
      <p style={eyebrowStyle}>Trace Browser · {userDisplayName ?? "signed in"}</p>
      <h1 style={emptyTitleStyle}>Couldn’t open traces</h1>
      <p style={errorBoxStyle}>
        {code}: {message}
      </p>
      <a href={`/projects/${encodeURIComponent(projectId)}/traces`} style={primaryLinkStyle}>
        Retry
      </a>
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
  fontSize: "clamp(40px, 5vw, 68px)",
  fontWeight: 500,
  lineHeight: 1.04,
  fontVariationSettings: '"opsz" 144, "SOFT" 50',
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

const secondaryLinkStyle: CSSProperties = {
  ...primaryLinkStyle,
  background: "transparent",
  color: "var(--ink)",
  border: "1px solid var(--border-strong)",
};

const tabsStyle: CSSProperties = {
  display: "flex",
  gap: "8px",
  flexWrap: "wrap",
  marginBottom: "18px",
};

const tabStyle: CSSProperties = {
  display: "inline-flex",
  alignItems: "center",
  minHeight: "38px",
  padding: "0 14px",
  border: "1px solid transparent",
  borderRadius: "999px",
  color: "var(--muted)",
  fontFamily: "var(--font-mono)",
  fontSize: "11px",
  letterSpacing: "0.14em",
  textTransform: "uppercase",
};

const activeTabStyle: CSSProperties = {
  ...tabStyle,
  borderColor: "var(--border-strong)",
  background: "var(--ink)",
  color: "var(--canvas)",
};

const traceTableStyle: CSSProperties = {
  overflow: "hidden",
};

const traceHeaderRowStyle: CSSProperties = {
  display: "grid",
  gridTemplateColumns: "132px minmax(240px, 1fr) 104px 132px 72px",
  gap: "18px",
  padding: "14px 20px",
  borderBottom: "1px solid var(--border)",
  color: "var(--muted)",
  fontFamily: "var(--font-mono)",
  fontSize: "10px",
  letterSpacing: "0.14em",
  textTransform: "uppercase",
};

const traceRowStyle: CSSProperties = {
  display: "grid",
  gridTemplateColumns: "132px minmax(240px, 1fr) 104px 132px 72px",
  gap: "18px",
  alignItems: "start",
  padding: "18px 20px",
  borderTop: "1px solid var(--border)",
  background: "var(--canvas-raised)",
};

const selectedTraceRowStyle: CSSProperties = {
  ...traceRowStyle,
  borderColor: "var(--magic-primary-strong)",
  boxShadow:
    "inset 4px 0 0 var(--magic-primary-strong), 0 0 0 4px color-mix(in srgb, var(--magic-primary-strong) 18%, transparent)",
};

const traceIdStyle: CSSProperties = {
  color: "var(--muted)",
  fontFamily: "var(--font-mono)",
  fontSize: "11px",
  textDecoration: "none",
  overflowWrap: "anywhere",
};

const traceDecisionCellStyle: CSSProperties = {
  minWidth: 0,
};

const traceDecisionStyle: CSSProperties = {
  margin: 0,
  color: "var(--ink)",
  fontFamily: "var(--font-sans)",
  fontSize: "15px",
  fontWeight: 800,
  lineHeight: 1.35,
};

const traceRationaleStyle: CSSProperties = {
  margin: "6px 0 0",
  color: "var(--ink-muted)",
  fontSize: "13px",
  lineHeight: 1.5,
};

const traceMetaLineStyle: CSSProperties = {
  display: "flex",
  gap: "8px",
  flexWrap: "wrap",
  marginTop: "10px",
  color: "var(--muted)",
  fontFamily: "var(--font-mono)",
  fontSize: "10px",
  letterSpacing: "0.08em",
};

function stateBadgeStyle(state: string): CSSProperties {
  const swan = state === "Swan";
  const rejected = state === "Rejected";
  return {
    justifySelf: "start",
    padding: "4px 10px",
    borderRadius: "6px",
    background: swan
      ? "color-mix(in srgb, var(--magic-primary) 25%, var(--canvas-raised))"
      : rejected
        ? "color-mix(in srgb, var(--danger) 12%, var(--canvas-raised))"
        : "var(--problem)",
    color: swan ? "var(--magic-primary-strong)" : rejected ? "var(--danger)" : "var(--canvas)",
    fontFamily: "var(--font-mono)",
    fontSize: "10px",
    fontWeight: 700,
    letterSpacing: "0.04em",
  };
}

const timeStyle: CSSProperties = {
  color: "var(--muted)",
  fontFamily: "var(--font-mono)",
  fontSize: "11px",
};

const refsStyle: CSSProperties = {
  color: "var(--muted)",
  fontFamily: "var(--font-mono)",
  fontSize: "11px",
  textAlign: "right",
};

const sourceRefListStyle: CSSProperties = {
  display: "flex",
  flexWrap: "wrap",
  gap: "8px",
  margin: "12px 0 0",
  padding: 0,
  listStyle: "none",
};

const sourceRefStyle: CSSProperties = {
  display: "inline-flex",
  alignItems: "center",
  gap: "6px",
  padding: "5px 9px",
  border: "1px solid var(--border)",
  borderRadius: "999px",
  color: "var(--ink-muted)",
  fontFamily: "var(--font-mono)",
  fontSize: "10px",
};

const sourceDotStyle: CSSProperties = {
  width: "6px",
  height: "6px",
  borderRadius: "999px",
  background: "var(--magic-primary-strong)",
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
