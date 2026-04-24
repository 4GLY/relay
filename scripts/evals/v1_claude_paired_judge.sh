#!/usr/bin/env bash
set -euo pipefail

RESULT_FILE=""
MODEL="${RELAY_EVAL_JUDGE_MODEL:-opus}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd -P)"

source "${SCRIPT_DIR}/lib/claude_structured_output.sh"

usage() {
  cat <<EOF
Usage:
  v1_claude_paired_judge.sh --result-file PATH [--model MODEL]

Runs a blind A/B judge over a Relay V1 acceptance run using claude CLI.

Required input:
  result.json from scripts/acceptance/v1_canonical_handoff.sh

Outputs next to result.json:
  paired-comparison-prompt.md
  claude-judge.json
  paired-comparison.json

Environment:
  RELAY_EVAL_CLAUDE_STRUCTURED_MAX_ATTEMPTS controls structured-output retries. Default: 3.
EOF
}

parse_args() {
  while [[ $# -gt 0 ]]; do
    case "$1" in
      --result-file)
        RESULT_FILE="${2:?result file required}"
        shift 2
        ;;
      --model)
        MODEL="${2:?model required}"
        shift 2
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
}

require_command() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "$1 is required" >&2
    exit 1
  fi
}

main() {
  parse_args "$@"
  require_command claude
  require_command jq
  require_command python3

  local result_dir output_root prompt_file raw_response_file comparison_file ledger_file judge_schema
  result_dir="$(cd "$(dirname "$RESULT_FILE")" && pwd)"
  output_root="$(cd "${result_dir}/.." && pwd)"
  RESULT_FILE="$(cd "$(dirname "$RESULT_FILE")" && pwd)/$(basename "$RESULT_FILE")"
  prompt_file="${result_dir}/paired-comparison-prompt.md"
  raw_response_file="${result_dir}/claude-judge.json"
  comparison_file="${result_dir}/paired-comparison.json"
  ledger_file="${output_root}/usage-validation.jsonl"

  local style_packet_file control_packet_file
  style_packet_file="$(jq -r '.style_packet_file // empty' "$RESULT_FILE")"
  control_packet_file="$(jq -r '.control_packet_file // empty' "$RESULT_FILE")"
  if [[ -z "$style_packet_file" || -z "$control_packet_file" ]]; then
    echo "result file does not include style_packet_file/control_packet_file; rerun acceptance with the current runner" >&2
    exit 1
  fi
  [[ "$style_packet_file" = /* ]] || style_packet_file="$(cd "$(dirname "$RESULT_FILE")" && pwd)/$(basename "$style_packet_file")"
  [[ "$control_packet_file" = /* ]] || control_packet_file="$(cd "$(dirname "$RESULT_FILE")" && pwd)/$(basename "$control_packet_file")"

  local style_body control_body packet_a_body packet_b_body packet_a_kind packet_b_kind
  style_body="$(jq -r '.rendered_body // .body // ""' "$style_packet_file")"
  control_body="$(jq -r '.rendered_body // .body // ""' "$control_packet_file")"

  if (( $(date +%s) % 2 == 0 )); then
    packet_a_kind="style-aware"
    packet_b_kind="control"
    packet_a_body="$style_body"
    packet_b_body="$control_body"
  else
    packet_a_kind="control"
    packet_b_kind="style-aware"
    packet_a_body="$control_body"
    packet_b_body="$style_body"
  fi

  cat >"$prompt_file" <<EOF
# Relay V1 Blind Paired Comparison

You are judging two Relay handoff packets for a fresh AI coding agent.

The target behavior:
- Continue the same Relay V1 implementation without reading prior chat history.
- Preserve the user's decision style: explicit contracts over magic inference, API-first boundaries, public MCP stays packet-centric, human approval controls durable heuristics.
- Prefer the packet that better enables immediate continuation with fewer wrong assumptions.

Scoring guidance:
- preferred_packet: choose A or B.
- style_match_*: use 1-5, where 5 means the packet strongly preserves the target decision style.
- continuation_without_summary_*: true when the packet is sufficient without a manual chat summary.
- reason: short reason.
- risks: short list of concrete risks.

## Packet A

$(printf '%s' "$packet_a_body")

## Packet B

$(printf '%s' "$packet_b_body")
EOF

  judge_schema="$(jq -nc '{
    type: "object",
    properties: {
      preferred_packet: {type: "string", enum: ["A", "B"]},
      style_match_a: {type: "integer", minimum: 1, maximum: 5},
      style_match_b: {type: "integer", minimum: 1, maximum: 5},
      continuation_without_summary_a: {type: "boolean"},
      continuation_without_summary_b: {type: "boolean"},
      reason: {type: "string"},
      risks: {
        type: "array",
        items: {type: "string"}
      }
    },
    required: [
      "preferred_packet",
      "style_match_a",
      "style_match_b",
      "continuation_without_summary_a",
      "continuation_without_summary_b",
      "reason",
      "risks"
    ],
    additionalProperties: false
  }')"

  run_claude_structured_output "$MODEL" "$judge_schema" "$prompt_file" "$raw_response_file" "paired comparison judge"

  local judge_json preferred_packet preferred_continuation style_match
  judge_json="$(jq -c '.structured_output' "$raw_response_file")"
  preferred_packet="$(jq -r '.preferred_packet' <<<"$judge_json")"
  case "$preferred_packet" in
    A)
      preferred_continuation="$packet_a_kind"
      ;;
    B)
      preferred_continuation="$packet_b_kind"
      ;;
    *)
      preferred_continuation="unscored"
      ;;
  esac
  if [[ "$preferred_continuation" == "style-aware" ]]; then
    if [[ "$packet_a_kind" == "style-aware" ]]; then
      style_match="$(jq -r '.style_match_a' <<<"$judge_json")"
    else
      style_match="$(jq -r '.style_match_b' <<<"$judge_json")"
    fi
  else
    style_match=0
  fi

  jq -n \
    --arg result_file "$RESULT_FILE" \
    --arg prompt_file "$prompt_file" \
    --arg raw_response_file "$raw_response_file" \
    --arg model "$MODEL" \
    --arg packet_a_kind "$packet_a_kind" \
    --arg packet_b_kind "$packet_b_kind" \
    --arg preferred_continuation "$preferred_continuation" \
    --argjson style_match "$style_match" \
    --argjson judge "$judge_json" \
    '{
      result_file: $result_file,
      prompt_file: $prompt_file,
      raw_response_file: $raw_response_file,
      judge_model: $model,
      mapping: {
        A: $packet_a_kind,
        B: $packet_b_kind
      },
      preferred_continuation: $preferred_continuation,
      style_match: $style_match,
      judge: $judge
    }' >"$comparison_file"

  jq -nc \
    --arg recorded_at "$(python3 -c 'import datetime; print(datetime.datetime.now(datetime.UTC).isoformat())')" \
    --arg run_id "$(jq -r '.run_id' "$RESULT_FILE")" \
    --arg fixture_id "$(jq -r '.fixture_id' "$RESULT_FILE")" \
    --arg project "$(jq -r '.project' "$RESULT_FILE")" \
    --arg result_file "$RESULT_FILE" \
    --arg comparison_file "$comparison_file" \
    --arg judge_model "$MODEL" \
    --arg preferred_continuation "$preferred_continuation" \
    --argjson style_match "$style_match" \
    --argjson budget_pass "$(jq '.budget.pass' "$RESULT_FILE")" \
    '{
      recorded_at: $recorded_at,
      event: "paired-comparison",
      run_id: $run_id,
      fixture_id: $fixture_id,
      project: $project,
      result_file: $result_file,
      comparison_file: $comparison_file,
      judge_model: $judge_model,
      preferred_continuation: $preferred_continuation,
      style_match: $style_match,
      budget_pass: $budget_pass
    }' >>"$ledger_file"
}

main "$@"
