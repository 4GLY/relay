import { cookies } from "next/headers";
import { redirect } from "next/navigation";

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

  const me = await resolveSession(cookieHeader);
  if (!me) {
    redirect("/?return=/style-memory");
  }

  // V2.5: `/v1/auth/me` exposes default_project_id for Project Explorer entry
  // routing, while Style Memory keeps `?project=<id>` as its explicit review
  // surface contract.
  const projectId = (params.project ?? "").trim();

  if (!projectId) {
    return <MissingProject userDisplayName={me.display_name} />;
  }

  const headers: HeadersInit = { cookie: cookieHeader };
  const [pendingResult, approvedResult, rejectedResult] = await Promise.allSettled([
    listPendingProposals(projectId, { headers, limit: 50 }),
    listApprovedHeuristics(projectId, { headers, limit: 100 }),
    listRejectedProposals(projectId, { headers, limit: 50 }),
  ]);

  if (pendingResult.status === "rejected") {
    return <FullPageError reason={String(pendingResult.reason)} projectId={projectId} />;
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

function MissingProject({ userDisplayName }: { userDisplayName?: string }) {
  return (
    <main
      style={{
        maxWidth: "640px",
        margin: "0 auto",
        padding: "120px 32px",
        fontFamily: "var(--font-sans)",
        color: "var(--ink)",
      }}
    >
      <p
        style={{
          fontFamily: "var(--font-mono)",
          fontSize: "11px",
          letterSpacing: "0.18em",
          textTransform: "uppercase",
          color: "var(--muted)",
          marginBottom: "16px",
        }}
      >
        Style Memory · {userDisplayName ?? "signed in"}
      </p>
      <h1
        style={{
          fontFamily: "var(--font-display)",
          fontWeight: 500,
          fontSize: "clamp(36px, 6vw, 56px)",
          lineHeight: 1.05,
          letterSpacing: "-0.025em",
          marginBottom: "20px",
          fontVariationSettings: '"opsz" 144, "SOFT" 50',
        }}
      >
        Pick a project
      </h1>
      <p
        style={{
          fontFamily: "var(--font-display)",
          fontStyle: "italic",
          fontSize: "16px",
          lineHeight: 1.55,
          color: "var(--ink-muted)",
          fontVariationSettings: '"opsz" 48',
        }}
      >
        Style Memory needs a project to read proposals from. Add{" "}
        <code style={{ fontFamily: "var(--font-mono)", fontSize: "13px" }}>?project=&lt;id&gt;</code>{" "}
        to the URL, or finish onboarding to land here automatically.
      </p>
    </main>
  );
}

function FullPageError({
  reason,
  projectId,
}: {
  reason: string;
  projectId: string;
}) {
  return (
    <main
      style={{
        maxWidth: "640px",
        margin: "0 auto",
        padding: "120px 32px",
        fontFamily: "var(--font-sans)",
        color: "var(--ink)",
      }}
    >
      <p
        style={{
          fontFamily: "var(--font-mono)",
          fontSize: "11px",
          letterSpacing: "0.18em",
          textTransform: "uppercase",
          color: "var(--danger)",
          marginBottom: "16px",
        }}
      >
        Couldn’t load proposals
      </p>
      <h1
        style={{
          fontFamily: "var(--font-display)",
          fontWeight: 500,
          fontSize: "clamp(28px, 4vw, 40px)",
          lineHeight: 1.1,
          marginBottom: "16px",
        }}
      >
        Something is wrong with the queue right now.
      </h1>
      <p
        style={{
          fontFamily: "var(--font-mono)",
          fontSize: "12px",
          color: "var(--ink-muted)",
          background: "var(--canvas-raised)",
          border: "1px solid var(--border)",
          borderRadius: "8px",
          padding: "12px",
          marginBottom: "20px",
          whiteSpace: "pre-wrap",
        }}
      >
        {reason}
      </p>
      <a
        href={`/style-memory?project=${encodeURIComponent(projectId)}`}
        style={{
          display: "inline-block",
          fontFamily: "var(--font-sans)",
          fontWeight: 600,
          fontSize: "13px",
          padding: "10px 18px",
          borderRadius: "8px",
          background: "var(--ink)",
          color: "var(--canvas)",
          textDecoration: "none",
        }}
      >
        Retry
      </a>
    </main>
  );
}
