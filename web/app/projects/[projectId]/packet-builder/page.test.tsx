import { render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import PacketBuilderPage from "./page";

const mocks = vi.hoisted(() => ({
  redirect: vi.fn((path: string) => {
    throw new Error(`NEXT_REDIRECT:${path}`);
  }),
  cookies: vi.fn(),
  relayFetch: vi.fn(),
  getLatestPacketSnapshot: vi.fn(),
  pathname: "/projects/proj_1/packet-builder",
  search: "",
}));

vi.mock("next/navigation", () => ({
  redirect: mocks.redirect,
  usePathname: () => mocks.pathname,
  useSearchParams: () => new URLSearchParams(mocks.search),
}));
vi.mock("next/headers", () => ({ cookies: mocks.cookies }));
vi.mock("@/lib/api", () => ({
  RELAY_API_URL: "https://relay.4gly.dev",
  relayFetch: mocks.relayFetch,
}));
vi.mock("@/lib/packet-builder", async () => {
  const actual = await vi.importActual<typeof import("@/lib/packet-builder")>(
    "@/lib/packet-builder",
  );
  return {
    ...actual,
    getLatestPacketSnapshot: mocks.getLatestPacketSnapshot,
  };
});

function cookieStore(value = "relay_session=test") {
  return {
    toString: () => value,
  };
}

function authResponse(status: number, body: unknown, ok = status >= 200 && status < 300) {
  return {
    status,
    ok,
    json: async () => body,
  } as Response;
}

const snapshot = {
  snapshotId: "psnap_1",
  projectId: "proj_1",
  schemaVersion: "relay.packet.v1",
  type: "resume",
  target: "codex",
  taskSummary: "Continue the QA pass",
  renderedBody: "# Relay Packet\n\nUse specific recovery paths.",
  styleCues: [
    {
      heuristicId: "heur_1",
      canonicalText: "Prefer specific recovery actions.",
      whySelected: "matches task",
      sourceSummary: "approved trace",
      sourceRefs: ["trace_1"],
    },
  ],
  supportingNotes: [{ noteId: "note_1", excerpt: "Packet builder scope" }],
  supportingDecisions: [{ decisionId: "dec_1", summary: "Ship graph first" }],
  supportingQuestions: [{ questionId: "q_1", summary: "Where should publish live?" }],
  supportingArtifacts: [{ artifactId: "art_1", sourcePath: "docs/v2-5-packet-builder-wysiwyg-scope.md" }],
  whyIncluded: ["latest snapshot"],
  approvedHeuristicIds: ["heur_1"],
  decisionIds: ["dec_1"],
  openQuestionIds: ["q_1"],
  sourceArtifactIds: ["art_1"],
  missingContext: [],
  publicReadable: true,
  publicToken: "psnap_public",
  createdAt: "2026-04-30T00:00:00Z",
};

describe("<PacketBuilderPage>", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mocks.pathname = "/projects/proj_1/packet-builder";
    mocks.search = "";
    mocks.cookies.mockResolvedValue(cookieStore());
  });

  it("renders the latest packet snapshot document", async () => {
    mocks.relayFetch.mockResolvedValueOnce(
      authResponse(200, {
        ok: true,
        command: "relay auth me",
        data: {
          user_id: "user_1",
          display_name: "Hoon",
          onboarding_complete: true,
          default_project_id: "proj_1",
        },
        warnings: [],
      }),
    );
    mocks.getLatestPacketSnapshot.mockResolvedValueOnce(snapshot);

    render(await PacketBuilderPage({ params: Promise.resolve({ projectId: "proj_1" }) }));

    expect(screen.getByRole("heading", { name: "Compose a handoff." })).toBeVisible();
    expect(screen.getByText(/Use specific recovery paths/i)).toBeVisible();
    expect(screen.getByRole("link", { name: /Open public snapshot/i })).toHaveAttribute(
      "href",
      "/p/psnap_public",
    );
    expect(screen.getByRole("link", { name: "Decision Graph" })).toHaveAttribute(
      "href",
      "/projects/proj_1/graph",
    );
    expect(screen.getByText("Source evidence")).toBeVisible();
    expect(mocks.getLatestPacketSnapshot).toHaveBeenCalledWith("proj_1", {
      headers: { cookie: "relay_session=test" },
    });
  });

  it("shows sign-in when there is no session", async () => {
    mocks.relayFetch.mockResolvedValueOnce(
      authResponse(401, {
        ok: false,
        command: "relay auth me",
        error: { code: "UNAUTHORIZED", message: "missing session", retryable: false },
      }),
    );

    render(await PacketBuilderPage({ params: Promise.resolve({ projectId: "proj_1" }) }));

    expect(screen.getByRole("heading", { name: "Sign in first" })).toBeVisible();
    expect(screen.getByRole("link", { name: "Continue with GitHub" })).toHaveAttribute(
      "href",
      "https://relay.4gly.dev/v1/auth/github/start?redirect_to=%2Fprojects%2Fproj_1%2Fpacket-builder",
    );
  });

  it("redirects authenticated users who still need onboarding", async () => {
    mocks.relayFetch.mockResolvedValueOnce(
      authResponse(200, {
        ok: true,
        command: "relay auth me",
        data: {
          user_id: "user_1",
          display_name: "Hoon",
          onboarding_complete: false,
        },
        warnings: [],
      }),
    );

    await expect(
      PacketBuilderPage({ params: Promise.resolve({ projectId: "proj_1" }) }),
    ).rejects.toThrow("NEXT_REDIRECT:/onboarding");
    expect(mocks.redirect).toHaveBeenCalledWith("/onboarding");
  });
});
