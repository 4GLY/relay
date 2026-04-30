# User API Keys and Korean i18n Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use `superpowers:executing-plans`
> to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for
> tracking. Do not skip the backend ownership tests. The security boundary is the
> whole point of this change.

Created: 2026-04-30
Branch: `main`
Mode: `gstack-autoplan` + `superpowers:writing-plans`

## Plan Summary

Relay already has two adjacent but different settings surfaces:

- Provider credentials: user-owned Anthropic key stored in
  `user_provider_credentials`, currently exposed at `/settings/providers`.
- Relay API keys: bearer tokens stored in `api_keys`, currently exposed only through
  admin-only `/v1/api-keys/*` endpoints.

This plan adds a user-facing Relay API key settings page and a small i18n layer so
Korean users see Korean UI without changing the core onboarding/product routing.

## Premises

1. Korean users should not have to read English UI for first-run and settings flows.
2. A logged-in user should be able to create and revoke their own Relay client keys
   without receiving the bootstrap admin token.
3. Provider keys and Relay API keys must remain visibly separate because they solve
   different jobs.
4. The first i18n pass should avoid a full route migration such as `/ko/...` until
   the product actually needs shareable locale-prefixed URLs.

## User-Facing Outcome

Authenticated users get:

- `/settings/api-keys`: issue, list, copy once, and revoke Relay API keys.
- `/settings/providers`: Anthropic provider credential remains separate.
- Korean UI when the browser or saved preference is Korean.
- English fallback for all strings.

Unauthenticated users get:

- A localized sign-in state.
- No access to key list, key issue, or key revoke actions.

## Current Code Leverage

| Need | Existing leverage | Gap |
| --- | --- | --- |
| Authenticated settings route | `web/app/settings/providers/page.tsx` | Need `/settings/api-keys` sibling |
| User session auth | `requireSessionOrAdmin`, `/v1/auth/me` | Need session-owned API key routes |
| API key persistence | `api_keys.owner_user_id`, `domain.APIKey.OwnerUserID` | Store lacks owner-filtered list/revoke |
| Admin API key issue/list/revoke | `internal/services/api_keys.go` | Admin-only service must not be reused directly for users |
| Frontend API wrapper | `web/lib/api.ts`, `web/lib/provider-credentials.ts` | Need `web/lib/user-api-keys.ts` |
| Existing settings QA | `web/e2e/relay-live-smoke.spec.ts` provider test | Add user API key settings smoke |
| i18n | None | Need dictionary and locale resolver |

## Architecture Decision

Do not expose the current admin endpoints to the frontend.

Add user-owned endpoints under `/v1/settings/api-keys`:

- `GET /v1/settings/api-keys`
- `POST /v1/settings/api-keys`
- `POST /v1/settings/api-keys/revoke`

These endpoints require session auth and operate only on `owner_user_id =
auth.UserID`.

Admin endpoints remain unchanged:

- `GET /v1/api-keys`
- `POST /v1/api-keys/issue`
- `POST /v1/api-keys/revoke`

## Backend Implementation Plan

1. Extend `repositories.APIKeyStore`.
   - `ListAPIKeysByOwner(ctx, userID string)`
   - `RevokeAPIKeyByOwner(ctx, userID string, keyID string)`

2. Extend Postgres store.
   - Filter list by `owner_user_id = $1`.
   - Revoke with `WHERE id = $1 AND owner_user_id = $2`.
   - Keep token hash and raw token write-only.

3. Add service methods.
   - `IssueUserAPIKey(ctx, input)`
   - `ListUserAPIKeys(ctx)`
   - `RevokeUserAPIKey(ctx, input)`

4. Service rules.
   - Require `AuthInfo.UserID`.
   - Default user keys to `scope: global` for now.
   - Set `OwnerUserID` on create.
   - Return raw `token` only on create.
   - List/revoke returns only metadata.
   - Do not allow a user to issue an admin key.

5. Add handlers and routes.
   - Register under `requireSessionOrAdmin`.
   - Route names stay under settings because this is a user settings concern.

6. Update OpenAPI and API docs.
   - Document the settings endpoints separately from admin API key endpoints.

## Frontend Implementation Plan

1. Add `web/lib/user-api-keys.ts`.
   - `listUserAPIKeys`
   - `issueUserAPIKey`
   - `revokeUserAPIKey`
   - typed response contracts mirroring Go envelopes

2. Add `/settings/api-keys`.
   - Server page forwards cookie and loads current user keys.
   - Unauthenticated state mirrors `/settings/providers`.
   - Client component handles issue/revoke.

3. UI behavior.
   - Key name input.
   - Optional scope display, initially `global`.
   - New token shown once after issue.
   - Copy button for the newly issued token.
   - List shows name, prefix, scope, project binding if present, revoked state.
   - Revoke uses an explicit confirm step, not instant destructive click.

4. Navigation.
   - Add cross-links between Provider settings and API key settings.
   - Add Project Explorer settings links without turning the page into a dashboard.

## i18n Implementation Plan

Start with a repo-owned dictionary layer rather than a route-level i18n library.

1. Add `web/lib/i18n.ts`.
   - Supported locales: `en`, `ko`.
   - Resolve locale from cookie first, then `Accept-Language`, then `en`.
   - `ko-KR`, `ko`, and Korean-weighted browser headers map to `ko`.

2. Add dictionaries.
   - `web/lib/i18n/dictionaries/en.ts`
   - `web/lib/i18n/dictionaries/ko.ts`

3. Add locale cookie action or endpoint.
   - Minimal option: `POST /settings/language` via Next route handler.
   - Store a cookie such as `relay_locale=ko`.

4. Wire high-value surfaces first.
   - `/`
   - `/onboarding`
   - `/settings/providers`
   - `/settings/api-keys`

5. Keep dynamic user data outside dictionaries.
   - Use interpolation at render time.
   - Do not put API errors directly into dictionaries unless mapping known error
     codes to friendly copy.

## Design Review

The settings experience should make the difference between keys obvious:

- Provider key: "Claude가 Relay 안에서 기능을 실행할 때 쓰는 외부 서비스 키"
- Relay API key: "외부 agent나 도구가 Relay API/MCP에 접근할 때 쓰는 키"

Recommended settings layout:

- Page title: "Settings" / "설정"
- Two sibling sections:
  - "Provider credentials" / "Provider 인증 정보"
  - "Relay API keys" / "Relay API 키"

Do not put both key types into one dense form. Users will paste the wrong key into
the wrong box.

## Engineering Review

### Dependency Graph

```text
Browser
  -> Next /settings/api-keys
      -> web/lib/user-api-keys.ts
          -> Go /v1/settings/api-keys
              -> services.Issue/List/RevokeUserAPIKey
                  -> repositories.APIKeyStore owner-filtered methods
                      -> postgres api_keys.owner_user_id

Browser
  -> Next pages/layout components
      -> web/lib/i18n.ts
          -> en/ko dictionaries
          -> relay_locale cookie + Accept-Language fallback
```

### Main Risks

| Risk | Severity | Mitigation |
| --- | --- | --- |
| User can list or revoke another user's key | Critical | Owner-filtered store methods and service tests |
| Raw token leaks after initial issue | Critical | Never persist or list raw token, create-only response |
| Admin key UI accidentally exposed | High | Keep admin routes untouched and separate settings routes |
| Korean text overflows existing large serif layouts | Medium | Screenshot mobile/desktop for Korean strings |
| API errors remain English-only | Medium | Map known error codes in the client |
| Locale choice changes public snapshot output unexpectedly | Medium | Keep public snapshot locale-neutral for this pass |

### Test Plan

Backend:

- `go test ./internal/services` for user API key service rules.
- `go test ./internal/api` for session-only routes and owner isolation.
- `go test ./internal/storage/postgres` if store tests exist or add focused fake
  coverage at service/API layer if not.

Frontend:

- `npm --prefix web test -- user-api-keys`
- `npm --prefix web test -- settings`
- `npm --prefix web test -- i18n`
- `npm --prefix web run lint`
- `npm --prefix web run typecheck`

E2E:

- Authenticated `/settings/api-keys` issue/revoke flow.
- Korean locale smoke for `/`, `/onboarding`, `/settings/providers`,
  `/settings/api-keys`.
- Mobile screenshot check for Korean strings.

## DX Review

Developer-facing docs must name the two key types clearly:

- Bootstrap admin token: deployment/operator secret.
- Relay client API key: issued key for API/MCP consumers.
- Provider credential: user-owned external provider key.

Update:

- `README.md`
- `docs/api.md`
- `docs/openapi.yaml`
- `docs/mcp-first-dogfood.md` if examples mention how a user obtains a client key.

## Not In Scope

- Locale-prefixed routes such as `/ko/settings/api-keys`.
- Translating every V2.5 surface in one pass.
- Multiple provider credential types beyond Anthropic.
- Role-based team key management.
- Public snapshot localization.
- Billing, quotas, or per-key usage analytics.

## Decision Audit Trail

| # | Phase | Decision | Classification | Principle | Rationale | Rejected |
| --- | --- | --- | --- | --- | --- | --- |
| 1 | CEO | Add self-service user API key settings instead of exposing admin key routes | Mechanical | Security boundary | Current `/v1/api-keys/*` requires admin auth and must remain operator-only | Calling admin endpoints from Next |
| 2 | CEO | Keep provider credentials and Relay API keys as separate settings surfaces | Mechanical | Explicit over clever | The two key types have different users, storage, and risk | One combined key form |
| 3 | Design | Start i18n with repo-owned dictionaries and cookie/header resolution | Taste | Pragmatic | Small blast radius and enough for Korean UI without route migration | Full locale-prefixed routing now |
| 4 | Eng | Add owner-filtered store/service methods rather than filtering in frontend | Mechanical | Completeness | Authorization must live in service/storage boundaries | Client-side filtering |
| 5 | DX | Update docs to distinguish admin token, client API key, and provider credential | Mechanical | User impact | Developers will otherwise paste the wrong secret into the wrong place | UI-only change |

## Recommended Build Order

1. Backend user API key service and routes.
2. Frontend `/settings/api-keys`.
3. i18n dictionary/resolver and Korean copy for root/onboarding/settings.
4. Tests and live QA.
5. Docs update and deployment.

## Final Gate Recommendation

Proceed with this plan, with one explicit choice:

- Recommended: dictionary-first i18n now, locale-prefixed routes later.
- Alternative: install route-level i18n now and move pages under locale segments.

The recommended path is smaller and fits the current product. The alternative is
more complete for a public multilingual SaaS, but it touches routing and QA scope
before Relay needs that complexity.

---

## Implementation Plan

### Phase 1: Backend User API Key Contract

- [x] **Step 1: Add service-level user API key tests**

  Files:

  - `internal/services/services_test.go`
  - `internal/services/api_keys.go`
  - `internal/services/types.go`

  Add tests before implementation:

  - logged-in user can issue a Relay API key without admin auth
  - issued key has `OwnerUserID == auth.UserID`
  - issued key defaults to `scope: global`
  - list returns only keys owned by the current user
  - revoke succeeds only for a key owned by the current user
  - unauthenticated context is rejected
  - raw token is returned only from issue, never from list or revoke

  Expected first run:

  ```bash
  go test ./internal/services -run 'UserAPIKey|APIKey'
  ```

  It should fail because the user-facing methods do not exist yet.

- [x] **Step 2: Extend API key store interface and fakes**

  Files:

  - `internal/storage/repositories/interfaces.go`
  - `internal/services/services_test.go`
  - `internal/api/server_test.go`
  - `internal/mcpserver/server_test.go`
  - any other fake implementing `repositories.APIKeyStore`

  Add:

  ```go
  ListAPIKeysByOwner(ctx context.Context, userID string) ([]domain.APIKey, error)
  RevokeAPIKeyByOwner(ctx context.Context, userID string, keyID string) (domain.APIKey, error)
  ```

  Fakes must enforce owner filtering, not return all keys and rely on callers.

- [x] **Step 3: Implement Postgres owner-filtered methods**

  File:

  - `internal/storage/postgres/stores.go`

  Add:

  - `ListAPIKeysByOwner`
  - `RevokeAPIKeyByOwner`

  SQL requirements:

  - `WHERE owner_user_id = $1`
  - revoke must include both `id = $1` and `owner_user_id = $2`
  - return `API_KEY_NOT_FOUND_BY_ID` or equivalent not-found error when the key is
    missing or owned by another user

- [x] **Step 4: Implement service methods**

  Files:

  - `internal/services/api_keys.go`
  - `internal/services/types.go`

  Add explicit user-owned methods rather than weakening existing admin methods:

  ```go
  IssueUserAPIKey(ctx context.Context, input IssueUserAPIKeyInput) (IssueAPIKeyResult, error)
  ListUserAPIKeys(ctx context.Context) (ListAPIKeysResult, error)
  RevokeUserAPIKey(ctx context.Context, input RevokeAPIKeyInput) (RevokeAPIKeyResult, error)
  ```

  Rules:

  - require `AuthInfo.UserID`
  - validate `name`
  - generate with `lib.NewSecretToken("relay_live")`
  - set `OwnerUserID`
  - set `Scope: APIKeyScopeGlobal`
  - do not accept user-supplied admin scope
  - do not return raw token except in issue result

- [x] **Step 5: Add API handler tests**

  Files:

  - `internal/api/server_test.go`
  - optionally `internal/api/handlers_api_keys.go` or new `handlers_user_api_keys.go`

  Tests:

  - `GET /v1/settings/api-keys` requires session
  - `POST /v1/settings/api-keys` requires session and returns raw token once
  - `POST /v1/settings/api-keys/revoke` requires session
  - user A cannot revoke user B key
  - existing admin `/v1/api-keys/*` routes still require admin bearer

- [x] **Step 6: Add settings API routes**

  Files:

  - `internal/api/server.go`
  - `internal/api/handlers_api_keys.go` or new `internal/api/handlers_user_api_keys.go`

  Register:

  ```text
  /v1/settings/api-keys
  /v1/settings/api-keys/revoke
  ```

  Use `requireSessionOrAdmin`, but service logic must still require `UserID` for
  user-owned operations. Admin auth without a user should not create a user-owned key.

- [x] **Step 7: Verify backend**

  Commands:

  ```bash
  go test ./internal/services
  go test ./internal/api
  go test ./internal/storage/postgres
  go test ./...
  ```

  If `go test ./internal/storage/postgres` has no focused tests or needs external DB,
  record the exact reason and rely on service/API isolation plus full `go test ./...`.

### Phase 2: Frontend API Key Settings Page

- [x] **Step 1: Add frontend API client tests**

  Files:

  - `web/lib/user-api-keys.ts`
  - `web/lib/user-api-keys.test.ts`

  Cover:

  - list success
  - issue success with one-time token
  - revoke success
  - Relay envelope error mapping

  Command:

  ```bash
  npm --prefix web test -- user-api-keys
  ```

- [x] **Step 2: Implement `web/lib/user-api-keys.ts`**

  Types:

  ```ts
  type UserAPIKeySummary = {
    key_id: string;
    name: string;
    token_prefix: string;
    scope: "global" | "project";
    project_id?: string;
    revoked: boolean;
  };
  ```

  Functions:

  - `listUserAPIKeys(headers?: HeadersInit)`
  - `issueUserAPIKey(name: string)`
  - `revokeUserAPIKey(keyID: string)`

- [x] **Step 3: Add `/settings/api-keys` server page**

  Files:

  - `web/app/settings/api-keys/page.tsx`
  - `web/app/settings/api-keys/api-key-settings-client.tsx`
  - `web/app/settings/api-keys/api-key-settings-client.test.tsx`

  Page behavior:

  - forward cookies from server component
  - show localized sign-in panel when unauthenticated
  - render initial key list for authenticated users
  - include links back to Project Explorer and Provider settings

- [x] **Step 4: Build the client interaction**

  Required UX:

  - name input
  - issue button disabled while empty or busy
  - one-time token reveal panel after issue
  - copy button for the one-time token
  - key list with prefix and scope
  - revoke confirm step per key
  - success/error states with `aria-live`

  Security UX:

  - never render a fake reusable token after refresh
  - make "copy now, it will not be shown again" explicit
  - do not use localStorage/sessionStorage for the token

- [x] **Step 5: Add settings navigation links**

  Files to inspect and update as appropriate:

  - `web/app/settings/providers/page.tsx`
  - `web/app/settings/providers/provider-settings-client.tsx`
  - `web/app/onboarding/onboarding-client.tsx`
  - `web/app/projects/[projectId]/page.tsx`

  Keep links utilitarian. No marketing page, no nested cards.

- [x] **Step 6: Verify frontend settings page**

  Commands:

  ```bash
  npm --prefix web test -- user-api-keys
  npm --prefix web test -- settings
  npm --prefix web run lint
  npm --prefix web run typecheck
  ```

### Phase 3: i18n Foundation and Korean Copy

- [x] **Step 1: Add i18n unit tests**

  Files:

  - `web/lib/i18n.ts`
  - `web/lib/i18n.test.ts`
  - `web/lib/i18n/dictionaries/en.ts`
  - `web/lib/i18n/dictionaries/ko.ts`

  Test:

  - `relay_locale=ko` resolves Korean
  - `relay_locale=en` resolves English
  - `Accept-Language: ko-KR,ko;q=0.9,en;q=0.8` resolves Korean
  - unsupported values fall back to English
  - missing dictionary keys fail loudly in tests

- [x] **Step 2: Implement i18n resolver**

  Use a small local helper:

  ```ts
  type Locale = "en" | "ko";
  function resolveLocale(input: { cookie?: string; acceptLanguage?: string }): Locale
  function getDictionary(locale: Locale): Dictionary
  ```

  Keep it dependency-free for this pass.

- [x] **Step 3: Add language preference endpoint**

  Minimal path:

  - `web/app/settings/language/route.ts`

  Behavior:

  - accepts `locale=en|ko`
  - sets `relay_locale`
  - rejects unsupported locale
  - uses `SameSite=Lax`

- [x] **Step 4: Localize high-value surfaces**

  Files:

  - `web/app/layout.tsx`
  - `web/app/page.tsx`
  - `web/app/onboarding/page.tsx`
  - `web/app/onboarding/onboarding-client.tsx`
  - `web/app/settings/providers/page.tsx`
  - `web/app/settings/providers/provider-settings-client.tsx`
  - `web/app/settings/api-keys/page.tsx`
  - `web/app/settings/api-keys/api-key-settings-client.tsx`

  Requirements:

  - `<html lang>` reflects resolved locale
  - Korean copy is natural, not literal machine translation
  - error messages map known Relay error codes to Korean
  - unsupported unknown errors can fall back to raw server message

- [x] **Step 5: Add Korean UI tests**

  Tests:

  - root renders Korean sign-in copy when locale resolves to `ko`
  - onboarding renders Korean first-run copy
  - provider settings renders Korean credential copy
  - API key settings renders Korean issue/revoke copy

  Command:

  ```bash
  npm --prefix web test -- i18n settings onboarding page
  ```

### Phase 4: Docs and Contract

- [x] **Step 1: Update OpenAPI**

  File:

  - `docs/openapi.yaml`

  Add schemas and paths for:

  - `GET /v1/settings/api-keys`
  - `POST /v1/settings/api-keys`
  - `POST /v1/settings/api-keys/revoke`

  Mark raw token as create-only in descriptions.

- [x] **Step 2: Update API docs**

  Files:

  - `README.md`
  - `docs/api.md`
  - `docs/mcp-first-dogfood.md`

  Clarify:

  - bootstrap admin token is operator-only
  - Relay client API key is for API/MCP consumers
  - provider credential is for external AI provider access
  - logged-in users can issue their own client key from Settings

- [x] **Step 3: Update QA docs**

  Files:

  - `docs/live-e2e-qa.md`
  - `docs/v2-5-closure.md` only if the current closure state needs a note

  Add the new live QA evidence requirements for user API keys and Korean locale.

### Phase 5: Live E2E and Release

- [x] **Step 1: Add Playwright coverage**

  File:

  - `web/e2e/relay-live-smoke.spec.ts`

  Add:

  - authenticated API key issue/revoke flow, Chromium only if shared state risk remains
  - Korean locale smoke for root, onboarding, provider settings, API key settings
  - mobile viewport screenshot for Korean API key page

- [x] **Step 2: Run local verification**

  Commands:

  ```bash
  go test ./...
  npm --prefix web test
  npm --prefix web run lint
  npm --prefix web run typecheck
  npm --prefix web run build
  ```

- [ ] **Step 3: Run live QA with authenticated cookies**

  Use the existing gstack browser/cookie setup if needed.

  Required checks:

  - `/settings/api-keys` loads for logged-in user
  - issue returns a visible one-time token
  - refresh hides the raw token and keeps only prefix metadata
  - revoke removes or marks the key revoked
  - Korean locale renders on `/`, `/onboarding`, `/settings/providers`,
    `/settings/api-keys`
  - no console errors
  - mobile Korean layout has no overlapping text

- [ ] **Step 4: Commit and PR**

  Commit shape:

  ```text
  feat: add user API key settings and Korean i18n
  ```

  PR body must include:

  - backend ownership boundary summary
  - i18n scope
  - test commands and results
  - live QA screenshots or run id

- [ ] **Step 5: Merge, publish, deploy, Argo**

  Follow the existing release flow:

  - merge feature PR
  - watch `publish-relay-api`
  - watch `publish-relay-web`
  - merge deploy PRs if created
  - verify Argo `Synced/Healthy`
  - run live smoke again on `https://relay.4gly.dev`

## Completion Criteria

- Users can create and revoke their own Relay client API keys from the web UI.
- Users cannot see or revoke another user's keys.
- Admin API key endpoints remain admin-only.
- Provider credentials remain separate from Relay API keys.
- Korean users see Korean copy on root, onboarding, provider settings, and API key settings.
- Raw API token is visible only immediately after creation.
- Unit, backend, typecheck, lint, build, and live QA evidence are recorded.
