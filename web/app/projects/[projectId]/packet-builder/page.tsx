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
import { RelayTopRail } from "@/components/relay-app-shell";

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
  const sources = packetSources(snapshot);
  const previewTitle = snapshot.taskSummary || `${snapshot.type} · ${snapshot.target}`;
  const publicSnapshotHref =
    snapshot.publicReadable && snapshot.publicToken ? `/p/${encodeURIComponent(snapshot.publicToken)}` : undefined;

  return (
    <>
      <RelayTopRail
        activeStep="Transform"
        userLabel={userDisplayName ?? "signed in"}
        projectHref={`/projects/${encodeURIComponent(snapshot.projectId)}`}
      />
      <main style={pageStyle}>
        <section style={pageHeadStyle} aria-labelledby="packet-title">
          <div>
            <p style={eyebrowStyle}>{snapshot.projectId} · Packet Builder</p>
            <h1 id="packet-title" style={titleStyle}>
              Compose a handoff.
            </h1>
          </div>
          <nav style={actionsStyle} aria-label="Packet actions">
            <button type="button" style={ghostButtonStyle} disabled>
              Save draft
            </button>
            <Link href={`/projects/${encodeURIComponent(snapshot.projectId)}/graph`} style={ghostLinkStyle}>
              Decision Graph
            </Link>
            {publicSnapshotHref ? (
              <Link href={publicSnapshotHref} style={primaryLinkStyle}>
                Open public snapshot →
              </Link>
            ) : (
              <span style={disabledActionStyle}>Build snapshot →</span>
            )}
          </nav>
        </section>

        <section style={metaGridStyle} aria-label="Packet metadata">
          <Metric label="Snapshot" value={snapshot.snapshotId} />
          <Metric label="Target" value={snapshot.target} />
          <Metric label="Evidence" value={`${evidenceCount}`} />
          <Metric label="Visibility" value={snapshot.publicReadable ? "public" : "private"} />
        </section>

        <section className="relay-packet-builder-grid" style={builderGridStyle}>
          <article style={compositionCardStyle} aria-label="Packet composition">
            <span style={kickerStyle}>cover note · Fraunces italic</span>
            <textarea
              readOnly
              value={previewTitle}
              aria-label="Packet cover note"
              style={coverNoteStyle}
            />

            <section style={sourceSectionStyle} aria-label="Included sources">
              <span style={kickerStyle}>included sources · {sources.length}</span>
              {sources.length > 0 ? (
                <div style={sourceListStyle}>
                  {sources.map((source) => (
                    <SourceRow key={source.id} kind={source.kind} label={source.label} />
                  ))}
                </div>
              ) : (
                <div style={emptyDropzoneStyle}>No sources were selected for this packet snapshot.</div>
              )}
            </section>

            <div style={emptyDropzoneStyle}>Drop a trace, note, or artifact here. Or pick from the Inspector.</div>

            <details open style={documentDetailsStyle}>
              <summary style={summaryStyle}>Rendered packet body</summary>
              <pre style={packetBodyStyle}>{snapshot.renderedBody || "No rendered packet body."}</pre>
            </details>
          </article>

          <aside style={previewColumnStyle} aria-label="Packet preview">
            <span style={kickerStyle}>snapshot preview</span>
            <div style={previewCardStyle}>
              <div style={previewHeaderStyle}>
                <div style={kickerStyle}>{snapshot.snapshotId}</div>
                <h2 style={previewTitleStyle}>{previewTitle}</h2>
                <p style={previewMetaStyle}>
                  {snapshot.styleCues.length} heuristics ·{" "}
                  {snapshot.supportingDecisions.length + snapshot.supportingQuestions.length} traces ·{" "}
                  {snapshot.supportingNotes.length + snapshot.supportingArtifacts.length} sources
                </p>
              </div>
              <div style={previewBodyStyle}>
                <span style={kickerStyle}>recipient</span>
                <span style={monoValueStyle}>{snapshot.target}</span>
                <span style={{ ...kickerStyle, marginTop: "8px" }}>visibility</span>
                <span style={monoValueStyle}>
                  {snapshot.publicReadable ? "Public link" : "Private workspace"}
                </span>
                <span style={{ ...kickerStyle, marginTop: "8px" }}>created</span>
                <span style={monoValueStyle}>{snapshot.createdAt}</span>
              </div>
            </div>

            <div style={inspectorStyle}>
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
            </div>
          </aside>
        </section>
      </main>
    </>
  );
}

function packetSources(snapshot: PacketBuilderSnapshot) {
  return [
    ...snapshot.styleCues.map((item) => ({
      id: `style:${item.heuristicId}`,
      kind: "swan" as const,
      label: `Heuristic · ${item.canonicalText || item.heuristicKey || item.heuristicId}`,
    })),
    ...snapshot.supportingDecisions.map((item) => ({
      id: `decision:${item.decisionId}`,
      kind: "trace" as const,
      label: `Decision · ${item.summary}`,
    })),
    ...snapshot.supportingQuestions.map((item) => ({
      id: `question:${item.questionId}`,
      kind: "trace" as const,
      label: `Question · ${item.summary}`,
    })),
    ...snapshot.supportingNotes.map((item) => ({
      id: `note:${item.noteId}`,
      kind: "note" as const,
      label: `Note · ${item.source || item.excerpt}`,
    })),
    ...snapshot.supportingArtifacts.map((item) => ({
      id: `artifact:${item.artifactId}`,
      kind: "artifact" as const,
      label: `Artifact · ${item.sourcePath || item.artifactId}`,
    })),
  ];
}

function SourceRow({
  kind,
  label,
}: {
  kind: "swan" | "trace" | "note" | "artifact";
  label: string;
}) {
  return (
    <div style={sourceRowStyle}>
      <span style={{ ...sourceDotStyle, background: sourceKindColor(kind) }} />
      <span style={sourceLabelStyle}>{label}</span>
      <span style={sourceRemoveStyle}>remove</span>
    </div>
  );
}

function sourceKindColor(kind: "swan" | "trace" | "note" | "artifact") {
  if (kind === "swan") return "var(--magic-primary-strong)";
  if (kind === "trace") return "var(--magic-accent-strong)";
  if (kind === "note") return "var(--success)";
  return "var(--muted)";
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
  padding: "44px 28px 96px",
};

const pageHeadStyle: React.CSSProperties = {
  display: "flex",
  gap: "20px",
  alignItems: "center",
  justifyContent: "space-between",
  flexWrap: "wrap",
  marginBottom: "28px",
  paddingBottom: "24px",
  borderBottom: "1px solid var(--border)",
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
  fontSize: "clamp(44px, 6vw, 68px)",
  fontWeight: 500,
  lineHeight: 1,
  fontVariationSettings: '"opsz" 144, "SOFT" 50',
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

const ghostLinkStyle: React.CSSProperties = {
  ...primaryLinkStyle,
  background: "transparent",
  color: "var(--ink)",
  border: "1px solid var(--border-strong)",
};

const ghostButtonStyle: React.CSSProperties = {
  ...ghostLinkStyle,
  fontFamily: "var(--font-sans)",
  fontSize: "14px",
  opacity: 0.62,
  cursor: "not-allowed",
};

const disabledActionStyle: React.CSSProperties = {
  ...ghostLinkStyle,
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
  gap: "20px",
  alignItems: "start",
};

const compositionCardStyle: React.CSSProperties = {
  padding: "24px",
  border: "1px solid var(--border)",
  borderRadius: "12px",
  background: "var(--canvas-raised)",
  overflow: "hidden",
};

const kickerStyle: React.CSSProperties = {
  color: "var(--muted)",
  fontFamily: "var(--font-mono)",
  fontSize: "11px",
  letterSpacing: "0.14em",
  textTransform: "uppercase",
};

const coverNoteStyle: React.CSSProperties = {
  width: "100%",
  minHeight: "96px",
  marginTop: "10px",
  padding: "14px",
  border: "1px solid var(--border-strong)",
  borderRadius: "8px",
  background: "var(--canvas)",
  color: "var(--ink)",
  fontFamily: "var(--font-display)",
  fontSize: "17px",
  fontStyle: "italic",
  lineHeight: 1.5,
  resize: "vertical",
  fontVariationSettings: '"opsz" 48',
};

const sourceSectionStyle: React.CSSProperties = {
  marginTop: "22px",
};

const sourceListStyle: React.CSSProperties = {
  display: "flex",
  flexDirection: "column",
  gap: "8px",
  marginTop: "12px",
};

const sourceRowStyle: React.CSSProperties = {
  display: "flex",
  alignItems: "center",
  gap: "10px",
  padding: "10px 14px",
  border: "1px solid var(--border)",
  borderRadius: "8px",
  background: "var(--canvas)",
};

const sourceDotStyle: React.CSSProperties = {
  width: "8px",
  height: "8px",
  flex: "0 0 auto",
  borderRadius: "999px",
};

const sourceLabelStyle: React.CSSProperties = {
  minWidth: 0,
  flex: 1,
  color: "var(--ink)",
  fontSize: "13px",
  overflowWrap: "anywhere",
};

const sourceRemoveStyle: React.CSSProperties = {
  color: "var(--muted)",
  fontFamily: "var(--font-mono)",
  fontSize: "11px",
};

const emptyDropzoneStyle: React.CSSProperties = {
  marginTop: "14px",
  padding: "22px",
  border: "1px dashed var(--border-strong)",
  borderRadius: "12px",
  background: "color-mix(in srgb, var(--magic-primary) 8%, var(--canvas))",
  color: "var(--ink-muted)",
  fontFamily: "var(--font-display)",
  fontSize: "17px",
  fontStyle: "italic",
  textAlign: "center",
};

const documentDetailsStyle: React.CSSProperties = {
  marginTop: "18px",
};

const packetBodyStyle: React.CSSProperties = {
  margin: 0,
  padding: "18px 2px 0",
  whiteSpace: "pre-wrap",
  overflowWrap: "anywhere",
  color: "var(--ink)",
  fontFamily: "var(--font-mono)",
  fontSize: "15px",
  lineHeight: 1.7,
};

const previewColumnStyle: React.CSSProperties = {
  minWidth: 0,
};

const previewCardStyle: React.CSSProperties = {
  marginTop: "12px",
  border: "1px solid var(--border)",
  borderRadius: "12px",
  background: "var(--canvas-raised)",
  overflow: "hidden",
};

const previewHeaderStyle: React.CSSProperties = {
  padding: "20px 22px 16px",
  borderBottom: "1px solid var(--border)",
};

const previewTitleStyle: React.CSSProperties = {
  margin: "8px 0 0",
  color: "var(--ink)",
  fontFamily: "var(--font-display)",
  fontSize: "22px",
  fontWeight: 500,
  lineHeight: 1.2,
  fontVariationSettings: '"opsz" 72',
};

const previewMetaStyle: React.CSSProperties = {
  margin: "10px 0 0",
  color: "var(--muted)",
  fontFamily: "var(--font-mono)",
  fontSize: "11px",
};

const previewBodyStyle: React.CSSProperties = {
  display: "flex",
  flexDirection: "column",
  gap: "8px",
  padding: "16px 22px",
};

const monoValueStyle: React.CSSProperties = {
  color: "var(--ink)",
  fontFamily: "var(--font-mono)",
  fontSize: "13px",
  overflowWrap: "anywhere",
};

const inspectorStyle: React.CSSProperties = {
  display: "grid",
  gap: "12px",
  marginTop: "16px",
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
