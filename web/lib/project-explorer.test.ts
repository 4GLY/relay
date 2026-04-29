import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { getProjectExplorer, ProjectExplorerError } from "./project-explorer";

const originalFetch = globalThis.fetch;

function mockResponse(body: unknown, status = 200): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { "content-type": "application/json" },
  });
}

beforeEach(() => {
  globalThis.fetch = vi.fn();
});

afterEach(() => {
  globalThis.fetch = originalFetch;
  vi.clearAllMocks();
});

describe("getProjectExplorer", () => {
  it("maps the project explorer read model", async () => {
    const fetchMock = globalThis.fetch as unknown as ReturnType<typeof vi.fn>;
    fetchMock.mockResolvedValueOnce(
      mockResponse({
        ok: true,
        command: "relay project explorer",
        data: {
          project: { project_id: "proj_1", name: "Personal", status: "active" },
          counts: {
            notes: 1,
            artifacts: 2,
            decisions: 3,
            open_questions: 4,
            judgment_traces: 5,
            pending_proposals: 6,
            approved_heuristics: 7,
            rejected_proposals: 8,
            packet_snapshots: 9,
          },
          latest_snapshot: {
            snapshot_id: "psnap_1",
            packet_kind: "handoff",
            target: "design",
            task_summary: "handoff summary",
            created_at: "2026-04-29T00:00:00Z",
            public_readable: true,
          },
          style_memory: {
            next_proposal_id: "hprop_1",
            next_proposal_text: "Prefer specific recovery actions.",
          },
          recent_activity: [
            {
              kind: "approved_heuristic",
              id: "heur_1",
              title: "Prefer specific recovery actions.",
              created_at: "2026-04-29T00:00:00Z",
            },
          ],
        },
        warnings: [],
      }),
    );

    const result = await getProjectExplorer("proj_1", {
      headers: { cookie: "relay_session=test" },
    });

    expect(result.project.projectId).toBe("proj_1");
    expect(result.counts.openQuestions).toBe(4);
    expect(result.counts.pendingProposals).toBe(6);
    expect(result.latestSnapshot?.snapshotId).toBe("psnap_1");
    expect(result.styleMemory.nextProposalId).toBe("hprop_1");
    expect(result.recentActivity[0].kind).toBe("approved_heuristic");

    expect(fetchMock.mock.calls[0][0]).toContain("/v1/projects/proj_1/explorer");
    expect(fetchMock.mock.calls[0][1]).toMatchObject({
      method: "GET",
      headers: { cookie: "relay_session=test" },
    });
  });

  it("throws ProjectExplorerError on API failures", async () => {
    const fetchMock = globalThis.fetch as unknown as ReturnType<typeof vi.fn>;
    fetchMock.mockResolvedValueOnce(
      mockResponse(
        {
          ok: false,
          command: "relay project explorer",
          error: { code: "FORBIDDEN", message: "no", retryable: false },
        },
        403,
      ),
    );

    await expect(getProjectExplorer("proj_1")).rejects.toBeInstanceOf(ProjectExplorerError);
  });
});
