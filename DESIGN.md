---
version: alpha
name: Relay
description: 4gly Labs product design system for Relay, a calm engineering workspace that turns chaotic AI work into reusable decisions, style memory, and handoff packets.
colors:
  primary: "#0E1A35"
  canvas: "#FAF8F3"
  canvas-raised: "#FFFFFF"
  text: "#0E1A35"
  text-muted: "#4A5669"
  text-subtle: "#687386"
  surface-inverse: "#1E2230"
  surface-inverse-muted: "#2B3140"
  accent: "#A7C4FF"
  accent-strong: "#6F96DB"
  accent-secondary: "#C7B8FF"
  accent-secondary-strong: "#8B76E4"
  border: "#E7ECF3"
  border-strong: "#D4DCE8"
  success: "#2F9B73"
  danger: "#C84646"
  on-accent: "#0E1A35"
  on-inverse: "#FAF8F3"
  on-danger: "#FFFFFF"
  dark-canvas: "#0B1020"
  dark-canvas-raised: "#121930"
  dark-text: "#EEF4FF"
  dark-text-muted: "#A5B1C8"
  dark-text-subtle: "#9AA6B8"
  dark-surface-inverse: "#06091A"
  dark-surface-inverse-muted: "#1A1F30"
  dark-accent: "#7FB6FF"
  dark-accent-strong: "#A7C4FF"
  dark-accent-secondary: "#A78BFA"
  dark-accent-secondary-strong: "#C7B8FF"
  dark-border: "#273244"
  dark-border-strong: "#3A4660"
  dark-success: "#64D6A4"
  dark-danger: "#FF7777"
typography:
  display:
    fontFamily: Fraunces
    fontSize: 40px
    fontWeight: 650
    lineHeight: 1.1
    letterSpacing: "-0.02em"
    fontVariation: "\"opsz\" 96, \"SOFT\" 40"
  page-title:
    fontFamily: Fraunces
    fontSize: 28px
    fontWeight: 650
    lineHeight: 1.1
    letterSpacing: "-0.02em"
    fontVariation: "\"opsz\" 96, \"SOFT\" 35"
  body:
    fontFamily: Nunito
    fontSize: 14px
    fontWeight: 500
    lineHeight: 1.55
    letterSpacing: "0em"
  body-large:
    fontFamily: Nunito
    fontSize: 16px
    fontWeight: 500
    lineHeight: 1.55
    letterSpacing: "0em"
  editorial:
    fontFamily: Fraunces
    fontSize: 20px
    fontWeight: 450
    lineHeight: 1.5
    letterSpacing: "0em"
    fontVariation: "\"opsz\" 48"
  mono-label:
    fontFamily: JetBrains Mono
    fontSize: 11px
    fontWeight: 500
    lineHeight: 1.4
    letterSpacing: "0.12em"
rounded:
  chip: 6px
  control: 8px
  card: 12px
  chrome: 14px
  pill: 999px
spacing:
  xxs: 2px
  xs: 4px
  sm: 8px
  md: 12px
  lg: 16px
  xl: 24px
  2xl: 32px
  3xl: 48px
  4xl: 64px
  5xl: 96px
components:
  app-shell:
    backgroundColor: "{colors.canvas}"
    textColor: "{colors.primary}"
    typography: "{typography.body}"
    width: 1440px
  top-rail:
    backgroundColor: "{colors.canvas}"
    textColor: "{colors.text}"
    height: 56px
    padding: "{spacing.xl}"
  navigation-rail:
    backgroundColor: "{colors.canvas-raised}"
    textColor: "{colors.text-muted}"
    width: 240px
    padding: "{spacing.lg}"
  inspector-panel:
    backgroundColor: "{colors.canvas-raised}"
    textColor: "{colors.text-muted}"
    width: 320px
    rounded: "{rounded.chrome}"
    padding: "{spacing.xl}"
  action-primary:
    backgroundColor: "{colors.text}"
    textColor: "{colors.on-inverse}"
    typography: "{typography.mono-label}"
    rounded: "{rounded.control}"
    padding: "{spacing.md}"
  action-secondary:
    backgroundColor: "{colors.canvas-raised}"
    textColor: "{colors.text-muted}"
    typography: "{typography.mono-label}"
    rounded: "{rounded.control}"
    padding: "{spacing.md}"
  action-danger:
    backgroundColor: "{colors.danger}"
    textColor: "{colors.on-danger}"
    typography: "{typography.mono-label}"
    rounded: "{rounded.control}"
    padding: "{spacing.md}"
  content-card:
    backgroundColor: "{colors.canvas-raised}"
    textColor: "{colors.text}"
    typography: "{typography.body}"
    rounded: "{rounded.card}"
    padding: "{spacing.xl}"
  metadata-label:
    backgroundColor: "{colors.canvas}"
    textColor: "{colors.text-subtle}"
    typography: "{typography.mono-label}"
  source-chip:
    backgroundColor: "{colors.canvas-raised}"
    textColor: "{colors.text-muted}"
    typography: "{typography.mono-label}"
    rounded: "{rounded.pill}"
    padding: "{spacing.sm}"
  status-neutral:
    backgroundColor: "{colors.canvas-raised}"
    textColor: "{colors.text-muted}"
    typography: "{typography.mono-label}"
    rounded: "{rounded.pill}"
    padding: "{spacing.sm}"
  status-pending:
    backgroundColor: "{colors.surface-inverse}"
    textColor: "{colors.on-inverse}"
    typography: "{typography.mono-label}"
    rounded: "{rounded.chip}"
    padding: "{spacing.sm}"
  status-complete:
    backgroundColor: "{colors.accent}"
    textColor: "{colors.on-accent}"
    typography: "{typography.mono-label}"
    rounded: "{rounded.chip}"
    padding: "{spacing.sm}"
  status-success:
    backgroundColor: "{colors.text}"
    textColor: "{colors.success}"
    typography: "{typography.mono-label}"
    rounded: "{rounded.pill}"
    padding: "{spacing.sm}"
  status-danger:
    backgroundColor: "{colors.danger}"
    textColor: "{colors.on-danger}"
    typography: "{typography.mono-label}"
    rounded: "{rounded.pill}"
    padding: "{spacing.sm}"
  comparison-before:
    backgroundColor: "{colors.surface-inverse-muted}"
    textColor: "{colors.on-inverse}"
    typography: "{typography.mono-label}"
    rounded: "{rounded.control}"
    padding: "{spacing.md}"
  comparison-after:
    backgroundColor: "{colors.accent}"
    textColor: "{colors.on-accent}"
    typography: "{typography.mono-label}"
    rounded: "{rounded.control}"
    padding: "{spacing.md}"
  accent-emphasis:
    backgroundColor: "{colors.accent-secondary}"
    textColor: "{colors.text}"
    rounded: "{rounded.card}"
    padding: "{spacing.lg}"
  accent-control:
    backgroundColor: "{colors.text}"
    textColor: "{colors.accent-strong}"
    typography: "{typography.mono-label}"
    rounded: "{rounded.control}"
    padding: "{spacing.md}"
  accent-secondary-control:
    backgroundColor: "{colors.text}"
    textColor: "{colors.accent-secondary-strong}"
    typography: "{typography.mono-label}"
    rounded: "{rounded.control}"
    padding: "{spacing.md}"
  divider:
    backgroundColor: "{colors.border}"
    height: 1px
  divider-strong:
    backgroundColor: "{colors.border-strong}"
    height: 1px
  dark-app-shell:
    backgroundColor: "{colors.dark-canvas}"
    textColor: "{colors.dark-text}"
    typography: "{typography.body}"
    width: 1440px
  dark-content-card:
    backgroundColor: "{colors.dark-canvas-raised}"
    textColor: "{colors.dark-text}"
    typography: "{typography.body}"
    rounded: "{rounded.card}"
    padding: "{spacing.xl}"
  dark-metadata-label:
    backgroundColor: "{colors.dark-canvas}"
    textColor: "{colors.dark-text-subtle}"
    typography: "{typography.mono-label}"
  dark-source-chip:
    backgroundColor: "{colors.dark-canvas-raised}"
    textColor: "{colors.dark-text-muted}"
    typography: "{typography.mono-label}"
    rounded: "{rounded.pill}"
    padding: "{spacing.sm}"
  dark-status-pending:
    backgroundColor: "{colors.dark-surface-inverse}"
    textColor: "{colors.dark-text}"
    typography: "{typography.mono-label}"
    rounded: "{rounded.chip}"
    padding: "{spacing.sm}"
  dark-comparison-before:
    backgroundColor: "{colors.dark-surface-inverse-muted}"
    textColor: "{colors.dark-text}"
    typography: "{typography.mono-label}"
    rounded: "{rounded.control}"
    padding: "{spacing.md}"
  dark-status-complete:
    backgroundColor: "{colors.dark-accent}"
    textColor: "{colors.dark-canvas}"
    typography: "{typography.mono-label}"
    rounded: "{rounded.chip}"
    padding: "{spacing.sm}"
  dark-status-success:
    backgroundColor: "{colors.dark-canvas-raised}"
    textColor: "{colors.dark-success}"
    typography: "{typography.mono-label}"
    rounded: "{rounded.pill}"
    padding: "{spacing.sm}"
  dark-action-danger:
    backgroundColor: "{colors.dark-danger}"
    textColor: "{colors.dark-canvas}"
    typography: "{typography.mono-label}"
    rounded: "{rounded.control}"
    padding: "{spacing.md}"
  dark-accent-control:
    backgroundColor: "{colors.dark-canvas-raised}"
    textColor: "{colors.dark-accent-strong}"
    typography: "{typography.mono-label}"
    rounded: "{rounded.control}"
    padding: "{spacing.md}"
  dark-accent-emphasis:
    backgroundColor: "{colors.dark-accent-secondary}"
    textColor: "{colors.dark-canvas}"
    rounded: "{rounded.card}"
    padding: "{spacing.lg}"
  dark-accent-secondary-control:
    backgroundColor: "{colors.dark-canvas-raised}"
    textColor: "{colors.dark-accent-secondary-strong}"
    typography: "{typography.mono-label}"
    rounded: "{rounded.control}"
    padding: "{spacing.md}"
  dark-divider:
    backgroundColor: "{colors.dark-border}"
    height: 1px
  dark-divider-strong:
    backgroundColor: "{colors.dark-border-strong}"
    height: 1px
---

# Design System — Relay (project-swan · 4gly Labs)

> **Simplicity out of Complexity.**
> We handle the Ugly, so you can be Elegant.

Relay is a product shipped by **4gly Labs**. Before making any visual or UI decision, re-read `/Users/hoon-ch/repos/4gly/4gly_foundation.md`. This file is downstream of that one.

---

## Overview

### Product Context

- **What this is:** An API-first second-brain backend for long-running AI-assisted work, now getting its first web UI.
- **Who it's for:** Senior engineers, researchers, and PMs running many parallel AI workstreams who need to capture notes, audit judgments, learn style memory, and hand off packets to AI consumers (Claude, Codex).
- **Space / industry:** Developer tools × AI memory systems × second-brain knowledge bases.
- **Project type:** Internal dense web application + minimal public surface. Desktop-first (>=1280px). Mobile deferred.
- **Priority screens (in order):** Style Memory (signature) → Project Explorer → Judgment Traces → Decision Graph → Packet Builder.

### The Memorable Thing

> **"A quiet engine that turns chaos into swans."**

First 3 seconds, a user should feel: *"this is a serious engineering tool, but it has a soul."*

Every design decision downstream serves this sentence. If a decision doesn't — reject it.

### 4gly 4-Step Engine to Relay Domain Mapping

| Step | 4gly | Relay surface |
|------|------|--------------|
| 1 · **Face** | Meet the duckling head-on | **Capture** — raw Note / Artifact / Open Question land in the project |
| 2 · **Dissect** | Decompose the complexity | **Trace** — Judgment Trace exposes workflow · alternatives · rationale |
| 3 · **Refine** | Strip the unnecessary | **Propose → Approve** — Heuristic Proposal reviewed into Style Memory |
| 4 · **Transform** | Reborn as swan | **Packet** — compiled handoff document for the next AI consumer |

The UI carries a thin "Face → Dissect → Refine → Transform" ribbon in the top rail. The active step is faintly tinted with `--magic-primary`. This is the engine's heartbeat, always visible, never loud.

### Aesthetic Direction

- **Direction:** **Sophisticated Cuteness Plus** — 4gly's Magic Lab × Toy Workshop warmed with editorial authority and anchored with stronger engineering charcoal. Professional but not cold. Minimal but not empty. Playful only at signature moments.
- **Decoration level:** **Intentional** — no ornamentation. Allowed: very subtle grain on `--problem` surfaces, pastel halo glow for elevation, the 900 ms duckling→swan contour transform.
- **Mood:** *"Someone who reads and thinks built this."* Engineer-serious on the surface, with rare toy-workshop warmth at transformation moments.
- **Density:** Comfortable-leaning. Tighter than Granola, looser than Linear. Reference density is Reflect.app with engineering additions.

### Brand Voice

- **Professional, but not cold.** — `"Three proposals waiting for your judgment."` not `"Pending items: 3"`.
- **Minimal, but not empty.** — Empty states show context, not shrug emojis.
- **Serious problems, playful solutions.** — Action button label: **"Approve → Swan"** (not `Approve`).
- **Show the swan, not the pipes.** — Curator job state, lease counts, attempt numbers live in a power-user drawer, not on the primary surface.

**Canonical action labels**
- `Approve → Swan` (primary, heuristic approval)
- `Edit & approve` (ghost, lightly-amended approval)
- `Reject` (danger-tinted ghost)
- `Compose handoff` (packet build)

## Colors

Approach: **Restrained.** Pastel magic (blue + purple) never washes the canvas — it appears only where transformation is about to happen or just did. Charcoal does the engineering work. Deep navy is the text.

### Light theme tokens

```css
--canvas:              #FAF8F3;   /* warm off-white — never pure white */
--canvas-raised:       #FFFFFF;   /* card surfaces, half a step above canvas */
--ink:                 #0E1A35;   /* Deep Navy — primary text, trust, authorship */
--ink-muted:           #4A5669;   /* secondary text */
--problem:             #1E2230;   /* pushed charcoal — duckling / unresolved states */
--problem-soft:        #2B3140;   /* charcoal one step softer (diff-before, chips) */
--magic-primary:       #A7C4FF;   /* pastel blue — transformation moments only */
--magic-primary-strong:#6F96DB;   /* interactive magic — hover, focus, wordmark dot */
--magic-accent:        #C7B8FF;   /* soft purple — the swan moment */
--magic-accent-strong: #8B76E4;   /* interactive accent */
--muted:               #687386;   /* tertiary text, meta */
--border:              #E7ECF3;   /* Mist */
--border-strong:       #D4DCE8;   /* emphasized border */
--success:             #2F9B73;   /* forest — never lime */
--danger:              #C84646;   /* muted crimson — failure is not loud */
--halo:                rgba(167, 196, 255, 0.35);   /* pastel elevation glow */
--grain-opacity:       0.03;
```

### Dark theme tokens

```css
--canvas:              #0B1020;
--canvas-raised:       #121930;
--ink:                 #EEF4FF;
--ink-muted:           #A5B1C8;
--problem:             #06091A;
--problem-soft:        #1A1F30;
--magic-primary:       #7FB6FF;
--magic-primary-strong:#A7C4FF;
--magic-accent:        #A78BFA;
--magic-accent-strong: #C7B8FF;
--muted:               #9AA6B8;
--border:              #273244;
--border-strong:       #3A4660;
--success:             #64D6A4;
--danger:              #FF7777;
--halo:                rgba(127, 182, 255, 0.28);
--grain-opacity:       0.045;
```

### Color rules

1. **`--magic-*` tokens appear only at transformation moments.** Hover-to-approve glow, the 900 ms swan contour, the `ScopeMatrix` heatmap, the Current Transform inspector tile. They are **not** ambient background.
2. **`--problem`** gets used for Ugly-side states: pending proposals (not in focus), failed curator jobs, unresolved Open Questions, diff-before panels.
3. **No gradients** except the singular pastel rim-light on signature moments.
4. **Dark mode is 1st-class.** Every component must be designed and tested in both themes. Dark canvas is deep navy, never pure black.

## Typography

Three faces, four roles. Fraunces covers both display and editorial italic to save a font load and reinforce the family voice.

| Role | Font | Weights | Usage |
|------|------|---------|-------|
| Display / Hero / Section headers | **Fraunces Variable** (Google Fonts) | 500 / 600 / 700 | Use `opsz 96–144`, `SOFT 30–50`, `letter-spacing: -0.025em`. Wordmark, page headers, mockup chrome titles. |
| Body / UI | **Nunito** (Google Fonts) | 400 / 500 / 600 / 700 | Default body, buttons, labels, table rows. Never use weight 300 (too thin against navy). Never use weight 900. |
| Editorial / Authored | **Fraunces Italic** (same family) | 400 / 500 | Decision rationale, Approved Heuristic summaries, Packet cover notes, trace-card quote. Use `opsz 48`. This is the "authored knowledge" surface. |
| Mono | **Commit Mono** (self-hosted, paid) — stand-in: **JetBrains Mono** (Google Fonts) | 400 / 500 | Trace IDs, heuristic IDs, code spans, provenance metadata, scope chips, timestamps. Always `font-variant-numeric: tabular-nums`. |

**Scale** (root 16px):

| Role | px | rem | line-height |
|------|-----|------|-------------|
| micro / mono-label | 10 | 0.625 | 1.4 |
| meta / chip | 11 | 0.6875 | 1.4 |
| UI-small | 12 | 0.75 | 1.5 |
| body | 13–14 | 0.8125–0.875 | 1.55 |
| body-large | 16 | 1 | 1.55 |
| editorial italic | 18–22 | 1.125–1.375 | 1.5 |
| ws-heading | 28 | 1.75 | 1.1 |
| section-title | 40 | 2.5 | 1.1 |
| hero | 72–168 (clamp) | — | 0.92 |

**Loading:** Google Fonts CDN in development, self-host in production via `next/font` for CLS safety. License Commit Mono when shipping; JetBrains Mono is the fallback.

**Bans:** Inter, Roboto, Arial, Helvetica, Open Sans, Lato, Montserrat, Poppins, Space Grotesk, Clash Display, system-ui, -apple-system (as display or body).

## Layout

### Spacing and Grid

- **Base unit:** 4 px.
- **Scale:** `2 · 4 · 8 · 12 · 16 · 24 · 32 · 48 · 64 · 96`.
- **Density:** Comfortable. Card internal padding is `20–24 px`, not `12–14`.
- **Grid (desktop 1440×900 reference):** three-region `240 px · 1fr · 320 px`. Inspector (`320 px`) is collapsible.
- **Max content width for non-app surfaces** (hero, docs): `1360 px`.

### App Shell

- **Approach:** Hybrid. Three-region engineering workspace for app screens. Editorial serif surfaces for authored content (Decision rationale, Approved Heuristic summaries, Packet cover notes).
- **App shell:**
  - **Top rail (56 px):** flush-left Fraunces wordmark (`Relay.` with a `--magic-primary-strong` dot). Centered 4 Steps ribbon. Flush-right theme toggle + user avatar.
  - **Left rail (240 px):** dense Project Explorer. Projects stacked by recency with small mono status glyphs: duckling count badge (`--problem` filled) + swan count badge (`--magic-primary` tinted).
  - **Main workspace:** the hero object of the current view. Tabs at top, content below, everything honors the 4 px grid.
  - **Right inspector (320 px, collapsible):** "Current Transform" tile + Scope Matrix heatmap + project stats.
  - **Bottom tray (collapsed by default):** Packet Builder.

### First Viewport Rule

**First viewport = poster, not document.** The Style Memory screen has one gravitational card (hover-to-approve state, pastel halo). Every other element supports that card.

## Elevation & Depth

### Halo Elevation

- **Elevation = pastel halo,** not drop-shadow. Cards use `box-shadow: 0 0 0 4px var(--halo)` patterns, never dark-opacity shadows.
- Use pastel halo glow for important raised surfaces and signature transformation states.
- Keep charcoal surfaces visually heavy for unresolved or Ugly-side states, with very subtle grain controlled by `--grain-opacity`.
- Use the singular pastel rim-light only on signature moments.

### Motion

Framer Motion. CSS transitions for simple hover/focus.

- **Base easing:** `cubic-bezier(0.2, 0.8, 0.2, 1)` (controlled ease-out).
- **Durations:**
  - Controls (hover, focus, theme toggle): **140 ms**
  - Panels (tabs, card hover, collapse/expand): **220 ms**
  - Graph transitions (Decision Graph, Scope Matrix re-layout): **420 ms**
  - **Signature: 900 ms duckling → swan contour transform.**
- **Signature details.** When a heuristic proposal is approved:
  1. 0–240 ms: charcoal "duckling" text cluster contracts slightly (`scale: 1 → 0.96`), rotates `-4°`, picks up micro-grain.
  2. 240–640 ms: pastel rim-light sweeps left-to-right (`--magic-primary-strong` at 15% opacity, `blur(12px)`).
  3. 640–900 ms: a swan-contour silhouette is drawn by graph edges, resolves into the actual approved-heuristic chip (or packet cover, or decision node). The contour uses `stroke-dasharray` reveal.
- **Do not:** confetti, scale-bounce, hover-springs above `1.02`, entrance animations on every element, more than one signature moment on screen at once.

## Shapes

### Radius Scale

- Cards: `12 px`
- Inspector / mockup chrome: `14 px`
- Buttons / inputs: `8 px`
- Chips / pills: `6 px` for rectangular, `999 px` for pill-shaped (source chips, scope chips)
- **Never** uniform 9999 px on every element.

### Iconography and Illustration

- **Icon set:** Custom line icons, 1.5 px stroke, inheriting `currentColor`. Lucide as placeholder during early dev.
- **Duckling / Swan:** used only at (a) very subtle empty-state hint, (b) the 900 ms contour transform, (c) the wordmark's post-dot. **Never** a cartoon mascot. **Never** a decorative illustration.
- **Status glyphs in Project Explorer:** small mono characters (`◈`, `△`, `○`) colored per `--problem` / `--magic-primary` role. No emoji.

## Components

### TraceCard
- Collapsed: `pill(scope chip in mono) · timestamp(mono) · rationale(Fraunces italic, single line)`.
- Expanded: `workflow · alternatives · rationale · linked source chips · 'propose heuristic' button`.

### HeuristicDiff
- Two-column `before` (problem-soft bg) · `after` (magic-primary 22% tint, magic-primary border).
- Mono content. Each side has an uppercase meta label (`Current heuristic` / `Proposed refinement`).
- Rationale below in Fraunces italic, left-bordered by `magic-accent` 2 px.

### PacketPreview
- Markdown rendered in Nunito body + Fraunces italic for quoted lines.
- Section-level **source provenance halo**: hover → short column of `source-chip`s linking back to the originating note / trace / decision.

### SourceChip
- Pill, 10.5 px mono, 1 px border. Dot prefix colored by type (`trace` → magic-accent, `note` → success, `artifact` → muted).

### ScopeMatrixCell
- `60 px · repeat(N, 1fr)` grid. Cells shaded by `color-mix(in oklab, var(--magic-primary) <density>%, var(--canvas))` where density is `0–60%`. Cell text is the approved-heuristic count in tabular-nums. Never multicolor.

### DucklingBadge / SwanBadge
- Mono, 10 px, `2px 6px` padding, `6 px` radius.
- `DucklingBadge`: `var(--problem)` bg, canvas text.
- `SwanBadge`: `color-mix(magic-primary 25%, canvas-raised)` bg, `--magic-primary-strong` text.

### TransformRibbon
- Top-rail centered. Mono uppercase labels separated by `→`. Active step has `color-mix(magic-primary 30%, canvas)` pill bg.

## Do's and Don'ts

### Do

- Use `--magic-*` only when transformation is about to happen or just happened.
- Use `--problem` for unresolved, pending, failed, or ugly-side states.
- Use pastel halo elevation for important surfaces.
- Preserve the visible `Face -> Dissect -> Refine -> Transform` engine in app workspaces.
- Keep Style Memory as the signature screen.
- Use sentence-case product copy that is professional but not cold.
- Respect `prefers-reduced-motion` for signature animation.

### Don't

- Do not use `--magic-*` as ambient page background.
- Do not use generic dark-opacity drop shadows as the primary elevation model.
- Do not use confetti, bounce, emoji, or mascot decoration.
- Do not expose curator job state, lease counters, or attempt numbers on primary surfaces.
- Do not use Inter, Roboto, Arial, Helvetica, Open Sans, Lato, Montserrat, Poppins, Space Grotesk, Clash Display, or system UI fonts as display or body.
- Do not apply pill radius uniformly to every element.
- Do not add more than one signature transformation moment on screen at once.

## Accessibility

- WCAG AA minimum. Test all `ink`/`ink-muted` × `canvas` pairs at ≥ 4.5:1. Pastel magic over canvas pairs are UI-chrome only, not for text.
- Focus rings: `outline: 2px solid var(--halo); outline-offset: 2px; border-color: var(--magic-primary-strong)`.
- Keyboard-first navigation across Project Explorer, tabs, proposal cards, approve/reject actions.
- Motion respects `prefers-reduced-motion`: the 900 ms signature collapses to a 160 ms fade.

## Tech Stack

- **Framework:** Next.js (App Router)
- **CSS:** Tailwind v4 with CSS variables driving tokens
- **Components:** Shadcn/UI base, customized to this system
- **Motion:** Framer Motion
- **3D / future hero:** Three.js / Spline (kept out of app shell; used only for marketing hero if we build one)
- **Fonts:** `next/font` (Fraunces, Nunito, JetBrains Mono). Commit Mono self-hosted when licensed.
- Do **not** design anything this stack cannot render. Graceful degradation is fine; fantasy is not.

## Non-Scope

- Onboarding, marketing landing, signup funnels
- Mobile-first layouts (design responsive collapse points only)
- Admin / org-management consoles (API Keys management fits inside one inspector panel)
- Multi-tenant switching (single-user-per-account for v0.1)

## Success Bar

1. **The 3-second rule.** A first-time viewer feels: *"this tool turns chaos into swans."* No exceptions.
2. **Show the Swan rule.** User finishes a session without ever seeing `curator_jobs.state`, lease counters, or attempt numbers on the primary surface. The information is accessible one click deeper — never hidden, never in the way.
3. **Signature rule.** The Style Memory screen alone explains what Relay is. If that screen is removed, the product's differentiator disappears with it.
4. **Density + calm rule.** Density is Linear-class. Breath is Arc-class. Feel is 4gly-class.
5. **The Anchor.** Re-read `4gly_foundation.md §12`. If this UI would make that document's author flinch, rebuild.

## References

- **Foundation:** `/Users/hoon-ch/repos/4gly/4gly_foundation.md`
- **Preview artifact (light + dark):** `~/.gstack/projects/relay/designs/design-system-20260425/preview.html`
- **Rendered preview (light):** `/tmp/relay-preview-light.png`
- **Rendered preview (dark):** `/tmp/relay-preview-dark.png`
- **Competitor visual scan (2026-04-24):** Linear · Raycast · Reflect · Granola (screenshots in `/tmp/relay-research/`)

## Decisions Log

| Date | Decision | Rationale |
|------|----------|-----------|
| 2026-04-25 | Memorable thing locked: *"A quiet engine that turns chaos into swans."* | User selection from 4 options; matches 4gly foundation metaphor most directly |
| 2026-04-25 | Keep 4gly's pastel blue/purple as `--magic-*` (rejected Subagent's gold alternative) | Brand continuity with 4gly parent non-negotiable at v0.1 |
| 2026-04-25 | Keep Nunito as body (rejected Subagent's DM Sans alternative) | 4gly declared font inheritance; warmth preserved in daily UI text |
| 2026-04-25 | Add Fraunces as display + editorial italic | Both Codex and Claude subagent converged on serif for "authored knowledge" surface |
| 2026-04-25 | Add Commit Mono (stand-in JetBrains Mono) as 1st-class for trace IDs and code | Both outside voices + Linear/Raycast convergence; engineering trust signal |
| 2026-04-25 | Push `--problem` charcoal more aggressively than 4gly declared | Both outside voices converged; Ugly side must feel heavy for Transformation to matter |
| 2026-04-25 | Signature = 900 ms graph contour transform, not decorative swan mascot | Rejects mascot branding that senior engineers resist; makes the Decision Graph the brand object |
| 2026-04-25 | Layout = hybrid 3-region engineering workspace + editorial serif surfaces | Codex's 3-region workspace + Subagent's authored-content insight |
| 2026-04-25 | Initial design system created | `/gstack-design-consultation` session based on 4gly foundation + Relay domain exploration + 3-voice synthesis (Claude main + Codex + designer subagent) + competitor scan |
