# Relay — TODOS

Deferred work tracked outside the active CEO/eng/design plans. Each item references the source decision so context is recoverable.

## V2 implementation status

- **S1 (auth foundation)** ✅ DONE 2026-04-25 on branch `feat/s1-auth-foundation` — migration `0007_users.sql`, domain types, repositories, OAuth provider abstraction (GitHub + Google with the E4 mitigation), cookie session middleware, `/v1/auth/{provider}/start|callback`, `/v1/auth/me`, `/v1/auth/logout`, project-owner authorization for style-memory writes (R1 admin path preserved), unit tests, OpenAPI updates.
  - **Session refresh policy** (V2.5 stable-token update, 2026-04-28): user sessions enforce 30-day TTL + 7-day rolling refresh on `/v1/auth/me` only. Refresh keeps the `relay_session` token value stable and extends `user_sessions.expires_at`, so concurrent `/me` polls cannot invalidate a sibling tab's cookie. Other authenticated endpoints (style-memory, etc.) validate without extending expiry.
  - **Future session work**: token-family/grace-window storage is deferred until Relay needs device/session management, "sign out other devices," token-family analytics, or theft-detection semantics.


## Deferred from V2 CEO Plan Rev 2 (2026-04-25, post-Codex reframe)

Source: `~/.gstack/projects/relay/ceo-plans/2026-04-25-v2-end-user-surface.md` (Rev 2)

V2 scope (post-reframe): Minimal authenticated Style Memory UI + A (Sharable Packet Snapshot URL) + C (1-click Onboarding Flow).

Items below are **NOT in V2** but tracked for V2.5 / V3.

### V2.5 (load-bearing — slated for the next product cycle after V2 ships)

**1. DESIGN.md remaining priority screens** — Decision Graph, Packet Builder WYSIWYG
- Status: Project Explorer and Trace Browser shipped in V2.5 on 2026-04-30 with live E2E coverage.
- Effort remaining: M/L UI + M backend read-model polish.
- Why still tracked: Decision Graph needs the canonical graph projection to include Style Memory and packet evidence, and Packet Builder still needs an authoring/review surface rather than latest-snapshot reading only.
- Unblocks: full DESIGN.md realization, dense workspace experience, signature ribbon
- Depends on: V2 ship + at least one cycle of demand evidence
- Source: CEO plan Rev 2 Scope Decision #4
- V2.5 entry slice: `docs/v2-5-project-explorer-scope.md`
- Decision Graph slice: `docs/v2-5-decision-graph-scope.md`
- Next Packet Builder slice: `docs/v2-5-packet-builder-wysiwyg-scope.md`

**2. B: Public Style Profile** — `/u/{username}/style` (the LinkedIn-for-engineering-judgment surface)
- Effort: L/XL (full effort estimate revised after Codex finding #1: cross-project aggregation, slug reservation, privacy state, publish UX. User model lands in V2 baseline so partial reuse.)
- Why deferred: Codex finding #5 — depends on cross-project aggregation that's V3 territory. Original Rev 1 estimate (M ~1d) was wrong by an order of magnitude.
- Unblocks: portfolio surface, identity moment, viral entry point beyond individual snapshot URLs
- Depends on: V2 demand validation, V3 candidate A (Multi-Project Style Transfer) for cross-project heuristic aggregation
- Source: CEO plan Rev 2 Scope Decision #5

**3. F: Multi-agent Live Workspace** — Claude + Codex simultaneous driver visualization
- Effort: L (new event product per Codex finding #6 — current MCP stateless request-response, needs new realtime surface)
- Why deferred: Codex flagged this as a new event product, not a viz layer. High value but architectural cost is L not M. After V2 demand validation justifies the realtime infrastructure.
- Unblocks: thesis demo (multi-agent Style Memory consumption), competitive differentiation vs single-agent Mem/Reflect/Notion AI
- Depends on: V2 ship, realtime infrastructure decision (SSE? WebSocket? polling?), Codex trace ingestion path resolved (Open Q #7 from Rev 1)
- Source: CEO plan Rev 2 Scope Decision #6

**4. Inline Curator Suggestions** — confidence + reasoning side panel during approval
- Effort: M (~2d after backend curator extension)
- Why deferred: backend curator does not yet emit confidence/reasoning. Backend extension prerequisite.
- Depends on: backend curator change to emit per-proposal confidence + reasoning fields
- Source: CEO plan Rev 2 Scope Decision #8

### Eng Review failure-mode mitigations (must land in V2 implementation)

Source: `/gstack-plan-eng-review` 2026-04-25, Section 4 failure modes table

**E1. Onboarding KMS error UX (failure mode #1)**
- Production failure: KMS encrypt fails (IAM/rate limit) → silent 500 → user retries indefinitely
- Mitigation: explicit error codes from `internal/services/onboarding.go` → recoverable UX in Frame 2 with message "Couldn't securely store your key. We'll try again automatically." + retry button
- Effort: S (~2h CC), part of Slice 4 (Onboarding service)

**E2. CDN cache invalidation on snapshot revoke (failure mode #2)**
- Production failure: CDN serves stale OG image after `public_readable=false` toggle → external recipient sees outdated preview while page is 410 Gone
- Mitigation: hook CDN cache invalidation API into `POST /v1/snapshots/{id}/publish` toggle (Vercel Cache-Tag invalidation if Vercel-hosted, or Cloudflare purge API)
- Effort: S (~2h CC), part of Slice 3 (Sharable URL)
- Note: needs CDN provider decision in eng implementation

**E3. 900ms transform "Saving..." chip after 1.5s in aria-busy (failure mode #3)**
- Production failure: API takes 5s mid-transform → card frozen in `aria-busy` → user thinks UI is frozen → second click ignored without feedback
- Mitigation: after 1.5s in `aria-busy`, show subtle Fraunces italic "Saving..." chip; after 5s show "Still saving... [Cancel]" option
- Effort: S (~1h CC), part of Slice 2 (Style Memory screen)

**E4. GitHub OAuth missing email scope detection (failure mode #4)** ✅ IMPLEMENTED 2026-04-25
- Production failure: GH returns valid token but user has private email → cannot create user account → silent fail / generic error
- Mitigation shipped in `internal/lib/oauth/github.go` (`Exchange` method): after the `/user` fetch we always call `/user/emails` and pick the primary verified address. If still empty (very rare), the OAuth flow keeps going with a NULL email user record (schema allows it) and the auto-link step is skipped.
- Effort: S (~1-2h CC), part of Slice 1 (Auth refactor)

### V2.5 / V3 candidates (delight, not load-bearing)

**5. Embed Widget** — `<script>` tag for blog/README packet preview embedding
- Effort: M (~3d CC)
- Why deferred: viral layer beyond V2 baseline. Add only if A (Sharable URL, now in V2) shows real share traffic
- Depends on: V2 A traffic data, CSP/iframe security model
- Source: CEO plan Rev 2 Scope Decision #7

**6. Style-aware vs Control Inline Diff Toggle** — UI-side surface for V1's eval pipeline comparison
- Effort: S (~4h CC)
- Why deferred: already in eval scripts. UI replication is nice-to-have
- Source: CEO plan Rev 2 Scope Decision #9

**7. Cmd+K Command Palette** — Linear-style action palette
- Effort: S (~4h CC)
- Why deferred: power-user delight. First 5 external users will not measure V2 by command palette presence
- Source: CEO plan Rev 2 Scope Decision #10

**8. Project Pulse Heatmap** — Linear-style activity heatmap, but for judgment density per project
- Effort: S (~4h CC)
- Why deferred: surface enhancement, does not move PMF needle
- Source: CEO plan Rev 2 Scope Decision #11

**9. Replay Last Handoff Debugger** — UI-side rewind through packet build steps with selection rationale
- Effort: M (~1d CC)
- Why deferred: ops/internal tool. Does not generate demand evidence
- Source: CEO plan Rev 2 Scope Decision #12

## V3 Candidates (post-V2.5)

Source: CEO plan Rev 2 §"Phase 2 / Phase 3 Trajectory"

- **V3 candidate A**: Multi-Project Style Transfer (the V1 design doc's stated V2 expansion) — generalizes Style Memory across projects. B (V2.5) depends on a primitive version of this; full version is V3.
- **V3 candidate B**: Embeddable trace + widget + integrations (viral platform layer)
- **V3 candidate C**: Multi-user / org accounts / team workspaces (only if individual demand validated)

## Design debt from V2 plan-design-review (2026-04-25)

Source: `~/.gstack/projects/relay/ceo-plans/2026-04-25-v2-end-user-surface.md` Rev 2 + cross-model design review (Codex outside voice + Claude subagent, both scored 6/10)

### CRITICAL — must address before V2 implementation

**D1. Re-author sharable-url HTML at 100% fidelity** ✅ DONE 2026-04-25
- Rev 2 finalized via `/gstack-design-html` with Pretext-native layout + design review CRITICAL findings addressed
- File: `~/.gstack/projects/relay/designs/sharable-url-20260425/finalized.html` (rev 2)
- Pretext sibling: `~/.gstack/projects/relay/designs/sharable-url-20260425/pretext.js` (30KB vendored)
- All findings closed: hero CTA, swan-contour seal, first-heuristic-above-fold, mood reduction, density modulation, memorable thing footer
- See finalized.json for full revision history

**D2. Re-author onboarding HTML at 100% fidelity** ✅ DONE 2026-04-25
- Rev 2 finalized via `/gstack-design-html` with Pretext-native layout + all 9 design review CRITICAL findings addressed
- File: `~/.gstack/projects/relay/designs/onboarding-20260425/finalized.html` (rev 2)
- Pretext sibling: `~/.gstack/projects/relay/designs/onboarding-20260425/pretext.js` (30KB vendored)
- All findings closed: memorable thing in F1 hero, F4 LIVE 900ms transform (3 layered animations), "Capture your first judgment" CTA, F2 with 4 error state panels (invalid_anthropic_key/anthropic_quota/relay_url_unreachable/relay_unauthorized), F2 magic-primary-strong glow + editorial typography, F3 magic-primary 30% pill on active step, removed storyboard chrome (each frame is product screen now), F1 outcome preview (stacked rotated cards showing actual heuristic content), mood repetition removed
- See finalized.json for full revision history

**D3. Style Memory single gravitational card refactor**
- DESIGN.md §8 violation per both reviewers
- Action: per Open Q #15. Edit existing finalized HTML to rank proposals by confidence, apply post-approval state preview to rank-1 only with halo + magic-accent border + Fraunces italic
- Effort: S (~1-2h CC)

**D4. 900ms signature transform failure-mode design**
- Per Open Q #10
- Action: extend Style Memory finalized HTML with: optimistic UI render, `aria-busy` lock, 220ms rewind on failure, `--danger` chip "Couldn't save — retry?", height lock during transform
- Effort: S (~1-2h CC)

### HIGH — address during V2 implementation slices

**D5. Sharable URL static brand signal (frozen swan-contour seal)**
- Per Open Q #12. Sharable URL is read-only so motion cannot play; static seal is the brand surrogate.
- Effort: S (~1h CC)

**D6. Sharable URL hero CTA**
- Per Open Q #13. Codex HARD REJECTION
- Effort: trivial (~30min CC)

**D7. Style Memory time-depth meta line**
- Per Open Q #16. Adds 5-year reflective dimension cheaply
- Effort: trivial (~30min CC)

**D8. OG image generation pipeline + visual spec**
- Per CEO plan Open Q #6 + design review specificity gap
- Visual spec: 1200×630, --canvas background, Fraunces 96px headline = first heuristic title, Fraunces italic 28px rationale, swan-contour silhouette at 30% opacity bottom-right, mono meta strip at base
- Effort: M (~2-3h CC)

### MEDIUM — V2.5 territory (deferred surface design issues)

**D9. Decision Graph magic-color misuse fix** (V2.5) ✅ ADDRESSED IN FIRST SLICE 2026-04-30
- Codex finding: "Magic colors used as graph vocabulary, not transformation moments. Reserve magic for selected/yielded transform path."
- DESIGN.md §6 rule 1 violation
- Action: first Decision Graph slice uses neutral edge strokes by default and reserves the magic accent for the active route / transform path.
- Follow-up: keep this rule when adding interactive filtering and graph layout controls.

**D10. Packet Builder source panel default-closed** (V2.5)
- Codex finding: "Source panel open by default fights the packet document"
- Action: default sources panel to collapsed; toggle button promotes them
- Effort: trivial (~30min CC). To address before V2.5

**D11. Project Explorer ops stats demoted to inspector drawer** (V2.5)
- Codex finding: "Curator success exposes pipe-level system detail too early"
- Action: move ops/curator stats from main metric strip into inspector drawer
- Effort: S (~1h CC). To address before V2.5

**D12. Judgment Traces first-run state** (V2.5)
- Codex finding: "Filter surface is dense before user intent is clear. Default to one trace narrative, then filters."
- Effort: S (~1-2h CC). To address before V2.5

### Missing first-run / empty / error / failure states (consolidated, ~40 items)

Across all 7 surfaces, the cross-model design review inventoried ~40 distinct missing states. Eng review should map each to backend semantics and route appropriate UI states. Top items:

**Style Memory**: no pending proposals, no approved heuristics, approval failure mid-transform, edit validation, reject undo, auth expired, no project selected, sync conflict from another device, single-proposal layout

**Sharable URL**: private snapshot, expired/deleted snapshot, owner revoked public access, empty packet (no style cues), missing OG image, share/copy success, broken/expired link

**Onboarding**: invalid Anthropic key, Relay URL unreachable, OAuth cancelled, secret-storage consent prompt, no prior traces (curator empty), first packet build failure, Anthropic rate limit / quota, "user already has an account" (returning user short-circuit)

**V2.5 surfaces** (Decision Graph, Packet Builder): remaining states should be inventoried during each surface slice. Project Explorer and Judgment Traces now have first shipped states plus live E2E coverage.

Each state needs: visual spec (DESIGN.md token map), backend trigger (which API state produces this), keyboard/accessibility behavior. Track per-state during V2 eng review.

## Operational Improvements (gstack-related)

- **Slug mismatch resolved upstream**: `gstack-slug` returns `relay`, `remote-slug` returns `4GLY-relay`. Skill bash that uses `remote-slug` for project artifact lookup misses the `relay/` directory. File a gstack issue or local override. Logged as a learning at `~/.gstack/projects/relay/learnings.jsonl` 2026-04-25.

## Rev History

- **Rev 1 (2026-04-25)** — initial deferred set after V2 cherry-pick (B + F accepted, A + C deferred)
- **Rev 2 (2026-04-25, post-Codex reframe)** — V2 cherry-pick flipped after cross-model tension. A + C now in V2; B + F + DESIGN.md remaining 4 screens now in V2.5.
