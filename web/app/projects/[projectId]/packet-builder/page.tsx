import { cookies } from "next/headers";
import Link from "next/link";
import { redirect } from "next/navigation";

import { RELAY_API_URL, relayFetch, type RelayEnvelope } from "@/lib/api";
import type { AuthMe } from "@/lib/onboarding";
import {
  getLatestPacketSnapshot,
  PacketBuilderError,
  type PacketBuilderSnapshot,
} from "@/lib/packet-builder";

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
  url.searchParams.set("redirect_to", `/projects/${projectId}/packet-builder`);
  return url.toString();
}

export default async function PacketBuilderPage({
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

  let snapshot: PacketBuilderSnapshot;
  try {
    snapshot = await getLatestPacketSnapshot(projectId, {
      headers: { cookie: cookieHeader },
    });
  } catch (error) {
    return <PacketBuilderFallback projectId={projectId} error={error} userDisplayName={me.display_name} />;
  }

  return <PacketBuilder snapshot={snapshot} userDisplayName={me.display_name} />;
}

function PacketBuilder({
  snapshot,
  userDisplayName,
}: {
  snapshot: PacketBuilderSnapshot;
  userDisplayName?: string;
}) {
  const evidenceCount =
    snapshot.styleCues.length +
    snapshot.supportingNotes.length +
    snapshot.supportingDecisions.length +
    snapshot.supportingQuestions.length +
    snapshot.supportingArtifacts.length;

  return (
    <main style={pageStyle}>
      <header style={topbarStyle}>
        <Link href="/" style={brandStyle}>
          Relay<span style={{ color: "var(--magic-primary-strong)" }}>.</span>
        </Link>
        <span style={userStyle}>{userDisplayName ?? "signed in"}</span>
      </header>

      <section style={heroStyle} aria-labelledby="packet-title">
        <p style={eyebrowStyle}>Project Explorer · Packet Builder</p>
        <div style={heroRowStyle}>
          <div>
            <h1 id="packet-title" style={titleStyle}>
              Packet Builder
            </h1>
            <p style={subtitleStyle}>{snapshot.taskSummary || `${snapshot.type} · ${snapshot.target}`}</p>
          </div>
          <nav style={actionsStyle} aria-label="Packet actions">
            <a href={`/projects/${encodeURIComponent(snapshot.projectId)}`} style={secondaryLinkStyle}>
              Project Explorer
            </a>
            <a href={`/projects/${encodeURIComponent(snapshot.projectId)}/graph`} style={secondaryLinkStyle}>
              Decision Graph
            </a>
            {snapshot.publicReadable && snapshot.publicToken ? (
              <a href={`/p/${encodeURIComponent(snapshot.publicToken)}`} style={primaryLinkStyle}>
                Open public snapshot
              </a>
            ) : (
              <span style={disabledActionStyle}>Private snapshot</span>
            )}
          </nav>
        </div>
      </section>

      <section style={metaGridStyle} aria-label="Packet metadata">
        <Metric label="Snapshot" value={snapshot.snapshotId} />
        <Metric label="Target" value={snapshot.target} />
        <Metric label="Evidence" value={`${evidenceCount}`} />
        <Metric label="Visibility" value={snapshot.publicReadable ? "public" : "private"} />
      </section>

      <section style={builderGridStyle}>
        <article style={documentStyle} aria-label="Packet document">
          <pre style={packetBodyStyle}>{snapshot.renderedBody || "No rendered packet body."}</pre>
        </article>

        <aside style={inspectorStyle} aria-label="Packet inspector">
          <details>
            <summary style={summaryStyle}>Source evidence</summary>
            <EvidenceList title="Style cues" items={snapshot.styleCues.map((item) => item.canonicalText || item.heuristicId)} />
            <EvidenceList title="Notes" items={snapshot.supportingNotes.map((item) => item.excerpt)} />
            <EvidenceList title="Decisions" items={snapshot.supportingDecisions.map((item) => item.summary)} />
            <EvidenceList title="Questions" items={snapshot.supportingQuestions.map((item) => item.summary)} />
            <EvidenceList title="Artifacts" items={snapshot.supportingArtifacts.map((item) => item.sourcePath || item.artifactId)} />
          </details>

          <details>
            <summary style={summaryStyle}>Publish controls</summary>
            <p style={quietCopyStyle}>
              Publish and revoke remain on the admin snapshot API for this slice. This builder reads the
              latest snapshot and shows the public route when the snapshot is already published.
            </p>
          </details>
        </aside>
      </section>
    </main>
  );
}

function EvidenceList({ title, items }: { title: string; items: string[] }) {
  return (
    <section style={evidenceSectionStyle}>
      <h2 style={inspectorTitleStyle}>{title}</h2>
      {items.length > 0 ? (
        <ul style={evidenceListStyle}>
          {items.map((item, index) => (
            <li key={`${title}:${index}`} style={evidenceItemStyle}>
              {item}
            </li>
          ))}
        </ul>
      ) : (
        <p style={quietCopyStyle}>None.</p>
      )}
    </section>
  );
}

function Metric({ label, value }: { label: string; value: string }) {
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
      <p style={eyebrowStyle}>Packet Builder</p>
      <h1 style={emptyTitleStyle}>Sign in first</h1>
      <p style={quietCopyStyle}>Packet snapshots are private to the project workspace.</p>
      <a href={signInURL(projectId)} style={primaryLinkStyle}>
        Continue with GitHub
      </a>
    </main>
  );
}

function PacketBuilderFallback({
  projectId,
  error,
  userDisplayName,
}: {
  projectId: string;
  error: unknown;
  userDisplayName?: string;
}) {
  const code = error instanceof PacketBuilderError ? error.code : "UNKNOWN";
  const message = error instanceof Error ? error.message : "Packet Builder failed to load.";
  const noSnapshot = code === "NOT_FOUND";
  return (
    <main style={emptyPageStyle}>
      <p style={eyebrowStyle}>Packet Builder · {userDisplayName ?? "signed in"}</p>
      <h1 style={emptyTitleStyle}>{noSnapshot ? "No packet yet" : "Couldn’t open packet"}</h1>
      <p style={noSnapshot ? quietCopyStyle : errorBoxStyle}>
        {noSnapshot ? "Build a packet snapshot first, then return here to inspect it." : `${code}: ${message}`}
      </p>
      <a href={`/projects/${encodeURIComponent(projectId)}`} style={primaryLinkStyle}>
        Project Explorer
      </a>
    </main>
  );
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

const disabledActionStyle: React.CSSProperties = {
  ...secondaryLinkStyle,
  color: "var(--muted)",
};

const metaGridStyle: React.CSSProperties = {
  display: "grid",
  gridTemplateColumns: "repeat(auto-fit, minmax(150px, 1fr))",
  gap: "12px",
  marginBottom: "18px",
};

const metricStyle: React.CSSProperties = {
  minWidth: 0,
  padding: "16px",
  border: "1px solid var(--border)",
  borderRadius: "8px",
  background: "var(--canvas-raised)",
};

const metricValueStyle: React.CSSProperties = {
  display: "block",
  color: "var(--ink)",
  fontFamily: "var(--font-mono)",
  fontSize: "15px",
  overflowWrap: "anywhere",
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

const builderGridStyle: React.CSSProperties = {
  display: "grid",
  gridTemplateColumns: "minmax(0, 1fr) minmax(260px, 340px)",
  gap: "18px",
  alignItems: "start",
};

const documentStyle: React.CSSProperties = {
  border: "1px solid var(--border)",
  borderRadius: "8px",
  background: "var(--canvas-raised)",
  overflow: "hidden",
};

const packetBodyStyle: React.CSSProperties = {
  margin: 0,
  padding: "28px",
  whiteSpace: "pre-wrap",
  overflowWrap: "anywhere",
  color: "var(--ink)",
  fontFamily: "var(--font-mono)",
  fontSize: "15px",
  lineHeight: 1.7,
};

const inspectorStyle: React.CSSProperties = {
  display: "grid",
  gap: "12px",
};

const summaryStyle: React.CSSProperties = {
  cursor: "pointer",
  padding: "16px",
  border: "1px solid var(--border)",
  borderRadius: "8px",
  background: "var(--canvas-raised)",
  color: "var(--ink)",
  fontFamily: "var(--font-mono)",
  fontSize: "12px",
  letterSpacing: "0.14em",
  textTransform: "uppercase",
};

const evidenceSectionStyle: React.CSSProperties = {
  padding: "14px 16px 0",
};

const inspectorTitleStyle: React.CSSProperties = {
  margin: "0 0 8px",
  color: "var(--ink)",
  fontFamily: "var(--font-display)",
  fontSize: "20px",
  fontWeight: 500,
};

const evidenceListStyle: React.CSSProperties = {
  margin: 0,
  paddingLeft: "18px",
};

const evidenceItemStyle: React.CSSProperties = {
  marginBottom: "8px",
  color: "var(--ink-muted)",
  fontSize: "14px",
  lineHeight: 1.5,
  overflowWrap: "anywhere",
};

const quietCopyStyle: React.CSSProperties = {
  margin: 0,
  color: "var(--ink-muted)",
  fontSize: "16px",
  lineHeight: 1.6,
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
