# Relay V2 Baseline Closure

Date: 2026-04-28

## Status

Relay V2 baseline is closed as shipped and verified.

The original gstack V2 baseline scope is complete:

- S1: Auth foundation
- S2: Style Memory auth refactor
- S3: Sharable packet snapshot backend
- S4: Onboarding backend
- S5: Next web scaffold
- S6: Style Memory UI
- S7: Public snapshot route decision
- S8: Keyless onboarding and provider settings
- S9: Backend integration coverage
- S10: Playwright smoke and visual evidence

## Final Decisions

### S7 Canonical Public Snapshot Surface

Go `/p/{token}` is the canonical public snapshot surface.

Rationale:

- The Go API already owns public token lookup.
- It serves the canonical HTML with OG metadata.
- It serves `/p/{token}/og.png`.
- It owns revoke behavior and returns `410 Gone` for unknown or revoked tokens.
- Keeping a separate Next public snapshot implementation would duplicate the route contract and risk drift.

The Next `web/app/p/[snapshotId]` route is now only a same-path redirect fallback for requests that reach the web service instead of the API gateway.

### S9 Backend Integration Coverage

The backend integration suite now covers the full public snapshot lifecycle through the mux:

1. Admin publishes a packet snapshot.
2. Publish response returns `public_url`, `og_image_url`, and `public_token`.
3. Public HTML route returns `200` and includes OG metadata.
4. OG PNG route returns `200 image/png`.
5. Admin revokes the snapshot.
6. Public HTML route returns `410 Gone`.

### S10 Browser Smoke Coverage

Playwright smoke coverage now exists for live or local Relay web targets:

- `/`
- `/onboarding`
- `/style-memory?project=<project>`
- `/settings/providers`
- `/p/unknown_s10_snapshot_token`
- optional positive `/p/<token>` check when `RELAY_QA_PUBLIC_SNAPSHOT_TOKEN` is set

The token-dependent public snapshot positive case is intentionally skipped without a supplied live token.

## Implementation Evidence

Code merge:

- `bc2147a test: close s7 s9 s10 qa plan`
- `e1470b2 merge: close s7 s9 s10 qa plan`

Deploy PRs:

- PR #75: `chore: deploy relay-web image sha-e1470b279116cf9c52ad770fbe1e356107917a7d`
- PR #76: `chore: deploy relay-api image sha-e1470b279116cf9c52ad770fbe1e356107917a7d`

Final deployed images:

- `relay-web`: `ghcr.io/4gly/relay-web:sha-e1470b279116cf9c52ad770fbe1e356107917a7d`
- `relay-api`: `ghcr.io/4gly/relay-api:sha-e1470b279116cf9c52ad770fbe1e356107917a7d`
- `relay-curator-worker`: `ghcr.io/4gly/relay-api:sha-e1470b279116cf9c52ad770fbe1e356107917a7d`

Argo:

- Application: `relay`
- Revision: `3ef4b67defe28f8b92e7291ddba6cfc528cb0cf2`
- Status: `Synced Healthy Succeeded`

Runtime:

- `relay-web`: `1/1`
- `relay-api`: `1/1`
- `relay-curator-worker`: `1/1`

## Verification

Local verification before merge:

```bash
go test ./...
cd web
npm run test
npm run lint
npm run typecheck
npx playwright test --list
```

Live verification after deploy:

```bash
curl -i https://relay.4gly.dev/healthz
curl -i https://relay.4gly.dev/p/unknown_s10_snapshot_token
cd web
RELAY_WEB_BASE_URL=https://relay.4gly.dev npm run qa:e2e
```

Observed live results:

- `/healthz`: `200 OK`
- `/p/unknown_s10_snapshot_token`: `410 Gone`
- Playwright live smoke: `10 passed`, `2 skipped`

The two skipped checks are the optional positive public snapshot checks that require `RELAY_QA_PUBLIC_SNAPSHOT_TOKEN`.

Post-closure live QA hardening:

- `scripts/qa/run_live_e2e.sh` now seeds temporary live QA state and runs authenticated browser coverage without manual cookie setup.
- The live authenticated suite covers completed-onboarding redirect, owned Style Memory queue rendering, Provider Settings validation/store/delete, and positive public snapshot HTML + OG PNG.
- Latest run: `17 passed`, `1 skipped`; the skipped case is the mobile duplicate of the provider credential mutation to avoid cross-browser state races.

## QA Artifacts

Tracked implementation plan:

- `docs/superpowers/plans/2026-04-28-s7-s9-s10-closure.md`

Local gstack evidence:

- `.gstack/qa-reports/qa-report-relay-4gly-dev-2026-04-27.md`
- `.gstack/qa-reports/screenshots/s10-onboarding.png`
- `.gstack/qa-reports/screenshots/s10-style-memory.png`
- `.gstack/qa-reports/screenshots/s10-settings-providers.png`
- `.gstack/qa-reports/screenshots/s10-public-snapshot-410.png`

## Outcome

V2 baseline is ready to close.

Next product planning should start from V2.5, with the first slice likely centered on Project Explorer plus the read-model APIs that later V2.5 surfaces depend on.
