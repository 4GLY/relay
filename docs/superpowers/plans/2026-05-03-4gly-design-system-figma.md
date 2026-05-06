# 4gly Design System Figma Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Convert the existing `4gly Design System` Figma file into a 4gly-owned design system by adopting the imported expert scaffold, adding a status layer, retheming semantic tokens, and mapping Relay as the first implemented product layer.

**Architecture:** The Figma file is treated as the source artifact. Implementation proceeds through small `use_figma` calls: inspect first, add governance frames, retheme variables/styles, classify existing components with status markers, add Relay mapping boards, then validate by reading the file back. Existing structure, taxonomy, contracts, and variant coverage stay intact unless a value conflicts with 4gly semantic rules.

**Tech Stack:** Figma Plugin API through `mcp__codex_apps__figma._use_figma`, Figma variables/styles/components, local Relay docs (`DESIGN.md`, `docs/design-system.md`, `web/app/globals.css`), source zip (`/Users/hoon-ch/Downloads/Relay Design System.zip`).

---

## File And Artifact Map

**Figma target**
- Modify: `https://www.figma.com/design/aX8b4hj4snppygfl07VARd/4gly-Design-System`

**Local references**
- Read: `/Users/hoon-ch/repos/relay/docs/superpowers/specs/2026-05-03-4gly-design-system-figma-design.md`
- Read: `/Users/hoon-ch/repos/relay/DESIGN.md`
- Read: `/Users/hoon-ch/repos/relay/docs/design-system.md`
- Read: `/Users/hoon-ch/repos/relay/web/app/globals.css`
- Read: `/Users/hoon-ch/repos/relay/web/components/relay/*`
- Read: `/Users/hoon-ch/repos/4gly/4gly_foundation.md`
- Read: `/Users/hoon-ch/Downloads/Relay Design System.zip`

**Figma pages to preserve**
- `Overview`
- `Updates`
- `————  Component  ————`
- `1 Theme`
- `2 Element`
- `3 Component`
- `Resource`
- `————    Guideline    ————`
- `⓪ Foundation`
- ` ┗ Color - Atomic`
- ` ┗ Color - Semantic`
- ` ┗ Typography`
- ` ┗ Grid`
- `① Theme`
- ` ┗ Icon`
- ` ┗ Logo`
- `② Element`
- ` ┗ Basic`
- ` ┗ Spacing`
- ` ┗ Decorate`
- `③ Component`
- ` ┗ Guidelines`
- `Ⓜ Makers’ Principle`
- `————     Example     ————`
- `Work`

**Figma pages or frames to add if missing**
- Add frame on `Overview`: `4gly v1 / Start here`
- Add frame on `Updates`: `4gly v1 / Migration log`
- Add frame on `Ⓜ Makers’ Principle`: `4gly maker principles`
- Add frame on ` ┗ Color - Semantic`: `4gly semantic tokens`
- Add frame on `③ Component`: `Component status model`
- Add frame on `Work`: `Relay product layer`

## Implementation Rules

- Always load `figma-use` before any `use_figma` call.
- Always pass `skillNames: "figma-use"` to `use_figma`.
- Do not use `figma.notify()`, `figma.closePlugin()`, or `getPluginData()`.
- Use `await figma.setCurrentPageAsync(page)` when switching pages.
- Return created and mutated node IDs from every mutating call.
- Do not delete imported pages, components, variants, variables, or styles in v1.
- Add visible status labels instead of removing uncertain imported assets.
- Prefer Figma fonts already available in the file. If a brand font is unavailable, use Figma's default loaded font and write the intended brand font in the board text.

## 4gly Tokens For Implementation

Use these semantic colors for Light/Dark documentation and variables.

```js
const GLY_TOKENS = {
  light: {
    canvas: "#faf8f3",
    "canvas-raised": "#ffffff",
    ink: "#0e1a35",
    "ink-muted": "#4a5669",
    muted: "#687386",
    problem: "#1e2230",
    "problem-soft": "#2b3140",
    "magic-primary": "#a7c4ff",
    "magic-primary-strong": "#6f96db",
    "magic-accent": "#c7b8ff",
    "magic-accent-strong": "#8b76e4",
    border: "#e7ecf3",
    "border-strong": "#d4dce8",
    success: "#2f9b73",
    danger: "#c84646",
    halo: "#a7c4ff"
  },
  dark: {
    canvas: "#0b1020",
    "canvas-raised": "#121930",
    ink: "#eef4ff",
    "ink-muted": "#a5b1c8",
    muted: "#9aa6b8",
    problem: "#06091a",
    "problem-soft": "#1a1f30",
    "magic-primary": "#7fb6ff",
    "magic-primary-strong": "#a7c4ff",
    "magic-accent": "#a78bfa",
    "magic-accent-strong": "#c7b8ff",
    border: "#273244",
    "border-strong": "#3a4660",
    success: "#64d6a4",
    danger: "#ff7777",
    halo: "#7fb6ff"
  }
};
```

---

### Task 1: Capture Baseline Inventory

**Files:**
- Modify: Figma file `aX8b4hj4snppygfl07VARd`
- Local read: `docs/superpowers/specs/2026-05-03-4gly-design-system-figma-design.md`

- [ ] **Step 1: Confirm local worktree is clean**

Run:

```bash
git status --short
```

Expected: no output, or only changes intentionally made by the current implementation pass.

- [ ] **Step 2: Run high-level Figma inventory**

Use `mcp__codex_apps__figma._use_figma` with `fileKey: "aX8b4hj4snppygfl07VARd"` and `skillNames: "figma-use"`:

```js
const pages = [];
for (const page of figma.root.children) {
  await figma.setCurrentPageAsync(page);
  pages.push({
    id: page.id,
    name: page.name,
    childCount: page.children.length,
    topLevel: page.children.slice(0, 40).map((node) => ({
      id: node.id,
      name: node.name,
      type: node.type,
      childCount: "children" in node ? node.children.length : 0,
      x: Math.round(node.x || 0),
      y: Math.round(node.y || 0),
      width: Math.round(node.width || 0),
      height: Math.round(node.height || 0)
    }))
  });
}

const collections = await figma.variables.getLocalVariableCollectionsAsync();
const variableCollections = collections.map((collection) => ({
  id: collection.id,
  name: collection.name,
  modes: collection.modes.map((m) => ({ id: m.modeId, name: m.name })),
  variableCount: collection.variableIds.length
}));

const paintStyles = await figma.getLocalPaintStylesAsync();
const textStyles = await figma.getLocalTextStylesAsync();
const effectStyles = await figma.getLocalEffectStylesAsync();

return {
  fileName: figma.root.name,
  pages,
  variableCollections,
  styleCounts: {
    paint: paintStyles.length,
    text: textStyles.length,
    effect: effectStyles.length
  },
  samplePaintStyles: paintStyles.slice(0, 60).map((s) => ({ id: s.id, name: s.name })),
  sampleTextStyles: textStyles.slice(0, 60).map((s) => ({ id: s.id, name: s.name })),
  sampleEffectStyles: effectStyles.slice(0, 30).map((s) => ({ id: s.id, name: s.name }))
};
```

Expected: returns page list including `Overview`, `1 Theme`, `2 Element`, `3 Component`, Foundation subpages, `Ⓜ Makers’ Principle`, and `Work`.

- [ ] **Step 3: Run component taxonomy inventory**

Use `mcp__codex_apps__figma._use_figma`:

```js
const results = [];
for (const page of figma.root.children) {
  await figma.setCurrentPageAsync(page);
  const nodes = page.findAll((node) => node.type === "COMPONENT" || node.type === "COMPONENT_SET");
  results.push({
    page: page.name,
    componentCount: nodes.length,
    samples: nodes.slice(0, 80).map((node) => ({
      id: node.id,
      name: node.name,
      type: node.type,
      width: Math.round(node.width),
      height: Math.round(node.height)
    }))
  });
}

return {
  totalComponents: results.reduce((sum, page) => sum + page.componentCount, 0),
  pages: results.filter((page) => page.componentCount > 0)
};
```

Expected: returns a per-page component count. Do not mutate anything in this step.

- [ ] **Step 4: Record baseline in Figma `Updates`**

Use `mcp__codex_apps__figma._use_figma`:

```js
const target = figma.root.children.find((page) => page.name === "Updates");
if (!target) throw new Error("Updates page not found");
await figma.setCurrentPageAsync(target);

await figma.loadFontAsync({ family: "Inter", style: "Regular" });
await figma.loadFontAsync({ family: "Inter", style: "Semi Bold" });

function textNode(name, text, size, weight) {
  const node = figma.createText();
  node.name = name;
  node.characters = text;
  node.fontName = { family: "Inter", style: weight };
  node.fontSize = size;
  node.lineHeight = { unit: "PIXELS", value: Math.round(size * 1.45) };
  node.fills = [{ type: "SOLID", color: { r: 0.055, g: 0.102, b: 0.208 } }];
  return node;
}

const frame = figma.createFrame();
frame.name = "4gly v1 / Migration log";
frame.x = 760;
frame.y = 0;
frame.resize(720, 420);
frame.layoutMode = "VERTICAL";
frame.itemSpacing = 16;
frame.paddingTop = 32;
frame.paddingRight = 32;
frame.paddingBottom = 32;
frame.paddingLeft = 32;
frame.fills = [{ type: "SOLID", color: { r: 0.980, g: 0.973, b: 0.953 } }];
frame.strokes = [{ type: "SOLID", color: { r: 0.906, g: 0.925, b: 0.953 } }];
frame.strokeWeight = 1;
frame.cornerRadius = 12;

const title = textNode("Title", "4gly v1 migration log", 28, "Semi Bold");
const body = textNode(
  "Body",
  [
    "Baseline captured before 4gly adoption.",
    "Strategy: full scaffold adoption.",
    "Existing IA, component taxonomy, contracts, and variants are preserved.",
    "4gly semantic rules override conflicting imported visual values.",
    "Relay is the first implemented product layer, not the whole system."
  ].join("\\n"),
  15,
  "Regular"
);

frame.appendChild(title);
frame.appendChild(body);
target.appendChild(frame);

return {
  createdNodeIds: [frame.id, title.id, body.id],
  mutatedNodeIds: [target.id]
};
```

Expected: a new `4gly v1 / Migration log` frame appears on `Updates`.

- [ ] **Step 5: Validate Task 1**

Use `mcp__codex_apps__figma._use_figma`:

```js
const page = figma.root.children.find((p) => p.name === "Updates");
if (!page) throw new Error("Updates page missing");
await figma.setCurrentPageAsync(page);
const found = page.findAll((node) => node.name === "4gly v1 / Migration log");
return { count: found.length, nodeIds: found.map((node) => node.id) };
```

Expected: `{ count: 1, nodeIds: [...] }`.

---

### Task 2: Add Governance And Status Layer

**Files:**
- Modify: Figma pages `Overview`, `————    Guideline    ————`, `Ⓜ Makers’ Principle`

Note: Task 2 only mutates `Overview` and `Ⓜ Makers’ Principle`. The Guideline
page is part of the governance area, but its concrete absorption guide is added
in Task 5.

- [ ] **Step 1: Add Start Here frame to `Overview`**

Use `mcp__codex_apps__figma._use_figma`:

```js
const page = figma.root.children.find((p) => p.name === "Overview");
if (!page) throw new Error("Overview page not found");
await figma.setCurrentPageAsync(page);

await figma.loadFontAsync({ family: "Inter", style: "Regular" });
await figma.loadFontAsync({ family: "Inter", style: "Semi Bold" });

const colors = {
  canvas: { r: 0.980, g: 0.973, b: 0.953 },
  raised: { r: 1, g: 1, b: 1 },
  ink: { r: 0.055, g: 0.102, b: 0.208 },
  muted: { r: 0.290, g: 0.337, b: 0.412 },
  border: { r: 0.906, g: 0.925, b: 0.953 },
  magic: { r: 0.655, g: 0.769, b: 1.0 }
};

function makeText(name, text, size, style = "Regular") {
  const node = figma.createText();
  node.name = name;
  node.characters = text;
  node.fontName = { family: "Inter", style };
  node.fontSize = size;
  node.lineHeight = { unit: "PIXELS", value: Math.round(size * 1.45) };
  node.fills = [{ type: "SOLID", color: colors.ink }];
  return node;
}

const frame = figma.createFrame();
frame.name = "4gly v1 / Start here";
frame.x = 0;
frame.y = 6400;
frame.resize(960, 760);
frame.layoutMode = "VERTICAL";
frame.itemSpacing = 20;
frame.paddingTop = 40;
frame.paddingRight = 40;
frame.paddingBottom = 40;
frame.paddingLeft = 40;
frame.fills = [{ type: "SOLID", color: colors.canvas }];
frame.strokes = [{ type: "SOLID", color: colors.border }];
frame.strokeWeight = 1;
frame.cornerRadius = 12;

const title = makeText("Title", "4gly Design System v1", 36, "Semi Bold");
const intro = makeText(
  "Intro",
  "This file adopts the imported expert scaffold as its baseline. Structure, component taxonomy, contracts, and variant coverage are preserved. 4gly owns the semantic layer: For Ugly, the 4-step Magic Engine, voice, token meaning, motion, and Relay product mapping.",
  16
);
intro.resize(860, 120);

const status = makeText(
  "Status model",
  "Status model\\nImplemented: approved and already represented in Relay.\\nDesign-ready: approved for 4gly, not implemented yet.\\nReference: preserved from imported scaffold, not adopted yet.\\nDeprecated: conflicts with 4gly rules and should not be used for new work.",
  16
);
status.resize(860, 150);

const rules = makeText(
  "Override rules",
  "Override rules\\nMagic colors are for transformation moments only. Elevation prefers pastel halo, not generic dark shadows. Cards use 12px radius, buttons and inputs 8px, rectangular chips 6px. UI copy uses sentence case and no emoji.",
  16
);
rules.resize(860, 130);

frame.appendChild(title);
frame.appendChild(intro);
frame.appendChild(status);
frame.appendChild(rules);
page.appendChild(frame);

return {
  createdNodeIds: [frame.id, title.id, intro.id, status.id, rules.id],
  mutatedNodeIds: [page.id]
};
```

Expected: `Overview` contains a visible `4gly v1 / Start here` frame.

- [ ] **Step 2: Add status chips to the Start Here frame**

Use `mcp__codex_apps__figma._use_figma`:

```js
const page = figma.root.children.find((p) => p.name === "Overview");
if (!page) throw new Error("Overview page not found");
await figma.setCurrentPageAsync(page);
await figma.loadFontAsync({ family: "Inter", style: "Semi Bold" });

const frame = page.findOne((node) => node.name === "4gly v1 / Start here");
if (!frame || frame.type !== "FRAME") throw new Error("Start here frame not found");

const statuses = [
  ["Implemented", "#2f9b73"],
  ["Design-ready", "#6f96db"],
  ["Reference", "#687386"],
  ["Deprecated", "#c84646"]
];

function hexToRgb(hex) {
  const clean = hex.replace("#", "");
  return {
    r: parseInt(clean.slice(0, 2), 16) / 255,
    g: parseInt(clean.slice(2, 4), 16) / 255,
    b: parseInt(clean.slice(4, 6), 16) / 255
  };
}

const row = figma.createFrame();
row.name = "Status chips";
row.layoutMode = "HORIZONTAL";
row.itemSpacing = 12;
row.fills = [];
row.resize(860, 40);

const created = [row.id];
for (const [label, color] of statuses) {
  const chip = figma.createFrame();
  chip.name = `Status / ${label}`;
  chip.layoutMode = "HORIZONTAL";
  chip.primaryAxisSizingMode = "AUTO";
  chip.counterAxisSizingMode = "AUTO";
  chip.paddingTop = 8;
  chip.paddingRight = 12;
  chip.paddingBottom = 8;
  chip.paddingLeft = 12;
  chip.cornerRadius = 999;
  chip.fills = [{ type: "SOLID", color: hexToRgb(color), opacity: 0.14 }];
  chip.strokes = [{ type: "SOLID", color: hexToRgb(color) }];
  chip.strokeWeight = 1;

  const text = figma.createText();
  text.name = `Label / ${label}`;
  text.characters = label;
  text.fontName = { family: "Inter", style: "Semi Bold" };
  text.fontSize = 12;
  text.fills = [{ type: "SOLID", color: hexToRgb(color) }];
  chip.appendChild(text);
  row.appendChild(chip);
  created.push(chip.id, text.id);
}

frame.appendChild(row);
return { createdNodeIds: created, mutatedNodeIds: [frame.id] };
```

Expected: four status chips are visible inside the Start Here frame.

- [ ] **Step 3: Add maker principles frame**

Use `mcp__codex_apps__figma._use_figma`:

```js
const page = figma.root.children.find((p) => p.name === "Ⓜ Makers’ Principle");
if (!page) throw new Error("Makers’ Principle page not found");
await figma.setCurrentPageAsync(page);
await figma.loadFontAsync({ family: "Inter", style: "Regular" });
await figma.loadFontAsync({ family: "Inter", style: "Semi Bold" });

function t(name, text, size, style = "Regular") {
  const node = figma.createText();
  node.name = name;
  node.characters = text;
  node.fontName = { family: "Inter", style };
  node.fontSize = size;
  node.lineHeight = { unit: "PIXELS", value: Math.round(size * 1.45) };
  node.fills = [{ type: "SOLID", color: { r: 0.055, g: 0.102, b: 0.208 } }];
  return node;
}

const frame = figma.createFrame();
frame.name = "4gly maker principles";
frame.x = 3200;
frame.y = 0;
frame.resize(860, 620);
frame.layoutMode = "VERTICAL";
frame.itemSpacing = 18;
frame.paddingTop = 36;
frame.paddingRight = 36;
frame.paddingBottom = 36;
frame.paddingLeft = 36;
frame.fills = [{ type: "SOLID", color: { r: 1, g: 1, b: 1 } }];
frame.strokes = [{ type: "SOLID", color: { r: 0.906, g: 0.925, b: 0.953 } }];
frame.strokeWeight = 1;
frame.cornerRadius = 12;

const title = t("Title", "4gly maker principles", 32, "Semi Bold");
const body = t(
  "Body",
  [
    "Face the ugly problem first.",
    "Reduce complexity before adding UI.",
    "Show the swan, not the pipes.",
    "Use professional warmth, not empty decoration.",
    "Borrow the expert scaffold, own the 4gly meaning.",
    "Relay is the first product layer; it does not limit the full system."
  ].join("\\n"),
  17
);
body.resize(760, 260);

frame.appendChild(title);
frame.appendChild(body);
page.appendChild(frame);

return { createdNodeIds: [frame.id, title.id, body.id], mutatedNodeIds: [page.id] };
```

Expected: `Ⓜ Makers’ Principle` contains `4gly maker principles`.

- [ ] **Step 4: Validate Task 2**

Use `mcp__codex_apps__figma._use_figma`:

```js
const checks = [];
for (const [pageName, frameName] of [
  ["Overview", "4gly v1 / Start here"],
  ["Ⓜ Makers’ Principle", "4gly maker principles"]
]) {
  const page = figma.root.children.find((p) => p.name === pageName);
  if (!page) {
    checks.push({ pageName, frameName, found: false, reason: "page missing" });
    continue;
  }
  await figma.setCurrentPageAsync(page);
  const matches = page.findAll((node) => node.name === frameName);
  checks.push({ pageName, frameName, found: matches.length === 1, count: matches.length });
}
return checks;
```

Expected: both checks return `found: true`.

---

### Task 3: Retheme Semantic Variables And Token Board

**Files:**
- Modify: Figma variable collection `Theme`
- Modify: Figma page ` ┗ Color - Semantic`

- [ ] **Step 1: Add or update 4gly semantic variables in `Theme`**

Use `mcp__codex_apps__figma._use_figma`:

```js
const tokenNames = [
  "canvas",
  "canvas-raised",
  "ink",
  "ink-muted",
  "muted",
  "problem",
  "problem-soft",
  "magic-primary",
  "magic-primary-strong",
  "magic-accent",
  "magic-accent-strong",
  "border",
  "border-strong",
  "success",
  "danger",
  "halo",
  "halo-strong"
];

const values = {
  Light: {
    canvas: "#faf8f3",
    "canvas-raised": "#ffffff",
    ink: "#0e1a35",
    "ink-muted": "#4a5669",
    muted: "#687386",
    problem: "#1e2230",
    "problem-soft": "#2b3140",
    "magic-primary": "#a7c4ff",
    "magic-primary-strong": "#6f96db",
    "magic-accent": "#c7b8ff",
    "magic-accent-strong": "#8b76e4",
    border: "#e7ecf3",
    "border-strong": "#d4dce8",
    success: "#2f9b73",
    danger: "#c84646",
    halo: "#a7c4ff",
    "halo-strong": "#a7c4ff"
  },
  Dark: {
    canvas: "#0b1020",
    "canvas-raised": "#121930",
    ink: "#eef4ff",
    "ink-muted": "#a5b1c8",
    muted: "#9aa6b8",
    problem: "#06091a",
    "problem-soft": "#1a1f30",
    "magic-primary": "#7fb6ff",
    "magic-primary-strong": "#a7c4ff",
    "magic-accent": "#a78bfa",
    "magic-accent-strong": "#c7b8ff",
    border: "#273244",
    "border-strong": "#3a4660",
    success: "#64d6a4",
    danger: "#ff7777",
    halo: "#7fb6ff",
    "halo-strong": "#7fb6ff"
  }
};

function hexToRgb(hex) {
  const clean = hex.replace("#", "");
  return {
    r: parseInt(clean.slice(0, 2), 16) / 255,
    g: parseInt(clean.slice(2, 4), 16) / 255,
    b: parseInt(clean.slice(4, 6), 16) / 255,
    a: 1
  };
}

let collection = (await figma.variables.getLocalVariableCollectionsAsync()).find((c) => c.name === "Theme");
if (!collection) {
  collection = figma.variables.createVariableCollection("Theme");
}

const modeByName = Object.fromEntries(collection.modes.map((mode) => [mode.name, mode.modeId]));
let lightModeId = modeByName.Light;
let darkModeId = modeByName.Dark;
if (!lightModeId) {
  lightModeId = collection.modes[0].modeId;
  collection.renameMode(lightModeId, "Light");
}
if (!darkModeId) {
  darkModeId = collection.addMode("Dark");
}

const existing = {};
for (const id of collection.variableIds) {
  const variable = await figma.variables.getVariableByIdAsync(id);
  if (variable) existing[variable.name] = variable;
}

const created = [];
const mutated = [];
for (const token of tokenNames) {
  const name = `4gly/semantic/${token}`;
  let variable = existing[name];
  if (!variable) {
    variable = figma.variables.createVariable(name, collection, "COLOR");
    /* Figma rejects mixing fill-specific scopes in this file with:
       "If ALL_FILLS is set, other fill scopes cannot be set".
       Use explicit fill picker coverage here; Task 4 paint/effect styles
       cover designer-facing style discoverability for other uses. */
    variable.scopes = ["ALL_FILLS"];
    created.push(variable.id);
  } else {
    mutated.push(variable.id);
  }
  variable.setValueForMode(lightModeId, hexToRgb(values.Light[token]));
  variable.setValueForMode(darkModeId, hexToRgb(values.Dark[token]));
}

return {
  collectionId: collection.id,
  lightModeId,
  darkModeId,
  createdVariableIds: created,
  mutatedVariableIds: mutated
};
```

Expected: `Theme` collection contains `4gly/semantic/*` variables for Light and Dark.

- [ ] **Step 2: Add semantic token board**

Use `mcp__codex_apps__figma._use_figma`:

```js
const page = figma.root.children.find((p) => p.name === " ┗ Color - Semantic");
if (!page) throw new Error("Color - Semantic page not found");
await figma.setCurrentPageAsync(page);

const existingBoard = page.findOne((node) => node.name === "4gly semantic tokens");
if (existingBoard) {
  return { createdNodeIds: [], mutatedNodeIds: [], existingNodeId: existingBoard.id };
}

await figma.loadFontAsync({ family: "Inter", style: "Regular" });
await figma.loadFontAsync({ family: "Inter", style: "Semi Bold" });

const tokens = [
  ["canvas", "#faf8f3", "#0b1020", "Base canvas"],
  ["canvas-raised", "#ffffff", "#121930", "Raised panels and cards"],
  ["ink", "#0e1a35", "#eef4ff", "Primary text"],
  ["ink-muted", "#4a5669", "#a5b1c8", "Secondary text"],
  ["problem", "#1e2230", "#06091a", "Ugly or unresolved side"],
  ["magic-primary", "#a7c4ff", "#7fb6ff", "Transformation moments"],
  ["magic-accent", "#c7b8ff", "#a78bfa", "Swan moment accent"],
  ["border", "#e7ecf3", "#273244", "Default line"],
  ["success", "#2f9b73", "#64d6a4", "Positive state"],
  ["danger", "#c84646", "#ff7777", "Destructive/error state"]
];

function hexToRgb(hex) {
  const clean = hex.replace("#", "");
  return {
    r: parseInt(clean.slice(0, 2), 16) / 255,
    g: parseInt(clean.slice(2, 4), 16) / 255,
    b: parseInt(clean.slice(4, 6), 16) / 255
  };
}

function makeText(name, text, size, style = "Regular") {
  const node = figma.createText();
  node.name = name;
  node.characters = text;
  node.fontName = { family: "Inter", style };
  node.fontSize = size;
  node.lineHeight = { unit: "PIXELS", value: Math.round(size * 1.4) };
  node.fills = [{ type: "SOLID", color: hexToRgb("#0e1a35") }];
  return node;
}

const frame = figma.createFrame();
frame.name = "4gly semantic tokens";
frame.x = 10200;
frame.y = 0;
frame.resize(1040, 980);
frame.layoutMode = "VERTICAL";
frame.itemSpacing = 18;
frame.paddingTop = 40;
frame.paddingRight = 40;
frame.paddingBottom = 40;
frame.paddingLeft = 40;
frame.fills = [{ type: "SOLID", color: hexToRgb("#faf8f3") }];
frame.strokes = [{ type: "SOLID", color: hexToRgb("#e7ecf3") }];
frame.strokeWeight = 1;
frame.cornerRadius = 12;

const title = makeText("Title", "4gly semantic tokens", 34, "Semi Bold");
const note = makeText("Note", "Semantic tokens are the primary contract. Atomic colors can exist, but product work should use semantic meaning first.", 15);
note.resize(920, 48);
frame.appendChild(title);
frame.appendChild(note);

for (const [name, light, dark, usage] of tokens) {
  const row = figma.createFrame();
  row.name = `Token row / ${name}`;
  row.layoutMode = "HORIZONTAL";
  row.itemSpacing = 16;
  row.counterAxisAlignItems = "CENTER";
  row.resize(920, 56);
  row.fills = [];

  const lightSwatch = figma.createRectangle();
  lightSwatch.name = `Light / ${name}`;
  lightSwatch.resize(48, 48);
  lightSwatch.cornerRadius = 8;
  lightSwatch.fills = [{ type: "SOLID", color: hexToRgb(light) }];
  lightSwatch.strokes = [{ type: "SOLID", color: hexToRgb("#d4dce8") }];

  const darkSwatch = figma.createRectangle();
  darkSwatch.name = `Dark / ${name}`;
  darkSwatch.resize(48, 48);
  darkSwatch.cornerRadius = 8;
  darkSwatch.fills = [{ type: "SOLID", color: hexToRgb(dark) }];
  darkSwatch.strokes = [{ type: "SOLID", color: hexToRgb("#d4dce8") }];

  const label = makeText(`Label / ${name}`, `${name}\\n${usage}`, 14, "Regular");
  label.resize(760, 48);

  row.appendChild(lightSwatch);
  row.appendChild(darkSwatch);
  row.appendChild(label);
  frame.appendChild(row);
}

page.appendChild(frame);
return { createdNodeIds: [frame.id], mutatedNodeIds: [page.id] };
```

Expected: ` ┗ Color - Semantic` contains `4gly semantic tokens`.

- [ ] **Step 3: Validate Task 3**

Use `mcp__codex_apps__figma._use_figma`:

```js
const collections = await figma.variables.getLocalVariableCollectionsAsync();
const theme = collections.find((c) => c.name === "Theme");
if (!theme) throw new Error("Theme collection missing");

const names = [];
for (const id of theme.variableIds) {
  const variable = await figma.variables.getVariableByIdAsync(id);
  if (variable && variable.name.startsWith("4gly/semantic/")) names.push(variable.name);
}

const page = figma.root.children.find((p) => p.name === " ┗ Color - Semantic");
if (!page) throw new Error("Color - Semantic page missing");
await figma.setCurrentPageAsync(page);
const boards = page.findAll((node) => node.name === "4gly semantic tokens");

return {
  semanticVariableCount: names.length,
  semanticVariables: names.sort(),
  boardCount: boards.length
};
```

Expected: `semanticVariableCount` is at least `17`; `boardCount` is `1`.

---

### Task 4: Retheme Styles And Mark Shadow Policy

**Files:**
- Modify: Figma local paint styles
- Modify: Figma local effect styles
- Modify: Figma page ` ┗ Decorate`

- [ ] **Step 1: Create 4gly paint styles if missing**

Use `mcp__codex_apps__figma._use_figma`:

```js
const styles = await figma.getLocalPaintStylesAsync();
const byName = Object.fromEntries(styles.map((style) => [style.name, style]));
const tokens = {
  "4gly/Semantic/canvas": "#faf8f3",
  "4gly/Semantic/canvas-raised": "#ffffff",
  "4gly/Semantic/ink": "#0e1a35",
  "4gly/Semantic/ink-muted": "#4a5669",
  "4gly/Semantic/problem": "#1e2230",
  "4gly/Semantic/magic-primary": "#a7c4ff",
  "4gly/Semantic/magic-primary-strong": "#6f96db",
  "4gly/Semantic/magic-accent": "#c7b8ff",
  "4gly/Semantic/border": "#e7ecf3",
  "4gly/Semantic/success": "#2f9b73",
  "4gly/Semantic/danger": "#c84646"
};

function hexToRgb(hex) {
  const clean = hex.replace("#", "");
  return {
    r: parseInt(clean.slice(0, 2), 16) / 255,
    g: parseInt(clean.slice(2, 4), 16) / 255,
    b: parseInt(clean.slice(4, 6), 16) / 255
  };
}

const created = [];
const mutated = [];
for (const [name, hex] of Object.entries(tokens)) {
  let style = byName[name];
  if (!style) {
    style = figma.createPaintStyle();
    style.name = name;
    created.push(style.id);
  } else {
    mutated.push(style.id);
  }
  style.paints = [{ type: "SOLID", color: hexToRgb(hex) }];
  style.description = "4gly v1 semantic paint style. Product work should prefer semantic styles over imported atomic colors.";
}

return { createdStyleIds: created, mutatedStyleIds: mutated };
```

Expected: 4gly paint styles exist under `4gly/Semantic/*`.

- [ ] **Step 2: Create halo effect styles and mark legacy shadows**

Use `mcp__codex_apps__figma._use_figma`:

```js
const effects = await figma.getLocalEffectStylesAsync();
let style = effects.find((effect) => effect.name === "4gly/Halo/Default");
const created = [];
const mutated = [];
if (!style) {
  style = figma.createEffectStyle();
  style.name = "4gly/Halo/Default";
  created.push(style.id);
} else {
  mutated.push(style.id);
}
style.description = "4gly elevation model: blurred pastel halo for elevated cards. Mirrors CSS box-shadow: 0 18px 48px var(--halo).";
style.effects = [{
  type: "DROP_SHADOW",
  color: { r: 0.655, g: 0.769, b: 1.0, a: 0.35 },
  offset: { x: 0, y: 18 },
  radius: 48,
  spread: 0,
  visible: true,
  blendMode: "NORMAL"
}];

let ring = effects.find((effect) => effect.name === "4gly/Halo/Ring");
if (!ring) {
  ring = figma.createEffectStyle();
  ring.name = "4gly/Halo/Ring";
  created.push(ring.id);
} else {
  mutated.push(ring.id);
}
ring.description = "4gly ring halo for focus, selected states, and toast rim-light; not the default elevation style.";
ring.effects = [{
  type: "DROP_SHADOW",
  color: { r: 0.655, g: 0.769, b: 1.0, a: 0.35 },
  offset: { x: 0, y: 0 },
  radius: 0,
  spread: 4,
  visible: true,
  blendMode: "NORMAL"
}];

const legacyPrefix = "Reference: imported scaffold shadow. Do not use for new 4gly work unless reviewed. Prefer 4gly/Halo/Default or 4gly/Halo/Ring.";
for (const effect of effects) {
  if (!effect.name.startsWith("Shadow/")) continue;
  if (!effect.description.startsWith(legacyPrefix)) {
    effect.description = effect.description
      ? `${legacyPrefix}\n\n${effect.description}`
      : legacyPrefix;
    mutated.push(effect.id);
  }
}

return {
  createdStyleIds: created,
  mutatedStyleIds: mutated,
  legacyShadowCount: effects.filter((effect) => effect.name.startsWith("Shadow/")).length
};
```

Expected: effect styles `4gly/Halo/Default` and `4gly/Halo/Ring` exist. Imported
`Shadow/*` styles are marked as reference-only in their descriptions.

- [ ] **Step 3: Add decoration policy board**

Use `mcp__codex_apps__figma._use_figma`:

```js
const page = figma.root.children.find((p) => p.name === " ┗ Decorate");
if (!page) throw new Error("Decorate page not found");
await figma.setCurrentPageAsync(page);
await figma.loadFontAsync({ family: "Inter", style: "Regular" });
await figma.loadFontAsync({ family: "Inter", style: "Semi Bold" });

function t(name, text, size, style = "Regular") {
  const node = figma.createText();
  node.name = name;
  node.characters = text;
  node.fontName = { family: "Inter", style };
  node.fontSize = size;
  node.lineHeight = { unit: "PIXELS", value: Math.round(size * 1.45) };
  node.fills = [{ type: "SOLID", color: { r: 0.055, g: 0.102, b: 0.208 } }];
  return node;
}

const frame = figma.createFrame();
frame.name = "4gly decoration and elevation policy";
frame.x = 7200;
frame.y = 0;
frame.resize(860, 520);
frame.layoutMode = "VERTICAL";
frame.itemSpacing = 18;
frame.paddingTop = 36;
frame.paddingRight = 36;
frame.paddingBottom = 36;
frame.paddingLeft = 36;
frame.fills = [{ type: "SOLID", color: { r: 0.980, g: 0.973, b: 0.953 } }];
frame.strokes = [{ type: "SOLID", color: { r: 0.906, g: 0.925, b: 0.953 } }];
frame.strokeWeight = 1;
frame.cornerRadius = 12;

const title = t("Title", "Decoration and elevation", 30, "Semi Bold");
const body = t(
  "Body",
  [
    "Elevation uses pastel halo, not generic dark shadows.",
    "Magic color appears at transformation moments only.",
    "No ambient gradients in app surfaces.",
    "No bounce, confetti, or decorative motion on every element.",
    "Use duckling/swan marks sparingly: empty hint, transform signature, brand detail."
  ].join("\\n"),
  16
);
body.resize(760, 220);

frame.appendChild(title);
frame.appendChild(body);
page.appendChild(frame);

return { createdNodeIds: [frame.id, title.id, body.id], mutatedNodeIds: [page.id] };
```

Expected: ` ┗ Decorate` contains policy board.

- [ ] **Step 4: Validate Task 4**

Use `mcp__codex_apps__figma._use_figma`:

```js
const paint = await figma.getLocalPaintStylesAsync();
const effects = await figma.getLocalEffectStylesAsync();
const page = figma.root.children.find((p) => p.name === " ┗ Decorate");
if (!page) throw new Error("Decorate page missing");
await figma.setCurrentPageAsync(page);
return {
  semanticPaintCount: paint.filter((style) => style.name.startsWith("4gly/Semantic/")).length,
  hasHalo: effects.some((style) => style.name === "4gly/Halo/Default"),
  decorationPolicyCount: page.findAll((node) => node.name === "4gly decoration and elevation policy").length
};
```

Expected: `semanticPaintCount >= 10`, `hasHalo: true`, `decorationPolicyCount: 1`.

---

### Task 5: Add Component Status Model And Relay Implemented Subset

**Files:**
- Modify: Figma page `③ Component`
- Modify: Figma page ` ┗ Guidelines`

- [ ] **Step 1: Add component status model board**

Use `mcp__codex_apps__figma._use_figma`:

```js
const page = figma.root.children.find((p) => p.name === "③ Component");
if (!page) throw new Error("③ Component page not found");
await figma.setCurrentPageAsync(page);
await figma.loadFontAsync({ family: "Inter", style: "Regular" });
await figma.loadFontAsync({ family: "Inter", style: "Semi Bold" });

function t(name, text, size, style = "Regular") {
  const node = figma.createText();
  node.name = name;
  node.characters = text;
  node.fontName = { family: "Inter", style };
  node.fontSize = size;
  node.lineHeight = { unit: "PIXELS", value: Math.round(size * 1.45) };
  node.fills = [{ type: "SOLID", color: { r: 0.055, g: 0.102, b: 0.208 } }];
  return node;
}

const frame = figma.createFrame();
frame.name = "Component status model";
frame.x = 920;
frame.y = 0;
frame.resize(920, 680);
frame.layoutMode = "VERTICAL";
frame.itemSpacing = 18;
frame.paddingTop = 36;
frame.paddingRight = 36;
frame.paddingBottom = 36;
frame.paddingLeft = 36;
frame.fills = [{ type: "SOLID", color: { r: 1, g: 1, b: 1 } }];
frame.strokes = [{ type: "SOLID", color: { r: 0.906, g: 0.925, b: 0.953 } }];
frame.strokeWeight = 1;
frame.cornerRadius = 12;

const title = t("Title", "Component status model", 30, "Semi Bold");
const body = t(
  "Body",
  [
    "Preserve imported component taxonomy, contracts, and variants.",
    "Implemented: exists in Relay code or product UI.",
    "Design-ready: approved for 4gly but not built in Relay yet.",
    "Reference: useful imported scaffold, adoption undecided.",
    "Deprecated: conflicts with 4gly rules.",
    "Do not delete variants just because Relay does not implement them yet."
  ].join("\\n"),
  16
);
body.resize(800, 260);

const subset = t(
  "Relay implemented subset",
  "Relay implemented subset\\nRelayButton, RelayLinkButton, RelayCard, RelayField, RelayTextInput, RelayFeedback, RelayStatusBadge, RelaySourceChip, RelayTabs, RelayMetricTile, RelayListRow, RelayMetaGrid, RelayTopRail, RelayAppShell, RelayPageHead",
  15
);
subset.resize(800, 120);

frame.appendChild(title);
frame.appendChild(body);
frame.appendChild(subset);
page.appendChild(frame);

return { createdNodeIds: [frame.id, title.id, body.id, subset.id], mutatedNodeIds: [page.id] };
```

Expected: `③ Component` contains `Component status model`.

- [ ] **Step 2: Add component category mapping board**

Use `mcp__codex_apps__figma._use_figma`:

```js
const page = figma.root.children.find((p) => p.name === " ┗ Guidelines");
if (!page) throw new Error("Guidelines page not found");
await figma.setCurrentPageAsync(page);
await figma.loadFontAsync({ family: "Inter", style: "Regular" });
await figma.loadFontAsync({ family: "Inter", style: "Semi Bold" });

function t(name, text, size, style = "Regular") {
  const node = figma.createText();
  node.name = name;
  node.characters = text;
  node.fontName = { family: "Inter", style };
  node.fontSize = size;
  node.lineHeight = { unit: "PIXELS", value: Math.round(size * 1.45) };
  node.fills = [{ type: "SOLID", color: { r: 0.055, g: 0.102, b: 0.208 } }];
  return node;
}

const frame = figma.createFrame();
frame.name = "4gly component absorption guide";
frame.x = 1880;
frame.y = 0;
frame.resize(980, 740);
frame.layoutMode = "VERTICAL";
frame.itemSpacing = 16;
frame.paddingTop = 36;
frame.paddingRight = 36;
frame.paddingBottom = 36;
frame.paddingLeft = 36;
frame.fills = [{ type: "SOLID", color: { r: 0.980, g: 0.973, b: 0.953 } }];
frame.strokes = [{ type: "SOLID", color: { r: 0.906, g: 0.925, b: 0.953 } }];
frame.strokeWeight = 1;
frame.cornerRadius = 12;

const title = t("Title", "Component absorption guide", 30, "Semi Bold");
const body = t(
  "Body",
  [
    "1 Layout: shell, rail, page head, grid, rows.",
    "2 Action: buttons and action links.",
    "3 Selection and Input: fields, text input, tabs, language switch.",
    "4 Content: cards, metrics, badges, chips.",
    "5 Loading: quiet skeletons and loading rows.",
    "6 Navigation: top rail, project rail, active transform states.",
    "7 Feedback: empty, warning, error, success states.",
    "8 Presentation: public snapshot and packet preview surfaces.",
    "",
    "For each imported component: inventory, preserve contract, retheme, assign status, add migration note."
  ].join("\\n"),
  15
);
body.resize(880, 420);

frame.appendChild(title);
frame.appendChild(body);
page.appendChild(frame);

return { createdNodeIds: [frame.id, title.id, body.id], mutatedNodeIds: [page.id] };
```

Expected: ` ┗ Guidelines` contains `4gly component absorption guide`.

- [ ] **Step 3: Validate Task 5**

Use `mcp__codex_apps__figma._use_figma`:

```js
const checks = [];
for (const [pageName, frameName] of [
  ["③ Component", "Component status model"],
  [" ┗ Guidelines", "4gly component absorption guide"]
]) {
  const page = figma.root.children.find((p) => p.name === pageName);
  if (!page) {
    checks.push({ pageName, frameName, found: false, reason: "page missing" });
    continue;
  }
  await figma.setCurrentPageAsync(page);
  const matches = page.findAll((node) => node.name === frameName);
  checks.push({ pageName, frameName, count: matches.length, found: matches.length === 1 });
}
return checks;
```

Expected: both checks return `found: true`.

---

### Task 6: Add Relay Product Layer And Examples

**Files:**
- Modify: Figma page `Work`
- Modify: Figma page `————     Example     ————` if it has usable space

- [ ] **Step 1: Add Relay product mapping board**

Use `mcp__codex_apps__figma._use_figma`:

```js
const page = figma.root.children.find((p) => p.name === "Work");
if (!page) throw new Error("Work page not found");
await figma.setCurrentPageAsync(page);
await figma.loadFontAsync({ family: "Inter", style: "Regular" });
await figma.loadFontAsync({ family: "Inter", style: "Semi Bold" });

function t(name, text, size, style = "Regular") {
  const node = figma.createText();
  node.name = name;
  node.characters = text;
  node.fontName = { family: "Inter", style };
  node.fontSize = size;
  node.lineHeight = { unit: "PIXELS", value: Math.round(size * 1.45) };
  node.fills = [{ type: "SOLID", color: { r: 0.055, g: 0.102, b: 0.208 } }];
  return node;
}

const frame = figma.createFrame();
frame.name = "Relay product layer";
frame.x = 5000;
frame.y = 0;
frame.resize(1040, 760);
frame.layoutMode = "VERTICAL";
frame.itemSpacing = 18;
frame.paddingTop = 40;
frame.paddingRight = 40;
frame.paddingBottom = 40;
frame.paddingLeft = 40;
frame.fills = [{ type: "SOLID", color: { r: 0.980, g: 0.973, b: 0.953 } }];
frame.strokes = [{ type: "SOLID", color: { r: 0.906, g: 0.925, b: 0.953 } }];
frame.strokeWeight = 1;
frame.cornerRadius = 12;

const title = t("Title", "Relay as the first 4gly product layer", 32, "Semi Bold");
const mapping = t(
  "4-step mapping",
  [
    "Face → Capture raw notes, artifacts, and open questions.",
    "Dissect → Judgment traces and decision rationale.",
    "Refine → Heuristic proposals and Style Memory approval.",
    "Transform → Packet and public snapshot handoff."
  ].join("\\n"),
  17
);
mapping.resize(920, 160);

const note = t(
  "Note",
  "Relay-specific patterns live here as product recipes. They do not replace the general 4gly component library. The implemented subset is marked honestly so Figma can lead design without pretending every component already exists in code.",
  15
);
note.resize(920, 100);

frame.appendChild(title);
frame.appendChild(mapping);
frame.appendChild(note);
page.appendChild(frame);

return { createdNodeIds: [frame.id, title.id, mapping.id, note.id], mutatedNodeIds: [page.id] };
```

Expected: `Work` contains `Relay product layer`.

- [ ] **Step 2: Add compact Relay examples board**

Use `mcp__codex_apps__figma._use_figma`:

```js
const page = figma.root.children.find((p) => p.name === "Work");
if (!page) throw new Error("Work page not found");
await figma.setCurrentPageAsync(page);
await figma.loadFontAsync({ family: "Inter", style: "Regular" });
await figma.loadFontAsync({ family: "Inter", style: "Semi Bold" });

function hexToRgb(hex) {
  const clean = hex.replace("#", "");
  return {
    r: parseInt(clean.slice(0, 2), 16) / 255,
    g: parseInt(clean.slice(2, 4), 16) / 255,
    b: parseInt(clean.slice(4, 6), 16) / 255
  };
}

function t(name, text, size, style = "Regular") {
  const node = figma.createText();
  node.name = name;
  node.characters = text;
  node.fontName = { family: "Inter", style };
  node.fontSize = size;
  node.lineHeight = { unit: "PIXELS", value: Math.round(size * 1.4) };
  node.fills = [{ type: "SOLID", color: hexToRgb("#0e1a35") }];
  return node;
}

function card(name, title, body) {
  const frame = figma.createFrame();
  frame.name = name;
  frame.resize(420, 180);
  frame.layoutMode = "VERTICAL";
  frame.itemSpacing = 10;
  frame.paddingTop = 20;
  frame.paddingRight = 20;
  frame.paddingBottom = 20;
  frame.paddingLeft = 20;
  frame.fills = [{ type: "SOLID", color: hexToRgb("#ffffff") }];
  frame.strokes = [{ type: "SOLID", color: hexToRgb("#e7ecf3") }];
  frame.strokeWeight = 1;
  frame.cornerRadius = 12;
  const h = t(`Title / ${title}`, title, 18, "Semi Bold");
  const p = t(`Body / ${title}`, body, 13);
  p.resize(360, 86);
  frame.appendChild(h);
  frame.appendChild(p);
  return frame;
}

const board = figma.createFrame();
board.name = "Relay composition examples";
board.x = 5000;
board.y = 840;
board.resize(1040, 620);
board.layoutMode = "VERTICAL";
board.itemSpacing = 20;
board.paddingTop = 40;
board.paddingRight = 40;
board.paddingBottom = 40;
board.paddingLeft = 40;
board.fills = [{ type: "SOLID", color: hexToRgb("#faf8f3") }];
board.strokes = [{ type: "SOLID", color: hexToRgb("#e7ecf3") }];
board.strokeWeight = 1;
board.cornerRadius = 12;

const title = t("Title", "Relay composition examples", 32, "Semi Bold");
const grid = figma.createFrame();
grid.name = "Example grid";
grid.layoutMode = "HORIZONTAL";
grid.layoutWrap = "WRAP";
grid.itemSpacing = 20;
grid.counterAxisSpacing = 20;
grid.fills = [];
grid.resize(920, 420);

grid.appendChild(card("Example / Style Memory review card", "Style Memory review card", "Refine surface. Proposal copy, source chips, Approve → Swan action, and magic halo only at transform moment."));
grid.appendChild(card("Example / Settings API key form", "Settings API key form", "Practical form surface. Field, feedback, status badge, and code-first error message pattern."));
grid.appendChild(card("Example / Project rail row", "Project rail row", "Navigation surface. Project name, mono glyphs, duckling/swan counts, active state."));
grid.appendChild(card("Example / Public snapshot card", "Public snapshot card", "Presentation surface. Authored packet preview with quiet provenance and editorial voice."));

board.appendChild(title);
board.appendChild(grid);
page.appendChild(board);

return { createdNodeIds: [board.id, title.id, grid.id], mutatedNodeIds: [page.id] };
```

Expected: `Work` contains `Relay composition examples` with four cards.

- [ ] **Step 3: Validate Task 6**

Use `mcp__codex_apps__figma._use_figma`:

```js
const page = figma.root.children.find((p) => p.name === "Work");
if (!page) throw new Error("Work page missing");
await figma.setCurrentPageAsync(page);
return {
  relayProductLayerCount: page.findAll((node) => node.name === "Relay product layer").length,
  relayExamplesCount: page.findAll((node) => node.name === "Relay composition examples").length,
  exampleCardCount: page.findAll((node) => node.name.startsWith("Example / ")).length
};
```

Expected: `relayProductLayerCount: 1`, `relayExamplesCount: 1`, `exampleCardCount >= 4`.

---

### Task 7: Final Validation And Local Plan Closure

**Files:**
- Read: Figma file `aX8b4hj4snppygfl07VARd`
- Read: `/Users/hoon-ch/repos/relay/docs/superpowers/specs/2026-05-03-4gly-design-system-figma-design.md`

- [ ] **Step 1: Run final Figma validation**

Use `mcp__codex_apps__figma._use_figma`:

```js
const requiredFrames = [
  ["Updates", "4gly v1 / Migration log"],
  ["Overview", "4gly v1 / Start here"],
  ["Ⓜ Makers’ Principle", "4gly maker principles"],
  [" ┗ Color - Semantic", "4gly semantic tokens"],
  [" ┗ Decorate", "4gly decoration and elevation policy"],
  ["③ Component", "Component status model"],
  [" ┗ Guidelines", "4gly component absorption guide"],
  ["Work", "Relay product layer"],
  ["Work", "Relay composition examples"]
];

const frameChecks = [];
for (const [pageName, frameName] of requiredFrames) {
  const page = figma.root.children.find((p) => p.name === pageName);
  if (!page) {
    frameChecks.push({ pageName, frameName, ok: false, reason: "page missing" });
    continue;
  }
  await figma.setCurrentPageAsync(page);
  const count = page.findAll((node) => node.name === frameName).length;
  frameChecks.push({ pageName, frameName, ok: count === 1, count });
}

const collections = await figma.variables.getLocalVariableCollectionsAsync();
const theme = collections.find((collection) => collection.name === "Theme");
let semanticVariableCount = 0;
if (theme) {
  for (const id of theme.variableIds) {
    const variable = await figma.variables.getVariableByIdAsync(id);
    if (variable && variable.name.startsWith("4gly/semantic/")) semanticVariableCount += 1;
  }
}

const paintStyles = await figma.getLocalPaintStylesAsync();
const effectStyles = await figma.getLocalEffectStylesAsync();

return {
  frameChecks,
  allFramesOk: frameChecks.every((check) => check.ok),
  hasThemeCollection: Boolean(theme),
  semanticVariableCount,
  semanticPaintCount: paintStyles.filter((style) => style.name.startsWith("4gly/Semantic/")).length,
  hasHaloEffect: effectStyles.some((style) => style.name === "4gly/Halo/Default")
};
```

Expected:

- `allFramesOk: true`
- `hasThemeCollection: true`
- `semanticVariableCount >= 17`
- `semanticPaintCount >= 10`
- `hasHaloEffect: true`

- [ ] **Step 2: Verify local repo status**

Run:

```bash
git status --short
```

Expected: no unexpected local code changes from Figma work. If the only changes are plan tracking checkboxes, either commit them intentionally or leave them unstaged with a clear final note.

- [ ] **Step 3: Final human QA in Figma**

Open:

```text
https://www.figma.com/design/aX8b4hj4snppygfl07VARd/4gly-Design-System
```

Expected visual checks:

- `Overview` explains full scaffold adoption and status model.
- `Updates` records migration baseline.
- `Color - Semantic` shows 4gly Light/Dark semantic tokens.
- `Decorate` explains halo and magic-color restrictions.
- `③ Component` preserves imported taxonomy and explains status model.
- `Work` shows Relay as a product layer with composition examples.

- [ ] **Step 4: Report completion**

Final response should include:

```text
Figma file updated: https://www.figma.com/design/aX8b4hj4snppygfl07VARd/4gly-Design-System
Validation passed:
- governance frames present
- 4gly semantic variables present
- 4gly paint/effect styles present
- component status model present
- Relay product mapping present
```

Expected: the user has enough information to inspect the Figma file directly.

---

## Self-Review

### Spec coverage

- Full scaffold adoption is covered by Task 2 and Task 5.
- Status layer is covered by Task 2 and Task 5.
- Semantic token override is covered by Task 3 and Task 4.
- Component taxonomy and variant preservation are covered by Task 5.
- Relay as first product layer is covered by Task 6.
- Validation is covered by Task 7.

### Open-item scan

This plan uses explicit steps for each work item. Any component not implemented
in Relay yet receives a status instead of being deleted or left unclassified.

### Type and API consistency

All Figma calls use top-level `await`, `figma.setCurrentPageAsync`, explicit
return values, and no `figma.notify()` or `figma.closePlugin()`.
