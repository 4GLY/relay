# Relay — Claude Code working instructions

Relay is an API-first second-brain backend for long-running AI-assisted work, shipped by **4gly Labs**. Read these before making changes.

## Design System

Always read `DESIGN.md` before making any visual or UI decision. All font choices, colors, spacing, and aesthetic direction are defined there. Do not deviate without explicit user approval.

When you write or review any UI code:
- Use only tokens defined in `DESIGN.md` `Colors` and the token front matter.
- Use canonical token language in new docs: `accent`, `surface-inverse`, `status-pending`, `status-complete`, `focus-ring`.
- Treat existing legacy runtime CSS variables and badge class names as compatibility aliases until the web CSS migration removes them.
- Use the declared typefaces: **Fraunces** (display + editorial italic), **Nunito** (body / UI), **Commit Mono** (mono, with **JetBrains Mono** as stand-in until licensed).
- Never reintroduce banned fonts (Inter, Roboto, Arial, system-ui, Space Grotesk, Poppins, Montserrat, Helvetica, Open Sans, Lato, Clash Display).
- Accent tokens appear only for focus, review, selected paths, and completion emphasis. They are never ambient background.
- Elevation targets soft focus-ring emphasis. Existing shared primitives still contain legacy shadow styles; do not expand them, and migrate touched surfaces toward the canonical focus-ring model.

In QA mode, flag any code that does not match `DESIGN.md`.

## Parent brand

The product's voice and visual identity live in `4gly_foundation.md` at `/Users/hoon-ch/repos/4gly/4gly_foundation.md`. When the design system and the foundation disagree, the foundation wins — re-sync `DESIGN.md` rather than drifting.

The product promise: **"A calm workspace that turns unstructured work into reviewable decisions."**
Every UI change must serve that sentence.
Older runtime metadata and localized copy may lag this promise during the staged
migration; treat `DESIGN.md` as canonical for new UI and docs.

## Skill routing

When the user's request matches an available skill, invoke it via the Skill tool. The skill has multi-step workflows, checklists, and quality gates that produce better results than an ad-hoc answer. When in doubt, invoke the skill. A false positive is cheaper than a false negative.

Key routing rules:
- Product ideas, "is this worth building", brainstorming → invoke `/office-hours`
- Strategy, scope, "think bigger", "what should we build" → invoke `/plan-ceo-review`
- Architecture, "does this design make sense" → invoke `/plan-eng-review`
- Design system, brand, "how should this look" → invoke `/design-consultation`
- Design review of a plan → invoke `/plan-design-review`
- Developer experience of a plan → invoke `/plan-devex-review`
- "Review everything", full review pipeline → invoke `/autoplan`
- Bugs, errors, "why is this broken", "wtf", "this doesn't work" → invoke `/investigate`
- Test the site, find bugs, "does this work" → invoke `/qa` (or `/qa-only` for report only)
- Code review, check the diff, "look at my changes" → invoke `/review`
- Visual polish, design audit, "this looks off" → invoke `/design-review`
- Developer experience audit, try onboarding → invoke `/devex-review`
- Ship, deploy, create a PR, "send it" → invoke `/ship`
- Merge + deploy + verify → invoke `/land-and-deploy`
- Configure deployment → invoke `/setup-deploy`
- Post-deploy monitoring → invoke `/canary`
- Update docs after shipping → invoke `/document-release`
- Weekly retro, "how'd we do" → invoke `/retro`
- Second opinion, codex review → invoke `/codex`
- Safety mode, careful mode, lock it down → invoke `/careful` or `/guard`
- Restrict edits to a directory → invoke `/freeze` or `/unfreeze`
- Upgrade gstack → invoke `/gstack-upgrade`
- Save progress, "save my work" → invoke `/context-save`
- Resume, restore, "where was I" → invoke `/context-restore`
- Security audit, OWASP, "is this secure" → invoke `/cso`
- Make a PDF, document, publication → invoke `/make-pdf`
- Launch real browser for QA → invoke `/open-gstack-browser`
- Import cookies for authenticated testing → invoke `/setup-browser-cookies`
- Performance regression, page speed, benchmarks → invoke `/benchmark`
- Review what gstack has learned → invoke `/learn`
- Tune question sensitivity → invoke `/plan-tune`
- Code quality dashboard → invoke `/health`

## Repo conventions (load-bearing — don't guess)

- Language: **Go**.
- Surface: HTTP API at `/v1/*` + MCP at `/mcp` (both bearer-auth).
- Local agent bootstrap: `./skills/relay-api-agent/scripts/setup.sh`.
- Canonical API contract: `docs/openapi.yaml`.
- Migration history: `migrations/`.

## Non-negotiables

- Do not commit without user approval.
- Do not push to remote without user approval.
- Do not add a UI feature that bypasses `DESIGN.md`.
- Do not add decorative metaphor or character work on the primary surface — evidence, review state, and graph behavior carry the product identity.
