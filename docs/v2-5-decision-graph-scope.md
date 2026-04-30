# Relay V2.5 Decision Graph Scope

Date: 2026-04-30

## Goal

Decision Graph turns the existing project graph projection into an authenticated
workspace surface. It should answer one question quickly: what decisions,
traces, heuristics, and packet snapshots shaped this project?

## First Slice

- Extend `GET /v1/projects/{project_id}/graph` with Style Memory and packet
  evidence nodes:
  - `judgment_trace`
  - `heuristic_proposal`
  - `approved_heuristic`
  - latest `packet_snapshot`
- Add canonical edges:
  - project `includes` all graph nodes
  - proposals `derived_from` origin/source traces
  - approved heuristics `derived_from` origin proposal and source traces
  - packet snapshots `derived_from` approved heuristics, decisions, open
    questions, and source artifacts
- Add a Next route at `/projects/[projectId]/graph`.
- Link Project Explorer to Decision Graph.
- Add Playwright live smoke coverage for authenticated desktop and mobile graph
  rendering.

## Visual Contract

- Graph vocabulary uses neutral ink/mist strokes.
- The magic accent is reserved for the selected or yielded transformation path,
  not for static edge categories.
- The first viewport should show a dense but readable evidence map, not a
  marketing hero.
- Empty state should explain that graph evidence appears after captures,
  decisions, Style Memory, or packet snapshots exist.

## Deferred

- Force-directed layout.
- Dragging, zooming, and graph filtering.
- Full packet snapshot history listing beyond the latest graph evidence node.
- Query-conditioned subgraph retrieval.

## Acceptance

- An onboarded user can open Decision Graph from Project Explorer.
- The graph route renders current project nodes and edges from authenticated
  APIs.
- Live E2E covers Project Explorer -> Decision Graph on desktop Chromium and
  mobile Chrome.
