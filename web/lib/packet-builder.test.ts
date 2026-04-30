import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { getLatestPacketSnapshot, PacketBuilderError } from "./packet-builder";

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

describe("getLatestPacketSnapshot", () => {
  it("maps latest packet snapshot evidence", async () => {
    const fetchMock = globalThis.fetch as unknown as ReturnType<typeof vi.fn>;
    fetchMock.mockResolvedValueOnce(
      mockResponse({
        ok: true,
        command: "relay latest packet snapshot",
        data: {
          snapshot_id: "psnap_1",
          project_id: "proj_1",
          schema_version: "relay.packet.v1",
          type: "resume",
          target: "codex",
          task_summary: "Continue the QA pass",
          rendered_body: "# Relay Packet\n\nUse specific recovery paths.",
          style_cues: [
            {
              heuristic_id: "heur_1",
              heuristic_key: "qa.recovery",
              canonical_text: "Prefer specific recovery actions.",
              why_selected: "matches task",
              source_summary: "approved from trace",
              source_refs: ["trace_1"],
            },
          ],
          supporting_notes: [{ note_id: "note_1", source: "docs", excerpt: "Packet builder scope" }],
          supporting_decisions: [{ decision_id: "dec_1", summary: "Ship graph first" }],
          supporting_questions: [{ question_id: "q_1", summary: "Where should publish live?" }],
          supporting_artifacts: [{ artifact_id: "art_1", type: "doc", source_path: "docs/scope.md" }],
          why_included: ["latest snapshot"],
          approved_heuristic_ids: ["heur_1"],
          decision_ids: ["dec_1"],
          open_question_ids: ["q_1"],
          source_artifact_ids: ["art_1"],
          missing_context: [],
          public_readable: true,
          public_token: "psnap_public",
          created_at: "2026-04-30T00:00:00Z",
        },
        warnings: [],
      }),
    );

    const result = await getLatestPacketSnapshot("proj_1", {
      headers: { cookie: "relay_session=test" },
    });

    expect(result.snapshotId).toBe("psnap_1");
    expect(result.renderedBody).toContain("Relay Packet");
    expect(result.styleCues[0]).toMatchObject({ heuristicId: "heur_1", sourceRefs: ["trace_1"] });
    expect(result.supportingArtifacts[0].sourcePath).toBe("docs/scope.md");
    expect(result.publicToken).toBe("psnap_public");
    expect(fetchMock.mock.calls[0][0]).toContain("/v1/projects/proj_1/packet-snapshots/latest");
  });

  it("throws PacketBuilderError on API failures", async () => {
    const fetchMock = globalThis.fetch as unknown as ReturnType<typeof vi.fn>;
    fetchMock.mockResolvedValueOnce(
      mockResponse(
        {
          ok: false,
          command: "relay latest packet snapshot",
          error: { code: "NOT_FOUND", message: "no packet snapshot", retryable: false },
        },
        404,
      ),
    );

    await expect(getLatestPacketSnapshot("proj_1")).rejects.toBeInstanceOf(PacketBuilderError);
  });
});
