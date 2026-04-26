---
status: complete
priority: p1
issue_id: "001"
tags: [web, onboarding, settings, database]
dependencies: []
---

# S8 Keyless Onboarding And Provider Settings

## Problem Statement

S8 onboarding must let a user enter Relay without supplying an Anthropic key. The backend contract now supports keyless onboarding, but the web UI is still a placeholder, `npm run lint` is broken under Next 16, and provider credentials need a separate Settings-owned storage model instead of living in onboarding.

## Findings

- `POST /v1/onboarding` now accepts an empty object and returns `onboarding_complete` plus `default_project_id`.
- `npm run lint` currently calls `next lint`, which Next 16 treats as a project directory and fails with `/web/lint`.
- Provider key storage should move to a dedicated `user_provider_credentials` table so onboarding state remains independent from future provider connection flows.

## Proposed Solutions

### Option 1: Minimal Contract Completion

**Approach:** Fix lint script, build keyless onboarding UI, and add the provider credential schema + service API without a full credential settings UI.

**Pros:**
- Keeps S8 unblocked quickly.
- Establishes the durable provider credential boundary.
- Limits UI scope while the product flow is still forming.

**Cons:**
- Settings UI may still need a follow-up for full provider management.

**Effort:** 1 focused session

**Risk:** Medium

### Option 2: Full Onboarding And Settings UI

**Approach:** Build both onboarding and complete provider settings CRUD UI now.

**Pros:**
- End-user provider flow lands in one pass.

**Cons:**
- Larger blast radius.
- Provider product requirements are not yet as firm as keyless onboarding.

**Effort:** 2-3 sessions

**Risk:** Higher

## Recommended Action

Use Option 1. Fix lint first, implement S8 keyless onboarding UI, then add the dedicated provider credential persistence boundary with tests and OpenAPI/docs. Keep Anthropic key entry out of onboarding.

## Technical Details

Affected areas:
- `web/package.json` and ESLint config
- `web/app/onboarding/*`
- `web/lib/api.ts` or adjacent API wrapper
- `internal/domain`, `internal/storage/postgres`, `internal/services`, `internal/api`
- `migrations/*`
- `docs/openapi.yaml`

## Acceptance Criteria

- [x] `npm run lint` works under Next 16.
- [x] `/onboarding` provides a real keyless first-run flow and calls `POST /v1/onboarding {}`.
- [x] Successful onboarding routes the user into the app surface without provider key input.
- [x] Provider key persistence is modeled in `user_provider_credentials`, not onboarding.
- [x] Backend provider credential tests cover create/update/delete/status behavior.
- [x] OpenAPI/docs describe onboarding and provider credential boundaries.
- [x] Go and web verification pass.

## Work Log

### 2026-04-27 - Start Implementation

**By:** Codex

**Actions:**
- Created isolated worktree at `.worktrees/s8-onboarding-settings`.
- Verified baseline with `go test ./...`, `npm run typecheck`, `npm test`, and `npm run build`.
- Captured the ordered user checklist as this ready todo.

**Learnings:**
- Next 16 build/typecheck/test baseline is green.
- Existing lint script is the known blocker to address first.

### 2026-04-27 - Implementation Complete

**By:** Codex

**Actions:**
- Replaced `next lint` with ESLint CLI + flat config and verified `npm run lint`.
- Built the keyless `/onboarding` UI and client API wrapper for `POST /v1/onboarding {}`.
- Added `/settings/providers` UI and provider credential API wrappers.
- Added `user_provider_credentials` migration, domain model, storage, services, handlers, OpenAPI docs, and tests.
- Verified with `go test ./...`, `npm run lint`, `npm run typecheck`, `npm test`, `npm run build`, and OpenAPI YAML parsing.

**Learnings:**
- Provider credentials now have a dedicated Settings-owned boundary and no longer need to share onboarding state.
