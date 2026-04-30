import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { getProjectGraph, ProjectGraphError } from "./project-graph";

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

describe("getProjectGraph", () => {
  it("maps graph nodes and edges", async () => {
    const fetchMock = globalThis.fetch as unknown as ReturnType<typeof vi.fn>;
    fetchMock.mockResolvedValueOnce(
      mockResponse({
        ok: true,
        command: "relay project graph",
        data: {
          project_id: "proj_1",
          nodes: [
            {
              id: "trace_1",
              kind: "judgment_trace",
              title: "Prefer explicit recovery.",
              workflow: "qa_live",
              artifact_type: "style_memory",
            },
          ],
          edges: [
            {
              type: "derived_from",
              from: "hprop_1",
              to: "trace_1",
              why_included: "origin trace",
            },
          ],
        },
        warnings: [],
      }),
    );

    const result = await getProjectGraph("proj_1", {
      headers: { cookie: "relay_session=test" },
    });

    expect(result.projectId).toBe("proj_1");
    expect(result.nodes[0]).toMatchObject({
      id: "trace_1",
      kind: "judgment_trace",
      artifactType: "style_memory",
    });
    expect(result.edges[0]).toMatchObject({
      type: "derived_from",
      from: "hprop_1",
      to: "trace_1",
      whyIncluded: "origin trace",
    });
    expect(fetchMock.mock.calls[0][0]).toContain("/v1/projects/proj_1/graph");
  });

  it("throws ProjectGraphError on API failures", async () => {
    const fetchMock = globalThis.fetch as unknown as ReturnType<typeof vi.fn>;
    fetchMock.mockResolvedValueOnce(
      mockResponse(
        {
          ok: false,
          command: "relay project graph",
          error: { code: "FORBIDDEN", message: "no", retryable: false },
        },
        403,
      ),
    );

    await expect(getProjectGraph("proj_1")).rejects.toBeInstanceOf(ProjectGraphError);
  });
});
