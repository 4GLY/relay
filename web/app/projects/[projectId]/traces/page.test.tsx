import { render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import TraceBrowserPage from "./page";

const mocks = vi.hoisted(() => ({
  redirect: vi.fn((path: string) => {
    throw new Error(`NEXT_REDIRECT:${path}`);
  }),
  cookies: vi.fn(),
  relayFetch: vi.fn(),
  getProjectTraces: vi.fn(),
}));

vi.mock("next/navigation", () => ({ redirect: mocks.redirect }));
vi.mock("next/headers", () => ({ cookies: mocks.cookies }));
vi.mock("@/lib/api", () => ({
  RELAY_API_URL: "https://relay.4gly.dev",
  relayFetch: mocks.relayFetch,
}));
vi.mock("@/lib/project-traces", async () => {
  const actual = await vi.importActual<typeof import("@/lib/project-traces")>(
    "@/lib/project-traces",
  );
  return {
    ...actual,
    getProjectTraces: mocks.getProjectTraces,
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

const traces = {
  items: [
    {
      traceId: "trace_1",
      projectId: "proj_1",
      taskId: "task_1",
      agentId: "codex",
      workflow: "qa_live",
      artifactType: "style_memory",
      decision: "Prefer explicit recovery actions.",
      rationale: "Authenticated QA needs deterministic evidence.",
      sourceRefs: ["qa:live:e2e"],
      createdAt: "2026-04-29T00:00:00Z",
    },
  ],
};

describe("<TraceBrowserPage>", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mocks.cookies.mockResolvedValue(cookieStore());
  });

  it("renders judgment traces for an onboarded owner", async () => {
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
    mocks.getProjectTraces.mockResolvedValueOnce(traces);

    render(
      await TraceBrowserPage({
        params: Promise.resolve({ projectId: "proj_1" }),
        searchParams: Promise.resolve({ trace: "trace_1" }),
      }),
    );

    expect(screen.getByRole("heading", { name: "Trace Browser" })).toBeVisible();
    expect(screen.getByText("Prefer explicit recovery actions.")).toBeVisible();
    expect(screen.getByText("qa:live:e2e")).toBeVisible();
    expect(screen.getByRole("link", { name: "Project Explorer" })).toHaveAttribute(
      "href",
      "/projects/proj_1",
    );
    expect(mocks.getProjectTraces).toHaveBeenCalledWith("proj_1", {
      headers: { cookie: "relay_session=test" },
      limit: 50,
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

    render(await TraceBrowserPage({ params: Promise.resolve({ projectId: "proj_1" }) }));

    expect(screen.getByRole("heading", { name: "Sign in first" })).toBeVisible();
    expect(screen.getByRole("link", { name: "Continue with GitHub" })).toHaveAttribute(
      "href",
      "https://relay.4gly.dev/v1/auth/github/start?redirect_to=%2Fprojects%2Fproj_1%2Ftraces",
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
      TraceBrowserPage({ params: Promise.resolve({ projectId: "proj_1" }) }),
    ).rejects.toThrow("NEXT_REDIRECT:/onboarding");
    expect(mocks.redirect).toHaveBeenCalledWith("/onboarding");
  });
});
