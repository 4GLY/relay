# Relay V1 Canonical Handoff Eval

This eval locks the first V1 proof:

- same project
- model-to-model or session-to-session continuation
- explicit seed flow
- public MCP packet resume
- paired style-aware vs control packet comparison

## Runner

Use:

```bash
set -a; source .env; set +a
./scripts/acceptance/v1_canonical_handoff.sh \
  --base-url "${RELAY_BASE_URL:-http://127.0.0.1:8080}"
```

Required credentials:

- `RELAY_CLIENT_TOKEN`: issued client token for normal API and public MCP calls
- `RELAY_ADMIN_TOKEN`: bootstrap admin token for proposal approval

The runner writes:

- `.gstack/projects/relay/<run_id>/result.json`
- `.gstack/projects/relay/<run_id>/summary.md`
- `.gstack/projects/relay/<run_id>/style-packet.json`
- `.gstack/projects/relay/<run_id>/control-packet.json`
- `.gstack/projects/relay/usage-validation.jsonl`

## Flow

1. `POST /v1/capture` creates or updates the acceptance project and attaches trusted repo, handoff, and design artifacts.
2. `POST /v1/promote` creates one durable decision from the seed memory and links the captured artifacts as supporting evidence.
3. `POST /v1/promote` creates one open question from the same seed memory and links the same supporting artifacts.
4. `POST /v1/judgment-traces` stores the seed judgment.
5. `POST /v1/heuristic-proposals` creates a pending proposal from the trace.
6. `POST /v1/heuristic-proposals/review` approves it through the admin path.
7. `relay_build_packet` over public MCP builds a style-aware packet with `persist_snapshot: true`.
8. `relay_build_packet` over public MCP builds a control packet with `disable_style_cues: true`.
9. The runner records timings, snapshot IDs, reused heuristic IDs, and rubric fields.

## Blind Judge

After an acceptance run, ask an external model to judge the paired packets:

```bash
./scripts/evals/v1_claude_paired_judge.sh \
  --result-file .gstack/projects/relay/<run_id>/result.json \
  --model opus
```

The judge script writes:

- `.gstack/projects/relay/<run_id>/paired-comparison-prompt.md`
- `.gstack/projects/relay/<run_id>/claude-judge.json`
- `.gstack/projects/relay/<run_id>/paired-comparison.json`

The judge is intentionally blind to which packet is style-aware. It maps packet A/B back to `style-aware` or `control` only after the model returns its preference.

## Retrieval Baseline Judge

After an acceptance run, you can also compare the new retrieval-aware packet path against a ranking-only baseline:

```bash
./scripts/evals/v1_retrieval_baseline_judge.sh \
  --result-file .gstack/projects/relay/<run_id>/result.json \
  --model opus
```

The retrieval judge:

- rebuilds one packet with normal query-conditioned retrieval
- rebuilds one packet with `disable_retrieval: true`
- runs a blind paired judge over `retrieval-aware` vs `ranking-only`
- writes `retrieval-aware-packet.json`, `ranking-only-packet.json`, and `retrieval-baseline-comparison.json`

## Consumer Continuation Eval

After an acceptance run, run real consumer agents against the style-aware packet:

```bash
./scripts/evals/v1_consumer_continuation.sh \
  --result-file .gstack/projects/relay/<run_id>/result.json \
  --claude-model opus
```

Use `--reuse-existing` to rerun only the comparison judge when both consumer
outputs already exist.

This eval asks a fresh Claude consumer and a fresh Codex consumer to continue
from the packet only, then asks Claude/Opus to compare the two continuation
outputs.

It writes:

- `.gstack/projects/relay/<run_id>/consumer-continuation-prompt.md`
- `.gstack/projects/relay/<run_id>/claude-consumer-continuation.json`
- `.gstack/projects/relay/<run_id>/codex-consumer-continuation.json`
- `.gstack/projects/relay/<run_id>/consumer-continuation-comparison.json`

This is intentionally optional for now. It validates actual model-to-model or
session-to-session consumption behavior without making every PR depend on a
second consumer-agent run.

## Repeated Usage Validation

For the next-stage benchmark, run the fixture batch instead of a single canonical seed:

```bash
set -a; source .env; set +a
./scripts/evals/v1_usage_validation_batch.sh \
  --fixtures-file scripts/evals/fixtures/v1_usage_validation.json \
  --base-url "${RELAY_BASE_URL:-https://relay.4gly.dev}" \
  --model opus
```

To include real consumer-agent continuation checks for every fixture, add:

```bash
./scripts/evals/v1_usage_validation_batch.sh \
  --fixtures-file scripts/evals/fixtures/v1_usage_validation.json \
  --base-url "${RELAY_BASE_URL:-https://relay.4gly.dev}" \
  --model opus \
  --consumer-continuation
```

To measure consumer-score stability before turning it into a release gate, run
the same consumer batch repeatedly:

```bash
set -a; source .env; set +a
./scripts/evals/v1_consumer_continuation_stability.sh \
  --fixtures-file scripts/evals/fixtures/v1_usage_validation.json \
  --fixture-limit 1 \
  --runs 3 \
  --base-url "${RELAY_BASE_URL:-https://relay.4gly.dev}" \
  --model opus
```

The stability runner writes:

- `.gstack/projects/relay/stability/<prefix>/fixtures.json`
- `.gstack/projects/relay/stability/<prefix>/consumer-stability-summary.json`
- `.gstack/projects/relay/stability/<prefix>/consumer-stability-summary.md`

Use `--fixture-limit 0` only when the full five-fixture consumer run is worth
the extra model time. The first threshold candidate should be based on at least
three consumer continuation runs.

The batch runner:

- reuses the same acceptance contract for multiple scenarios
- attaches richer evidence pointers including code paths plus run-generated changed-files manifests and PR-diff snapshots
- runs the blind paired judge after each fixture
- runs the retrieval baseline judge after each fixture so `retrieval-aware` can be compared against `ranking-only`
- optionally runs the real Claude/Codex consumer continuation eval after each fixture when `--consumer-continuation` is set
- writes `batch-runs.jsonl` plus `batch-summary.json` and `batch-summary.md`
- evaluates a release gate from style-aware win rate, average `style_match`, retrieval-aware win rate, retrieval readiness/evidence scores, and budget-pass rate
- reports consumer continuation metrics when present, but does not gate releases on them yet

## CI Gate

The repo now includes a PR gate workflow at `.github/workflows/usage-validation-gate.yml`.

It runs the same repeated usage-validation benchmark against a local `relay-api` instance backed by a disposable Postgres service, then fails the job when the batch gate fails.

Workflow requirements:

- self-hosted Linux runner labeled `relay-evals`
- repository secret `CLAUDE_CODE_OAUTH_TOKEN` for headless `claude` CLI evaluation in GitHub Actions
- local operator runs can also reuse an existing `claude auth login` session
- GitHub Actions runner with Node.js 24+ so `npm install -g @anthropic-ai/claude-code` works and the repo is already opted into Node 24 JavaScript actions
- current retrieval gate defaults:
  - `RELAY_EVAL_MIN_RETRIEVAL_AWARE_WIN_RATE=0.6`
  - `RELAY_EVAL_MIN_AVG_RETRIEVAL_CONTINUATION_READINESS=3.5`
  - `RELAY_EVAL_MIN_AVG_RETRIEVAL_EVIDENCE_RELEVANCE=3.5`

Runner operations live in `docs/ops/self-hosted-runner.md`.

## Protected Main Publish

The repo also uses `.github/workflows/publish-relay-api.yml` to build `ghcr.io/4gly/relay-api:sha-<commit>` and sync `deploy/k8s/deployment.yaml`.

When `main` has required status checks, having the workflow push a manifest commit directly back to the protected branch conflicts with branch protection.

Current behavior:

- the workflow still uses `GITHUB_TOKEN` for GHCR publish
- when `deploy/k8s/deployment.yaml` changes, it commits that diff onto an automation branch
- it opens a PR back to `main` using a real repo token so the PR events still trigger downstream workflows
- it enables auto-merge so the PR lands only after the normal `usage-validation` required check passes

Repository requirement:

- repository setting `allow_auto_merge=true`
- repository secret `RELAY_PUSH_TOKEN` with a repo-scoped token for branch push and PR creation
- repository variable `RELAY_PUSH_USERNAME` with the GitHub login that owns that token

This keeps `main` protected while preserving the image-publish plus manifest-sync path.

Local and CI both use the same helper:

```bash
scripts/ci/run_usage_validation_gate.sh
```

That helper:

- runs migrations
- starts `relay-api` locally
- verifies `claude` authentication from `CLAUDE_CODE_OAUTH_TOKEN`, `ANTHROPIC_API_KEY`, or an existing `claude auth login` session
- runs `scripts/evals/v1_usage_validation_batch.sh`
- appends `batch-summary.md` to `GITHUB_STEP_SUMMARY` when running in Actions
- fails when either the style-aware gate or the retrieval-aware gate falls below threshold

Outputs land under:

- `.gstack/projects/relay/batches/<batch_id>/fixtures.json`
- `.gstack/projects/relay/batches/<batch_id>/generated-artifacts/<fixture_id>/changed-files.txt`
- `.gstack/projects/relay/batches/<batch_id>/generated-artifacts/<fixture_id>/pr-diff.md`
- `.gstack/projects/relay/batches/<batch_id>/batch-runs.jsonl`
- `.gstack/projects/relay/batches/<batch_id>/batch-summary.json`
- `.gstack/projects/relay/batches/<batch_id>/batch-summary.md`

Each fixture now declares `evidence_paths` instead of checking in static sample files. The batch runner generates:

- `changed-files.txt` from the current working-tree diff for those paths, with a fallback to the declared path set when the tree is clean
- `pr-diff.md` from the current working-tree diff for those paths, with a fallback to the latest commit touching them when no live diff exists

## Pass Conditions

The runner fails if timing budgets are exceeded:

- packet build: `<= 5s`
- MCP resume: `<= 10s`
- first usable response: `<= 45s`
- total handoff: `<= 60s`

The runner also requires the normal API and public MCP calls to succeed.

## Rubric

Set these env vars before running if scoring manually:

```bash
export RELAY_ACCEPTANCE_STYLE_MATCH=4
export RELAY_ACCEPTANCE_HEURISTIC_RELEVANCE=yes
export RELAY_ACCEPTANCE_MANUAL_CORRECTIONS=0
export RELAY_ACCEPTANCE_CONTINUATION_WITHOUT_SUMMARY=yes
export RELAY_ACCEPTANCE_PREFERRED_CONTINUATION=style-aware
```

Fields:

- `style_match`: `1` to `5`; `0` means unscored.
- `heuristic_relevance`: `yes` or `no` for the cited heuristic set.
- `manual_corrections`: number of user corrections needed after resume.
- `continuation_without_summary`: whether the consumer could continue without a manual chat summary.
- `preferred_continuation`: `style-aware`, `control`, or `unscored`.

## Interpretation

For V1, a pass means Relay can preserve and reuse one explicit user judgment as a style cue across a session/model boundary.

It does not yet prove broad implicit learning quality. That requires repeated usage-validation rows and later curator/provider evaluation.
