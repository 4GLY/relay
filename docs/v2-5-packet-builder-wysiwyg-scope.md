# Relay V2.5 Packet Builder WYSIWYG Scope

Date: 2026-04-30

## Goal

Packet Builder WYSIWYG is the next workspace surface after Decision Graph. It
turns Relay's packet snapshot machinery from a read-only continuation artifact
into an inspectable authoring surface.

The first version should let a user review the latest packet body, understand
which evidence shaped it, and publish or keep it private without exposing source
panels by default.

## Entry Criteria

- Decision Graph is deployed and live QA covers authenticated graph rendering.
- `GET /v1/projects/{project_id}/packet-snapshots/latest` remains the canonical
  packet read API.
- Snapshot publish/revoke routes remain Go-canonical.

## First Slice

- Add `/projects/[projectId]/packet-builder`.
- Show the latest packet snapshot body as the primary document.
- Keep source/evidence panels collapsed by default.
- Provide explicit actions:
  - open public snapshot when already public
  - publish snapshot
  - revoke public access
  - return to Project Explorer / Decision Graph
- Add Playwright coverage for latest snapshot rendering and publish/revoke
  recovery states.

## Visual Contract

- The packet document owns the page. Source evidence is an inspector, not the
  default main column.
- Avoid graph-like magic color vocabulary. Magic accent is reserved for the
  publish/revoke transformation moment.
- Long packet bodies must remain readable at mobile width.

## Deferred

- Rich text editing.
- Multi-snapshot history browsing.
- Inline source diffing.
- Regenerate packet with model controls.
- Collaborative review.

## Open Implementation Questions

- Whether publish/revoke belongs directly in this WYSIWYG surface or remains
  admin/API-only for one more slice.
- Whether packet editing creates a new snapshot row or a draft row.
- Whether OG image regeneration should be synchronous with publish or queued.
