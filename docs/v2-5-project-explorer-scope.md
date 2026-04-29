# Relay V2.5 Project Explorer Scope

Date: 2026-04-29

## Status

This document defines the first V2.5 product slice after the V2 baseline
closure.

V2.5 starts with Project Explorer because the other deferred workspace surfaces
depend on the same read model:

- Judgment Traces needs trace list and trace detail data.
- Decision Graph needs traces, heuristics, and snapshots represented in graph
  form.
- Packet Builder needs latest snapshot and provenance summaries.
- Public Style Profile later needs cross-project aggregation, which should not
  be designed before a single-project explorer is stable.

## Product Goal

Project Explorer should become the authenticated workspace entry after
onboarding, replacing the current `?project=<id>` dependency with a first-class
project surface.

The V2.5 slice is not a full multi-project dashboard. It is a single-project
explorer for the user's default project with enough read-model shape to support
the next workspace screens.

## Current Baseline

Already available:

- `GET /v1/auth/me` authenticates the browser session.
- `GET /v1/projects/{project_id}` returns note, artifact, decision, open
  question counts plus latest packet id.
- `GET /v1/projects/{project_id}/graph` returns project, note, artifact,
  decision, and open-question graph nodes.
- `GET /v1/projects/{project_id}/packet-snapshots/latest` returns latest packet
  snapshot body and provenance arrays.
- `GET /v1/heuristic-proposals?project_id=...&state=...` returns pending or
  rejected proposal summaries.
- `GET /v1/approved-heuristics?project_id=...` returns approved heuristic
  summaries.
- `/style-memory?project=<project_id>` renders and mutates proposal review
  state.

Important gaps:

- `GET /v1/auth/me` does not expose `default_project_id`, so the web app still
  depends on query-string project selection.
- `ProjectGraph` does not include judgment traces, heuristic proposals,
  approved heuristics, curator jobs, or packet snapshots.
- There is no session-authenticated trace list endpoint.
- There is no Explorer summary endpoint that aggregates counts, recent activity,
  latest snapshot, and Style Memory queue state in one request.
- There is no project list or project switcher contract. This remains out of
  scope for the first V2.5 slice unless the user has no default project.

## Slice Boundary

### In Scope

1. Default project discovery
   - Extend `GET /v1/auth/me` or add a small session-owned project endpoint so
     the web app can discover the user's default project without `?project=`.
   - The response must not expose projects owned by other users.

2. Explorer summary endpoint
   - Add a session-or-admin endpoint for one project:
     `GET /v1/projects/{project_id}/explorer`.
   - It should be a read-only aggregation endpoint for the first Explorer UI.

3. Trace list read model
   - Add a session-or-admin trace list endpoint:
     `GET /v1/projects/{project_id}/judgment-traces?limit=...&cursor=...`.
   - Return compact trace cards first; trace detail can be deferred unless the
     UI needs expansion in the same slice.

4. Graph enrichment
   - Extend the project graph to include:
     - `judgment_trace` nodes
     - `heuristic_proposal` nodes
     - `approved_heuristic` nodes
     - `packet_snapshot` nodes
   - Add edges:
     - proposal `derived_from` trace
     - approved heuristic `derived_from` proposal
     - packet snapshot `includes` approved heuristic / decision / question /
       artifact references

5. Web Project Explorer route
   - Add a Next route such as `/projects/[projectId]`.
   - Root and onboarding-complete users should land on this route once default
     project discovery exists.
   - Style Memory remains accessible as the review-focused surface, but
     Explorer should link to it rather than replace it.

6. QA
   - Backend integration tests for the explorer endpoint, trace list endpoint,
     graph enrichment, and owner authorization.
   - Playwright coverage for authenticated Explorer load on desktop and mobile.
   - Live QA seeding should extend `scripts/qa/run_live_e2e.sh` with at least
     one trace, one pending proposal, one approved heuristic, and one snapshot.

### Out of Scope

- Multi-project switcher beyond the default project.
- Public Style Profile.
- Cross-project style aggregation.
- Real-time multi-agent workspace.
- Full Packet Builder editing or WYSIWYG.
- Decision Graph layout polish beyond providing graph-compatible data.
- Curator confidence/reasoning side panel unless backend emits the fields in
  the same slice.

## Proposed API Contract

### `GET /v1/projects/{project_id}/explorer`

Auth: session cookie or admin bearer.

Response envelope:

```json
{
  "ok": true,
  "command": "relay project explorer",
  "data": {
    "project": {
      "project_id": "proj_123",
      "name": "Personal",
      "status": "active"
    },
    "counts": {
      "notes": 4,
      "artifacts": 2,
      "decisions": 3,
      "open_questions": 1,
      "judgment_traces": 8,
      "pending_proposals": 1,
      "approved_heuristics": 5,
      "rejected_proposals": 2,
      "packet_snapshots": 3
    },
    "latest_snapshot": {
      "snapshot_id": "psnap_123",
      "packet_kind": "handoff",
      "target": "design_doc",
      "task_summary": "Prepare a design handoff",
      "created_at": "2026-04-29T00:00:00Z",
      "public_readable": true
    },
    "style_memory": {
      "next_proposal_id": "hprop_123",
      "next_proposal_text": "Prefer specific recovery actions over generic error states."
    },
    "recent_activity": [
      {
        "kind": "judgment_trace",
        "id": "jtrace_123",
        "title": "Specific recovery action chosen",
        "created_at": "2026-04-29T00:00:00Z"
      }
    ]
  },
  "warnings": []
}
```

Notes:

- Counts should be cheap enough for first load. If count queries become
  expensive, cap or approximate only after measuring.
- `latest_snapshot` is summary only. Full packet content stays on the existing
  latest snapshot endpoint.
- `style_memory.next_proposal_*` is a preview, not a mutation contract.

### `GET /v1/projects/{project_id}/judgment-traces`

Auth: session cookie or admin bearer.

Query:

- `limit`: default 20, max 100
- `cursor`: opaque pagination cursor

Response data:

```json
{
  "items": [
    {
      "trace_id": "jtrace_123",
      "project_id": "proj_123",
      "task_id": "task_1",
      "agent_id": "codex",
      "workflow": "design_handoff",
      "artifact_type": "design_doc",
      "decision": "Show a specific recovery action instead of a generic error.",
      "rationale": "The user needs a next step, not a broad failure state.",
      "source_refs": ["ref #manual"],
      "created_at": "2026-04-29T00:00:00Z"
    }
  ],
  "next_cursor": ""
}
```

### `GET /v1/auth/me`

Preferred V2.5 addition:

```json
{
  "user_id": "user_123",
  "display_name": "hoon_ch",
  "default_project_id": "proj_123",
  "onboarding_completed": true
}
```

If `default_project_id` is omitted for compatibility reasons, add:

`GET /v1/projects/default`

## Web IA

Initial authenticated route behavior:

1. `/` checks `/v1/auth/me`.
2. If unauthenticated, show sign-in.
3. If authenticated but onboarding is incomplete, route to `/onboarding`.
4. If authenticated and a default project exists, route to
   `/projects/{default_project_id}`.
5. `/style-memory?project={project_id}` remains linked from the Explorer's
   Style Memory panel.

First Explorer screen:

- Top rail: existing Relay wordmark and account affordance.
- Left rail: one default project row only for this slice.
- Main panel: recent activity and Style Memory preview.
- Right inspector: counts, latest snapshot summary, Scope Matrix placeholder.
- Empty state: if the project has no traces/proposals/snapshots, show a
  capture-first state, not a fake populated dashboard.

## Implementation Order

1. Add backend read-model types and service methods for Explorer summary.
2. Add repository count/list helpers for traces, proposals, approved
   heuristics, and packet snapshots where missing.
3. Add `GET /v1/projects/{project_id}/explorer`.
4. Add `GET /v1/projects/{project_id}/judgment-traces`.
5. Extend project graph with trace, proposal, approved heuristic, and snapshot
   nodes.
6. Update OpenAPI and `docs/api.md`.
7. Add web `lib/projects.ts` typed wrappers.
8. Add `/projects/[projectId]` route and route authenticated users there.
9. Extend live E2E seed and Playwright coverage.

## Acceptance Criteria

- A completed-onboarding user can reach Project Explorer without manually
  appending `?project=`.
- The Explorer renders real project counts, recent traces, Style Memory preview,
  and latest snapshot summary from authenticated APIs.
- A user cannot read another user's Explorer data by guessing a project id.
- Existing Style Memory approve/reject flows continue to pass.
- Live E2E includes onboarding, Project Explorer, Style Memory, Provider
  Settings, and public snapshot positive/negative routes.
