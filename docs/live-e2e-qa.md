# Relay Live E2E QA

Date: 2026-04-30

## Purpose

`scripts/qa/run_live_e2e.sh` runs the browser smoke suite against a live Relay
deployment with deterministic temporary QA state:

- authenticated user session
- owned Style Memory project with one pending proposal
- completed onboarding state
- positive public snapshot token
- provider settings store/list/delete flow
- user-owned Relay API key issue/list/revoke flow
- Korean locale smoke for first-run and settings surfaces

The script removes the temporary user, project, provider credential, user-owned
API keys, and public snapshot fixture after Playwright exits.

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

## Current Coverage

The live suite covers:

- `/settings/api-keys` renders authenticated settings.
- API key settings issues a user-owned Relay client key.
- Refresh hides the raw one-time token and keeps only prefix metadata.
- API key settings revokes the user-owned key.
- Korean locale smoke covers `/`, `/onboarding`, `/settings/providers`, and
  `/settings/api-keys`.

## Latest Result

Run ID: `qa202604301114048db40a`

Target: `https://relay.4gly.dev`

Deployment evidence:

- Source commit under test: `e22ec25c041e11609e5ea4e6e248b84e05db9bbb`
- Deploy commit under test: `188e8e95f0970dd355c4711e57cbaa8bc0c52fa7`
- Argo revision: `188e8e95f0970dd355c4711e57cbaa8bc0c52fa7`
- Argo status: `Synced / Healthy`
- `relay-web`: `ghcr.io/4gly/relay-web:sha-e22ec25c041e11609e5ea4e6e248b84e05db9bbb`
- `relay-api`: `ghcr.io/4gly/relay-api:sha-e22ec25c041e11609e5ea4e6e248b84e05db9bbb`
- `relay-curator-worker`: `ghcr.io/4gly/relay-api:sha-e22ec25c041e11609e5ea4e6e248b84e05db9bbb`

Observed result:

- `30 passed`
- `2 skipped`
- public snapshot fixture: `/p/psnap_qa202604301114048db40a_token`

The skipped tests are the mobile projects for provider credential and API key
mutations. Those mutations intentionally run once on desktop Chromium to avoid
two browser projects racing on the same user-owned rows.

Additional authenticated coverage:

- `/onboarding` redirects completed users into Project Explorer.
- Project Explorer links to Style Memory and Trace Browser.
- Project Explorer exposes the latest public snapshot link when
  `latest_snapshot.public_token` is present.
- Trace Browser renders seeded judgment traces on desktop Chromium and mobile
  Chrome.
- Provider settings stores masked metadata and disconnects.
- API key settings issues a one-time `relay_live_` token and revokes the
  user-owned key.
- Korean locale renders first-run and settings copy.

Cleanup verification:

- temporary project rows: `0`
- temporary user rows: `0`
- temporary session rows: `0`
- temporary provider credential rows: `0`
- temporary API key rows: `0`
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

## Pre-Deploy Project Explorer + Trace Browser QA

Date: 2026-04-30

Target: `http://127.0.0.1:3000` web, backed by `https://relay.4gly.dev` API

Run ID: `qa20260430011628e3e478`

Scope:

- Project Explorer is covered by the automated authenticated live smoke suite.
- Project Explorer links to Style Memory and Trace Browser.
- Trace Browser renders live judgment trace fixtures on desktop Chromium and
  mobile Chrome.
- The script-created project, user, session, judgment trace, heuristic proposal,
  and packet snapshot fixtures were cleaned up by `scripts/qa/run_live_e2e.sh`.

Observed result:

- `16 passed`
- `6 skipped`
- temporary project: `proj_qa20260430011628e3e478`
- temporary public snapshot fixture:
  `/p/psnap_qa20260430011628e3e478_token`

Skip notes:

- Public snapshot route checks are skipped against local Next because `/p/*`
  is only a placeholder redirect there; canonical public snapshot routing is
  verified on the live Go surface.
- Provider credential mutation is skipped against local web because it requires
  same-origin live routing.
- The Project Explorer public snapshot link check is reserved for live deploy
  because the pre-deploy live API does not yet include `latest_snapshot.public_token`.
