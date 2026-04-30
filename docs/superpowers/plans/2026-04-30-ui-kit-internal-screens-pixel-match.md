# UI Kit Internal Screens Pixel Match Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use `superpowers:executing-plans`
> to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for
> tracking. Preserve existing API contracts and route behavior while matching the
> design handoff visually.

Created: 2026-04-30
Branch: `main`
Mode: `superpowers:writing-plans` fallback via local planning workflow

## Plan Summary

Implement the remaining Relay design handoff across the internal app screens,
using the exported UI kit as the visual reference:

- `ui_kits/web_app/Screens.jsx`
- `ui_kits/web_app/AuxScreens.jsx`
- `ui_kits/web_app/shell.css`
- `project/README.md`
- `project/colors_and_type.css`

The first pass already introduced shared brand fonts, `RelayTopRail`,
`RelayAppShell`, the transformation ribbon, Project Explorer rail, and Project
Explorer inspector. This plan finishes the interior screen matching for Style
Memory, Trace Browser, Decision Graph, Packet Builder, API Keys, Provider
Settings, and Onboarding.

## Product Intent

The design system is anchored on:

> A quiet engine that turns chaos into swans.

Internal screens should feel like a dense, calm engineering workspace. The
visual hierarchy should expose Relay's 4-step engine without turning app
surfaces into marketing pages:

1. Face: capture and inspect the unresolved work.
2. Dissect: expose judgment traces and decision evidence.
3. Refine: review proposals into Style Memory.
4. Transform: compose handoffs, manage keys, publish snapshots.

## Non-Negotiable Design Rules

- Keep sentence case for readable UI copy.
- Use uppercase wide-tracked mono only for eyebrows, labels, and metadata.
- Keep `--magic-*` tokens for transformation moments, not ambient background.
- Use pastel halo elevation, never dark drop shadows.
- Cards use 12px radius; buttons and inputs use 8px; rectangular chips use 6px.
- No emoji. Use mono status glyphs: `◈`, `△`, `○`, `●`.
- Do not show curator job internals, leases, or attempt counters on primary
  surfaces.
- Keep keyboard access and current route/API behavior intact.

## Current Implementation State

Already completed in the first design import pass:

- Self-hosted Fraunces, Nunito, and JetBrains Mono via `next/font/local`.
- Shared `RelayTopRail` and `RelayAppShell`.
- Shared CSS primitives for top rail, transform ribbon, rails, cards, metrics,
  badges, and focus rings.
- Project Explorer shell with left rail and right inspector.
- Shared top rail on Style Memory and Settings pages.

Remaining gap:

- Interior layouts still mostly follow the pre-handoff page-specific inline
  styles.
- Trace Browser, Decision Graph, Packet Builder, Provider Settings, and
  Onboarding do not yet match the UI kit's page-head, tabs, table/list, card, and
  composition patterns.
- Style Memory has the right brand tone, but its proposal card hierarchy and tab
  treatment should be aligned with the UI kit's gravitational-card reference.

## Target Screens and Acceptance Criteria

### 1. Shared App Primitives

- [x] Extract a small set of reusable primitives under `web/components/relay-*`
  or extend the existing `relay-app-shell.tsx`.
- [x] Add reusable components for:
  - [x] `RelayPageHead`
  - [x] `RelayTabs`
  - [x] `RelayCard`
  - [x] `RelayButton` or style helpers compatible with existing buttons
  - [x] `SourceChip`
  - [x] `StatusBadge`
  - [x] `MetricTile`
  - [x] `EmptyState`
- [x] Keep CSS token-driven and avoid inventing new colors.
- [x] Preserve existing shadcn/ui imports where already used, but do not force a
  large component migration just for aesthetics.

### 2. Style Memory

UI kit reference: `StyleMemoryScreen`.

- [x] Keep existing data loading, tabs, approval/reject APIs, optimistic refresh,
  and keyboard shortcuts.
- [x] Align the header to the UI kit page-head:
  - [x] eyebrow: `{project} · Style Memory`
  - [x] title state for pending count, not a generic status phrase
  - [x] actions: `Compose handoff` and queue view toggle if still needed
- [x] Make the focused proposal the gravitational card:
  - [x] 12px card radius
  - [x] selected magic border + halo
  - [x] proposal meta row using mono labels
  - [x] Fraunces title and editorial rationale area
  - [x] source chips using kind-colored dots
- [x] Keep `Approve → Swan`, `Edit & approve` if supported, and `Reject` label
  conventions.
- [x] Ensure approved/rejected tabs use the same card/list language, not a
  separate visual system.

### 3. Trace Browser

UI kit reference: `TraceBrowserScreen`.

- [x] Keep route `/projects/[projectId]/traces` and current trace query behavior.
- [x] Replace the current loose page composition with:
  - [x] page-head with project + Judgment Traces eyebrow
  - [x] count-based title
  - [x] tabs for `All`, `Decisions`, `Open questions`, `Notes` if data supports
    them; otherwise render disabled/empty-compatible tabs
  - [x] dense bordered trace table/list
- [x] Each trace row should include:
  - [x] truncated trace id in mono
  - [x] rationale/decision title
  - [x] state badge using `Swan`, `Duckling`, or `Rejected`
  - [x] timestamp
  - [x] source ref count
- [x] Preserve trace deep-linking and currently tested text.

### 4. Decision Graph

UI kit reference: `DecisionGraphScreen`.

- [x] Keep route `/projects/[projectId]/graph` and current graph API contract.
- [x] Align page-head and action labels:
  - [x] `Decision Graph`
  - [x] `How decisions reached swan.` when populated
  - [x] `Re-layout` as a ghost control if it can be no-op for now
  - [x] `Compose handoff` linking to Packet Builder
- [x] Replace ad hoc node styling with the UI kit graph visual language:
  - [x] warm raised canvas
  - [x] subtle dotted grid
  - [x] neutral edges
  - [x] magic-primary active edge
  - [x] node cards with mono labels
  - [x] legend using mono glyph language
- [x] Do not add a graph library unless the current data shape requires it.
  CSS/SVG is acceptable for v0.1 if it remains accessible enough for smoke QA.

### 5. Packet Builder

UI kit reference: `PacketBuilderScreen`.

- [x] Keep route `/projects/[projectId]/packet-builder` and current packet API
  behavior.
- [x] Align page-head:
  - [x] `{project} · Packet Builder`
  - [x] `Compose a handoff.`
  - [x] `Save draft` only if supported or rendered disabled
  - [x] `Build snapshot →` for the primary action
- [x] Use the two-column composition + preview layout on desktop:
  - [x] cover note/editorial textarea area
  - [x] included sources list
  - [x] dashed empty/drop zone for future source picking
  - [x] right-side snapshot preview card
- [x] Preserve actual build snapshot action and public snapshot link behavior.
- [x] Mobile can collapse to one column, but desktop is the pixel-match priority.

### 6. Settings: API Keys

UI kit reference: `ApiKeysScreen`.

- [x] Keep current user-owned API key routes:
  - [x] `GET /v1/settings/api-keys`
  - [x] `POST /v1/settings/api-keys`
  - [x] `POST /v1/settings/api-keys/revoke`
- [x] Reframe the page as a two-column settings surface on desktop:
  - [x] issue key card
  - [x] existing keys card/list
- [x] Keep raw token one-time reveal exactly as implemented.
- [x] Never display full secrets after reload.
- [x] Keep revoke confirmation.
- [x] Use code-first error format where possible.

### 7. Settings: Provider Credentials

- [x] Use the same settings page-head and card system as API Keys.
- [x] Keep provider credentials visually separate from Relay API keys.
- [x] Preserve Anthropic connected/disconnected, masked tail, replace, and
  disconnect behavior.
- [x] Make the difference explicit in copy:
  - [x] Provider key: lets Relay call an external provider.
  - [x] Relay API key: lets external agents/tools call Relay.

### 8. Onboarding

UI kit reference: `OnboardingScreen`.

- [x] Keep first-run keyless onboarding.
- [x] Match the design handoff's simplified project-start surface:
  - [x] `Create your Relay workspace`
  - [x] editorial explanation that provider keys stay in Settings
  - [x] primary `Continue` action
  - [x] secondary path to provider settings only after workspace creation
- [x] Do not reintroduce provider API key collection into onboarding.
- [x] Preserve GitHub sign-in state and completed-user redirect behavior.

## Implementation Sequence

1. Shared primitives first.
   - [x] Add/extend component primitives and CSS classes.
   - [x] Update tests only if snapshots/query expectations require it.

2. Style Memory.
   - [x] Align highest-value signature screen before lower-risk surfaces.
   - [x] Run focused proposal tests.

3. Trace Browser and Decision Graph.
   - [x] They share project page-head and evidence navigation language.
   - [x] Run page tests for both.

4. Packet Builder.
   - [x] It is downstream of graph/traces and should reuse source chip and preview
     primitives.

5. Settings API Keys and Provider Credentials.
   - [x] Keep security behavior unchanged while improving layout consistency.

6. Onboarding.
   - [x] Keep keyless onboarding guarantee.
   - [x] Verify Korean copy still resolves.

7. QA pass.
   - [x] Run local unit tests, lint, typecheck, build.
   - [x] Run local authenticated E2E smoke against live API fixtures.
   - [x] Run gstack/browser visual review for:
     - [x] `/projects/{projectId}`
     - [x] `/style-memory?project={projectId}`
     - [x] `/projects/{projectId}/traces`
     - [x] `/projects/{projectId}/graph`
     - [x] `/projects/{projectId}/packet-builder`
     - [x] `/settings/api-keys`
     - [x] `/settings/providers`
     - [x] `/onboarding`
   - [ ] Run live E2E only after deploy.

## Verification Commands

Local static checks:

```bash
npm --prefix web run lint
npm --prefix web run typecheck
npm --prefix web test
npm --prefix web run build
git diff --check
```

Browser checks:

```bash
npm --prefix web run dev -- --hostname 127.0.0.1 --port 3000
```

Live QA after deploy:

```bash
scripts/qa/run_live_e2e.sh
```

## Pixel-Match Standard

This is not a screenshot-perfect clone of prototype internals. It is a
production pixel match of the visible product language:

- shell dimensions: top rail 56px, left rail 240px, inspector 320px
- card radius/padding/borders match UI kit
- typography roles match UI kit
- transform ribbon is visible and active step is meaningful
- status glyphs and badges match UI kit
- desktop density matches the 1440px reference
- mobile remains non-overlapping and usable, but exact mobile pixel matching is
  not the primary target for this pass

## Risks

- Existing pages use large inline style blocks. Partial refactors can easily
  create duplicated style systems. Mitigation: extract only primitives that are
  reused immediately.
- Tests assert exact text on some pages. Keep user-facing labels stable or update
  tests deliberately.
- Design handoff examples use static sample data. Production screens must keep
  real API data and empty/error states.
- Settings screens carry security behavior. Visual changes must not alter token
  reveal, revoke, or masked metadata rules.

## Done Definition

- All target internal screens use shared Relay app primitives.
- The visible desktop UI matches the handoff's internal UI kit language.
- Existing behavior and API contracts remain intact.
- Local lint/typecheck/unit/build pass.
- gstack/browser visual QA finds no obvious overlap, blank screens, or broken
  navigation.
- Live E2E passes after deploy.
