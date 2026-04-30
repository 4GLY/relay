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

## Manual Project Explorer QA

Date: 2026-04-30

Target: `https://relay.4gly.dev`

Deployment evidence:

- Source commit under test: `4ec13cd25406682cccc9208d8dc3d35958421bd0`
- Deploy commit under test: `126e9e4a07e08362c174fbd1bc95166702b948d0`
- Argo status: `Synced / Healthy`
- `relay-web`: `ghcr.io/4gly/relay-web:sha-4ec13cd25406682cccc9208d8dc3d35958421bd0`
- `relay-api`: `ghcr.io/4gly/relay-api:sha-01a088020570ea3de8a22dbbd40cba80f5b0bff6`

Authenticated project:

- User: `hoon_ch`
- Project: `proj_28cc65685c63`
- Project owner: `usr_460ac62a4904`

Checks:

- `/` with a valid `relay_session` redirected to `/projects/proj_28cc65685c63`.
- Project Explorer rendered the `Personal` project, counts, Style Memory preview,
  latest snapshot, and recent activity.
- Explorer API returned `200 OK` for
  `/v1/projects/proj_28cc65685c63/explorer`.
- Explorer Style Memory link resolved to
  `/style-memory?project=proj_28cc65685c63`.
- Style Memory rendered authenticated tab counts:
  `Proposals 0`, `Approved 5`, `Rejected 2`.
- Desktop Project Explorer screenshot showed no visible overlap.
- Mobile Project Explorer screenshot at `390x844` showed no visible overlap.
- gstack browser console checks reported no console errors on Project Explorer
  and Style Memory.

Artifacts:

- `.gstack/qa-reports/screenshots/v2-5-project-explorer-desktop.png`
- `.gstack/qa-reports/screenshots/v2-5-project-explorer-mobile.png`
- `.gstack/qa-reports/screenshots/v2-5-style-memory-from-explorer.png`

Cleanup verification:

- QA browser sessions created for this run: `3`
- QA browser sessions revoked after the run: `3`
