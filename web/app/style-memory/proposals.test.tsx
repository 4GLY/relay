import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { Proposals } from "./proposals";
import type { ApprovedHeuristic, PendingProposal } from "@/lib/heuristics";

const fixturePending: PendingProposal[] = [
  {
    proposalId: "p-1",
    projectId: "proj",
    heuristicKey: "k1",
    canonicalText: "always prefer named return values in go funcs >2 args",
    normalizedText: "use named returns",
    state: "pending",
    sourceTraceIds: ["t1", "t2", "t3", "t4"],
    sourceRefs: [],
    workflow: "refactor",
    artifactType: "go",
    createdAt: "2026-04-26T00:00:00Z",
    updatedAt: "2026-04-26T00:00:00Z",
  },
  {
    proposalId: "p-2",
    projectId: "proj",
    heuristicKey: "k2",
    canonicalText: "flag any only outside test paths",
    normalizedText: "flag any everywhere",
    state: "pending",
    sourceTraceIds: ["t1"],
    sourceRefs: [],
    workflow: "review",
    artifactType: "ts",
    createdAt: "2026-04-26T01:00:00Z",
    updatedAt: "2026-04-26T01:00:00Z",
  },
  {
    proposalId: "p-3",
    projectId: "proj",
    heuristicKey: "k3",
    canonicalText: "third proposal",
    state: "pending",
    sourceTraceIds: [],
    sourceRefs: [],
    workflow: "ship",
    artifactType: "yaml",
    createdAt: "2026-04-26T02:00:00Z",
    updatedAt: "2026-04-26T02:00:00Z",
  },
];

const fixtureApproved: ApprovedHeuristic[] = [
  {
    heuristicId: "h-1",
    projectId: "proj",
    heuristicKey: "ka1",
    canonicalText: "approved one",
    state: "active",
    sourceTraceIds: [],
    sourceRefs: [],
    createdAt: "2026-04-25T00:00:00Z",
    updatedAt: "2026-04-25T00:00:00Z",
  },
];

const fixtureRejected: PendingProposal[] = [
  {
    proposalId: "r-1",
    projectId: "proj",
    heuristicKey: "kr1",
    canonicalText: "rejected one",
    state: "rejected",
    sourceTraceIds: [],
    sourceRefs: [],
    reviewNotes: "reason:stale",
    createdAt: "2026-04-25T00:00:00Z",
    updatedAt: "2026-04-25T00:00:00Z",
  },
];

beforeEach(() => {
  globalThis.fetch = vi.fn();
  window.localStorage.clear();
});

afterEach(() => {
  vi.clearAllMocks();
  vi.useRealTimers();
});

function renderProposals(overrides: Partial<Parameters<typeof Proposals>[0]> = {}) {
  return render(
    <Proposals
      projectId="proj"
      initialPending={fixturePending}
      initialApproved={fixtureApproved}
      initialRejected={fixtureRejected}
      approvedFetchFailed={false}
      rejectedFetchFailed={false}
      userId="user-1"
      userDisplayName="hoon"
      {...overrides}
    />,
  );
}

function mockFetchOk(body: unknown) {
  (globalThis.fetch as unknown as ReturnType<typeof vi.fn>).mockResolvedValueOnce(
    new Response(JSON.stringify(body), {
      status: 200,
      headers: { "content-type": "application/json" },
    }),
  );
}

function mockFetch409() {
  (globalThis.fetch as unknown as ReturnType<typeof vi.fn>).mockResolvedValueOnce(
    new Response(
      JSON.stringify({
        ok: false,
        command: "relay heuristic-proposal review",
        error: { code: "PROPOSAL_ALREADY_RESOLVED", message: "already", retryable: false },
      }),
      { status: 409, headers: { "content-type": "application/json" } },
    ),
  );
}

function mockPendingList(items: unknown[]) {
  mockFetchOk({
    ok: true,
    command: "relay heuristic-proposals list",
    data: { items },
    warnings: [],
  });
}

function mockApprovedList(items: unknown[]) {
  mockFetchOk({
    ok: true,
    command: "relay approved-heuristics list",
    data: { items },
    warnings: [],
  });
}

function mockRejectedList(items: unknown[]) {
  mockFetchOk({
    ok: true,
    command: "relay heuristic-proposals list",
    data: { items },
    warnings: [],
  });
}

function serverApproved(overrides: Record<string, unknown> = {}) {
  return {
    heuristic_id: "h-2",
    project_id: "proj",
    origin_proposal_id: "p-1",
    workflow: "review",
    artifact_type: "ts",
    heuristic_key: "k1",
    canonical_text: "approved from refetch",
    state: "active",
    source_trace_ids: [],
    source_refs: [],
    created_at: "2026-04-26T03:00:00Z",
    updated_at: "2026-04-26T03:00:00Z",
    ...overrides,
  };
}

function serverRejected(overrides: Record<string, unknown> = {}) {
  return {
    proposal_id: "p-1",
    project_id: "proj",
    workflow: "review",
    artifact_type: "ts",
    heuristic_key: "k1",
    canonical_text: "rejected from refetch",
    state: "rejected",
    source_trace_ids: [],
    source_refs: [],
    review_notes: "reason:stale",
    created_at: "2026-04-26T03:00:00Z",
    updated_at: "2026-04-26T03:00:00Z",
    ...overrides,
  };
}

describe("<Proposals> hero + queue render", () => {
  it("renders the highest-confidence card as hero and the rest as queued", () => {
    renderProposals();
    expect(screen.getByRole("heading", { name: "Style Memory" })).toBeInTheDocument();
    expect(screen.getByText(/queued · resolve hero first/i)).toBeInTheDocument();
    expect(screen.getByTestId("approve-p-1")).toBeInTheDocument();
    expect(screen.queryByTestId("approve-p-2")).toBeNull();
  });
});

describe("approve action", () => {
  it("triggers POST /v1/heuristic-proposals/review with action=approve", async () => {
    const user = userEvent.setup();
    mockFetchOk({
      ok: true,
      command: "relay heuristic-proposal review",
      data: { proposal_id: "p-1", project_id: "proj", state: "approved" },
      warnings: [],
    });
    mockApprovedList([serverApproved()]);
    renderProposals();
    await user.click(screen.getByTestId("approve-p-1"));
    await waitFor(() => expect(globalThis.fetch).toHaveBeenCalled());
    const fetchMock = globalThis.fetch as unknown as ReturnType<typeof vi.fn>;
    const [, init] = fetchMock.mock.calls[0];
    expect((init as RequestInit).method).toBe("POST");
    expect(JSON.parse((init as RequestInit).body as string)).toEqual({
      project_id: "proj",
      proposal_id: "p-1",
      action: "approve",
    });
  });

  it("refreshes the approved list after a successful approval", async () => {
    const user = userEvent.setup();
    mockFetchOk({
      ok: true,
      command: "relay heuristic-proposal review",
      data: {
        proposal_id: "p-1",
        approved_heuristic_id: "h-2",
        project_id: "proj",
        state: "approved",
      },
      warnings: [],
    });
    mockApprovedList([serverApproved()]);
    renderProposals();

    await user.click(screen.getByTestId("approve-p-1"));
    await waitFor(() => expect(globalThis.fetch).toHaveBeenCalledTimes(2));
    const fetchMock = globalThis.fetch as unknown as ReturnType<typeof vi.fn>;
    expect(fetchMock.mock.calls[1][0]).toContain("/v1/approved-heuristics?");

    await user.click(screen.getByRole("tab", { name: /approved1/i }));
    expect(await screen.findByText("approved from refetch")).toBeInTheDocument();
  });

  it("shows 'Already resolved' toast on 409", async () => {
    const user = userEvent.setup();
    mockFetch409();
    renderProposals();
    await user.click(screen.getByTestId("approve-p-1"));
    await waitFor(() =>
      expect(screen.getByText(/already resolved/i)).toBeInTheDocument(),
    );
  });
});

describe("reject overlay", () => {
  it("disables submit until exactly one chip is selected", async () => {
    const user = userEvent.setup();
    renderProposals();
    await user.click(screen.getAllByText("Reject")[0]);
    const submit = screen.getByTestId("reject-submit") as HTMLButtonElement;
    expect(submit.disabled).toBe(true);
    await user.click(screen.getByTestId("reject-chip-duplicate"));
    expect((screen.getByTestId("reject-submit") as HTMLButtonElement).disabled).toBe(false);
  });

  it("requires 10-char min for 'other' free text", async () => {
    const user = userEvent.setup();
    renderProposals();
    await user.click(screen.getAllByText("Reject")[0]);
    await user.click(screen.getByTestId("reject-chip-other"));

    const submit = screen.getByTestId("reject-submit") as HTMLButtonElement;
    expect(submit.disabled).toBe(true);

    const ta = screen.getByTestId("reject-other-textarea") as HTMLTextAreaElement;
    await user.type(ta, "too short");
    expect((screen.getByTestId("reject-submit") as HTMLButtonElement).disabled).toBe(true);

    await user.type(ta, " adding more chars now");
    expect((screen.getByTestId("reject-submit") as HTMLButtonElement).disabled).toBe(false);
  });

  it("refreshes the rejected list after a successful rejection", async () => {
    const user = userEvent.setup();
    mockFetchOk({
      ok: true,
      command: "relay heuristic-proposal review",
      data: { proposal_id: "p-1", project_id: "proj", state: "rejected" },
      warnings: [],
    });
    mockRejectedList([serverRejected()]);
    renderProposals({ initialRejected: [] });

    await user.click(screen.getAllByText("Reject")[0]);
    await user.click(screen.getByTestId("reject-chip-stale"));
    await user.click(screen.getByTestId("reject-submit"));

    await waitFor(() => expect(globalThis.fetch).toHaveBeenCalledTimes(2));
    const fetchMock = globalThis.fetch as unknown as ReturnType<typeof vi.fn>;
    expect(fetchMock.mock.calls[1][0]).toContain("/v1/heuristic-proposals?");
    expect(fetchMock.mock.calls[1][0]).toContain("state=rejected");

    await user.click(screen.getByRole("tab", { name: /rejected1/i }));
    expect(await screen.findByText("rejected from refetch")).toBeInTheDocument();
    expect(screen.getByText("reason:stale")).toBeInTheDocument();
  });
});

describe("view toggle persistence", () => {
  it("persists view choice to localStorage", async () => {
    const user = userEvent.setup();
    renderProposals();
    const toggle = screen.getByTestId("view-toggle");
    expect(window.localStorage.getItem("relay-style-memory-view")).toBe("single");
    await user.click(toggle);
    await waitFor(() =>
      expect(window.localStorage.getItem("relay-style-memory-view")).toBe("batch"),
    );
  });

  it("restores 'batch' from localStorage on mount", async () => {
    window.localStorage.setItem("relay-style-memory-view", "batch");
    renderProposals();
    // Batch mode: hint bar with j/k legend appears.
    await waitFor(() => expect(screen.getByText(/navigate/i)).toBeInTheDocument());
  });
});

describe("cross-tab refresh", () => {
  it("refreshes pending, approved, and rejected counts when the tab regains focus", async () => {
    const user = userEvent.setup();
    mockPendingList([]);
    mockApprovedList([serverApproved()]);
    mockRejectedList([]);
    renderProposals({ initialApproved: [], initialRejected: fixtureRejected });

    await user.click(screen.getByRole("tab", { name: /rejected1/i }));
    fireEvent.focus(window);

    await waitFor(() => expect(globalThis.fetch).toHaveBeenCalledTimes(3));
    const fetchMock = globalThis.fetch as unknown as ReturnType<typeof vi.fn>;
    expect(fetchMock.mock.calls[0][0]).toContain("state=pending");
    expect(fetchMock.mock.calls[1][0]).toContain("/v1/approved-heuristics?");
    expect(fetchMock.mock.calls[2][0]).toContain("state=rejected");

    await user.click(screen.getByRole("tab", { name: /approved1/i }));
    expect(await screen.findByText("approved from refetch")).toBeInTheDocument();
  });
});

describe("keyboard shortcuts (Contract C)", () => {
  it("hides the shortcut hint on mobile-width screens", async () => {
    const original = window.matchMedia;
    window.matchMedia = vi.fn().mockImplementation((query: string) => ({
      matches: query !== "(min-width: 640px)",
      media: query,
      onchange: null,
      addListener: vi.fn(),
      removeListener: vi.fn(),
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      dispatchEvent: vi.fn(),
    })) as unknown as typeof window.matchMedia;

    try {
      window.localStorage.setItem("relay-style-memory-view", "batch");
      renderProposals();

      await waitFor(() => expect(screen.queryByText(/navigate/i)).not.toBeInTheDocument());
    } finally {
      window.matchMedia = original;
    }
  });

  it("'j' moves focus to next card in batch mode", async () => {
    window.localStorage.setItem("relay-style-memory-view", "batch");
    renderProposals();
    await waitFor(() => expect(screen.getByText(/navigate/i)).toBeInTheDocument());

    fireEvent.keyDown(window, { key: "j" });
    await waitFor(() => {
      const focused = document.querySelector('[data-focused="true"]');
      expect(focused?.getAttribute("data-card")).toBe("p-2");
    });
  });

  it("'a' approves the focused card", async () => {
    window.localStorage.setItem("relay-style-memory-view", "batch");
    mockFetchOk({
      ok: true,
      command: "relay heuristic-proposal review",
      data: { proposal_id: "p-1", project_id: "proj", state: "approved" },
      warnings: [],
    });
    mockApprovedList([serverApproved()]);
    renderProposals();
    await waitFor(() => expect(screen.getByText(/navigate/i)).toBeInTheDocument());

    fireEvent.keyDown(window, { key: "a" });
    await waitFor(() => expect(globalThis.fetch).toHaveBeenCalled());
  });

  it("ignores shortcut when textarea is focused (INPUT focus exception)", async () => {
    const user = userEvent.setup();
    window.localStorage.setItem("relay-style-memory-view", "batch");
    renderProposals();
    await waitFor(() => expect(screen.getByText(/navigate/i)).toBeInTheDocument());

    // Open reject overlay on first card to mount textarea.
    await user.click(screen.getAllByText("Reject")[0]);
    await user.click(screen.getByTestId("reject-chip-other"));
    const ta = screen.getByTestId("reject-other-textarea") as HTMLTextAreaElement;
    ta.focus();

    // Pressing 'a' inside the textarea must not call fetch (no approve).
    fireEvent.keyDown(ta, { key: "a", target: ta });
    await new Promise((r) => setTimeout(r, 30));
    expect(globalThis.fetch).not.toHaveBeenCalled();
  });
});

describe("reduced motion", () => {
  it("renders without throwing when prefers-reduced-motion is set", () => {
    const original = window.matchMedia;
    window.matchMedia = vi.fn().mockReturnValue({
      matches: true,
      media: "(prefers-reduced-motion: reduce)",
      onchange: null,
      addListener: vi.fn(),
      removeListener: vi.fn(),
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      dispatchEvent: vi.fn(),
    }) as unknown as typeof window.matchMedia;

    expect(() => renderProposals()).not.toThrow();
    expect(screen.getByTestId("approve-p-1")).toBeInTheDocument();
    window.matchMedia = original;
  });
});
