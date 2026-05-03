import { cookies } from "next/headers";
import { getTranslations } from "next-intl/server";
import { redirect } from "next/navigation";

import {
  RelayEmptyState,
  RelayLinkButton,
  RelayPageHead,
  RelayPageKicker,
} from "@/components/relay";
import { relayFetch, type RelayEnvelope } from "@/lib/api";
import {
  listApprovedHeuristics,
  listPendingProposals,
  listRejectedProposals,
  type ApprovedHeuristic,
  type PendingProposal,
} from "@/lib/heuristics";

import { Proposals } from "./proposals";

export const dynamic = "force-dynamic";

type AuthMe = {
  user_id: string;
  email?: string;
  display_name?: string;
  avatar_url?: string;
};

type SearchParams = {
  project?: string;
  return?: string;
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

export default async function StyleMemoryPage({
  searchParams,
}: {
  searchParams: Promise<SearchParams>;
}) {
  const params = await searchParams;
  const cookieStore = await cookies();
  const cookieHeader = cookieStore.toString();
  const t = await getTranslations("StyleMemory.page");
  const actions = await getTranslations("StyleMemory.actions");

  const me = await resolveSession(cookieHeader);
  if (!me) {
    redirect("/?return=/style-memory");
  }

  // V2.5: `/v1/auth/me` exposes default_project_id for Project Explorer entry
  // routing, while Style Memory keeps `?project=<id>` as its explicit review
  // surface contract.
  const projectId = (params.project ?? "").trim();

  if (!projectId) {
    return <MissingProject userDisplayName={me.display_name} t={t} />;
  }

  const headers: HeadersInit = { cookie: cookieHeader };
  const [pendingResult, approvedResult, rejectedResult] = await Promise.allSettled([
    listPendingProposals(projectId, { headers, limit: 50 }),
    listApprovedHeuristics(projectId, { headers, limit: 100 }),
    listRejectedProposals(projectId, { headers, limit: 50 }),
  ]);

  if (pendingResult.status === "rejected") {
    return (
      <FullPageError
        reason={String(pendingResult.reason)}
        projectId={projectId}
        t={t}
        retryLabel={actions("retry")}
      />
    );
  }

  const initialPending = pendingResult.value.items;
  const approvedFetchFailed = approvedResult.status === "rejected";
  const initialApproved: ApprovedHeuristic[] = approvedFetchFailed
    ? []
    : approvedResult.value.items;
  const rejectedFetchFailed = rejectedResult.status === "rejected";
  const initialRejected: PendingProposal[] = rejectedFetchFailed
    ? []
    : rejectedResult.value.items;

  return (
    <Proposals
      projectId={projectId}
      initialPending={initialPending}
      initialApproved={initialApproved}
      initialRejected={initialRejected}
      approvedFetchFailed={approvedFetchFailed}
      rejectedFetchFailed={rejectedFetchFailed}
      userDisplayName={me.display_name}
      userId={me.user_id}
    />
  );
}

function MissingProject({
  userDisplayName,
  t,
}: {
  userDisplayName?: string;
  t: Awaited<ReturnType<typeof getTranslations>>;
}) {
  return (
    <main className="relay-empty-page">
      <RelayPageHead
        eyebrow={t("missingProjectEyebrow", { user: userDisplayName ?? t("signedInFallback") })}
        title={t("missingProjectTitle")}
        copy={
          <>
            {t("missingProjectCopyPrefix")}{" "}
            <code className="relay-inline-code">?project=&lt;id&gt;</code>
            {" "}{t("missingProjectCopySuffix")}
          </>
        }
      />
    </main>
  );
}

function FullPageError({
  reason,
  projectId,
  t,
  retryLabel,
}: {
  reason: string;
  projectId: string;
  t: Awaited<ReturnType<typeof getTranslations>>;
  retryLabel: string;
}) {
  return (
    <main className="relay-empty-page">
      <RelayPageKicker className="relay-danger-text">
        {t("loadErrorKicker")}
      </RelayPageKicker>
      <RelayEmptyState title={t("loadErrorTitle")} glyph="!">
      <p className="relay-error-pre">
        {reason}
      </p>
      <RelayLinkButton
        href={`/style-memory?project=${encodeURIComponent(projectId)}`}
        variant="primary"
      >
        {retryLabel}
      </RelayLinkButton>
      </RelayEmptyState>
    </main>
  );
}
