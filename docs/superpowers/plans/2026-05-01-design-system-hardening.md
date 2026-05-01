# Design System Hardening Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use `superpowers:executing-plans`
> or `superpowers:subagent-driven-development` to implement this plan task-by-task.
> Steps use checkbox (`- [ ]`) syntax for tracking. Keep behavior and API
> contracts stable while reducing page-local styling.

Created: 2026-05-01
Branch: `main`
Mode: `superpowers:writing-plans` via local planning workflow

## Plan Summary

Convert the current Relay UI kit application from a visual screen pass into a
durable design system implementation.

The previous pass successfully aligned the visible product language across the
main internal screens, but many pages still encode design decisions as
page-local `React.CSSProperties`. This plan extracts the repeated patterns into
stable Relay primitives, migrates target screens to those primitives, documents
the component contracts, and verifies the result with unit/build/live browser
checks.

## Product Intent

Relay should feel like a calm, dense engineering workspace:

- `Face`: capture and inspect unresolved work.
- `Dissect`: expose traces and decision evidence.
- `Refine`: approve style-memory proposals.
- `Transform`: compose packets, manage keys, and publish snapshots.

The design system should make this language reusable rather than copied per
route. A future screen should be buildable by composing Relay primitives instead
of inventing new page-local typography, cards, buttons, badges, or form styles.

## Current State

Already shipped:

- Self-hosted Fraunces, Nunito, and JetBrains Mono.
- Shared `RelayTopRail` and `RelayAppShell`.
- CSS tokens for color, type, radius, focus, and halo elevation.
- Shared shell classes for top rail, transform ribbon, rail, cards, badges, and
  responsive shell behavior.
- UI-kit language applied to:
  - Root/auth entry
  - Onboarding
  - Project Explorer
  - Trace Browser
  - Decision Graph
  - Packet Builder
  - Style Memory
  - Provider Settings
  - API Key Settings
- Live E2E passed after deploy against `https://relay.4gly.dev`.

Remaining gap:

- `web/components/relay-app-shell.tsx` holds shell primitives, but core page
  primitives such as page heads, tabs, cards, buttons, fields, status badges,
  metric tiles, empty states, and feedback callouts are not first-class React
  APIs yet.
- Page-local inline styles remain heavily used in:
  - `web/app/projects/[projectId]/page.tsx`
  - `web/app/projects/[projectId]/traces/page.tsx`
  - `web/app/projects/[projectId]/graph/page.tsx`
  - `web/app/projects/[projectId]/packet-builder/page.tsx`
  - `web/app/style-memory/proposals.tsx`
  - `web/app/onboarding/page.tsx`
  - `web/app/onboarding/onboarding-client.tsx`
  - `web/app/settings/providers/provider-settings-client.tsx`
  - `web/app/settings/api-keys/api-key-settings-client.tsx`
- Settings and onboarding duplicate card, heading, button, field, and feedback
  styling instead of sharing the same form primitives.
- The design system is documented only indirectly through implementation and the
  previous UI kit plan.

## Non-Goals

- Do not redesign the product language again.
- Do not change route behavior, API payloads, auth behavior, key masking,
  provider credential storage, or public snapshot behavior.
- Do not add Storybook unless explicitly chosen later; this plan prepares the
  component boundary so Storybook can be added cleanly.
- Do not chase screenshot-perfect matching for mobile. Mobile must remain
  readable, non-overlapping, and navigable.

## Target Architecture

Create a small component layer under `web/components/relay/` and keep
`web/components/relay-app-shell.tsx` either as a compatibility export or a thin
re-export.

Target primitives:

- `RelayTopRail`
- `RelayAppShell`
- `RelayPageHead`
- `RelayTabs`
- `RelayCard`
- `RelayMetricTile`
- `RelayButton`
- `RelayLinkButton`
- `RelaySourceChip`
- `RelayStatusBadge`
- `RelayField`
- `RelayTextInput`
- `RelayFeedback`
- `RelayEmptyState`
- `RelayListRow`
- `RelayMetaGrid`

CSS should live in `web/app/globals.css` using `relay-*` class names and the
existing token names. React components should expose semantic variants rather
than style objects.

## Implementation Tasks

### 1. Primitive Inventory And Contracts

- [x] Count page-local style objects and group them by pattern:
  - page heads
  - cards/panels
  - buttons/links
  - tabs
  - metric tiles
  - status badges
  - form fields
  - feedback/error states
  - empty states
  - meta grids/lists
- [x] Define TypeScript props for each primitive before migrating pages.
- [x] Keep class names stable and readable, using `relay-*` prefixes only.
- [x] Add `data-variant` or explicit variant props where the design language has
  real variants, not one-off styling.
- [x] Preserve accessible names, heading levels, `aria-live`, `role="alert"`,
  `aria-current`, and form labels.

### 2. Extract Core Components

- [x] Create `web/components/relay/shell.tsx` for shell exports, or move the
  existing shell there and keep `relay-app-shell.tsx` as a re-export.
- [x] Create `web/components/relay/page.tsx`:
  - [x] `RelayPageHead`
  - [x] `RelayPageActions`
  - [x] `RelayPageKicker`
- [x] Create `web/components/relay/card.tsx`:
  - [x] `RelayCard`
  - [x] `RelayCardHeader`
  - [x] `RelayCardTitle`
  - [x] `RelayCardKicker`
- [x] Create `web/components/relay/actions.tsx`:
  - [x] `RelayButton`
  - [x] `RelayLinkButton`
- [x] Create `web/components/relay/data-display.tsx`:
  - [x] `RelayTabs`
  - [x] `RelayMetricTile`
  - [x] `RelayStatusBadge`
  - [x] `RelaySourceChip`
  - [x] `RelayMetaGrid`
  - [x] `RelayListRow`
- [x] Create `web/components/relay/forms.tsx`:
  - [x] `RelayField`
  - [x] `RelayTextInput`
  - [x] `RelayFeedback`
  - [x] `RelayEmptyState`
- [x] Create `web/components/relay/index.ts` for public component exports.
- [x] Move shared CSS into a clearly marked `@layer components` section in
  `web/app/globals.css`.
- [x] Keep token definitions in one place and avoid new hard-coded colors unless
  added as named tokens.

### 3. Migrate Auth And Settings Surfaces

Start with settings because they have high duplication and security-sensitive
behavior.

- [x] Migrate `ProviderSettingsClient` to:
  - [x] `RelayPageHead`
  - [x] `RelayCard`
  - [x] `RelayField`
  - [x] `RelayTextInput`
  - [x] `RelayButton`
  - [x] `RelayStatusBadge`
  - [x] `RelayFeedback`
- [x] Migrate `APIKeySettingsClient` to the same primitives.
- [x] Migrate settings page sign-in/error fallback panels to shared primitives.
- [x] Verify provider credential behavior remains unchanged:
  - [x] invalid key validation
  - [x] save masked metadata
  - [x] disconnect
- [x] Verify user API key behavior remains unchanged:
  - [x] issue one-time token
  - [x] copy token
  - [x] revoke token
  - [x] revoked state remains visible

### 4. Migrate Onboarding And Root Entry

- [x] Migrate root sign-in entry to shared page/card/button primitives.
- [x] Migrate onboarding unauthenticated sign-in panel.
- [x] Migrate onboarding authenticated workspace creation panel.
- [x] Preserve keyless onboarding behavior.
- [x] Preserve redirect behavior for completed users.
- [x] Preserve Korean/English copy behavior.

### 5. Migrate Project Workspace Surfaces

- [x] Migrate Project Explorer page head, action links, metrics, panels, recent
  activity, inspector counts, and empty/error states.
- [x] Migrate Trace Browser page head, filters, table rows, selected state, and
  empty/error states.
- [x] Migrate Decision Graph page head, graph cards, node evidence panels, and
  empty/error states.
- [x] Migrate Packet Builder page head, composition grid, source chips, preview,
  publish feedback, and empty/error states.
- [x] Migrate Style Memory proposal cards, tabs, status badges, diff panels,
  approve/reject buttons, and toast/feedback UI.
- [x] Keep existing route-level data fetching and mutation code unchanged unless
  required by component props.

### 6. Remove Styling Drift

- [x] Remove duplicated page-local style constants after each migration.
- [x] Run a scan for remaining `const .*Style: React.CSSProperties`.
- [x] Keep only genuinely dynamic inline styles, such as:
  - CSS custom property values
  - small layout exceptions that cannot be expressed by variants
  - visually hidden or display contents cases
- [ ] Add a lightweight guard script or documented check:

```bash
rg -n "const [A-Za-z0-9]+Style: React\\.CSSProperties|style=\\{" web/app web/components
```

- [x] Document accepted exceptions so future work does not reintroduce page-local
  style systems.

### 7. Documentation

- [x] Create `docs/design-system.md`.
- [x] Document:
  - [x] design intent
  - [x] token contract
  - [x] component catalog
  - [x] variant names
  - [x] accessibility rules
  - [x] mobile expectations
  - [x] when inline styles are allowed
- [x] Link the previous UI kit implementation plan and this hardening plan.
- [x] Add examples for common page layouts:
  - [x] settings two-column form
  - [x] project workspace with inspector
  - [x] tabbed review queue
  - [x] dense trace list

### 8. Verification

Local static checks:

```bash
npm --prefix web run lint
npm --prefix web run typecheck
npm --prefix web test
npm --prefix web run build
git diff --check
```

Local browser checks:

```bash
npm --prefix web run dev -- --hostname 127.0.0.1 --port 3000
RELAY_WEB_BASE_URL=http://127.0.0.1:3000 scripts/qa/run_live_e2e.sh
```

Live checks after deploy:

```bash
RELAY_WEB_BASE_URL=https://relay.4gly.dev scripts/qa/run_live_e2e.sh
curl -sS -D - https://relay.4gly.dev/healthz
kubectl get applications -n argocd relay -o wide
kubectl -n relay rollout status deploy/relay-web --timeout=180s
```

Visual QA route set:

- [x] `/`
- [x] `/onboarding`
- [x] `/projects/:projectId`
- [x] `/projects/:projectId/traces`
- [x] `/projects/:projectId/graph`
- [x] `/projects/:projectId/packet-builder`
- [x] `/style-memory?project=:projectId`
- [x] `/settings/providers`
- [x] `/settings/api-keys`
- [x] `/p/:token`

Viewport set:

- [x] desktop chromium
- [x] 1024px tablet-ish width
- [x] mobile chromium / Pixel 5

Verification evidence:

- [x] `npm --prefix web run lint`
- [x] `npm --prefix web run typecheck`
- [x] `npm --prefix web test` — 22 files / 76 tests passed.
- [x] `npm --prefix web run build`
- [x] `git diff --check`
- [x] `RELAY_WEB_BASE_URL=https://relay.4gly.dev scripts/qa/run_live_e2e.sh`
  — 30 passed, 2 skipped by design for mobile-only mutation avoidance.
  Run id: `qa202605010259525a4464`.
- [x] `NEXT_PUBLIC_RELAY_API_URL=https://relay.4gly.dev npm --prefix web run dev -- --hostname 127.0.0.1 --port 3000`
  plus `RELAY_WEB_BASE_URL=http://127.0.0.1:3000 scripts/qa/run_live_e2e.sh`
  — local changed web UI against live API: 24 passed, 8 skipped by design
  for local `/p` placeholder routing and mutation-only-live checks.
  Run id: `qa202605010302249f8161`.
- [x] Local 1024px authenticated route smoke against live API — `/`,
  `/onboarding`, project explorer, traces, graph, packet builder, style memory,
  provider settings, and API key settings all rendered with no horizontal
  document overflow. Project id: `proj_tablet202605010304424413a5`.

## Acceptance Criteria

- [x] Target screens import shared Relay primitives for common UI patterns.
- [x] Page-local style constants are removed or reduced to documented dynamic
  exceptions.
- [x] Settings and onboarding use the same card, field, button, feedback, and
  page-head primitives as internal workspace screens.
- [x] Project screens use the same page-head, card, metric, tab, badge, and
  list-row primitives.
- [x] No visible regressions in desktop UI kit language.
- [x] Mobile screens remain readable, non-overlapping, and navigable.
- [x] Existing unit, build, and live E2E checks pass.
- [x] `docs/design-system.md` explains how to build future Relay screens without
  copying page-local styles.

## Suggested Commit / PR Split

1. `feat(web): extract relay design primitives`
   - component files
   - CSS classes
   - no broad page migration beyond smoke usage

2. `refactor(web): migrate settings and onboarding to relay primitives`
   - provider settings
   - API key settings
   - root/onboarding
   - targeted tests

3. `refactor(web): migrate project workspace screens to relay primitives`
   - explorer
   - traces
   - graph
   - packet builder
   - style memory
   - browser QA

4. `docs(web): document relay design system`
   - `docs/design-system.md`
   - inline-style exception policy
   - final verification evidence

For speed, these can be merged as one PR if the diff remains reviewable. Prefer
the split above if migration touches more than roughly 1,500 lines.

## Risks

- Component extraction can accidentally change accessible names or heading
  levels. Mitigation: preserve tests and prefer semantic wrappers.
- Settings screens are mutation-heavy. Mitigation: run live provider/API-key
  mutation smoke after migration.
- Large style refactors can create visual regressions without type failures.
  Mitigation: use gstack/browser screenshots at the viewport set above.
- Over-abstracting too early can make product-specific screens harder to build.
  Mitigation: extract only patterns already repeated across at least two screens.

## Done Definition

- Design system primitives exist as reusable React APIs.
- Main product screens consume those primitives.
- CSS tokens/classes are the primary styling source.
- Inline style usage is minimal and documented.
- Documentation and QA evidence exist.
- The live product passes E2E after deployment.
