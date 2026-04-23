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
- `.gstack/projects/relay/usage-validation.jsonl`

## Flow

1. `POST /v1/capture` creates or updates the acceptance project.
2. `POST /v1/judgment-traces` stores the seed judgment.
3. `POST /v1/heuristic-proposals` creates a pending proposal from the trace.
4. `POST /v1/heuristic-proposals/review` approves it through the admin path.
5. `relay_build_packet` over public MCP builds a style-aware packet with `persist_snapshot: true`.
6. `relay_build_packet` over public MCP builds a control packet with `disable_style_cues: true`.
7. The runner records timings, snapshot IDs, reused heuristic IDs, and rubric fields.

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
