import { cookies } from "next/headers";
import Link from "next/link";
import { redirect } from "next/navigation";

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
  return (
    <main style={pageStyle}>
      <header style={topbarStyle}>
        <Link href="/" style={brandStyle}>
          Relay<span style={{ color: "var(--magic-primary-strong)" }}>.</span>
        </Link>
        <span style={userStyle}>{userDisplayName ?? "signed in"}</span>
      </header>

      <section style={heroStyle} aria-labelledby="trace-title">
        <p style={eyebrowStyle}>Project Explorer · Judgment Traces</p>
        <div style={heroRowStyle}>
          <div>
            <h1 id="trace-title" style={titleStyle}>
              Trace Browser
            </h1>
            <p style={subtitleStyle}>
              {traces.length} trace{traces.length === 1 ? "" : "s"} captured for this project.
            </p>
          </div>
          <nav aria-label="Trace actions" style={actionsStyle}>
            <a href={`/projects/${encodeURIComponent(projectId)}`} style={secondaryLinkStyle}>
              Project Explorer
            </a>
            <a href={`/style-memory?project=${encodeURIComponent(projectId)}`} style={primaryLinkStyle}>
              Style Memory
            </a>
          </nav>
        </div>
      </section>

      {traces.length > 0 ? (
        <ol style={traceListStyle}>
          {traces.map((trace) => (
            <li
              key={trace.traceId}
              id={trace.traceId}
              style={trace.traceId === selectedTraceId ? selectedTraceCardStyle : traceCardStyle}
            >
              <div style={traceHeaderStyle}>
                <span style={traceBadgeStyle}>
                  {firstNonEmpty(trace.workflow, "workflow")} · {firstNonEmpty(trace.artifactType, "artifact")}
                </span>
                <time dateTime={trace.createdAt} style={timeStyle}>
                  {formatDate(trace.createdAt)}
                </time>
              </div>
              <h2 style={traceDecisionStyle}>{trace.decision}</h2>
              {trace.rationale ? <p style={traceRationaleStyle}>{trace.rationale}</p> : null}
              <dl style={traceMetaStyle}>
                <div>
                  <dt style={metaLabelStyle}>Trace</dt>
                  <dd style={metaValueStyle}>{trace.traceId}</dd>
                </div>
                {trace.taskId ? (
                  <div>
                    <dt style={metaLabelStyle}>Task</dt>
                    <dd style={metaValueStyle}>{trace.taskId}</dd>
                  </div>
                ) : null}
                {trace.agentId ? (
                  <div>
                    <dt style={metaLabelStyle}>Agent</dt>
                    <dd style={metaValueStyle}>{trace.agentId}</dd>
                  </div>
                ) : null}
              </dl>
              {trace.sourceRefs.length > 0 ? (
                <ul aria-label={`Source refs for ${trace.traceId}`} style={sourceRefListStyle}>
                  {trace.sourceRefs.map((sourceRef) => (
                    <li key={sourceRef} style={sourceRefStyle}>
                      {sourceRef}
                    </li>
                  ))}
                </ul>
              ) : null}
            </li>
          ))}
        </ol>
      ) : (
        <section style={emptyPanelStyle}>
          <h2 style={emptyTitleStyle}>No traces captured yet.</h2>
          <p style={quietCopyStyle}>Capture judgment traces to give Style Memory reviewable evidence.</p>
        </section>
      )}
    </main>
  );
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

const traceListStyle: React.CSSProperties = {
  display: "grid",
  gap: "16px",
  margin: 0,
  padding: 0,
  listStyle: "none",
};

const traceCardStyle: React.CSSProperties = {
  padding: "26px",
  border: "1px solid var(--border)",
  borderRadius: "8px",
  background: "var(--canvas-raised)",
};

const selectedTraceCardStyle: React.CSSProperties = {
  ...traceCardStyle,
  borderColor: "var(--magic-primary-strong)",
  boxShadow: "0 0 0 4px color-mix(in srgb, var(--magic-primary-strong) 18%, transparent)",
};

const traceHeaderStyle: React.CSSProperties = {
  display: "flex",
  justifyContent: "space-between",
  gap: "16px",
  flexWrap: "wrap",
  marginBottom: "16px",
};

const traceBadgeStyle: React.CSSProperties = {
  color: "var(--magic-primary-strong)",
  fontFamily: "var(--font-mono)",
  fontSize: "12px",
  letterSpacing: "0.14em",
  textTransform: "uppercase",
};

const timeStyle: React.CSSProperties = {
  color: "var(--muted)",
  fontSize: "13px",
};

const traceDecisionStyle: React.CSSProperties = {
  margin: "0 0 14px",
  color: "var(--ink)",
  fontFamily: "var(--font-display)",
  fontSize: "32px",
  fontWeight: 500,
  lineHeight: 1.2,
};

const traceRationaleStyle: React.CSSProperties = {
  margin: "0 0 20px",
  color: "var(--ink-muted)",
  fontSize: "17px",
  lineHeight: 1.6,
};

const traceMetaStyle: React.CSSProperties = {
  display: "grid",
  gridTemplateColumns: "repeat(auto-fit, minmax(180px, 1fr))",
  gap: "14px",
  margin: "0 0 20px",
};

const metaLabelStyle: React.CSSProperties = {
  color: "var(--muted)",
  fontFamily: "var(--font-mono)",
  fontSize: "11px",
  letterSpacing: "0.14em",
  textTransform: "uppercase",
};

const metaValueStyle: React.CSSProperties = {
  margin: "3px 0 0",
  color: "var(--ink)",
  fontFamily: "var(--font-mono)",
  fontSize: "13px",
  overflowWrap: "anywhere",
};

const sourceRefListStyle: React.CSSProperties = {
  display: "flex",
  flexWrap: "wrap",
  gap: "8px",
  margin: 0,
  padding: 0,
  listStyle: "none",
};

const sourceRefStyle: React.CSSProperties = {
  padding: "6px 10px",
  border: "1px solid var(--border)",
  borderRadius: "999px",
  color: "var(--ink-muted)",
  fontFamily: "var(--font-mono)",
  fontSize: "12px",
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
