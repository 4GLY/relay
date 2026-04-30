# Relay V2.5 Closure

Date: 2026-04-30

## Closure Boundary

V2.5 is closed when the authenticated workspace has the four deferred
DESIGN.md surfaces available in production with live E2E coverage:

- Project Explorer
- Trace Browser
- Decision Graph
- Packet Builder

This closure does not include V3 product bets such as Public Style Profile,
Multi-agent Live Workspace, multi-project style transfer, embed widgets, or
team workspaces.

## Shipped Surfaces

### Project Explorer

- Authenticated workspace entry after onboarding.
- Links core surfaces: Style Memory, Trace Browser, Decision Graph, Packet
  Builder, Provider Settings.
- Main metrics are user-facing: notes, decisions, snapshots.
- Operational/detail counts live in the collapsed Workspace inspector.

### Trace Browser

- Authenticated project trace route.
- Defaults to one featured narrative trace.
- Additional traces are available through a collapsed archive browser.
- `?trace={trace_id}` promotes the selected trace to the featured position,
  including traces that would otherwise live in the archive.
- Empty, sign-in, onboarding redirect, and load error states exist.

### Decision Graph

- Authenticated graph route from Project Explorer.
- Graph projection includes project, notes, artifacts, decisions, questions,
  judgment traces, heuristic proposals, approved heuristics, and latest packet
  snapshot evidence.
- Magic accent is reserved for selected derived paths, not generic graph
  vocabulary.

### Packet Builder

- Authenticated latest-snapshot review route.
- Rendered packet document owns the page.
- Source evidence and publish controls are collapsed by default.
- Public link appears only when the latest snapshot is already public.

## Deferred After V2.5

- Rich text packet editing.
- Draft packet rows.
- Snapshot history browsing.
- User-facing publish/revoke controls.
- Cross-project public style profiles.
- Multi-agent live workspace.
- Embed widgets and viral distribution surfaces.

These are product slices after V2.5, not blockers for V2.5 closure.

## QA Expectations

Closure requires:

- Unit/component coverage for each surface.
- Live E2E coverage for onboarding, Project Explorer, Trace Browser, Decision
  Graph, Packet Builder, Style Memory, Provider Settings, and public snapshot
  routes through `web/e2e/relay-live-smoke.spec.ts`.
- Deployment confirmation through publish workflows, deploy PRs, Argo
  Synced/Healthy, rollout status, and `/healthz`.
