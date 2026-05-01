import { cookies } from "next/headers";
import { redirect } from "next/navigation";

import { RELAY_API_URL, relayFetch, type RelayEnvelope } from "@/lib/api";
import type { AuthMe } from "@/lib/onboarding";
import {
  getLatestPacketSnapshot,
  PacketBuilderError,
  type PacketBuilderSnapshot,
} from "@/lib/packet-builder";
import {
  RelayCard,
  RelayFeedback,
  RelayLinkButton,
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
      <main className="relay-packet-page">
        <RelayPageHead
          eyebrow={`${snapshot.projectId} · Packet Builder`}
          title="Compose a handoff."
          titleId="packet-title"
          actions={
            <>
              <button type="button" className="relay-action" data-variant="secondary" disabled>
                Save draft
              </button>
              <RelayLinkButton href={`/projects/${encodeURIComponent(snapshot.projectId)}/graph`} variant="secondary">
                Decision Graph
              </RelayLinkButton>
              {publicSnapshotHref ? (
                <RelayLinkButton href={publicSnapshotHref} variant="primary">
                  Open public snapshot →
                </RelayLinkButton>
              ) : (
                <span className="relay-action" data-variant="secondary" aria-disabled="true">
                  Build snapshot →
                </span>
              )}
            </>
          }
        />

        <section className="relay-summary-grid" aria-label="Packet metadata">
          <RelayMetricTile label="Snapshot" value={snapshot.snapshotId} />
          <RelayMetricTile label="Target" value={snapshot.target} />
          <RelayMetricTile label="Evidence" value={`${evidenceCount}`} />
          <RelayMetricTile label="Visibility" value={snapshot.publicReadable ? "public" : "private"} />
        </section>

        <section className="relay-packet-builder-grid">
          <RelayCard className="relay-packet-composition" aria-label="Packet composition">
            <span className="relay-card-kicker">cover note · Fraunces italic</span>
            <textarea
              readOnly
              value={previewTitle}
              aria-label="Packet cover note"
              className="relay-packet-cover-note"
            />

            <section className="relay-packet-source-section" aria-label="Included sources">
              <span className="relay-card-kicker">included sources · {sources.length}</span>
              {sources.length > 0 ? (
                <div className="relay-packet-source-list">
                  {sources.map((source) => (
                    <SourceRow key={source.id} kind={source.kind} label={source.label} />
                  ))}
                </div>
              ) : (
                <div className="relay-packet-dropzone">No sources were selected for this packet snapshot.</div>
              )}
            </section>

            <div className="relay-packet-dropzone">Drop a trace, note, or artifact here. Or pick from the Inspector.</div>

            <details open className="relay-packet-details">
              <summary className="relay-details-summary">Rendered packet body</summary>
              <pre className="relay-packet-body">{snapshot.renderedBody || "No rendered packet body."}</pre>
            </details>
          </RelayCard>

          <aside className="relay-packet-preview-column" aria-label="Packet preview">
            <span className="relay-card-kicker">snapshot preview</span>
            <RelayCard className="relay-packet-preview-card">
              <div className="relay-packet-preview-header">
                <div className="relay-card-kicker">{snapshot.snapshotId}</div>
                <h2 className="relay-packet-preview-title">{previewTitle}</h2>
                <p className="relay-packet-preview-meta">
                  {snapshot.styleCues.length} heuristics ·{" "}
                  {snapshot.supportingDecisions.length + snapshot.supportingQuestions.length} traces ·{" "}
                  {snapshot.supportingNotes.length + snapshot.supportingArtifacts.length} sources
                </p>
              </div>
              <div className="relay-packet-preview-body">
                <span className="relay-card-kicker">recipient</span>
                <span className="relay-mono-value">{snapshot.target}</span>
                <span className="relay-card-kicker relay-kicker-spaced">visibility</span>
                <span className="relay-mono-value">
                  {snapshot.publicReadable ? "Public link" : "Private workspace"}
                </span>
                <span className="relay-card-kicker relay-kicker-spaced">created</span>
                <span className="relay-mono-value">{snapshot.createdAt}</span>
              </div>
            </RelayCard>

            <div className="relay-packet-inspector">
              <details>
                <summary className="relay-details-summary">Source evidence</summary>
                <EvidenceList title="Style cues" items={snapshot.styleCues.map((item) => item.canonicalText || item.heuristicId)} />
                <EvidenceList title="Notes" items={snapshot.supportingNotes.map((item) => item.excerpt)} />
                <EvidenceList title="Decisions" items={snapshot.supportingDecisions.map((item) => item.summary)} />
                <EvidenceList title="Questions" items={snapshot.supportingQuestions.map((item) => item.summary)} />
                <EvidenceList title="Artifacts" items={snapshot.supportingArtifacts.map((item) => item.sourcePath || item.artifactId)} />
              </details>

              <details>
                <summary className="relay-details-summary">Publish controls</summary>
                <p className="relay-quiet-copy">
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
    <div className="relay-packet-source-row">
      <span className="relay-packet-source-dot" data-kind={kind} />
      <span className="relay-packet-source-label">{label}</span>
      <span className="relay-packet-source-remove">remove</span>
    </div>
  );
}

function EvidenceList({ title, items }: { title: string; items: string[] }) {
  return (
    <section className="relay-evidence-section">
      <h2 className="relay-evidence-title">{title}</h2>
      {items.length > 0 ? (
        <ul className="relay-evidence-list">
          {items.map((item, index) => (
            <li key={`${title}:${index}`} className="relay-evidence-item">
              {item}
            </li>
          ))}
        </ul>
      ) : (
        <p className="relay-quiet-copy">None.</p>
      )}
    </section>
  );
}

function SignInRequired({ projectId }: { projectId: string }) {
  return (
    <main className="relay-empty-page">
      <RelayPageHead
        eyebrow="Packet Builder"
        title="Sign in first"
        copy="Packet snapshots are private to the project workspace."
        actions={
          <RelayLinkButton href={signInURL(projectId)} variant="primary">
            Continue with GitHub
          </RelayLinkButton>
        }
      />
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
    <main className="relay-empty-page">
      <RelayPageHead
        eyebrow={`Packet Builder · ${userDisplayName ?? "signed in"}`}
        title={noSnapshot ? "No packet yet" : "Couldn’t open packet"}
        copy={noSnapshot ? "Build a packet snapshot first, then return here to inspect it." : undefined}
        actions={
          <RelayLinkButton href={`/projects/${encodeURIComponent(projectId)}`} variant="primary">
            Project Explorer
          </RelayLinkButton>
        }
      />
      {!noSnapshot ? (
        <RelayFeedback role="alert" variant="error">
          {code}: {message}
        </RelayFeedback>
      ) : null}
    </main>
  );
}
