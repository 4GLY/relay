import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import {
  ProposalAlreadyResolvedError,
  REJECT_REASON_CODES,
  HeuristicsError,
  listApprovedHeuristics,
  listPendingProposals,
  reviewProposal,
  serializeRejectNotes,
} from "./heuristics";

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

describe("serializeRejectNotes", () => {
  it("serializes code-only when no free text", () => {
    expect(serializeRejectNotes("duplicate")).toBe("reason:duplicate");
  });

  it("appends free text after the code", () => {
    expect(serializeRejectNotes("other", "  this generalizes too widely  ")).toBe(
      "reason:other; this generalizes too widely",
    );
  });

  it("treats whitespace-only free text as missing", () => {
    expect(serializeRejectNotes("wrong", "   \n  ")).toBe("reason:wrong");
  });
});

describe("listPendingProposals", () => {
  it("maps server snake_case payloads to camelCase", async () => {
    const fetchMock = globalThis.fetch as unknown as ReturnType<typeof vi.fn>;
    fetchMock.mockResolvedValueOnce(
      mockResponse({
        ok: true,
        command: "relay heuristic-proposals list",
        data: {
          items: [
            {
              proposal_id: "prop-1",
              project_id: "proj-1",
              heuristic_key: "key-1",
              canonical_text: "do thing",
              normalized_text: "old thing",
              state: "pending",
              source_trace_ids: ["t1", "t2"],
              source_refs: [],
              workflow: "review",
              artifact_type: "go",
              created_at: "2026-04-26T00:00:00Z",
              updated_at: "2026-04-26T00:00:00Z",
            },
          ],
          next_cursor: "cur-1",
        },
        warnings: [],
      }),
    );

    const result = await listPendingProposals("proj-1", { limit: 10 });
    expect(result.items).toHaveLength(1);
    expect(result.items[0].proposalId).toBe("prop-1");
    expect(result.items[0].canonicalText).toBe("do thing");
    expect(result.items[0].sourceTraceIds).toEqual(["t1", "t2"]);
    expect(result.nextCursor).toBe("cur-1");

    const calledUrl = fetchMock.mock.calls[0][0] as string;
    expect(calledUrl).toContain("/v1/heuristic-proposals?");
    expect(calledUrl).toContain("project_id=proj-1");
    expect(calledUrl).toContain("state=pending");
    expect(calledUrl).toContain("limit=10");
  });

  it("throws HeuristicsError on 5xx responses", async () => {
    const fetchMock = globalThis.fetch as unknown as ReturnType<typeof vi.fn>;
    fetchMock.mockResolvedValueOnce(
      mockResponse(
        { ok: false, command: "list", error: { code: "INTERNAL", message: "boom", retryable: true } },
        500,
      ),
    );
    await expect(listPendingProposals("p1")).rejects.toBeInstanceOf(HeuristicsError);
  });
});

describe("listApprovedHeuristics", () => {
  it("maps approved heuristics", async () => {
    const fetchMock = globalThis.fetch as unknown as ReturnType<typeof vi.fn>;
    fetchMock.mockResolvedValueOnce(
      mockResponse({
        ok: true,
        command: "relay approved-heuristics list",
        data: {
          items: [
            {
              heuristic_id: "h1",
              project_id: "proj-1",
              heuristic_key: "k",
              canonical_text: "swan text",
              state: "active",
              source_trace_ids: ["t1"],
              source_refs: [],
              created_at: "2026-04-26T00:00:00Z",
              updated_at: "2026-04-26T00:00:00Z",
            },
          ],
        },
        warnings: [],
      }),
    );

    const result = await listApprovedHeuristics("proj-1");
    expect(result.items[0].heuristicId).toBe("h1");
    expect(result.items[0].canonicalText).toBe("swan text");
  });
});

describe("reviewProposal", () => {
  it("sends correct snake_case payload", async () => {
    const fetchMock = globalThis.fetch as unknown as ReturnType<typeof vi.fn>;
    fetchMock.mockResolvedValueOnce(
      mockResponse({
        ok: true,
        command: "relay heuristic-proposal review",
        data: {
          proposal_id: "prop-1",
          approved_heuristic_id: "h1",
          project_id: "proj-1",
          state: "approved",
        },
        warnings: [],
      }),
    );

    const result = await reviewProposal({
      projectId: "proj-1",
      proposalId: "prop-1",
      action: "approve",
    });

    expect(result.state).toBe("approved");
    expect(result.approvedHeuristicId).toBe("h1");

    const init = fetchMock.mock.calls[0][1] as RequestInit;
    expect(init.method).toBe("POST");
    expect(JSON.parse(init.body as string)).toEqual({
      project_id: "proj-1",
      proposal_id: "prop-1",
      action: "approve",
    });
  });

  it("forwards review_notes when provided", async () => {
    const fetchMock = globalThis.fetch as unknown as ReturnType<typeof vi.fn>;
    fetchMock.mockResolvedValueOnce(
      mockResponse({
        ok: true,
        command: "relay heuristic-proposal review",
        data: { proposal_id: "p1", project_id: "proj-1", state: "rejected" },
        warnings: [],
      }),
    );

    await reviewProposal({
      projectId: "proj-1",
      proposalId: "p1",
      action: "reject",
      reviewNotes: serializeRejectNotes("too_broad"),
    });

    const init = fetchMock.mock.calls[0][1] as RequestInit;
    expect(JSON.parse(init.body as string)).toEqual({
      project_id: "proj-1",
      proposal_id: "p1",
      action: "reject",
      review_notes: "reason:too_broad",
    });
  });

  it("surfaces 409 PROPOSAL_ALREADY_RESOLVED as typed error", async () => {
    const fetchMock = globalThis.fetch as unknown as ReturnType<typeof vi.fn>;
    fetchMock.mockResolvedValueOnce(
      mockResponse(
        {
          ok: false,
          command: "relay heuristic-proposal review",
          error: {
            code: "PROPOSAL_ALREADY_RESOLVED",
            message: "already resolved",
            retryable: false,
          },
        },
        409,
      ),
    );

    await expect(
      reviewProposal({ projectId: "proj-1", proposalId: "p1", action: "approve" }),
    ).rejects.toBeInstanceOf(ProposalAlreadyResolvedError);
  });
});

describe("REJECT_REASON_CODES", () => {
  it("contains the locked taxonomy", () => {
    expect(REJECT_REASON_CODES).toEqual([
      "duplicate",
      "wrong",
      "too_narrow",
      "too_broad",
      "stale",
      "other",
    ]);
  });
});
