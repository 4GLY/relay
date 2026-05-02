import { cookies } from "next/headers";
import { getLocale, getTranslations } from "next-intl/server";
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
  const locale = await getLocale();
  const t = await getTranslations("PacketBuilder");
  const me = await resolveSession(cookieHeader);

  if (!me) {
    return <SignInRequired projectId={projectId} t={t} />;
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
    return <PacketBuilderFallback projectId={projectId} error={error} userDisplayName={me.display_name} t={t} />;
  }

  return <PacketBuilder snapshot={snapshot} userDisplayName={me.display_name} t={t} locale={locale} />;
}

function PacketBuilder({
  snapshot,
  userDisplayName,
  t,
  locale,
}: {
  snapshot: PacketBuilderSnapshot;
  userDisplayName?: string;
  t: ProductT;
  locale: string;
}) {
  const evidenceCount =
    snapshot.styleCues.length +
    snapshot.supportingNotes.length +
    snapshot.supportingDecisions.length +
    snapshot.supportingQuestions.length +
    snapshot.supportingArtifacts.length;
  const sources = packetSources(snapshot, t);
  const previewTitle = snapshot.taskSummary || `${snapshot.type} · ${snapshot.target}`;
  const publicSnapshotHref =
    snapshot.publicReadable && snapshot.publicToken ? `/p/${encodeURIComponent(snapshot.publicToken)}` : undefined;

  return (
    <>
      <RelayTopRail
        activeStep="Transform"
        userLabel={userDisplayName ?? t("signedInFallback")}
        projectHref={`/projects/${encodeURIComponent(snapshot.projectId)}`}
      />
      <main className="relay-packet-page">
        <RelayPageHead
          eyebrow={t("eyebrow", { projectId: snapshot.projectId })}
          title={t("title")}
          titleId="packet-title"
          actions={
            <>
              <button type="button" className="relay-action" data-variant="secondary" disabled>
                {t("actions.saveDraft")}
              </button>
              <RelayLinkButton href={`/projects/${encodeURIComponent(snapshot.projectId)}/graph`} variant="secondary">
                {t("actions.decisionGraph")}
              </RelayLinkButton>
              {publicSnapshotHref ? (
                <RelayLinkButton href={publicSnapshotHref} variant="primary">
                  {t("actions.openPublicSnapshot")}
                </RelayLinkButton>
              ) : (
                <span className="relay-action" data-variant="secondary" aria-disabled="true">
                  {t("actions.buildSnapshot")}
                </span>
              )}
            </>
          }
        />

        <section className="relay-summary-grid" aria-label={t("labels.packetMetadata")}>
          <RelayMetricTile label={t("labels.snapshot")} value={snapshot.snapshotId} />
          <RelayMetricTile label={t("labels.target")} value={snapshot.target} />
          <RelayMetricTile label={t("labels.evidence")} value={`${evidenceCount}`} />
          <RelayMetricTile label={t("labels.visibility")} value={snapshot.publicReadable ? t("labels.public") : t("labels.private")} />
        </section>

        <section className="relay-packet-builder-grid">
          <RelayCard className="relay-packet-composition" aria-label={t("labels.packetComposition")}>
            <span className="relay-card-kicker">{t("labels.coverNote")}</span>
            <textarea
              readOnly
              value={previewTitle}
              aria-label={t("labels.packetCoverNote")}
              className="relay-packet-cover-note"
            />

            <section className="relay-packet-source-section" aria-label={t("labels.includedSources", { count: sources.length })}>
              <span className="relay-card-kicker">{t("labels.includedSources", { count: sources.length })}</span>
              {sources.length > 0 ? (
                <div className="relay-packet-source-list">
                  {sources.map((source) => (
                    <SourceRow key={source.id} kind={source.kind} label={source.label} t={t} />
                  ))}
                </div>
              ) : (
                <div className="relay-packet-dropzone">{t("empty.noSources")}</div>
              )}
            </section>

            <div className="relay-packet-dropzone">{t("empty.dropzone")}</div>

            <details open className="relay-packet-details">
              <summary className="relay-details-summary">{t("labels.renderedPacketBody")}</summary>
              <pre className="relay-packet-body">{snapshot.renderedBody || t("empty.noRenderedBody")}</pre>
            </details>
          </RelayCard>

          <aside className="relay-packet-preview-column" aria-label={t("labels.packetPreview")}>
            <span className="relay-card-kicker">{t("labels.snapshotPreview")}</span>
            <RelayCard className="relay-packet-preview-card">
              <div className="relay-packet-preview-header">
                <div className="relay-card-kicker">{snapshot.snapshotId}</div>
                <h2 className="relay-packet-preview-title">{previewTitle}</h2>
                <p className="relay-packet-preview-meta">
                  {snapshot.styleCues.length} {t("labels.heuristics")} ·{" "}
                  {snapshot.supportingDecisions.length + snapshot.supportingQuestions.length} {t("labels.traces")} ·{" "}
                  {snapshot.supportingNotes.length + snapshot.supportingArtifacts.length} {t("labels.sources")}
                </p>
              </div>
              <div className="relay-packet-preview-body">
                <span className="relay-card-kicker">{t("labels.recipient")}</span>
                <span className="relay-mono-value">{snapshot.target}</span>
                <span className="relay-card-kicker relay-kicker-spaced">{t("labels.visibilityKicker")}</span>
                <span className="relay-mono-value">
                  {snapshot.publicReadable ? t("labels.publicLink") : t("labels.privateWorkspace")}
                </span>
                <span className="relay-card-kicker relay-kicker-spaced">{t("labels.created")}</span>
                <span className="relay-mono-value">{formatDate(snapshot.createdAt, locale)}</span>
              </div>
            </RelayCard>

            <div className="relay-packet-inspector">
              <details>
                <summary className="relay-details-summary">{t("labels.sourceEvidence")}</summary>
                <EvidenceList title={t("labels.styleCues")} items={snapshot.styleCues.map((item) => item.canonicalText || item.heuristicId)} t={t} />
                <EvidenceList title={t("labels.notes")} items={snapshot.supportingNotes.map((item) => item.excerpt)} t={t} />
                <EvidenceList title={t("labels.decisions")} items={snapshot.supportingDecisions.map((item) => item.summary)} t={t} />
                <EvidenceList title={t("labels.questions")} items={snapshot.supportingQuestions.map((item) => item.summary)} t={t} />
                <EvidenceList title={t("labels.artifacts")} items={snapshot.supportingArtifacts.map((item) => item.sourcePath || item.artifactId)} t={t} />
              </details>

              <details>
                <summary className="relay-details-summary">{t("labels.publishControls")}</summary>
                <p className="relay-quiet-copy">
                  {t("publishCopy")}
                </p>
              </details>
            </div>
          </aside>
        </section>
      </main>
    </>
  );
}

function packetSources(snapshot: PacketBuilderSnapshot, t: ProductT) {
  return [
    ...snapshot.styleCues.map((item) => ({
      id: `style:${item.heuristicId}`,
      kind: "swan" as const,
      label: `${t("sourceKind.heuristic")} · ${item.canonicalText || item.heuristicKey || item.heuristicId}`,
    })),
    ...snapshot.supportingDecisions.map((item) => ({
      id: `decision:${item.decisionId}`,
      kind: "trace" as const,
      label: `${t("sourceKind.decision")} · ${item.summary}`,
    })),
    ...snapshot.supportingQuestions.map((item) => ({
      id: `question:${item.questionId}`,
      kind: "trace" as const,
      label: `${t("sourceKind.question")} · ${item.summary}`,
    })),
    ...snapshot.supportingNotes.map((item) => ({
      id: `note:${item.noteId}`,
      kind: "note" as const,
      label: `${t("sourceKind.note")} · ${item.source || item.excerpt}`,
    })),
    ...snapshot.supportingArtifacts.map((item) => ({
      id: `artifact:${item.artifactId}`,
      kind: "artifact" as const,
      label: `${t("sourceKind.artifact")} · ${item.sourcePath || item.artifactId}`,
    })),
  ];
}

function SourceRow({
  kind,
  label,
  t,
}: {
  kind: "swan" | "trace" | "note" | "artifact";
  label: string;
  t: ProductT;
}) {
  return (
    <div className="relay-packet-source-row">
      <span className="relay-packet-source-dot" data-kind={kind} />
      <span className="relay-packet-source-label">{label}</span>
      <span className="relay-packet-source-remove">{t("sourceKind.remove")}</span>
    </div>
  );
}

function EvidenceList({ title, items, t }: { title: string; items: string[]; t: ProductT }) {
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
        <p className="relay-quiet-copy">{t("empty.none")}</p>
      )}
    </section>
  );
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

function PacketBuilderFallback({
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
  const code = error instanceof PacketBuilderError ? error.code : "UNKNOWN";
  const message = error instanceof Error ? error.message : t("error.fallbackMessage");
  const noSnapshot = code === "NOT_FOUND";
  return (
    <main className="relay-empty-page">
      <RelayPageHead
        eyebrow={t("error.eyebrow", { user: userDisplayName ?? t("error.signedInFallback") })}
        title={noSnapshot ? t("error.noSnapshotTitle") : t("error.title")}
        copy={noSnapshot ? t("error.noSnapshotCopy") : undefined}
        actions={
          <RelayLinkButton href={`/projects/${encodeURIComponent(projectId)}`} variant="primary">
            {t("actions.projectExplorer")}
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
