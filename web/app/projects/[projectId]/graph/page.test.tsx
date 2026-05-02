import { render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import DecisionGraphPage from "./page";

const mocks = vi.hoisted(() => ({
  redirect: vi.fn((path: string) => {
    throw new Error(`NEXT_REDIRECT:${path}`);
  }),
  cookies: vi.fn(),
  relayFetch: vi.fn(),
  getProjectGraph: vi.fn(),
  pathname: "/projects/proj_1/graph",
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
vi.mock("@/lib/project-graph", async () => {
  const actual = await vi.importActual<typeof import("@/lib/project-graph")>(
    "@/lib/project-graph",
  );
  return {
    ...actual,
    getProjectGraph: mocks.getProjectGraph,
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

const graph = {
  projectId: "proj_1",
  nodes: [
    { id: "proj_1", kind: "project", title: "Personal" },
    { id: "trace_1", kind: "judgment_trace", title: "Prefer explicit recovery." },
    { id: "hprop_1", kind: "heuristic_proposal", title: "Show recovery action.", state: "pending" },
    { id: "heur_1", kind: "approved_heuristic", title: "Show recovery action.", state: "approved" },
    { id: "psnap_1", kind: "packet_snapshot", title: "Decision Graph snapshot", packetKind: "handoff" },
  ],
  edges: [
    { type: "includes", from: "proj_1", to: "trace_1" },
    { type: "derived_from", from: "hprop_1", to: "trace_1" },
    { type: "derived_from", from: "heur_1", to: "hprop_1" },
    { type: "derived_from", from: "psnap_1", to: "heur_1" },
  ],
};

describe("<DecisionGraphPage>", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mocks.pathname = "/projects/proj_1/graph";
    mocks.search = "";
    mocks.cookies.mockResolvedValue(cookieStore());
  });

  it("renders the decision graph for an onboarded owner", async () => {
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
    mocks.getProjectGraph.mockResolvedValueOnce(graph);

    render(await DecisionGraphPage({ params: Promise.resolve({ projectId: "proj_1" }) }));

    expect(screen.getByRole("heading", { name: "Decision Graph" })).toBeVisible();
    expect(screen.getByText("Prefer explicit recovery.")).toBeVisible();
    const projectExplorerLink = screen
      .getAllByRole("link", { name: /Project Explorer/ })
      .find((link) => link.getAttribute("href") === "/projects/proj_1");
    expect(projectExplorerLink).toHaveAttribute(
      "href",
      "/projects/proj_1",
    );
    expect(mocks.getProjectGraph).toHaveBeenCalledWith("proj_1", {
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

    render(await DecisionGraphPage({ params: Promise.resolve({ projectId: "proj_1" }) }));

    expect(screen.getByRole("heading", { name: "Sign in first" })).toBeVisible();
    expect(screen.getByRole("link", { name: "Continue with GitHub" })).toHaveAttribute(
      "href",
      "https://relay.4gly.dev/v1/auth/github/start?redirect_to=%2Fprojects%2Fproj_1%2Fgraph",
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
      DecisionGraphPage({ params: Promise.resolve({ projectId: "proj_1" }) }),
    ).rejects.toThrow("NEXT_REDIRECT:/onboarding");
    expect(mocks.redirect).toHaveBeenCalledWith("/onboarding");
  });
});
