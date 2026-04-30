import { render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import ProjectExplorerPage from "./page";

const mocks = vi.hoisted(() => ({
  redirect: vi.fn((path: string) => {
    throw new Error(`NEXT_REDIRECT:${path}`);
  }),
  cookies: vi.fn(),
  relayFetch: vi.fn(),
  getProjectExplorer: vi.fn(),
}));

vi.mock("next/navigation", () => ({ redirect: mocks.redirect }));
vi.mock("next/headers", () => ({ cookies: mocks.cookies }));
vi.mock("@/lib/api", () => ({
  RELAY_API_URL: "https://relay.4gly.dev",
  relayFetch: mocks.relayFetch,
}));
vi.mock("@/lib/project-explorer", async () => {
  const actual = await vi.importActual<typeof import("@/lib/project-explorer")>(
    "@/lib/project-explorer",
  );
  return {
    ...actual,
    getProjectExplorer: mocks.getProjectExplorer,
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

const explorer = {
  project: { projectId: "proj_1", name: "Personal", status: "active" },
  counts: {
    notes: 1,
    artifacts: 2,
    decisions: 3,
    openQuestions: 4,
    judgmentTraces: 5,
    pendingProposals: 1,
    approvedHeuristics: 7,
    rejectedProposals: 8,
    packetSnapshots: 1,
  },
  latestSnapshot: {
    snapshotId: "psnap_1",
    packetKind: "handoff",
    target: "design",
    taskSummary: "Design handoff",
    createdAt: "2026-04-29T00:00:00Z",
    publicReadable: true,
    publicToken: "psnap_public_token",
  },
  styleMemory: {
    nextProposalId: "hprop_1",
    nextProposalText: "Prefer specific recovery actions.",
  },
  recentActivity: [
    {
      kind: "judgment_trace",
      id: "trace_1",
      title: "Prefer explicit retry paths.",
      createdAt: "2026-04-29T00:01:00Z",
    },
    {
      kind: "approved_heuristic",
      id: "heur_1",
      title: "Prefer specific recovery actions.",
      createdAt: "2026-04-29T00:00:00Z",
    },
  ],
};

describe("<ProjectExplorerPage>", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mocks.cookies.mockResolvedValue(cookieStore());
  });

  it("renders the project explorer for an onboarded owner", async () => {
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
    mocks.getProjectExplorer.mockResolvedValueOnce(explorer);

    render(await ProjectExplorerPage({ params: Promise.resolve({ projectId: "proj_1" }) }));

    expect(screen.getByRole("heading", { name: "Personal" })).toBeVisible();
    expect(screen.getByRole("link", { name: "Style Memory" })).toHaveAttribute(
      "href",
      "/style-memory?project=proj_1",
    );
    expect(screen.getByRole("link", { name: "Trace Browser" })).toHaveAttribute(
      "href",
      "/projects/proj_1/traces",
    );
    expect(screen.getByRole("link", { name: "Open public snapshot" })).toHaveAttribute(
      "href",
      "/p/psnap_public_token",
    );
    expect(screen.getByRole("link", { name: "Prefer explicit retry paths." })).toHaveAttribute(
      "href",
      "/projects/proj_1/traces?trace=trace_1",
    );
    expect(screen.getAllByText("Prefer specific recovery actions.")).toHaveLength(2);
    expect(screen.getByText("Design handoff")).toBeVisible();
    expect(mocks.getProjectExplorer).toHaveBeenCalledWith("proj_1", {
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

    render(await ProjectExplorerPage({ params: Promise.resolve({ projectId: "proj_1" }) }));

    expect(screen.getByRole("heading", { name: "Sign in first" })).toBeVisible();
    expect(screen.getByRole("link", { name: "Continue with GitHub" })).toHaveAttribute(
      "href",
      "https://relay.4gly.dev/v1/auth/github/start?redirect_to=%2Fprojects%2Fproj_1",
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
      ProjectExplorerPage({ params: Promise.resolve({ projectId: "proj_1" }) }),
    ).rejects.toThrow("NEXT_REDIRECT:/onboarding");
    expect(mocks.redirect).toHaveBeenCalledWith("/onboarding");
  });
});
