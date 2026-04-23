#!/usr/bin/env bash
set -euo pipefail

RESULT_FILE=""
MODEL="${RELAY_EVAL_JUDGE_MODEL:-claude-opus-4.7}"

usage() {
  cat <<EOF
Usage:
  v1_copilot_paired_judge.sh --result-file PATH [--model MODEL]

Runs a blind A/B judge over a Relay V1 acceptance run using copilot CLI.

Required input:
  result.json from scripts/acceptance/v1_canonical_handoff.sh

Outputs next to result.json:
  paired-comparison-prompt.md
  copilot-opus-judge.jsonl
  copilot-opus-judge.md
  paired-comparison.json
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

extract_judge_json() {
  python3 - "$1" <<'PY'
import json
import re
import sys
from pathlib import Path

text = Path(sys.argv[1]).read_text()
matches = re.findall(r"\{[\s\S]*\}", text)
for candidate in matches:
    try:
        print(json.dumps(json.loads(candidate), indent=2, sort_keys=True))
        raise SystemExit(0)
    except json.JSONDecodeError:
        continue
raise SystemExit("judge response did not contain parseable JSON")
PY
}

extract_copilot_message() {
  python3 - "$1" <<'PY'
import json
import sys
from pathlib import Path

for line in Path(sys.argv[1]).read_text().splitlines():
    try:
        event = json.loads(line)
    except json.JSONDecodeError:
        continue
    if event.get("type") == "assistant.message":
        content = event.get("data", {}).get("content", "")
        if content:
            print(content)
PY
}

main() {
  parse_args "$@"
  require_command copilot
  require_command jq
  require_command python3

  local result_dir output_root prompt_file raw_jsonl_file raw_file comparison_file ledger_file
  result_dir="$(cd "$(dirname "$RESULT_FILE")" && pwd)"
  output_root="$(cd "${result_dir}/.." && pwd)"
  RESULT_FILE="$(cd "$(dirname "$RESULT_FILE")" && pwd)/$(basename "$RESULT_FILE")"
  prompt_file="${result_dir}/paired-comparison-prompt.md"
  raw_jsonl_file="${result_dir}/copilot-opus-judge.jsonl"
  raw_file="${result_dir}/copilot-opus-judge.md"
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

Return only JSON with this exact shape:

\`\`\`json
{
  "preferred_packet": "A",
  "style_match_a": 1,
  "style_match_b": 1,
  "continuation_without_summary_a": true,
  "continuation_without_summary_b": true,
  "reason": "short reason",
  "risks": ["short risk"]
}
\`\`\`

Use a 1-5 score for style_match, where 5 means the packet strongly preserves the target decision style.

## Packet A

$(printf '%s' "$packet_a_body")

## Packet B

$(printf '%s' "$packet_b_body")
EOF

  copilot \
    --model "$MODEL" \
    --silent \
    --stream off \
    --no-custom-instructions \
    --disable-builtin-mcps \
    --output-format json \
    --allow-all-tools \
    -p "$(cat "$prompt_file")" >"$raw_jsonl_file"

  extract_copilot_message "$raw_jsonl_file" >"$raw_file"

  local judge_json preferred_packet preferred_continuation style_match
  judge_json="$(extract_judge_json "$raw_file")"
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
    --arg raw_jsonl_file "$raw_jsonl_file" \
    --arg raw_file "$raw_file" \
    --arg model "$MODEL" \
    --arg packet_a_kind "$packet_a_kind" \
    --arg packet_b_kind "$packet_b_kind" \
    --arg preferred_continuation "$preferred_continuation" \
    --argjson style_match "$style_match" \
    --argjson judge "$judge_json" \
    '{
      result_file: $result_file,
      prompt_file: $prompt_file,
      raw_jsonl_file: $raw_jsonl_file,
      raw_file: $raw_file,
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

  printf 'prompt: %s\nraw_jsonl: %s\nraw: %s\ncomparison: %s\nledger: %s\n' "$prompt_file" "$raw_jsonl_file" "$raw_file" "$comparison_file" "$ledger_file"
}

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
  main "$@"
fi
