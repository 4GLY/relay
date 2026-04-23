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
./scripts/evals/v1_copilot_paired_judge.sh \
  --result-file .gstack/projects/relay/<run_id>/result.json \
  --model claude-opus-4.7
```

The judge script writes:

- `.gstack/projects/relay/<run_id>/paired-comparison-prompt.md`
- `.gstack/projects/relay/<run_id>/copilot-opus-judge.jsonl`
- `.gstack/projects/relay/<run_id>/copilot-opus-judge.md`
- `.gstack/projects/relay/<run_id>/paired-comparison.json`

The judge is intentionally blind to which packet is style-aware. It maps packet A/B back to `style-aware` or `control` only after the model returns its preference.

## Repeated Usage Validation

For the next-stage benchmark, run the fixture batch instead of a single canonical seed:

```bash
set -a; source .env; set +a
./scripts/evals/v1_usage_validation_batch.sh \
  --fixtures-file scripts/evals/fixtures/v1_usage_validation.json \
  --base-url "${RELAY_BASE_URL:-https://relay.4gly.dev}" \
  --model claude-opus-4.7
```

The batch runner:

- reuses the same acceptance contract for multiple scenarios
- attaches richer evidence pointers including code paths, changed-files manifests, and PR-diff summaries
- runs the blind paired judge after each fixture
- writes `batch-runs.jsonl` plus `batch-summary.json` and `batch-summary.md`
- evaluates a release gate from style-aware win rate, average `style_match`, and budget-pass rate

Outputs land under:

- `.gstack/projects/relay/batches/<batch_id>/fixtures.json`
- `.gstack/projects/relay/batches/<batch_id>/batch-runs.jsonl`
- `.gstack/projects/relay/batches/<batch_id>/batch-summary.json`
- `.gstack/projects/relay/batches/<batch_id>/batch-summary.md`

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
