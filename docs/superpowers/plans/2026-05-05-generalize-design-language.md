# Generalize DESIGN.md Language Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace Relay-specific metaphor terms in `DESIGN.md` front matter and Markdown body with portable semantic design language while keeping the current runtime UI behavior intact.

**Architecture:** Treat root `DESIGN.md` as the canonical design-language source and keep `web/app/globals.css` as the current runtime projection. Rename canonical design tokens and component contracts to generic semantic names, rewrite the normative Markdown body around neutral design-system concepts, then update supporting docs and CI guards so new canonical docs do not drift back to metaphor-first vocabulary.

**Tech Stack:** Markdown, YAML front matter, `@google/design.md` CLI through `npm --prefix web run design:*`, Node-based workflow guards, existing Relay web CSS token projection in `web/app/globals.css`.

---

## Scope

This plan generalizes the canonical design source and its documentation. It does not rename runtime CSS variables or React class names in this round.

Runtime names such as `--magic-primary`, `--problem`, `relay-badge-duckling`, and `relay-badge-swan` remain compatibility aliases until a separate UI-code migration changes `web/app/globals.css`, `web/components/*`, and rendered class names together. New prose should describe those aliases as implementation compatibility, not canonical design vocabulary.

## File Structure

- Modify: `DESIGN.md`
  - Rename front matter color tokens from metaphor-specific names to generic semantic names.
  - Expand `components:` with real generic variants so `design:lint` no longer reports unused palette/dark tokens.
  - Rewrite Markdown body to remove `duckling`, `swan`, `magic`, and `problem` as normative design language.
  - Keep product identity, target audience, density, typography, layout, motion, accessibility, and success criteria.
- Modify: `docs/design-system.md`
  - Update token contract wording to generic semantic token names.
  - Add a compatibility note that existing runtime CSS names are legacy implementation aliases derived from `DESIGN.md`.
  - Update component catalog examples to generic status names while acknowledging current class names where needed.
- Modify: `CLAUDE.md`
  - Update agent-facing design rules to use generic canonical terms.
  - Keep the instruction to read `DESIGN.md` before UI work.
- Modify: `web/README.md`
  - Update web package conventions to use generic canonical terms.
  - Make clear that runtime CSS aliases remain until the web UI migration.
- Modify: `.github/workflows/design-system.yml`
  - Update the globals projection guard from metaphor-specific tokens to compatibility aliases plus canonical documentation checks.
  - Add a Node guard that fails if `DESIGN.md` reintroduces forbidden metaphor terms in normative sections.
- Read only: `web/app/globals.css`
  - Do not edit in this plan except if a comment must be clarified during Task 4.

## Canonical Vocabulary

Use this vocabulary in `DESIGN.md` front matter and normative Markdown body:

| Current term | New canonical term |
|---|---|
| `primary` | `primary` (generic design.md required bridge token) |
| `problem` | `surface-inverse` |
| `problem-soft` | `surface-inverse-muted` |
| `on-problem` | `on-inverse` |
| `magic-primary` | `accent` |
| `magic-primary-strong` | `accent-strong` |
| `magic-accent` | `accent-secondary` |
| `magic-accent-strong` | `accent-secondary-strong` |
| `DucklingBadge` | `StatusPending` |
| `SwanBadge` | `StatusComplete` |
| `magic-glow` / pastel rim-light | `accent-emphasis` |
| `duckling -> swan` motion | `review-to-approved transition` |
| `Face / Dissect / Refine / Transform` | `Capture / Analyze / Refine / Deliver` |

Terms forbidden in `DESIGN.md` normative sections after this plan:

```text
duckling
swan
magic
problem
Ugly
Elegant
Toy Workshop
mascot
```

The `References` section may mention external file names and old artifact paths if the path itself contains these words. The `Decisions Log` should summarize old decisions in general language rather than repeating retired terms.

---

### Task 1: Rename Front Matter Tokens And Component Contracts

**Files:**
- Modify: `DESIGN.md`

- [ ] **Step 1: Verify current front matter still uses metaphor-specific token names**

Run:

```bash
node <<'NODE'
const fs = require("node:fs");
const text = fs.readFileSync("DESIGN.md", "utf8");
const frontMatter = text.split("---\n")[1];
const terms = ["problem:", "magic-primary:", "magic-accent:", "dark-problem:", "dark-magic-primary:"];
const missing = terms.filter((term) => !frontMatter.includes(term));
if (missing.length > 0) {
  console.error(`Expected current metaphor tokens are missing before migration: ${missing.join(", ")}`);
  process.exit(1);
}
console.log("Current metaphor-specific front matter tokens are present");
NODE
```

Expected:

```text
Current metaphor-specific front matter tokens are present
```

- [ ] **Step 2: Replace the `colors:` block**

In `DESIGN.md`, replace the entire `colors:` block with this exact block:

```yaml
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
```

`primary` is intentionally retained as a generic bridge token because `@google/design.md@0.1.1` reports `missing-primary` when it is absent. Treat `primary` as a package compatibility token, not a Relay metaphor.

- [ ] **Step 3: Replace the `components:` block**

In `DESIGN.md`, replace the entire `components:` block with this exact block. These are real design contracts, not artificial linter-only entries.

```yaml
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
```

- [ ] **Step 4: Run lint and confirm warnings are gone**

Run:

```bash
npm --prefix web run design:lint
```

Expected: PASS with:

```json
"errors": 0,
"warnings": 0
```

`app-shell.textColor` references `colors.primary` only to satisfy the package's required primary-token rule; `colors.primary` has the same value as `colors.text`. Interactive components should use `colors.text`, `colors.on-inverse`, and other semantic tokens instead of treating `primary` as a Relay-specific design concept.

These contrast-safe component pairs intentionally use `colors.text` as a dark anchor background where light accent/success foreground tokens would otherwise fail WCAG AA on `canvas-raised`. If a new contrast warning appears, prefer adjusting the affected component to the nearest readable semantic pair and rerun.

- [ ] **Step 5: Verify removed token names are gone from front matter**

Run:

```bash
node <<'NODE'
const fs = require("node:fs");
const text = fs.readFileSync("DESIGN.md", "utf8");
const frontMatter = text.split("---\n")[1];
const forbidden = [
  "problem:",
  "problem-soft:",
  "on-problem:",
  "magic-primary:",
  "magic-primary-strong:",
  "magic-accent:",
  "magic-accent-strong:",
  "dark-problem:",
  "dark-problem-soft:",
  "dark-magic-primary:",
  "dark-magic-primary-strong:",
  "dark-magic-accent:",
  "dark-magic-accent-strong:"
];
const found = forbidden.filter((term) => frontMatter.includes(term));
if (found.length > 0) {
  console.error(`Forbidden front matter terms remain: ${found.join(", ")}`);
  process.exit(1);
}
console.log("DESIGN.md front matter uses generic token names");
NODE
```

Expected:

```text
DESIGN.md front matter uses generic token names
```

- [ ] **Step 6: Commit**

```bash
git add DESIGN.md
git commit -m "docs(design): generalize design token names"
```

---

### Task 2: Rewrite DESIGN.md Markdown Body To Generic Language

**Files:**
- Modify: `DESIGN.md`

- [ ] **Step 1: Verify current body still contains retired metaphor language**

Run:

```bash
node <<'NODE'
const fs = require("node:fs");
const text = fs.readFileSync("DESIGN.md", "utf8");
const body = text.slice(text.indexOf("# Design System"));
const terms = ["duckling", "swan", "magic", "problem", "Ugly", "Elegant", "Toy Workshop"];
const found = terms.filter((term) => new RegExp(term, "i").test(body));
if (found.length === 0) {
  console.error("Expected retired metaphor language before rewrite, but none was found");
  process.exit(1);
}
console.log(`Retired body terms still present before rewrite: ${found.join(", ")}`);
NODE
```

Expected: PASS and prints at least these terms:

```text
Retired body terms still present before rewrite:
```

- [ ] **Step 2: Replace product slogan and overview language**

In `DESIGN.md`, change:

```markdown
> **Simplicity out of Complexity.**
> We handle the Ugly, so you can be Elegant.
```

to:

```markdown
> **Clarity out of Complexity.**
> Make ambiguous work legible, reviewable, and ready to hand off.
```

Change `### The Memorable Thing` to `### Product Promise`, and replace that subsection with:

```markdown
> **A calm workspace that turns unstructured work into reviewable decisions.**

First 3 seconds, a user should feel: *"this is a serious engineering tool that makes complex work easier to continue."*

Every design decision downstream serves this sentence. If a decision makes the workspace louder, less legible, or less trustworthy, reject it.
```

- [ ] **Step 3: Replace workflow mapping**

Replace `### 4gly 4-Step Engine to Relay Domain Mapping` with:

```markdown
### Workflow Model

| Step | Relay surface | User outcome |
|------|--------------|--------------|
| 1 · **Capture** | Raw Note / Artifact / Open Question land in the project | Information is saved without premature structure |
| 2 · **Analyze** | Judgment Trace exposes workflow, alternatives, and rationale | Reasoning becomes inspectable |
| 3 · **Refine** | Heuristic Proposal is reviewed into Style Memory | Reusable judgment patterns become explicit |
| 4 · **Deliver** | Packet compiles the handoff document for the next AI consumer | Work can continue with context intact |

The UI carries a thin `Capture -> Analyze -> Refine -> Deliver` ribbon in the top rail. The active step is faintly tinted with the canonical `accent` token. This is orientation chrome, always visible, never loud.
```

- [ ] **Step 4: Replace aesthetic and brand voice language**

In `### Aesthetic Direction`, replace the current bullets with:

```markdown
- **Direction:** Warm technical clarity — a dense engineering workspace with editorial polish, restrained color, and careful status hierarchy.
- **Decoration level:** Intentional — no ornamentation. Allowed: subtle inverse surfaces for unresolved states, accent emphasis for review/approval moments, and one controlled state transition for approval.
- **Mood:** *"Someone who reads and thinks built this."* Engineer-serious on the surface, with warmth coming from typography, spacing, and copy rather than illustration.
- **Density:** Comfortable-leaning. Tighter than Granola, looser than Linear. Reference density is Reflect.app with engineering additions.
```

In `### Brand Voice`, replace the bullets and canonical action labels with:

```markdown
- **Professional, but not cold.** — `"Three proposals waiting for review."` not `"Pending items: 3"`.
- **Minimal, but not empty.** — Empty states show context, not decorative filler.
- **Review-oriented, not theatrical.** — Action button label: **"Approve pattern"** or **"Approve refinement"**, not an ornamental metaphor.
- **Show the outcome, not the pipes.** — Curator job state, lease counts, attempt numbers live in a power-user drawer, not on the primary surface.

**Canonical action labels**
- `Approve pattern` (primary, heuristic approval)
- `Edit & approve` (ghost, lightly-amended approval)
- `Reject` (danger-tinted ghost)
- `Compose handoff` (packet build)
```

- [ ] **Step 5: Replace color prose and token names**

Rewrite `## Colors` prose and CSS examples to use canonical generic names:

```markdown
Approach: **Restrained.** Accent colors never wash the canvas; they appear only where review, approval, focus, or handoff needs emphasis. Inverse surfaces carry unresolved or before-state weight. Deep navy is the primary text color.
```

In light and dark CSS blocks, use the generic tokens from Task 1. For example:

```css
--canvas:                 #FAF8F3;
--canvas-raised:          #FFFFFF;
--text:                   #0E1A35;
--text-muted:             #4A5669;
--text-subtle:            #687386;
--surface-inverse:        #1E2230;
--surface-inverse-muted:  #2B3140;
--accent:                 #A7C4FF;
--accent-strong:          #6F96DB;
--accent-secondary:       #C7B8FF;
--accent-secondary-strong:#8B76E4;
--border:                 #E7ECF3;
--border-strong:          #D4DCE8;
--success:                #2F9B73;
--danger:                 #C84646;
--focus-ring:             rgba(167, 196, 255, 0.35);
--grain-opacity:          0.03;
```

Replace the color rules with:

```markdown
1. **Accent tokens appear only at moments of emphasis.** Use them for selected paths, approval affordances, focus states, and review-complete states. They are not ambient page backgrounds.
2. **Inverse surface tokens mark unresolved or before-state content.** Use them for pending proposals, failed jobs, unresolved questions, and before-side diff panels.
3. **No ambient gradients.** A single accent sweep is allowed only for the approval transition.
4. **Dark mode is first-class.** Every component must be designed and tested in both themes. Dark canvas is deep navy, never pure black.
```

- [ ] **Step 6: Replace layout, elevation, motion, shapes, component, accessibility, and success language**

Make these exact replacements where the old body uses metaphor terms:

```markdown
Top rail active state:
The active workflow step uses `accent` emphasis.

Left rail status summary:
Projects stack by recency with small mono status glyphs for pending and complete counts.

First viewport rule:
The Style Memory screen has one primary review card. Every other element supports that card.

Elevation:
Elevation uses accent focus rings and soft outline emphasis, not generic dark-opacity drop shadows.
Use accent emphasis for important raised surfaces and review-complete states.
Keep inverse surfaces visually weighty for unresolved states, with very subtle grain controlled by `--grain-opacity`.

Motion:
Signature: 900 ms review-to-approved transition.
0-240 ms: unresolved text cluster contracts slightly (`scale: 1 -> 0.96`), rotates `-4deg`, and picks up micro-grain.
240-640 ms: accent sweep moves left-to-right (`accent-strong` at 15% opacity, `blur(12px)`).
640-900 ms: a graph-edge contour resolves into the approved-heuristic chip, packet cover, or decision node.

Iconography and illustration:
Use custom line icons, 1.5 px stroke, inheriting `currentColor`. Lucide remains a placeholder during early dev.
Do not use mascots or decorative illustrations on primary work surfaces.
Status glyphs in Project Explorer use small mono characters (`◈`, `△`, `○`) colored by pending/complete roles. No emoji.

StatusPending / StatusComplete:
StatusPending: `surface-inverse` background, `on-inverse` text.
StatusComplete: `accent` background, `on-accent` text.
```

Update `Do's and Don'ts` to:

```markdown
### Do

- Use accent tokens only for focus, review, selected paths, and completion emphasis.
- Use inverse surface tokens for unresolved, pending, failed, or before-state content.
- Use soft focus-ring elevation for important surfaces.
- Preserve the visible `Capture -> Analyze -> Refine -> Deliver` workflow in app workspaces.
- Keep Style Memory as the signature review surface.
- Use sentence-case product copy that is professional but not cold.
- Respect `prefers-reduced-motion` for approval transitions.

### Don't

- Do not use accent tokens as ambient page background.
- Do not use generic dark-opacity drop shadows as the primary elevation model.
- Do not use confetti, bounce, emoji, mascots, or decorative metaphors.
- Do not expose curator job state, lease counters, or attempt numbers on primary surfaces.
- Do not use Inter, Roboto, Arial, Helvetica, Open Sans, Lato, Montserrat, Poppins, Space Grotesk, Clash Display, or system UI fonts as display or body.
- Do not apply pill radius uniformly to every element.
- Do not add more than one approval transition on screen at once.
```

Update `Success Bar` to:

```markdown
1. **The 3-second rule.** A first-time viewer feels: *"this tool makes complex work legible and ready to continue."*
2. **Show the outcome rule.** User finishes a session without seeing `curator_jobs.state`, lease counters, or attempt numbers on the primary surface.
3. **Signature rule.** The Style Memory screen alone explains what Relay is. If that screen is removed, the product's differentiator disappears with it.
4. **Density + calm rule.** Density is Linear-class. Breath is Arc-class. Feel is 4gly-class.
5. **The Anchor.** Re-read `4gly_foundation.md §12`. If this UI weakens that foundation, rebuild the design language before implementing.
```

- [ ] **Step 7: Rewrite Decisions Log without retired vocabulary**

Replace the `Decisions Log` rows with this content:

```markdown
| Date | Decision | Rationale |
|------|----------|-----------|
| 2026-04-25 | Product promise locked: *"A calm workspace that turns unstructured work into reviewable decisions."* | Preserves the original clarity-through-complexity direction while removing metaphor-first language from the canonical design source. |
| 2026-04-25 | Keep 4gly's pastel blue/purple as accent tokens | Brand continuity with 4gly parent remains important, but canonical token names are now semantic and reusable. |
| 2026-04-25 | Keep Nunito as body | 4gly declared font inheritance; warmth preserved in daily UI text. |
| 2026-04-25 | Add Fraunces as display + editorial italic | Both Codex and Claude subagent converged on serif for authored knowledge surfaces. |
| 2026-04-25 | Add Commit Mono (stand-in JetBrains Mono) as first-class mono | Both outside voices plus Linear/Raycast convergence; engineering trust signal. |
| 2026-04-25 | Use inverse surfaces for unresolved state weight | Unresolved and before-state content must feel materially distinct for review outcomes to matter. |
| 2026-04-25 | Signature motion = 900 ms graph-contour approval transition | Rejects mascot branding that senior engineers resist; makes evidence and graph behavior the brand object. |
| 2026-04-25 | Layout = hybrid 3-region engineering workspace + editorial serif surfaces | Codex's 3-region workspace plus subagent's authored-content insight. |
| 2026-04-25 | Initial design system created | `/gstack-design-consultation` session based on 4gly foundation, Relay domain exploration, 3-voice synthesis, and competitor scan. |
```

- [ ] **Step 8: Verify retired body language is gone**

Run:

```bash
node <<'NODE'
const fs = require("node:fs");
const text = fs.readFileSync("DESIGN.md", "utf8");
const body = text.slice(text.indexOf("# Design System"));
const forbidden = [
  /\bduckling\b/i,
  /\bswan\b/i,
  /\bmagic\b/i,
  /\bproblem\b/i,
  /\bUgly\b/,
  /\bElegant\b/,
  /Toy Workshop/i,
  /\bmascot\b/i
];
const found = forbidden.filter((pattern) => pattern.test(body)).map(String);
if (found.length > 0) {
  console.error(`Retired DESIGN.md body language remains: ${found.join(", ")}`);
  process.exit(1);
}
console.log("DESIGN.md body uses generic design language");
NODE
```

Expected:

```text
DESIGN.md body uses generic design language
```

- [ ] **Step 9: Run lint and export**

Run:

```bash
npm --prefix web run design:lint
npm --prefix web run design:export:tailwind >/tmp/relay-design-tailwind-theme.json
test -s /tmp/relay-design-tailwind-theme.json
rg --fixed-strings -- '"accent"' /tmp/relay-design-tailwind-theme.json
rg --fixed-strings -- '"surface-inverse"' /tmp/relay-design-tailwind-theme.json
```

Expected:

- `design:lint` exits 0 with `errors: 0`, `warnings: 0`.
- export file is non-empty and contains `"accent"` plus `"surface-inverse"`.

- [ ] **Step 10: Commit**

```bash
git add DESIGN.md
git commit -m "docs(design): generalize canonical design language"
```

---

### Task 3: Update Supporting Design Documentation

**Files:**
- Modify: `docs/design-system.md`
- Modify: `CLAUDE.md`
- Modify: `web/README.md`

- [ ] **Step 1: Verify supporting docs still use retired canonical language**

Run:

```bash
rg -n "duckling|swan|magic|problem|Ugly|Elegant|mascot|--magic|--problem|relay-badge-duckling|relay-badge-swan" docs/design-system.md CLAUDE.md web/README.md
```

Expected: PASS with matches.

- [ ] **Step 2: Update `docs/design-system.md` token contract**

In `docs/design-system.md`, replace section `### 2.2 색/표면 토큰` with:

```markdown
### 2.2 색/표면 토큰

- Surface
  - `--canvas`, `--canvas-raised`
  - `--surface-inverse`, `--surface-inverse-muted`
- Text
  - `--text`, `--text-muted`, `--text-subtle`, `--on-accent`, `--on-inverse`, `--on-danger`
- Accent / State
  - `--accent`, `--accent-strong`, `--accent-secondary`, `--accent-secondary-strong`
  - `--success`, `--danger`
- Border / Focus
  - `--border`, `--border-strong`, `--focus-ring`, `--focus-ring-strong`, `--grain-opacity`
- Typography
  - `--font-display`, `--font-sans`, `--font-mono`
- Shape
  - `--radius-card`, `--radius-pill`

Runtime note: the current web CSS still exposes compatibility aliases such as
`--magic-primary` and `--problem` while the React/CSS migration catches up.
Those aliases are implementation details. New design language and new docs use
the generic canonical tokens above.
```

- [ ] **Step 3: Update `docs/design-system.md` component catalog wording**

Replace the project navigation bullet currently listing `relay-badge-duckling`, `relay-badge-swan` with:

```markdown
- 프로젝트 내비게이션(현재 Shell 내부)
  - Canonical roles: `status-pending`, `status-complete`
  - Current runtime classes: `relay-project-rail`, `relay-rail-section`,
    `relay-rail-item`, `relay-rail-glyph`, `relay-rail-name`,
    `relay-badge-duckling`, `relay-badge-swan`
  - 상태: `relay-rail-item[data-active="true"]`, `relay-rail-glyph[data-kind="snapshot|pending|active"]`
```

This is the one allowed occurrence of the old class names in `docs/design-system.md` because it documents runtime compatibility.

- [ ] **Step 4: Update `CLAUDE.md` design rules**

Replace the visual rules under `When you write or review any UI code:` with:

```markdown
- Use only tokens defined in `DESIGN.md` `Colors` and the token front matter.
- Use canonical token language in new docs: `accent`, `surface-inverse`, `status-pending`, `status-complete`, `focus-ring`.
- Treat existing `--magic-*`, `--problem-*`, and badge class names as runtime compatibility aliases until the web CSS migration removes them.
- Use the declared typefaces: **Fraunces** (display + editorial italic), **Nunito** (body / UI), **Commit Mono** (mono, with **JetBrains Mono** as stand-in until licensed).
- Never reintroduce banned fonts (Inter, Roboto, Arial, system-ui, Space Grotesk, Poppins, Montserrat, Helvetica, Open Sans, Lato, Clash Display).
- Accent tokens appear only for focus, review, selected paths, and completion emphasis. They are never ambient background.
- Elevation uses soft focus-ring emphasis, not drop-shadow.
```

Replace the memorable thing and non-negotiable illustration rule with:

```markdown
The product promise: **"A calm workspace that turns unstructured work into reviewable decisions."**
Every UI change must serve that sentence.
```

and:

```markdown
- Do not add decorative metaphor or mascot work on the primary surface — evidence, review state, and graph behavior carry the product identity.
```

- [ ] **Step 5: Update `web/README.md` conventions**

Replace the S6 bullet:

```markdown
- **S6** — Style Memory authenticated UI with the 900 ms review-to-approved transition.
```

Replace the conventions with:

```markdown
- Use only `DESIGN.md` `Colors` and token front matter tokens such as `--canvas`, `--text`, `--accent`, `--surface-inverse`, and related light/dark values.
- Use only the declared typefaces: Fraunces, Nunito, JetBrains Mono, and LXGW WenKai KR for Korean UI.
- Accent tokens only appear for focus, review, selected paths, and completion emphasis — never as ambient background.
- Elevation is soft focus-ring emphasis, not drop-shadow.
- Existing `--magic-*` and `--problem-*` CSS variables are runtime compatibility aliases until the web CSS migration removes them.
```

- [ ] **Step 6: Verify supporting docs**

Run:

```bash
node <<'NODE'
const fs = require("node:fs");
const files = ["CLAUDE.md", "web/README.md"];
const forbidden = [/duckling/i, /swan/i, /magic/i, /problem/i, /Ugly/, /Elegant/, /mascot/i];
for (const file of files) {
  const text = fs.readFileSync(file, "utf8");
  const found = forbidden.filter((pattern) => pattern.test(text)).map(String);
  if (found.length > 0) {
    console.error(`${file} still contains retired language: ${found.join(", ")}`);
    process.exit(1);
  }
}
const designSystem = fs.readFileSync("docs/design-system.md", "utf8");
const required = [
  "Canonical roles: `status-pending`, `status-complete`",
  "runtime compatibility",
  "`--accent`",
  "`--surface-inverse`"
];
const missing = required.filter((needle) => !designSystem.includes(needle));
if (missing.length > 0) {
  console.error(`docs/design-system.md missing required generic contract text: ${missing.join(", ")}`);
  process.exit(1);
}
console.log("supporting docs use generic design language");
NODE
```

Expected:

```text
supporting docs use generic design language
```

- [ ] **Step 7: Run lint and commit**

Run:

```bash
npm --prefix web run design:lint
git diff --check -- docs/design-system.md CLAUDE.md web/README.md
```

Expected: `design:lint` exits 0 with `errors: 0`, `warnings: 0`; diff check exits 0.

Commit:

```bash
git add docs/design-system.md CLAUDE.md web/README.md
git commit -m "docs(design): align supporting docs to generic language"
```

---

### Task 4: Update CI Guards For Generic Canonical Language

**Files:**
- Modify: `.github/workflows/design-system.yml`

- [ ] **Step 1: Verify the workflow still guards old runtime aliases**

Run:

```bash
rg -n "magic|problem|docs/design\\.md|requiredGlobals|DESIGN.md token" .github/workflows/design-system.yml
```

Expected: PASS with matches for `magic` and `requiredGlobals`.

- [ ] **Step 2: Replace token projection guard**

In `.github/workflows/design-system.yml`, replace `requiredGlobals` with this version. It acknowledges runtime aliases but validates generic canonical docs:

```js
          const requiredGlobals = [
            "--color-canvas: #faf8f3",
            "--color-ink: #0e1a35",
            "--canvas: #faf8f3",
            "--ink: #0e1a35",
            "--magic-primary: #a7c4ff",
            "--problem: #1e2230"
          ];
```

Then add this `DESIGN.md` body guard after the globals check:

```js
          const design = fs.readFileSync("DESIGN.md", "utf8");
          const body = design.slice(design.indexOf("# Design System"));
          const retired = [
            /\bduckling\b/i,
            /\bswan\b/i,
            /\bmagic\b/i,
            /\bproblem\b/i,
            /\bUgly\b/,
            /\bElegant\b/,
            /Toy Workshop/i,
            /\bmascot\b/i
          ];
          const foundRetired = retired.filter((pattern) => pattern.test(body)).map(String);
          if (foundRetired.length > 0) {
            console.error(`Retired metaphor language remains in DESIGN.md body: ${foundRetired.join(", ")}`);
            process.exit(1);
          }
```

Keep the stale `docs/design.md` checks from the previous workflow.

- [ ] **Step 3: Add workflow path filters**

Ensure `paths:` includes all files touched by this plan:

```yaml
      - "DESIGN.md"
      - "docs/design-system.md"
      - "CLAUDE.md"
      - "web/README.md"
      - "docs/api.md"
      - "internal/api/server_test.go"
      - "internal/services/services_test.go"
      - "internal/services/style_memory_test.go"
      - "web/app/globals.css"
      - "web/package.json"
      - "web/package-lock.json"
      - ".github/workflows/design-system.yml"
```

- [ ] **Step 4: Run workflow guard locally**

Run:

```bash
node <<'NODE'
const fs = require("node:fs");

const forbidden = [
  ["docs/api.md", "docs/design.md"],
  ["internal/api/server_test.go", "docs/design.md"],
  ["internal/services/services_test.go", "docs/design.md"],
  ["internal/services/style_memory_test.go", "docs/design.md"]
];

const stale = forbidden.filter(([file, needle]) =>
  fs.readFileSync(file, "utf8").includes(needle)
);
if (stale.length > 0) {
  console.error(`Stale design document path references: ${stale.map(([file]) => file).join(", ")}`);
  process.exit(1);
}

const globals = fs.readFileSync("web/app/globals.css", "utf8");
const requiredGlobals = [
  "--color-canvas: #faf8f3",
  "--color-ink: #0e1a35",
  "--canvas: #faf8f3",
  "--ink: #0e1a35",
  "--magic-primary: #a7c4ff",
  "--problem: #1e2230"
];
const missing = requiredGlobals.filter((needle) => !globals.includes(needle));
if (missing.length > 0) {
  console.error(`Missing runtime token projection in globals.css: ${missing.join(", ")}`);
  process.exit(1);
}

const design = fs.readFileSync("DESIGN.md", "utf8");
const body = design.slice(design.indexOf("# Design System"));
const retired = [/\bduckling\b/i, /\bswan\b/i, /\bmagic\b/i, /\bproblem\b/i, /\bUgly\b/, /\bElegant\b/, /Toy Workshop/i, /\bmascot\b/i];
const foundRetired = retired.filter((pattern) => pattern.test(body)).map(String);
if (foundRetired.length > 0) {
  console.error(`Retired metaphor language remains in DESIGN.md body: ${foundRetired.join(", ")}`);
  process.exit(1);
}

console.log("canonical design references, runtime aliases, and generic DESIGN.md body are valid");
NODE
```

Expected:

```text
canonical design references, runtime aliases, and generic DESIGN.md body are valid
```

- [ ] **Step 5: Run remaining checks and commit**

Run:

```bash
if command -v actionlint >/dev/null 2>&1; then
  actionlint .github/workflows/design-system.yml
else
  echo "actionlint not installed; workflow syntax will be validated by GitHub Actions"
fi
npm --prefix web run design:lint
git diff --check -- .github/workflows/design-system.yml
```

Expected:

- `actionlint` passes if installed, otherwise prints fallback.
- `design:lint` exits 0 with `errors: 0`, `warnings: 0`.
- diff check exits 0.

Commit:

```bash
git add .github/workflows/design-system.yml
git commit -m "ci: guard generic design language"
```

---

### Task 5: Final Verification

**Files:**
- Read: `DESIGN.md`
- Read: `docs/design-system.md`
- Read: `CLAUDE.md`
- Read: `web/README.md`
- Read: `.github/workflows/design-system.yml`
- Read: `web/package.json`

- [ ] **Step 1: Run design.md lint and export**

Run:

```bash
npm --prefix web run design:lint
npm --prefix web run design:export:tailwind >/tmp/relay-design-tailwind-theme.json
test -s /tmp/relay-design-tailwind-theme.json
rg --fixed-strings -- '"accent"' /tmp/relay-design-tailwind-theme.json
rg --fixed-strings -- '"surface-inverse"' /tmp/relay-design-tailwind-theme.json
```

Expected:

- `design:lint` exits 0 with `errors: 0`, `warnings: 0`.
- export output includes `"accent"` and `"surface-inverse"`.

- [ ] **Step 2: Run docs language guard**

Run:

```bash
node <<'NODE'
const fs = require("node:fs");
const design = fs.readFileSync("DESIGN.md", "utf8");
const body = design.slice(design.indexOf("# Design System"));
const forbidden = [/\bduckling\b/i, /\bswan\b/i, /\bmagic\b/i, /\bproblem\b/i, /\bUgly\b/, /\bElegant\b/, /Toy Workshop/i, /\bmascot\b/i];
const found = forbidden.filter((pattern) => pattern.test(body)).map(String);
if (found.length > 0) {
  console.error(`Retired DESIGN.md body language remains: ${found.join(", ")}`);
  process.exit(1);
}
for (const file of ["CLAUDE.md", "web/README.md"]) {
  const text = fs.readFileSync(file, "utf8");
  const foundInFile = forbidden.filter((pattern) => pattern.test(text)).map(String);
  if (foundInFile.length > 0) {
    console.error(`${file} still contains retired design language: ${foundInFile.join(", ")}`);
    process.exit(1);
  }
}
console.log("canonical docs use generic design language");
NODE
```

Expected:

```text
canonical docs use generic design language
```

- [ ] **Step 3: Run workflow guard**

Run the same Node command from Task 4 Step 4.

Expected:

```text
canonical design references, runtime aliases, and generic DESIGN.md body are valid
```

- [ ] **Step 4: Run web and targeted backend checks**

Run:

```bash
npm --prefix web run lint
npm --prefix web run typecheck
npm --prefix web run test
go test ./internal/api ./internal/services
git diff --check
```

Expected:

- `npm --prefix web run lint` exits 0.
- `npm --prefix web run typecheck` exits 0.
- `npm --prefix web run test` exits 0 with all tests passing.
- `go test ./internal/api ./internal/services` exits 0.
- `git diff --check` exits 0.

- [ ] **Step 5: Confirm no runtime files were renamed**

Run:

```bash
git diff --name-only origin/main...HEAD | rg -n "web/app/globals\\.css|web/components|web/app" || true
```

Expected: No `web/components` or `web/app` runtime component files are listed unless Task 4 updated only the `web/app/globals.css` comment. This confirms the runtime CSS/class migration remains out of scope.

- [ ] **Step 6: Commit final cleanup if needed**

If Task 5 revealed a missing docs guard or formatting-only cleanup, commit it:

```bash
git add DESIGN.md docs/design-system.md CLAUDE.md web/README.md .github/workflows/design-system.yml
git commit -m "docs(design): finalize generic design language"
```

If Task 5 produced no changes, do not create an empty commit.

---

## Self-Review

**Spec coverage:** The plan covers generic front matter token names, warning-free component references, Markdown body rewrite, supporting docs, CI guard updates, and final verification. Runtime CSS/class rename is explicitly out of scope.

**Placeholder scan:** The plan contains no unresolved file paths and no instruction that depends on undefined helpers.

**Type consistency:** Canonical token references consistently use `{colors.accent}`, `{colors.surface-inverse}`, `{colors.text}`, `{colors.canvas}`, `{rounded.*}`, `{spacing.*}`, and `{typography.*}`. Runtime compatibility aliases remain documented as aliases only.

## Execution Handoff

Plan complete and saved to `docs/superpowers/plans/2026-05-05-generalize-design-language.md`. Two execution options:

**1. Subagent-Driven (recommended)** - I dispatch a fresh subagent per task, review between tasks, fast iteration

**2. Inline Execution** - Execute tasks in this session using executing-plans, batch execution with checkpoints

Which approach?
