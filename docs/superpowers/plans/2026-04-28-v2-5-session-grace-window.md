<!-- /autoplan restore point: /Users/hoon-ch/.gstack/projects/relay/main-autoplan-restore-20260428-204255.md -->
# V2.5 Stable Session Refresh Implementation Plan

**Goal:** Remove the V2 accepted edge case where concurrent `/v1/auth/me` requests during the 7-day refresh window can invalidate one browser tab's `relay_session` and force a surprise re-login.

**Architecture:** Keep the existing `user_sessions` table and existing `relay_session` cookie value stable. `/v1/auth/me` remains the only rolling-refresh endpoint, but refresh now extends `user_sessions.expires_at` without replacing `token_hash`. The handler still re-sets the cookie so browser expiry moves forward, but the raw token value stays the same. Concurrent tabs become idempotent because all tabs keep a valid token after refresh.

**Tech Stack:** Go services/storage, `httptest` auth handler tests, fake stores for service-level race coverage.

---

## Original V2 State

- `internal/services/users.go` rotates session tokens only in `ResolveSession`, used by `/v1/auth/me`.
- `internal/storage/postgres/users.go` stores one `token_hash` directly on `user_sessions`.
- `RotateUserSession` atomically replaces `user_sessions.token_hash`.
- If two tabs call `/v1/auth/me` near expiry, one tab may receive a new cookie and the other may keep an instantly stale cookie.
- Authenticated non-`/me` routes use `GetUserBySessionToken`, which validates without rotating to avoid making the race worse.

## Proposed Data Model

No migration. `user_sessions` remains canonical:

- `token_hash`: stable bearer token hash for the session.
- `expires_at`: rolling session expiry and cookie expiry source.
- `revoked_at`: immediate server-side revocation.

The V2.5 change is behavioral: refresh updates only `expires_at`; it never changes `token_hash`.

## Refresh Policy

- Fresh token: valid until session expires or is revoked.
- Near-expiry token on `/v1/auth/me`: extend `user_sessions.expires_at` by `userSessionTTL` and re-set the same cookie value with the new expiry.
- Concurrent `/v1/auth/me`: safe and idempotent. One request may extend expiry first; the next request sees a fresh session and skips refresh.
- Authenticated non-`/me` routes continue validating without extending expiry.
- Logout still revokes `user_sessions.revoked_at` and invalidates the stable token immediately.

## File Structure

- Modify `internal/storage/repositories/interfaces.go`
- Modify `internal/storage/postgres/users.go`
- Modify `internal/services/users.go`
- Modify fake stores:
  - `internal/services/auth_fake_stores_test.go`
  - `internal/api/auth_fakes_test.go`
- Modify tests:
  - `internal/services/users_test.go`
  - `internal/api/handlers_auth_test.go`

---

## Task 1: Storage Contract

- [x] Rename `RotateUserSession` to `RefreshUserSessionExpiry` in the repository interface.
- [x] Update Postgres implementation so it updates only `expires_at`, guarded by `id`, `token_hash`, `revoked_at IS NULL`, and `expires_at > NOW()`.
- [x] Keep `CreateUserSession`, `GetUserSessionByTokenHash`, and `RevokeUserSession` behavior unchanged.

## Task 2: Service Semantics

- [x] On sign-in, create the same `user_sessions` row as V2.
- [x] On fresh session lookup, return no refresh.
- [x] On near-expiry lookup, extend session expiry and return `Refreshed=true` with the original raw token.
- [x] If another request already extended expiry, authorize the user and skip re-setting the cookie.
- [x] On logout, revoke the session.
- [x] Update comments in `internal/services/users.go` so the V2.5 policy is clear.

## Task 3: Race And Handler Tests

- [x] Service test: near-expiry `ResolveSession` refreshes expiry while keeping the token value stable.
- [x] Service test: repeated/concurrent-ish `ResolveSession` calls using the same token both authorize the user.
- [x] Service test: logout still rejects the stable token.
- [x] API test: `/v1/auth/me` reissues the cookie with the same value.
- [x] API test: the original cookie still returns `200` after refresh.

## Task 4: Verification

Run:

```bash
go test ./internal/services ./internal/api ./internal/storage/postgres
go test ./...
```

No migration verification is required for A because no schema changes are introduced.

## Out Of Scope

- Refresh-token family analytics.
- Device/session management UI.
- Manual "sign out other devices."
- Cross-device session list.
- Token binding to device fingerprint or IP.
- Session-token family and grace-token storage.

## Acceptance Criteria

- A user with two tabs open does not get logged out because one tab refreshed `/v1/auth/me` first.
- The cookie token value remains stable across rolling refresh.
- Logout remains immediate for the session.
- Existing OAuth sign-in behavior is unchanged.
- Existing cookie shape remains `relay_session=<token>`.

---

## /autoplan Review Log

### Phase 0: Intake

- Plan summary: V2.5 fixes the accepted V2 race where concurrent `/v1/auth/me` refreshes can make a sibling browser tab hold a stale `relay_session`.
- UI scope: no. This is storage/service/API behavior.
- DX scope: yes. The plan changes repository contracts, migration behavior, auth semantics, test commands, and operator verification.
- Loaded review skill methodology from disk: CEO, Eng, and DX. Design review skipped because no UI scope was detected.
- Codex CLI: available and authenticated. Codex CEO, Eng, and DX voices ran in read-only mode.
- Claude subagent voice: degraded to local primary review because this Codex environment does not permit agent spawning unless the user explicitly asks for subagents.

### Phase 1: CEO Review

#### Premise Challenge

| Premise | Evaluation | Decision |
|---|---|---|
| Concurrent `/v1/auth/me` can stale another tab's cookie | Confirmed by current code comments and tests that assert old-token invalidation after rotation | Keep |
| The issue deserves a V2.5 item | Plausible, because live onboarding/style-memory QA now depends on stable cookie auth | Keep, but scope must stay lean |
| A child token table is the best default fix | Challenged by Codex CEO voice; stable-token rolling expiry may solve the same user problem with less risk | Surface as User Challenge |
| Logout must revoke every valid token in a session | Confirmed; any solution must preserve this | Keep |

#### What Already Exists

| Sub-problem | Existing code |
|---|---|
| Session issue on OAuth sign-in | `internal/services/users.go` creates `user_sessions` with one `token_hash` |
| Session lookup | `GetUserSessionByTokenHash` in `internal/storage/postgres/users.go` filters revoked/expired rows |
| Rolling refresh | `ResolveSession` rotates within `userSessionRefreshWindow` |
| Logout | `handleAuthLogout` resolves cookie to session ID, then calls `SignOut` |
| Regression coverage | `internal/services/users_test.go` and `internal/api/handlers_auth_test.go` currently pin immediate old-token invalidation |

#### Dream State Delta

```
CURRENT
  one session row = one token hash
  /me refresh replaces token
  losing tab can 401

THIS PLAN
  parent session + child token rows
  short grace window
  logout revokes token family

12-MONTH IDEAL
  stable session UX
  explicit device/session management if needed
  auth telemetry and cleanup policy
  token-family complexity only if product value needs it
```

#### Implementation Alternatives

| Alternative | Effort | Risk | Pros | Cons | Review outcome |
|---|---:|---:|---|---|---|
| Stable token + rolling expiry | low | low | Solves two-tab stale-cookie race without schema split | Less token churn/security theater | Recommended by CEO voice |
| Child token table + grace window | medium-high | medium | Future device/session features become easier | More migration and concurrency surface | Current draft, only if rotation is a real requirement |
| Defer and reduce client polling | low | medium | Avoids auth migration now | Does not fully solve backend race | Reject for completeness |

#### Error & Rescue Registry

| Failure | User-facing rescue | Operator rescue |
|---|---|---|
| Stale cookie after refresh | Keep the user authenticated or reissue a cookie deterministically | Track refresh conflict count |
| Migration backfill failure | Roll back before app deploy | Avoid `pgcrypto`; deterministic IDs |
| Logout does not revoke child tokens | Browser cookie cleared and server session revoked | Transactional parent+child revoke test |
| Grace token accepted too long | Bounded grace expiry | SQL check and repository tests |

#### Failure Modes Registry

| Failure mode | Severity | Mitigation |
|---|---|---|
| Grace token accidentally valid for 30 days | Critical | Explicit parent expiry vs token expiry semantics |
| Rotation churn from stale tabs | High | Prefer stable token, or make rotation idempotent |
| Unique current-token conflict | High | Lock parent row and define transaction semantics |
| `pgcrypto` missing in migration | High | Deterministic backfill IDs |
| Fake-only tests miss row-lock bugs | Medium | Add Postgres repository tests |

#### CEO Dual Voices Consensus

| Dimension | Local review | Codex | Consensus |
|---|---|---|---|
| Premises valid? | Yes | Yes | Confirmed |
| Right problem to solve? | Yes, but keep small | Maybe too small vs Relay core gaps | Taste |
| Scope calibration correct? | Child table may be heavy | Too heavy | User Challenge |
| Alternatives explored? | Needs stable-token alternative | Stable token was skipped | Confirmed gap |
| Competitive/product risk covered? | Not central | Core Relay value may be delayed | Confirmed risk |
| 6-month trajectory sound? | Only if future session UI is planned | Could look like auth over-investment | Taste |

Phase 1 complete. Codex: 6 concerns. Local review: 4 issues. Consensus: 3/6 confirmed, 3 surfaced at gate.

### Phase 2: Design Review

Skipped: no UI scope detected.

### Phase 3: Engineering Review

#### Scope Challenge

The current child-token design is implementable, but not implementation-ready until it pins expiry semantics, return types, transaction shape, and repository-level race tests. The most important engineering challenge is whether token rotation is a real requirement. If it is not, stable token + rolling expiry is materially simpler and safer.

#### Architecture Diagram

```
HTTP /v1/auth/me
  -> services.ResolveSession(raw cookie)
    -> UserSessionStore.GetUserSessionByTokenHash(hash)
      -> user_sessions parent row
      -> user_session_tokens matched token row
    -> maybe UserSessionStore.RotateUserSessionToken(...)
      -> transaction
        -> lock parent session
        -> demote current token to grace
        -> insert new current token
        -> update parent expires_at + legacy token_hash
        -> prune stale child tokens
  -> Set-Cookie when refreshed

HTTP protected routes
  -> requireSessionOrAdmin
    -> services.GetUserBySessionToken(raw cookie)
      -> same lookup, no rotation

HTTP /v1/auth/logout
  -> services.GetSessionByToken(raw cookie)
  -> services.SignOut(session_id)
    -> revoke parent session + child tokens
```

#### Test Diagram

| Code path | Branch | Test type | Current coverage | Required update |
|---|---|---|---|---|
| `ResolveSession` fresh token | no refresh | service unit | Exists | Keep |
| `ResolveSession` current near expiry | refresh | service unit + API | Exists but asserts old invalid | Invert expectation or stable-token assertion |
| `ResolveSession` stale/grace token | authorize within bound | service + API | Missing | Add |
| `ResolveSession` grace expiry | reject | service + repository | Missing | Add |
| `RotateUserSessionToken` transaction | conflict/unique index | Postgres repository | Missing | Add |
| `SignOut` | revoke all valid tokens | service + repository | Partial parent-only | Add child-token case |
| migration backfill | existing sessions | migration smoke | Missing | Add |

Test plan artifact: `/Users/hoon-ch/.gstack/projects/relay/main-test-plan-20260428-session-grace-window.md`

#### Eng Dual Voices Consensus

| Dimension | Local review | Codex | Consensus |
|---|---|---|---|
| Architecture sound? | Needs tightening | Not implementation-ready | Confirmed issue |
| Test coverage sufficient? | No | No | Confirmed issue |
| Performance risks addressed? | Token cleanup missing | Cleanup missing | Confirmed issue |
| Security threats covered? | Partially | Under-specified | Confirmed issue |
| Error paths handled? | Not enough | Logout and expiry ambiguous | Confirmed issue |
| Deployment risk manageable? | Needs phased rollout | Migration can fail | Confirmed issue |

Phase 3 complete. Codex: 6 findings. Local review: 5 findings. Consensus: 6/6 confirmed.

### Phase 3.5: DX Review

#### Developer Journey Map

| Stage | Current plan friction | Fix |
|---|---|---|
| Read plan | Agent-only instruction in repo doc | Removed |
| Understand data model | Parent/token expiry blurred | Added explicit expiry semantics |
| Implement migration | `pgcrypto` assumption | Switched to deterministic backfill ID |
| Implement interfaces | Return type vague | Added `UserSessionLookup` requirement |
| Implement rotation | Transaction shape vague | Added lock/demote/insert/update/prune sequence |
| Write tests | "Concurrent-ish" vague | Added service/API/Postgres test split |
| Verify migration | Optional command | Added explicit migrate command and SQL smoke |
| Roll out | Rollback invariant unclear | Added legacy token hash compatibility invariant |
| Operate | No cleanup/observability | Added cleanup; observability remains follow-up |

#### DX Scorecard

| Dimension | Score | Note |
|---|---:|---|
| Getting started | 7/10 | Clear file list; now needs selected architecture |
| API naming | 7/10 | `UserSessionLookup` improves ergonomics |
| Error messages | 6/10 | User-facing errors stay stable; internal reason logging still missing |
| Docs findability | 7/10 | Plan plus external test artifact are discoverable |
| Upgrade path | 6/10 | Needs final chosen rollout path |
| Dev environment | 7/10 | Uses existing Go test/migrate commands |
| Observability | 4/10 | Metrics/log fields still deferred |
| Cleanup/maintenance | 6/10 | Pruning added, retention policy still light |

#### DX Dual Voices Consensus

| Dimension | Local review | Codex | Consensus |
|---|---|---|---|
| Getting started under 5 min? | No, because schema choice is open | No | Confirmed issue |
| API naming guessable? | Improve with lookup type | Needs concrete type | Confirmed issue |
| Error messages actionable? | Stable externally | Missing internal reasons | Confirmed issue |
| Docs complete? | Mostly after patch | Needs API docs note | Confirmed issue |
| Upgrade path safe? | Needs phased deploy | Needs phased deploy | Confirmed issue |
| Dev environment friction-free? | Mostly | Migration DB needed | Confirmed issue |

Phase 3.5 complete. DX overall: 6.25/10. TTHW: 45-90 min if child-token path, 20-40 min if stable-token path.

### Cross-Phase Themes

- **Theme: child-token table may be over-scoped** — flagged in CEO and Eng. High-confidence because the target race can be solved without introducing a token family if token rotation is not a product/security requirement.
- **Theme: expiry semantics must be explicit** — flagged in Eng and DX. Parent session expiry and token validity cannot be inferred from one `expires_at` column without mistakes.
- **Theme: migration safety matters more than unit coverage** — flagged in Eng and DX. This change is primarily storage semantics, so fake-only tests are insufficient.

## Decision Audit Trail

| # | Phase | Decision | Classification | Principle | Rationale | Rejected |
|---|---|---|---|---|---|---|
| 1 | Phase 0 | Skip Design review | Mechanical | Explicit over clever | No UI scope detected | Running irrelevant UI review |
| 2 | Phase 1 | Keep the premise that stale-cookie refresh is real | Mechanical | Bias toward action | Existing comments and tests confirm the behavior | Re-litigating observed V2 edge |
| 3 | Phase 1 | Surface stable-token rolling expiry as a user challenge | User Challenge | Pragmatic | Both CEO and engineering review see it as simpler for the same UX goal | Auto-changing the user's chosen architecture |
| 4 | Phase 3 | Remove `gen_random_bytes` backfill | Mechanical | Explicit over clever | Existing migrations avoid pgcrypto; deterministic IDs reduce deploy risk | Adding pgcrypto dependency |
| 5 | Phase 3 | Require explicit parent/token expiry semantics | Mechanical | Completeness | Avoids accidental 30-day grace tokens | Leaving semantics to implementer |
| 6 | Phase 3 | Require concrete lookup return type | Mechanical | Explicit over clever | Service needs matched token status without overloading `UserSession` | Implicit status inference |
| 7 | Phase 3 | Require Postgres repository tests | Mechanical | Completeness | Race behavior depends on SQL transaction/index behavior | Fake-only concurrent-ish tests |
| 8 | Phase 3.5 | Add migration command and smoke check | Mechanical | Bias toward action | Gives implementer a concrete verification path | Optional migration verification |
| 9 | Phase 3.5 | Keep observability as a follow-up unless implementation touches logging infra | Taste | Pragmatic | Useful but not required to prove the V2.5 fix | Blocking the auth fix on metrics plumbing |

## Final Approval Gate

### User Challenge 1: Choose the session-refresh architecture

You originally approved continuing with a child-token grace-window direction.

Both CEO and engineering review recommend changing the default to **stable token + rolling expiry** unless there is a specific security/product requirement for rotating bearer material.

Why:

- It fixes the actual UX failure: two tabs do not invalidate each other.
- It removes the need for `user_session_tokens`, grace expiry, one-current uniqueness, pruning, and transactional demotion.
- It makes out-of-order browser `Set-Cookie` responses harmless because the token value does not change.
- Current Relay does not yet have device/session UI, token-family analytics, or theft detection that would justify token-family complexity.

What context the reviews might be missing:

- If you intentionally want token-family groundwork for future device/session management, the child-token table is a reasonable stepping stone.
- If token rotation is considered a security requirement for Relay regardless of current product surface, stable token may be too conservative.

If the reviews are wrong, the cost is:

- Choosing stable-token now may require a later migration when Relay adds device-level session controls.

### Taste Decision 1: Observability now or later

Recommendation: defer structured auth metrics/log fields unless the implementation already touches shared logging. The core correctness work is tests + storage semantics. Adding observability is useful, but it should not block the narrow V2.5 fix.

### Deferred to TODOS

- Device/session management UI.
- "Sign out other devices."
- Token-family analytics and theft detection.
- Removing legacy `user_sessions.token_hash` after a separate rollback window.

### Approval Outcome

User selected **A** at the final gate. The implementation plan was rewritten to the stable-token rolling-expiry variant, and code changes were applied in this branch:

- `relay_session` raw token value remains stable across `/v1/auth/me` refresh.
- `RefreshUserSessionExpiry` updates only `user_sessions.expires_at`.
- Near-expiry `/v1/auth/me` reissues the same cookie value with a newer expiry.
- Original cookies remain valid after refresh, so sibling tabs do not surprise-logout.

Verification:

```bash
go test ./internal/services ./internal/api ./internal/storage/postgres
go test ./...
```
