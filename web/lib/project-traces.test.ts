import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { getProjectTraces, ProjectTracesError } from "./project-traces";

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

describe("getProjectTraces", () => {
  it("maps compact judgment traces", async () => {
    const fetchMock = globalThis.fetch as unknown as ReturnType<typeof vi.fn>;
    fetchMock.mockResolvedValueOnce(
      mockResponse({
        ok: true,
        command: "relay judgment-traces list",
        data: {
          items: [
            {
              trace_id: "trace_1",
              project_id: "proj_1",
              task_id: "task_1",
              agent_id: "codex",
              workflow: "qa_live",
              artifact_type: "style_memory",
              decision: "Prefer explicit recovery actions.",
              rationale: "Specific actions are easier to retry.",
              source_refs: ["qa:live:e2e"],
              created_at: "2026-04-29T00:00:00Z",
            },
          ],
        },
        warnings: [],
      }),
    );

    const result = await getProjectTraces("proj_1", {
      headers: { cookie: "relay_session=test" },
      limit: 25,
    });

    expect(result.items[0]).toMatchObject({
      traceId: "trace_1",
      decision: "Prefer explicit recovery actions.",
      sourceRefs: ["qa:live:e2e"],
    });
    expect(fetchMock.mock.calls[0][0]).toContain("/v1/projects/proj_1/judgment-traces?limit=25");
    expect(fetchMock.mock.calls[0][1]).toMatchObject({
      method: "GET",
      headers: { cookie: "relay_session=test" },
    });
  });

  it("throws ProjectTracesError on API failures", async () => {
    const fetchMock = globalThis.fetch as unknown as ReturnType<typeof vi.fn>;
    fetchMock.mockResolvedValueOnce(
      mockResponse(
        {
          ok: false,
          command: "relay judgment-traces list",
          error: { code: "FORBIDDEN", message: "no", retryable: false },
        },
        403,
      ),
    );

    await expect(getProjectTraces("proj_1")).rejects.toBeInstanceOf(ProjectTracesError);
  });
});
