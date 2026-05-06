# 4gly Design System Figma Design

Date: 2026-05-03
Status: Draft for user review

## Goal

Create the first practical 4gly Design System inside the existing Figma file:

- Figma file: `4gly Design System`
- File key: `aX8b4hj4snppygfl07VARd`
- Parent brand: 4gly Labs
- First product layer: Relay
- Source assets: `/Users/hoon-ch/Downloads/Relay Design System.zip`
- Repo references: `DESIGN.md`, `docs/design-system.md`, `web/app/globals.css`,
  and `web/components/relay/*`

The work should turn the imported design system into a 4gly-owned system without
throwing away the expert scaffold already present in the file.

## Decisions

- Adopt the existing expert scaffold as the baseline.
- Keep the current broad information architecture:
  - `Overview`
  - `Updates`
  - `Component` management/separator pages
  - `1 Theme`
  - `2 Element`
  - `3 Component`
  - `Resource`
  - `Guideline`
  - `Foundation`
  - `Color - Atomic`
  - `Color - Semantic`
  - `Typography`
  - `Grid`
  - `Icon`
  - `Logo`
  - `Basic`
  - `Spacing`
  - `Decorate`
  - `Makers' Principle`
  - `Example`
  - `Work`
- Keep existing component taxonomy, contract/property model, and variant
  coverage wherever possible.
- Do not delete components or variants just because Relay does not implement
  them yet.
- Add a status layer for every major component, variant, style, or section:
  `Implemented`, `Design-ready`, `Reference`, or `Deprecated`.
- Let 4gly semantic rules override imported visual values when they conflict.
- Treat Relay as the first product implementation layer, not as the whole design
  system.

## Strategy

The Figma file is not a blank-file rebuild. It is a design-system acquisition.
The imported system appears to have a mature hub-and-spoke structure with
Foundation, Theme, Element, Component, Guideline, Resource, and Example areas.
That structure should be preserved because it encodes design-system expertise
the 4gly team does not need to recreate from scratch.

The ownership rule is:

> Structure borrowed, meaning owned.

The imported scaffold can define how the file is organized, how components are
classified, and how variants are represented. 4gly defines what the tokens mean,
when they are allowed, how copy sounds, and how Relay uses the system.

## Status Layer

Every major item should have one of these statuses.

### Implemented

The item is both approved in the 4gly system and already represented in Relay
code or product UI. Examples include the current Relay primitive subset:

- `RelayButton`
- `RelayLinkButton`
- `RelayCard`
- `RelayField`
- `RelayTextInput`
- `RelayFeedback`
- `RelayStatusBadge`
- `RelaySourceChip`
- `RelayTabs`
- `RelayMetricTile`
- `RelayListRow`
- `RelayMetaGrid`
- `RelayTopRail`
- `RelayAppShell`
- `RelayPageHead`

### Design-ready

The item is approved for 4gly design use but is not implemented in Relay yet.
Existing imported components that fit 4gly after retheming should usually land
here instead of being deleted.

### Reference

The item remains from the imported scaffold, but 4gly has not decided whether to
adopt it. Reference items are useful for future design work, but should not be
used in new production-facing Relay designs without review.

### Deprecated

The item conflicts with 4gly rules and should not be used for new work. Examples
include visual values or patterns that rely on:

- generic dark-opacity drop shadows as the primary elevation model
- ambient brand-color gradients
- overuse of magic colors outside transformation moments
- rounded-pill treatment on every control
- Title Case as the default UI casing
- emoji in product copy
- bounce/confetti-style motion

## 4gly Semantic Overrides

Imported names and component structures can remain, but these 4gly rules take
priority over imported values.

### Brand Meaning

4gly exists to refine ugly complexity into elegant outcomes:

- `For Ugly`
- `Simplicity out of Complexity`
- `We handle the Ugly, so you can be Elegant`
- `Turning Ugly Ducklings into Elegant Swans`
- `Your complexity is our fuel`

The four-step engine is canonical:

1. `Face`
2. `Dissect`
3. `Refine`
4. `Transform`

### Color

4gly semantic tokens are the primary design contract:

- `canvas`
- `canvas-raised`
- `ink`
- `ink-muted`
- `muted`
- `problem`
- `problem-soft`
- `magic-primary`
- `magic-primary-strong`
- `magic-accent`
- `magic-accent-strong`
- `border`
- `border-strong`
- `success`
- `danger`
- `halo`
- `halo-strong`

`magic-*` colors are reserved for transformation moments. They should not become
ambient page backgrounds or the default treatment for every primary action.

### Elevation

Elevation should prefer pastel halo effects over generic dark shadows. Existing
shadow styles may remain as `Reference` or `Deprecated` depending on whether a
4gly-safe use case exists.

### Shape

4gly radius rules:

- cards: `12px`
- inspector/chrome: `14px`
- buttons and inputs: `8px`
- rectangular chips: `6px`
- pill chips: `999px`

The system should avoid applying pill radius uniformly to every element.

### Typography

Role-based type contract:

- Fraunces: display, headings, editorial italic
- Nunito: body and UI
- JetBrains Mono: mono fallback for Commit Mono
- LXGW WenKai KR: Korean body/display fallback
- LXGW WenKai Mono KR: Korean mono fallback

Korean support is part of v1, not a later add-on. Mixed Korean/English text
should preserve the Latin track for Latin glyphs and use WenKai for Hangul.

### Voice

The system uses sentence case for UI copy and keeps the voice professional but
not cold. Product copy should avoid emoji, marketing puffery, and exposing
internal pipe details on the primary surface.

Errors follow:

```text
CODE: Human sentence.
```

## Existing Structure Mapping

### Overview

Keep as the entry point. Add:

- what 4gly Design System v1 is
- how to use the status layer
- how imported scaffold adoption works
- how Relay maps to the system

### Updates

Keep as migration log. Record:

- imported scaffold baseline date
- 4gly semantic token adoption
- Relay implemented subset mapping
- later component absorption decisions

### Component Management Pages

Keep existing separator or dummy pages as internal management areas. They can
hold index frames, taxonomy notes, or staging content.

### 1 Theme

Keep the Theme area. Use it for brand identity, logo, icon direction, and theme
mode decisions.

### 2 Element

Keep Basic, Spacing, and Decorate. Redefine them around 4gly rules:

- 4px spacing scale
- grid behavior
- radius rules
- halo elevation
- controlled motion
- restrained decoration

### 3 Component

Keep the existing component categories:

- `1 Layout`
- `2 Action`
- `3 Selection and Input`
- `4 Content`
- `5 Loading`
- `6 Navigation`
- `7 Feedback`
- `8 Presentation`

Absorb existing imported components into these categories. Retheme them with
4gly semantics and attach a status.

### Resource

Keep as asset storage:

- fonts from the Relay Design System zip
- wordmark SVGs
- swan contour asset
- exported brand references
- source notes for licensed or substitute fonts

### Guideline

Keep as usage guidance:

- do and don't examples
- accessibility rules
- i18n rules
- motion rules
- token usage rules
- component status definitions

### Foundation and Subpages

Keep:

- `Color - Atomic`
- `Color - Semantic`
- `Typography`
- `Grid`

Retheme the semantic layer first. Atomic palettes can remain broader, but
production-facing usage should point designers toward semantic tokens.

### Icon and Logo

Keep the structure. Retheme around 4gly and Relay:

- 1.5px currentColor line icons
- Lucide as the early substitute icon set
- no emoji as iconography
- duckling/swan marks used sparingly
- Relay wordmark remains `Relay.`

### Makers' Principle

Use this page actively. It should explain how 4gly makers decide:

- face the ugly problem first
- reduce complexity before adding UI
- show the swan, not the pipes
- keep structure borrowed but meaning owned

### Example and Work

Use for Relay examples and product recipes. v1 examples should be component
composition examples, not a full screen redesign:

- Style Memory review card
- Settings API key form
- Project rail row
- Public snapshot card

## Component Absorption Pipeline

Existing imported components that Relay does not currently have can be absorbed.
Use this pipeline.

1. Inventory the component by category, contract, variant axes, and states.
2. Preserve its taxonomy, property model, and variant coverage by default.
3. Retheme visual values to 4gly semantic tokens.
4. Apply 4gly radius, typography, elevation, motion, and voice rules.
5. Assign one status: `Implemented`, `Design-ready`, `Reference`, or
   `Deprecated`.
6. Add a migration note if the component was adapted from the imported scaffold.
7. Map to Relay code only when a corresponding component exists or is planned.

The goal is not to shrink the design system down to current Relay code. The goal
is to keep a useful Figma library while making implementation status explicit.

## Relay Product Layer

Relay is the first 4gly product layer. It maps the 4gly engine like this:

| 4gly step | Relay surface |
| --- | --- |
| `Face` | Capture raw notes, artifacts, and open questions |
| `Dissect` | Judgment traces and decision rationale |
| `Refine` | Heuristic proposals and Style Memory approval |
| `Transform` | Packet and public snapshot handoff |

Relay-specific patterns should not distort the general 4gly component library.
They should live as product mappings, recipes, and implemented-subset notes.

## Execution Plan

### 1. Snapshot inventory

Before changing the file, capture the current file structure:

- pages
- variable collections
- local paint, text, and effect styles
- component taxonomy
- major component sets and variants

### 2. Add governance layer

Update `Overview`, `Updates`, and guideline areas with:

- full scaffold adoption principle
- status definitions
- 4gly semantic override rules
- Relay implemented subset

### 3. Retheme variables

Keep the existing variable collections:

- `Atomic`
- `Theme`
- `Component`
- `Frame`

Start with `Theme` Light/Dark semantic values and align them to 4gly tokens.

### 4. Retheme styles

Align local styles with the semantic layer:

- paint styles for color semantics
- text styles for Fraunces/Nunito/Mono/WenKai roles
- effect styles for halo elevation
- imported shadows marked as `Reference` or `Deprecated`

### 5. Absorb components

For each component category, preserve structure and variant coverage, then apply
4gly semantics and status labels.

### 6. Add Relay product mapping

Document Relay's 4-step mapping and implemented primitive subset.

### 7. Add examples

Create small composition examples:

- Style Memory review card
- Settings API key form
- Project rail row
- Public snapshot card

### 8. Validate

Read the Figma file again and verify:

- existing IA remains intact
- variable collections still exist
- status layer is visible
- semantic tokens are represented in Light/Dark
- component taxonomy is preserved
- Relay implemented subset is documented

## Non-Goals

- Do not rebuild the Figma file from scratch.
- Do not delete existing components just because Relay lacks implementation.
- Do not make Relay the entire design system.
- Do not complete every full-screen Relay design in v1.
- Do not introduce a new product visual language outside 4gly foundation.
- Do not publish or push production code changes as part of this design spec.

## Open Implementation Notes

- Figma font availability should be checked before writing text-heavy boards.
  If a brand font is unavailable in Figma, use a documented fallback rather than
  silently changing the type contract.
- Existing variable names may remain if changing them would break imported
  component bindings. In that case, add 4gly semantic documentation on top.
- The first Figma mutation pass should be small: governance/status frames and
  semantic token boards before component retheming.
