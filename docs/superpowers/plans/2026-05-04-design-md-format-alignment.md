# DESIGN.md Format Alignment Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Convert Relay's existing `DESIGN.md` into a `google-labs-code/design.md` compatible source of truth while preserving the current 4gly/Relay visual language.

**Architecture:** Keep `DESIGN.md` as the canonical product design source, add machine-readable YAML front matter for tokens, and reorganize the prose into the canonical `design.md` section order. Add local validation through the official `@google/design.md` CLI so future token or prose drift is caught before merge.

**Tech Stack:** Markdown, YAML front matter, `@google/design.md` CLI, npm scripts in `web/package.json`, existing Relay design docs under `docs/design-system.md`, Tailwind v4 token CSS in `web/app/globals.css`.

---

## File Structure

- Modify: `DESIGN.md`
  - Add `design.md` YAML front matter at the top.
  - Reorder the body to canonical `##` sections: `Overview`, `Colors`, `Typography`, `Layout`, `Elevation & Depth`, `Shapes`, `Components`, `Do's and Don'ts`.
  - Preserve Relay-specific sections as post-spec appendices: `Accessibility`, `Tech Stack`, `Non-Scope`, `Success Bar`, `References`, `Decisions Log`.
- Modify: `docs/design-system.md`
  - Mark `DESIGN.md` as the normative design source.
  - Keep `docs/design-system.md` as the implementation contract for `web/components/relay/*` and `web/app/globals.css`.
  - Add the validation command used by contributors.
- Modify: `web/package.json`
  - Add `@google/design.md` as a dev dependency through npm.
  - Add `design:lint`, `design:spec`, and `design:export:tailwind` scripts.
- Modify: `web/package-lock.json`
  - Let `npm --prefix web install --save-dev @google/design.md@^0.1.1` update the lockfile.
- Create: `.github/workflows/design-system.yml`
  - Run `npm ci` inside `web`.
  - Run `npm --prefix web run design:lint`.
  - Run only for `DESIGN.md`, `docs/design-system.md`, the new workflow, and `web/package*.json` changes.

## External Spec Facts Used

- A `DESIGN.md` file may combine YAML front matter tokens with Markdown rationale.
- Token groups include `colors`, `typography`, `rounded`, `spacing`, and `components`.
- Component token properties accepted by the spec include `backgroundColor`, `textColor`, `typography`, `rounded`, `padding`, `size`, `height`, and `width`.
- The canonical Markdown section order is `Overview`, `Colors`, `Typography`, `Layout`, `Elevation & Depth`, `Shapes`, `Components`, `Do's and Don'ts`.
- `design.md lint DESIGN.md` exits `1` only when errors are found; warnings are acceptable but should be reviewed.

---

### Task 1: Add Official DESIGN.md CLI Validation

**Files:**
- Modify: `web/package.json`
- Modify: `web/package-lock.json`

- [ ] **Step 1: Verify the validation script does not exist yet**

Run:

```bash
npm --prefix web run design:lint
```

Expected: FAIL with this npm script error:

```text
Missing script: "design:lint"
```

- [ ] **Step 2: Install the CLI package**

Run:

```bash
npm --prefix web install --save-dev @google/design.md@^0.1.1
```

Expected: PASS and `web/package.json` plus `web/package-lock.json` are modified.

- [ ] **Step 3: Add validation scripts**

Edit `web/package.json` so the `scripts` block contains these exact entries:

```json
{
  "dev": "next dev",
  "build": "next build",
  "start": "next start",
  "lint": "eslint .",
  "typecheck": "tsc --noEmit",
  "qa:e2e": "playwright test",
  "test": "vitest run",
  "test:watch": "vitest",
  "design:lint": "design.md lint --format json ../DESIGN.md",
  "design:spec": "design.md spec --rules",
  "design:export:tailwind": "design.md export --format tailwind ../DESIGN.md"
}
```

- [ ] **Step 4: Run validation against the current file**

Run:

```bash
npm --prefix web run design:lint
```

Expected: The command completes, but the JSON output is not yet acceptable because the current `DESIGN.md` has no machine-readable front matter and uses numbered section headings rather than the canonical section order. Keep the output open for comparison while completing Task 2 and Task 3.

- [ ] **Step 5: Commit the CLI setup**

```bash
git add web/package.json web/package-lock.json
git commit -m "chore(web): add design md validation"
```

---

### Task 2: Add Machine-Readable Token Front Matter

**Files:**
- Modify: `DESIGN.md`

- [ ] **Step 1: Verify `DESIGN.md` does not start with YAML front matter**

Run:

```bash
node - <<'NODE'
const fs = require("fs");
const text = fs.readFileSync("DESIGN.md", "utf8");
if (!text.startsWith("---\n")) {
  console.error("DESIGN.md is missing YAML front matter");
  process.exit(1);
}
console.log("DESIGN.md has YAML front matter");
NODE
```

Expected: FAIL with:

```text
DESIGN.md is missing YAML front matter
```

- [ ] **Step 2: Insert this exact front matter at the top of `DESIGN.md`**

Place this block before the existing `# Design System — Relay (project-swan · 4gly Labs)` heading:

```yaml
---
version: alpha
name: Relay
description: 4gly Labs product design system for Relay, a calm engineering workspace that turns chaotic AI work into reusable decisions, style memory, and handoff packets.
colors:
  primary: "#0E1A35"
  secondary: "#4A5669"
  tertiary: "#6F96DB"
  neutral: "#FAF8F3"
  canvas: "#FAF8F3"
  canvas-raised: "#FFFFFF"
  ink: "#0E1A35"
  ink-muted: "#4A5669"
  muted: "#687386"
  problem: "#1E2230"
  problem-soft: "#2B3140"
  magic-primary: "#A7C4FF"
  magic-primary-strong: "#6F96DB"
  magic-accent: "#C7B8FF"
  magic-accent-strong: "#8B76E4"
  border: "#E7ECF3"
  border-strong: "#D4DCE8"
  success: "#2F9B73"
  danger: "#C84646"
  on-primary: "#FFFFFF"
  on-problem: "#FAF8F3"
  dark-canvas: "#0B1020"
  dark-canvas-raised: "#121930"
  dark-ink: "#EEF4FF"
  dark-ink-muted: "#A5B1C8"
  dark-problem: "#06091A"
  dark-problem-soft: "#1A1F30"
  dark-magic-primary: "#7FB6FF"
  dark-magic-primary-strong: "#A7C4FF"
  dark-magic-accent: "#A78BFA"
  dark-magic-accent-strong: "#C7B8FF"
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
    textColor: "{colors.ink}"
    typography: "{typography.body}"
    width: 1440px
  top-rail:
    backgroundColor: "{colors.canvas}"
    textColor: "{colors.ink}"
    height: 56px
    padding: 24px
  project-rail:
    backgroundColor: "{colors.canvas-raised}"
    textColor: "{colors.ink-muted}"
    width: 240px
    padding: 16px
  inspector:
    backgroundColor: "{colors.canvas-raised}"
    textColor: "{colors.ink-muted}"
    width: 320px
    rounded: "{rounded.chrome}"
    padding: 24px
  button-primary:
    backgroundColor: "{colors.primary}"
    textColor: "{colors.on-primary}"
    typography: "{typography.mono-label}"
    rounded: "{rounded.control}"
    padding: 12px
  button-secondary:
    backgroundColor: "{colors.canvas-raised}"
    textColor: "{colors.ink-muted}"
    typography: "{typography.mono-label}"
    rounded: "{rounded.control}"
    padding: 12px
  button-danger:
    backgroundColor: "{colors.danger}"
    textColor: "{colors.on-primary}"
    typography: "{typography.mono-label}"
    rounded: "{rounded.control}"
    padding: 12px
  card:
    backgroundColor: "{colors.canvas-raised}"
    textColor: "{colors.ink}"
    typography: "{typography.body}"
    rounded: "{rounded.card}"
    padding: 24px
  source-chip:
    backgroundColor: "{colors.canvas-raised}"
    textColor: "{colors.ink-muted}"
    typography: "{typography.mono-label}"
    rounded: "{rounded.pill}"
    padding: 8px
  status-badge:
    backgroundColor: "{colors.canvas-raised}"
    textColor: "{colors.ink-muted}"
    typography: "{typography.mono-label}"
    rounded: "{rounded.pill}"
    padding: 8px
  transform-step-active:
    backgroundColor: "{colors.magic-primary}"
    textColor: "{colors.magic-primary-strong}"
    typography: "{typography.mono-label}"
    rounded: "{rounded.pill}"
    padding: 8px
  diff-before:
    backgroundColor: "{colors.problem-soft}"
    textColor: "{colors.on-problem}"
    typography: "{typography.mono-label}"
    rounded: "{rounded.control}"
    padding: 12px
  diff-after:
    backgroundColor: "{colors.magic-primary}"
    textColor: "{colors.ink}"
    typography: "{typography.mono-label}"
    rounded: "{rounded.control}"
    padding: 12px
---
```

- [ ] **Step 3: Verify the front matter exists**

Run:

```bash
node - <<'NODE'
const fs = require("fs");
const text = fs.readFileSync("DESIGN.md", "utf8");
if (!text.startsWith("---\nversion: alpha\nname: Relay\n")) {
  console.error("DESIGN.md front matter does not start with the expected Relay header");
  process.exit(1);
}
if (!text.includes('button-primary:') || !text.includes('backgroundColor: "{colors.primary}"')) {
  console.error("DESIGN.md front matter is missing component token references");
  process.exit(1);
}
console.log("DESIGN.md front matter is present");
NODE
```

Expected:

```text
DESIGN.md front matter is present
```

- [ ] **Step 4: Run the official linter**

Run:

```bash
npm --prefix web run design:lint
```

Expected: exit code `0`, and the JSON output contains:

```json
"errors": 0
```

Warnings are acceptable at this stage only if they are contrast-ratio informational warnings or orphaned-token warnings for dark-mode tokens.

- [ ] **Step 5: Commit the token front matter**

```bash
git add DESIGN.md
git commit -m "docs(design): add design md token front matter"
```

---

### Task 3: Reorganize `DESIGN.md` Into Canonical Section Order

**Files:**
- Modify: `DESIGN.md`

- [ ] **Step 1: Verify the current canonical sections are missing**

Run:

```bash
node - <<'NODE'
const fs = require("fs");
const text = fs.readFileSync("DESIGN.md", "utf8");
const required = [
  "## Overview",
  "## Colors",
  "## Typography",
  "## Layout",
  "## Elevation & Depth",
  "## Shapes",
  "## Components",
  "## Do's and Don'ts"
];
const missing = required.filter((heading) => !text.includes(heading));
if (missing.length > 0) {
  console.error(`Missing canonical sections: ${missing.join(", ")}`);
  process.exit(1);
}
console.log("Canonical sections are present");
NODE
```

Expected before the edit: FAIL with missing canonical sections.

- [ ] **Step 2: Rename and move the existing prose into this exact section order**

Keep the existing title and quote after the YAML front matter. Then rewrite the top-level `##` headings so they appear in this order:

```markdown
## Overview

### Product Context

### The Memorable Thing

### 4gly 4-Step Engine to Relay Domain Mapping

### Aesthetic Direction

### Brand Voice

## Colors

### Light theme tokens

### Dark theme tokens

### Color rules

## Typography

### Type roles

### Scale

### Loading

### Bans

## Layout

### Spacing and Grid

### App Shell

### First Viewport Rule

## Elevation & Depth

### Halo Elevation

### Motion

## Shapes

### Radius Scale

### Iconography and Illustration

## Components

### TraceCard

### HeuristicDiff

### PacketPreview

### SourceChip

### ScopeMatrixCell

### DucklingBadge and SwanBadge

### TransformRibbon

## Do's and Don'ts

### Do

### Don't

## Accessibility

## Tech Stack

## Non-Scope

## Success Bar

## References

## Decisions Log
```

Move the current content as follows:

- Move current `## 0. Product Context` under `## Overview` as `### Product Context`.
- Move current `## 1. The Memorable Thing` under `## Overview` as `### The Memorable Thing`.
- Move current `## 2. 4gly 4-Step Engine → Relay Domain Mapping` under `## Overview` as `### 4gly 4-Step Engine to Relay Domain Mapping`.
- Move current `## 3. Aesthetic Direction` under `## Overview` as `### Aesthetic Direction`.
- Move current `## 4. Brand Voice (4gly foundation, applied to UI copy)` under `## Overview` as `### Brand Voice`.
- Rename current `## 6. Color` to `## Colors` and keep its subsections.
- Rename current `## 5. Typography` to `## Typography` and keep its tables under the new subsection names above.
- Split current `## 7. Spacing & Grid` and `## 8. Layout` into the new `## Layout` section.
- Move the elevation bullets from current `## 6. Color` and the current `## 9. Motion` content into `## Elevation & Depth`.
- Move current radius bullets from `## 7. Spacing & Grid` plus current `## 10. Iconography & Illustration` into `## Shapes`.
- Rename current `## 11. Component Contracts` to `## Components`.
- Create `## Do's and Don'ts` using the explicit rules already present in the current document:

```markdown
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
```

- [ ] **Step 3: Verify canonical section order**

Run:

```bash
node - <<'NODE'
const fs = require("fs");
const text = fs.readFileSync("DESIGN.md", "utf8");
const required = [
  "## Overview",
  "## Colors",
  "## Typography",
  "## Layout",
  "## Elevation & Depth",
  "## Shapes",
  "## Components",
  "## Do's and Don'ts"
];
let previous = -1;
for (const heading of required) {
  const index = text.indexOf(heading);
  if (index === -1) {
    console.error(`Missing ${heading}`);
    process.exit(1);
  }
  if (index <= previous) {
    console.error(`${heading} appears out of order`);
    process.exit(1);
  }
  previous = index;
}
console.log("Canonical DESIGN.md sections are present and ordered");
NODE
```

Expected:

```text
Canonical DESIGN.md sections are present and ordered
```

- [ ] **Step 4: Run official linter again**

Run:

```bash
npm --prefix web run design:lint
```

Expected: exit code `0`, and the JSON output contains:

```json
"errors": 0
```

There should be no `section-order` warning for the canonical sections.

- [ ] **Step 5: Commit the section reorganization**

```bash
git add DESIGN.md
git commit -m "docs(design): align design md section order"
```

---

### Task 4: Update Implementation Contract Documentation

**Files:**
- Modify: `docs/design-system.md`

- [ ] **Step 1: Verify the implementation doc does not name the canonical source**

Run:

```bash
node - <<'NODE'
const fs = require("fs");
const text = fs.readFileSync("docs/design-system.md", "utf8");
if (!text.includes("DESIGN.md is the canonical visual identity source")) {
  console.error("docs/design-system.md does not declare DESIGN.md as canonical");
  process.exit(1);
}
console.log("docs/design-system.md declares the canonical source");
NODE
```

Expected before the edit: FAIL with:

```text
docs/design-system.md does not declare DESIGN.md as canonical
```

- [ ] **Step 2: Add this source-of-truth note after the purpose section**

Insert this block after the initial bullet list in `docs/design-system.md` section `0) 목적`:

```markdown
### Canonical Source Relationship

`DESIGN.md` is the canonical visual identity source for Relay and follows the
`google-labs-code/design.md` format: YAML token front matter plus canonical
Markdown rationale sections. This document is the implementation contract for
how those tokens and decisions appear in Relay code.

- `DESIGN.md`: brand meaning, normative tokens, component token references,
  section-ordered design rationale.
- `web/app/globals.css`: Tailwind v4 token projection and concrete `relay-*`
  class behavior.
- `web/components/relay/*`: React primitive API for screens.
- `docs/design-system.md`: engineering rules for applying the system without
  re-litigating brand decisions.

Validate the canonical source before changing implementation details:

```bash
npm --prefix web run design:lint
```
```

- [ ] **Step 3: Add a reference to the external spec**

Append this bullet to the `## 7) References` list:

```markdown
- DESIGN.md format specification:
  `https://github.com/google-labs-code/design.md`
```

- [ ] **Step 4: Verify the doc link and command**

Run:

```bash
node - <<'NODE'
const fs = require("fs");
const text = fs.readFileSync("docs/design-system.md", "utf8");
const required = [
  "DESIGN.md is the canonical visual identity source",
  "npm --prefix web run design:lint",
  "https://github.com/google-labs-code/design.md"
];
const missing = required.filter((item) => !text.includes(item));
if (missing.length > 0) {
  console.error(`Missing docs contract strings: ${missing.join(", ")}`);
  process.exit(1);
}
console.log("docs/design-system.md references the canonical DESIGN.md contract");
NODE
```

Expected:

```text
docs/design-system.md references the canonical DESIGN.md contract
```

- [ ] **Step 5: Commit the implementation docs update**

```bash
git add docs/design-system.md
git commit -m "docs(design): document design md source contract"
```

---

### Task 5: Add Pull Request Validation

**Files:**
- Create: `.github/workflows/design-system.yml`

- [ ] **Step 1: Verify no design-system workflow exists**

Run:

```bash
test ! -f .github/workflows/design-system.yml
```

Expected: PASS.

- [ ] **Step 2: Create the workflow**

Create `.github/workflows/design-system.yml` with this exact content:

```yaml
name: design-system

on:
  pull_request:
    branches:
      - main
    paths:
      - "DESIGN.md"
      - "docs/design-system.md"
      - "web/package.json"
      - "web/package-lock.json"
      - ".github/workflows/design-system.yml"
  workflow_dispatch:

permissions:
  contents: read

concurrency:
  group: design-system-${{ github.event.pull_request.number || github.ref }}
  cancel-in-progress: true

jobs:
  design-md:
    runs-on: ubuntu-latest
    timeout-minutes: 10
    steps:
      - name: Checkout
        uses: actions/checkout@v6

      - name: Set up Node.js
        uses: actions/setup-node@v6
        with:
          node-version: "24"
          cache: npm
          cache-dependency-path: web/package-lock.json

      - name: Install web dependencies
        working-directory: web
        run: npm ci

      - name: Lint DESIGN.md
        run: npm --prefix web run design:lint
```

- [ ] **Step 3: Validate workflow syntax with actionlint if available**

Run:

```bash
if command -v actionlint >/dev/null 2>&1; then
  actionlint .github/workflows/design-system.yml
else
  echo "actionlint not installed; workflow syntax will be validated by GitHub Actions"
fi
```

Expected if `actionlint` is installed: no output and exit code `0`.

Expected if `actionlint` is not installed:

```text
actionlint not installed; workflow syntax will be validated by GitHub Actions
```

- [ ] **Step 4: Run local validation**

Run:

```bash
npm --prefix web run design:lint
```

Expected: exit code `0`, and the JSON output contains:

```json
"errors": 0
```

- [ ] **Step 5: Commit the workflow**

```bash
git add .github/workflows/design-system.yml
git commit -m "ci: validate relay design source"
```

---

### Task 6: Final Verification And Export Smoke

**Files:**
- Read: `DESIGN.md`
- Read: `docs/design-system.md`
- Read: `web/package.json`
- Read: `.github/workflows/design-system.yml`

- [ ] **Step 1: Run the official lint command**

Run:

```bash
npm --prefix web run design:lint
```

Expected: exit code `0`, and the JSON output contains:

```json
"errors": 0
```

- [ ] **Step 2: Run the Tailwind export smoke**

Run:

```bash
npm --prefix web run design:export:tailwind >/tmp/relay-design-tailwind-theme.css
test -s /tmp/relay-design-tailwind-theme.css
rg --fixed-strings -- '"primary"' /tmp/relay-design-tailwind-theme.css
rg --fixed-strings -- '"display"' /tmp/relay-design-tailwind-theme.css
```

Expected: all commands pass, proving the token front matter can be exported to Tailwind configuration JSON.

- [ ] **Step 3: Run existing documentation and web checks**

Run:

```bash
npm --prefix web run lint
npm --prefix web run typecheck
npm --prefix web run test
```

Expected: all commands pass. If an existing unrelated test is red before this branch, rerun the same command on `origin/main` and record the baseline result in the PR body.

- [ ] **Step 4: Review the diff for unintended UI code changes**

Run:

```bash
git diff -- DESIGN.md docs/design-system.md web/package.json web/package-lock.json .github/workflows/design-system.yml
git diff -- web/app web/components
```

Expected:

- First command shows only design-source, docs, package, lockfile, and workflow changes.
- Second command prints no diff.

- [ ] **Step 5: Final commit if Task 6 produced cleanup changes**

If Task 6 required a formatting or documentation correction, commit it:

```bash
git add DESIGN.md docs/design-system.md web/package.json web/package-lock.json .github/workflows/design-system.yml
git commit -m "docs(design): finalize design md validation"
```

If Task 6 produced no changes, do not create an empty commit.

---

## Self-Review

**Spec coverage:** The plan covers YAML front matter, token schema groups, component token references, canonical Markdown section order, official CLI linting, Tailwind export smoke, and PR validation.

**Placeholder scan:** The plan contains no unresolved file paths and no instruction that depends on undefined functions or components.

**Type consistency:** Token references use the same YAML paths throughout: `{colors.*}`, `{typography.*}`, `{rounded.*}`, and `{spacing.*}`. npm scripts consistently run from `web` and reference `../DESIGN.md`.

## Execution Handoff

Plan complete and saved to `docs/superpowers/plans/2026-05-04-design-md-format-alignment.md`. Two execution options:

**1. Subagent-Driven (recommended)** - I dispatch a fresh subagent per task, review between tasks, fast iteration

**2. Inline Execution** - Execute tasks in this session using executing-plans, batch execution with checkpoints

Which approach?
