# Relay Live E2E QA

Date: 2026-04-29

## Purpose

`scripts/qa/run_live_e2e.sh` runs the browser smoke suite against a live Relay
deployment with deterministic temporary QA state:

- authenticated user session
- owned Style Memory project with one pending proposal
- completed onboarding state
- positive public snapshot token
- provider settings store/list/delete flow

The script removes the temporary user, project, provider credential, and public
snapshot fixture after Playwright exits.

## Command

```bash
scripts/qa/run_live_e2e.sh
```

Optional overrides:

```bash
RELAY_WEB_BASE_URL=https://relay.4gly.dev scripts/qa/run_live_e2e.sh
RELAY_DATABASE_URL=postgresql://... scripts/qa/run_live_e2e.sh
```

If `RELAY_DATABASE_URL` is not set, the script reads
`relay-secrets/database_url` from the `relay` namespace.

## Latest Result

Run ID: `qa20260429000502c10725`

Target: `https://relay.4gly.dev`

Deployment evidence:

- Source commit under test: `4553d0e79f67acc1409ef9eb33e6f6f51203c659`
- Argo revision: `fa6bc7bd5d5a011de30fe0310a82f4de8da2bc81`
- Argo status: `Synced / Healthy`
- `relay-web`: `ghcr.io/4gly/relay-web:sha-4553d0e79f67acc1409ef9eb33e6f6f51203c659`
- `relay-api`: `ghcr.io/4gly/relay-api:sha-4553d0e79f67acc1409ef9eb33e6f6f51203c659`
- `relay-curator-worker`: `ghcr.io/4gly/relay-api:sha-4553d0e79f67acc1409ef9eb33e6f6f51203c659`

Observed result:

- `17 passed`
- `1 skipped`
- public snapshot fixture: `/p/psnap_qa20260429000502c10725_token`

The skipped test is the mobile project for the provider credential mutation.
That mutation intentionally runs once on desktop Chromium to avoid two browser
projects racing on the same user-owned credential row.

Cleanup verification:

- temporary project rows: `0`
- temporary user rows: `0`
- temporary public snapshot rows: `0`
