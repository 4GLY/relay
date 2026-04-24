#!/usr/bin/env bash
set -euo pipefail

RESULT_FILE=""
CLAUDE_MODEL="${RELAY_EVAL_CLAUDE_CONSUMER_MODEL:-opus}"
CODEX_MODEL="${RELAY_EVAL_CODEX_CONSUMER_MODEL:-}"
RUN_CLAUDE=1
RUN_CODEX=1
RUN_JUDGE=1
REUSE_EXISTING=0
REPO_ROOT="$(git rev-parse --show-toplevel 2>/dev/null || pwd -P)"

usage() {
  cat <<EOF
Usage:
  v1_consumer_continuation.sh --result-file PATH [options]

Runs a packet-only continuation check against real consumer agents.

Inputs:
  result.json from scripts/acceptance/v1_canonical_handoff.sh

Options:
  --result-file PATH      Required acceptance result.json
  --claude-model MODEL    Claude consumer and judge model. Default: ${CLAUDE_MODEL}
  --codex-model MODEL     Optional Codex consumer model. Default: Codex CLI config
  --skip-claude           Do not run the Claude consumer
  --skip-codex            Do not run the Codex consumer
  --skip-judge            Do not run the Claude comparison judge
  --reuse-existing        Reuse existing consumer outputs when present
EOF
}

parse_args() {
  while [[ $# -gt 0 ]]; do
    case "$1" in
      --result-file)
        RESULT_FILE="${2:?result file required}"
        shift 2
        ;;
      --claude-model)
        CLAUDE_MODEL="${2:?Claude model required}"
        shift 2
        ;;
      --codex-model)
        CODEX_MODEL="${2:?Codex model required}"
        shift 2
        ;;
      --skip-claude)
        RUN_CLAUDE=0
        shift
        ;;
      --skip-codex)
        RUN_CODEX=0
        shift
        ;;
      --skip-judge)
        RUN_JUDGE=0
        shift
        ;;
      --reuse-existing)
        REUSE_EXISTING=1
        shift
        ;;
      -h|--help)
        usage
        exit 0
        ;;
      *)
        echo "Unknown argument: $1" >&2
        usage >&2
        exit 1
        ;;
    esac
  done

  if [[ -z "$RESULT_FILE" ]]; then
    echo "--result-file is required" >&2
    usage >&2
    exit 1
  fi
  if [[ "$RUN_CLAUDE" -eq 0 && "$RUN_CODEX" -eq 0 ]]; then
    echo "at least one consumer must run" >&2
    exit 1
  fi
}

require_command() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "$1 is required" >&2
    exit 1
  fi
}

write_consumer_schema() {
  local path="$1"
  jq -nc '{
    type: "object",
    properties: {
      consumer_model: {type: "string"},
      packet_only: {type: "boolean"},
      continuation_summary: {type: "string"},
      next_actions: {type: "array", items: {type: "string"}, minItems: 3},
      assumptions: {type: "array", items: {type: "string"}},
      risks: {type: "array", items: {type: "string"}},
      style_signals_used: {type: "array", items: {type: "string"}},
      packet_sufficiency_score: {type: "integer", minimum: 1, maximum: 5}
    },
    required: [
      "consumer_model",
      "packet_only",
      "continuation_summary",
      "next_actions",
      "assumptions",
      "risks",
      "style_signals_used",
      "packet_sufficiency_score"
    ],
    additionalProperties: false
  }' >"$path"
}

write_judge_schema() {
  local path="$1"
  jq -nc '{
    type: "object",
    properties: {
      codex_packet_only_continuation: {type: "boolean"},
      claude_packet_only_continuation: {type: "boolean"},
      codex_style_match: {type: "integer", minimum: 1, maximum: 5},
      claude_style_match: {type: "integer", minimum: 1, maximum: 5},
      codex_continuation_readiness: {type: "integer", minimum: 1, maximum: 5},
      claude_continuation_readiness: {type: "integer", minimum: 1, maximum: 5},
      preferred_consumer: {type: "string", enum: ["codex", "claude", "tie"]},
      reason: {type: "string"},
      risks: {type: "array", items: {type: "string"}}
    },
    required: [
      "codex_packet_only_continuation",
      "claude_packet_only_continuation",
      "codex_style_match",
      "claude_style_match",
      "codex_continuation_readiness",
      "claude_continuation_readiness",
      "preferred_consumer",
      "reason",
      "risks"
    ],
    additionalProperties: false
  }' >"$path"
}

run_claude_consumer() {
  local prompt_file="$1"
  local schema_file="$2"
  local raw_response_file="$3"
  local output_file="$4"

  claude \
    -p \
    --output-format json \
    --model "$CLAUDE_MODEL" \
    --tools "" \
    --json-schema "$(cat "$schema_file")" \
    "$(cat "$prompt_file")" >"$raw_response_file"

  jq -e '.structured_output' "$raw_response_file" >"$output_file"
}

run_codex_consumer() {
  local prompt_file="$1"
  local schema_file="$2"
  local raw_response_file="$3"
  local log_file="$4"

  local -a cmd=(
    codex exec
    --cd "$REPO_ROOT"
    --sandbox read-only
    --ephemeral
    --output-schema "$schema_file"
    --output-last-message "$raw_response_file"
  )
  if [[ -n "$CODEX_MODEL" ]]; then
    cmd+=(--model "$CODEX_MODEL")
  fi
  cmd+=("$(cat "$prompt_file")")

  "${cmd[@]}" >"$log_file" 2>&1
  jq -e . "$raw_response_file" >/dev/null
}

main() {
  parse_args "$@"
  require_command jq
  require_command python3
  if [[ "$RUN_CLAUDE" -eq 1 || "$RUN_JUDGE" -eq 1 ]]; then
    require_command claude
  fi
  if [[ "$RUN_CODEX" -eq 1 ]]; then
    require_command codex
  fi

  local result_dir output_root style_packet_file prompt_file consumer_schema_file judge_schema_file
  result_dir="$(cd "$(dirname "$RESULT_FILE")" && pwd)"
  output_root="$(cd "${result_dir}/.." && pwd)"
  RESULT_FILE="$(cd "$(dirname "$RESULT_FILE")" && pwd)/$(basename "$RESULT_FILE")"
  style_packet_file="$(jq -r '.style_packet_file // empty' "$RESULT_FILE")"
  if [[ -z "$style_packet_file" ]]; then
    echo "result file does not include style_packet_file" >&2
    exit 1
  fi
  [[ "$style_packet_file" = /* ]] || style_packet_file="${result_dir}/$(basename "$style_packet_file")"
  if [[ ! -f "$style_packet_file" ]]; then
    echo "style packet not found: $style_packet_file" >&2
    exit 1
  fi

  prompt_file="${result_dir}/consumer-continuation-prompt.md"
  consumer_schema_file="${result_dir}/consumer-continuation.schema.json"
  judge_schema_file="${result_dir}/consumer-continuation-judge.schema.json"

  local packet_body project fixture_id run_id task_summary
  packet_body="$(jq -r '.rendered_body // .body // ""' "$style_packet_file")"
  project="$(jq -r '.project // empty' "$RESULT_FILE")"
  fixture_id="$(jq -r '.fixture_id // "consumer-continuation"' "$RESULT_FILE")"
  run_id="$(jq -r '.run_id // "consumer-continuation"' "$RESULT_FILE")"
  task_summary="$(jq -r '.task_summary // empty' "$style_packet_file")"
  if [[ -z "$packet_body" ]]; then
    echo "style packet does not include rendered_body/body" >&2
    exit 1
  fi

  write_consumer_schema "$consumer_schema_file"
  write_judge_schema "$judge_schema_file"

  cat >"$prompt_file" <<EOF
# Relay Packet-Only Continuation Eval

You are a fresh AI coding agent joining the same Relay project.

Use only the handoff packet below as prior context. Do not rely on chat history.
Do not inspect repository files for this eval. Your job is to state how you
would continue the work while preserving the user's decision style.

Target behavior:
- Continue the same project/session from the packet alone.
- Preserve explicit contracts over magic inference.
- Preserve API-first boundaries and packet-centric public MCP behavior.
- Preserve human approval for durable heuristic changes.
- Be concrete enough that another agent can execute the next step.

Return JSON matching the provided schema.

Project: ${project}
Task summary: ${task_summary}

## Handoff Packet

$(printf '%s' "$packet_body")
EOF

  local claude_raw_file claude_output_file codex_raw_file codex_log_file comparison_prompt_file comparison_raw_file comparison_file ledger_file
  claude_raw_file="${result_dir}/claude-consumer-continuation.raw.json"
  claude_output_file="${result_dir}/claude-consumer-continuation.json"
  codex_raw_file="${result_dir}/codex-consumer-continuation.json"
  codex_log_file="${result_dir}/codex-consumer-continuation.log"
  comparison_prompt_file="${result_dir}/consumer-continuation-comparison-prompt.md"
  comparison_raw_file="${result_dir}/claude-consumer-continuation-judge.raw.json"
  comparison_file="${result_dir}/consumer-continuation-comparison.json"
  ledger_file="${output_root}/usage-validation.jsonl"

  if [[ "$RUN_CLAUDE" -eq 1 ]]; then
    if [[ "$REUSE_EXISTING" -eq 1 && -f "$claude_output_file" ]]; then
      jq -e . "$claude_output_file" >/dev/null
    else
      run_claude_consumer "$prompt_file" "$consumer_schema_file" "$claude_raw_file" "$claude_output_file"
    fi
  fi
  if [[ "$RUN_CODEX" -eq 1 ]]; then
    if [[ "$REUSE_EXISTING" -eq 1 && -f "$codex_raw_file" ]]; then
      jq -e . "$codex_raw_file" >/dev/null
    else
      run_codex_consumer "$prompt_file" "$consumer_schema_file" "$codex_raw_file" "$codex_log_file"
    fi
  fi

  if [[ "$RUN_JUDGE" -eq 1 ]]; then
    if [[ ! -f "$claude_output_file" || ! -f "$codex_raw_file" ]]; then
      echo "judge requires both Claude and Codex consumer outputs; rerun without skips or pass --skip-judge" >&2
      exit 1
    fi

    cat >"$comparison_prompt_file" <<EOF
# Relay Consumer Continuation Comparison

You are judging whether real consumer agents can continue from a Relay handoff
packet without prior chat history.

The target behavior:
- packet-only continuation is true only if the agent's plan clearly uses the
  handoff packet rather than generic Relay assumptions.
- style_match is 1-5 for preserving the user's decision style:
  explicit contracts, API-first boundaries, packet-centric public MCP, and
  human approval for durable heuristics.
- continuation_readiness is 1-5 for whether the output can guide the next
  implementation step without extra user recap.

## Original Handoff Packet

$(printf '%s' "$packet_body")

## Codex Consumer Output

$(jq . "$codex_raw_file")

## Claude Consumer Output

$(jq . "$claude_output_file")
EOF

    claude \
      -p \
      --output-format json \
      --model "$CLAUDE_MODEL" \
      --tools "" \
      --json-schema "$(cat "$judge_schema_file")" \
      "$(cat "$comparison_prompt_file")" >"$comparison_raw_file"

    jq -e '.structured_output' "$comparison_raw_file" >"$comparison_file"

    jq -nc \
      --arg recorded_at "$(python3 -c 'import datetime; print(datetime.datetime.now(datetime.UTC).isoformat())')" \
      --arg run_id "$run_id" \
      --arg fixture_id "$fixture_id" \
      --arg project "$project" \
      --arg result_file "$RESULT_FILE" \
      --arg prompt_file "$prompt_file" \
      --arg comparison_file "$comparison_file" \
      --arg claude_model "$CLAUDE_MODEL" \
      --arg codex_model "${CODEX_MODEL:-codex-config-default}" \
      --argjson comparison "$(jq -c . "$comparison_file")" \
      '{
        recorded_at: $recorded_at,
        event: "consumer-continuation-comparison",
        run_id: $run_id,
        fixture_id: $fixture_id,
        project: $project,
        result_file: $result_file,
        prompt_file: $prompt_file,
        comparison_file: $comparison_file,
        claude_model: $claude_model,
        codex_model: $codex_model,
        preferred_consumer: $comparison.preferred_consumer,
        codex_style_match: $comparison.codex_style_match,
        claude_style_match: $comparison.claude_style_match,
        codex_continuation_readiness: $comparison.codex_continuation_readiness,
        claude_continuation_readiness: $comparison.claude_continuation_readiness
      }' >>"$ledger_file"
  fi

  jq -n \
    --arg result_file "$RESULT_FILE" \
    --arg prompt_file "$prompt_file" \
    --arg claude_output_file "$claude_output_file" \
    --arg codex_output_file "$codex_raw_file" \
    --arg comparison_file "$comparison_file" \
    '{
      result_file: $result_file,
      prompt_file: $prompt_file,
      claude_output_file: $claude_output_file,
      codex_output_file: $codex_output_file,
      comparison_file: $comparison_file
    }'
}

main "$@"
