#!/usr/bin/env bash
set -euo pipefail

RESULT_FILE=""
BASE_URL="${RELAY_BASE_URL:-https://relay.4gly.dev}"
MCP_URL="${RELAY_MCP_URL:-${BASE_URL%/}/mcp}"
CLIENT_TOKEN="${RELAY_CLIENT_TOKEN:-${RELAY_MCP_TOKEN:-}}"
MODEL="${RELAY_EVAL_JUDGE_MODEL:-opus}"
WORKFLOW="${RELAY_EVAL_PACKET_WORKFLOW:-}"
ARTIFACT_TYPE="${RELAY_EVAL_PACKET_ARTIFACT_TYPE:-}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd -P)"

source "${SCRIPT_DIR}/lib/claude_structured_output.sh"

usage() {
  cat <<EOF
Usage:
  v1_retrieval_baseline_judge.sh --result-file PATH [options]

Builds and judges a blind retrieval-aware vs ranking-only packet comparison
for an existing Relay acceptance run.

Options:
  --result-file PATH   result.json from v1_canonical_handoff.sh
  --base-url URL       Relay API base URL. Default: ${BASE_URL}
  --mcp-url URL        Relay MCP URL. Default: \$base_url/mcp
  --client-token TOKEN Issued client token for public MCP calls
  --model MODEL        Judge model for claude CLI. Default: ${MODEL}
  --workflow NAME      Optional workflow selector to reuse while building packets
  --artifact-type TYPE Optional artifact_type selector to reuse while building packets
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
      --base-url)
        BASE_URL="${2:?base URL required}"
        shift 2
        ;;
      --mcp-url)
        MCP_URL="${2:?MCP URL required}"
        shift 2
        ;;
      --client-token)
        CLIENT_TOKEN="${2:?client token required}"
        shift 2
        ;;
      --model)
        MODEL="${2:?model required}"
        shift 2
        ;;
      --workflow)
        WORKFLOW="${2:?workflow required}"
        shift 2
        ;;
      --artifact-type)
        ARTIFACT_TYPE="${2:?artifact type required}"
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
  if [[ "${MCP_URL}" == "${BASE_URL}" ]]; then
    MCP_URL="${BASE_URL%/}/mcp"
  fi
}

require_command() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "$1 is required" >&2
    exit 1
  fi
}

curl_json() {
  local token="$1"
  local method="$2"
  local url="$3"
  local body="${4:-}"
  local -a cmd=(curl --fail-with-body --silent --show-error -X "$method" "$url" -H "Accept: application/json, text/event-stream")
  if [[ -n "$token" ]]; then
    cmd+=(-H "Authorization: Bearer ${token}")
  fi
  if [[ -n "$body" ]]; then
    cmd+=(-H "Content-Type: application/json" --data "$body")
  fi
  "${cmd[@]}"
}

mcp_call() {
  local tool_name="$1"
  local args_json="$2"
  local request
  request="$(jq -nc --arg name "$tool_name" --argjson args "$args_json" '{
    jsonrpc: "2.0",
    id: 1,
    method: "tools/call",
    params: {name: $name, arguments: $args}
  }')"
  curl_json "$CLIENT_TOKEN" POST "$MCP_URL" "$request"
}

structured_content() {
  local response="$1"
  if jq -e '.error? // empty' >/dev/null <<<"$response"; then
    jq '.error' <<<"$response" >&2
    exit 1
  fi
  jq -c '.result.structuredContent' <<<"$response"
}

main() {
  parse_args "$@"
  require_command claude
  require_command jq
  require_command python3

  if [[ -z "$CLIENT_TOKEN" ]]; then
    echo "RELAY_CLIENT_TOKEN or --client-token is required" >&2
    exit 1
  fi

  local result_dir output_root style_packet_file project task_summary fixture_id run_id
  result_dir="$(cd "$(dirname "$RESULT_FILE")" && pwd)"
  output_root="$(cd "${result_dir}/.." && pwd)"
  RESULT_FILE="$(cd "$(dirname "$RESULT_FILE")" && pwd)/$(basename "$RESULT_FILE")"
  style_packet_file="$(jq -r '.style_packet_file // empty' "$RESULT_FILE")"
  if [[ -z "$style_packet_file" ]]; then
    echo "result file does not include style_packet_file" >&2
    exit 1
  fi
  [[ "$style_packet_file" = /* ]] || style_packet_file="${result_dir}/$(basename "$style_packet_file")"
  project="$(jq -r '.project // empty' "$RESULT_FILE")"
  task_summary="$(jq -r '.task_summary // empty' "$style_packet_file")"
  fixture_id="$(jq -r '.fixture_id // "retrieval-baseline"' "$RESULT_FILE")"
  run_id="$(jq -r '.run_id // "retrieval-baseline"' "$RESULT_FILE")"
  if [[ -z "$project" || -z "$task_summary" ]]; then
    echo "result/style packet is missing project or task_summary" >&2
    exit 1
  fi

  local retrieval_packet_file ranking_packet_file prompt_file raw_response_file comparison_file ledger_file judge_schema
  retrieval_packet_file="${result_dir}/retrieval-aware-packet.json"
  ranking_packet_file="${result_dir}/ranking-only-packet.json"
  prompt_file="${result_dir}/retrieval-baseline-prompt.md"
  raw_response_file="${result_dir}/claude-retrieval-judge.json"
  comparison_file="${result_dir}/retrieval-baseline-comparison.json"
  ledger_file="${output_root}/usage-validation.jsonl"

  local retrieval_args ranking_args retrieval_response ranking_response retrieval_packet ranking_packet
  retrieval_args="$(jq -nc \
    --arg project "$project" \
    --arg fixture "$fixture_id" \
    --arg task_summary "$task_summary" \
    --arg workflow "$WORKFLOW" \
    --arg artifact_type "$ARTIFACT_TYPE" \
    '{
      project: $project,
      type: "resume",
      target: "codex",
      task_summary: $task_summary,
      persist_snapshot: true,
      idempotency_key: ($fixture + "-retrieval-aware-packet")
    }
    | if $workflow == "" then . else . + {workflow: $workflow} end
    | if $artifact_type == "" then . else . + {artifact_type: $artifact_type} end')"
  ranking_args="$(jq -nc \
    --arg project "$project" \
    --arg fixture "$fixture_id" \
    --arg task_summary "$task_summary" \
    --arg workflow "$WORKFLOW" \
    --arg artifact_type "$ARTIFACT_TYPE" \
    '{
      project: $project,
      type: "resume",
      target: "codex",
      task_summary: $task_summary,
      disable_retrieval: true,
      persist_snapshot: true,
      idempotency_key: ($fixture + "-ranking-only-packet")
    }
    | if $workflow == "" then . else . + {workflow: $workflow} end
    | if $artifact_type == "" then . else . + {artifact_type: $artifact_type} end')"

  retrieval_response="$(mcp_call relay_build_packet "$retrieval_args")"
  ranking_response="$(mcp_call relay_build_packet "$ranking_args")"
  retrieval_packet="$(structured_content "$retrieval_response")"
  ranking_packet="$(structured_content "$ranking_response")"
  jq . <<<"$retrieval_packet" >"$retrieval_packet_file"
  jq . <<<"$ranking_packet" >"$ranking_packet_file"

  local retrieval_body ranking_body packet_a_body packet_b_body packet_a_kind packet_b_kind
  retrieval_body="$(jq -r '.rendered_body // .body // ""' "$retrieval_packet_file")"
  ranking_body="$(jq -r '.rendered_body // .body // ""' "$ranking_packet_file")"

  if (( $(date +%s) % 2 == 0 )); then
    packet_a_kind="retrieval-aware"
    packet_b_kind="ranking-only"
    packet_a_body="$retrieval_body"
    packet_b_body="$ranking_body"
  else
    packet_a_kind="ranking-only"
    packet_b_kind="retrieval-aware"
    packet_a_body="$ranking_body"
    packet_b_body="$retrieval_body"
  fi

  cat >"$prompt_file" <<EOF
# Relay Retrieval Blind Paired Comparison

You are judging two Relay handoff packets for a fresh AI coding agent.

The target behavior:
- Continue the same Relay implementation without reading prior chat history.
- Prefer the packet that surfaces the most relevant evidence for the current task.
- Prefer the packet that lets the next agent continue with fewer wrong assumptions.

Scoring guidance:
- preferred_packet: choose A or B.
- continuation_readiness_*: use 1-5 for how ready the next agent is to continue.
- evidence_relevance_*: use 1-5 for how relevant the surfaced evidence is.
- reason: short reason.
- risks: short list of concrete risks.

Current task summary:
- ${task_summary}

## Packet A

$(printf '%s' "$packet_a_body")

## Packet B

$(printf '%s' "$packet_b_body")
EOF

  judge_schema="$(jq -nc '{
    type: "object",
    properties: {
      preferred_packet: {type: "string", enum: ["A", "B"]},
      continuation_readiness_a: {type: "integer", minimum: 1, maximum: 5},
      continuation_readiness_b: {type: "integer", minimum: 1, maximum: 5},
      evidence_relevance_a: {type: "integer", minimum: 1, maximum: 5},
      evidence_relevance_b: {type: "integer", minimum: 1, maximum: 5},
      reason: {type: "string"},
      risks: {
        type: "array",
        items: {type: "string"}
      }
    },
    required: [
      "preferred_packet",
      "continuation_readiness_a",
      "continuation_readiness_b",
      "evidence_relevance_a",
      "evidence_relevance_b",
      "reason",
      "risks"
    ],
    additionalProperties: false
  }')"

  run_claude_structured_output "$MODEL" "$judge_schema" "$prompt_file" "$raw_response_file" "retrieval baseline judge"

  local judge_json preferred_packet preferred_variant continuation_readiness evidence_relevance
  judge_json="$(jq -c '.structured_output' "$raw_response_file")"
  preferred_packet="$(jq -r '.preferred_packet' <<<"$judge_json")"
  case "$preferred_packet" in
    A)
      preferred_variant="$packet_a_kind"
      ;;
    B)
      preferred_variant="$packet_b_kind"
      ;;
    *)
      preferred_variant="unscored"
      ;;
  esac
  if [[ "$preferred_packet" == "A" ]]; then
    continuation_readiness="$(jq -r '.continuation_readiness_a' <<<"$judge_json")"
    evidence_relevance="$(jq -r '.evidence_relevance_a' <<<"$judge_json")"
  elif [[ "$preferred_packet" == "B" ]]; then
    continuation_readiness="$(jq -r '.continuation_readiness_b' <<<"$judge_json")"
    evidence_relevance="$(jq -r '.evidence_relevance_b' <<<"$judge_json")"
  else
    continuation_readiness=0
    evidence_relevance=0
  fi

  jq -n \
    --arg result_file "$RESULT_FILE" \
    --arg retrieval_packet_file "$retrieval_packet_file" \
    --arg ranking_packet_file "$ranking_packet_file" \
    --arg prompt_file "$prompt_file" \
    --arg raw_response_file "$raw_response_file" \
    --arg model "$MODEL" \
    --arg packet_a_kind "$packet_a_kind" \
    --arg packet_b_kind "$packet_b_kind" \
    --arg preferred_variant "$preferred_variant" \
    --argjson continuation_readiness "$continuation_readiness" \
    --argjson evidence_relevance "$evidence_relevance" \
    --argjson judge "$judge_json" \
    '{
      result_file: $result_file,
      retrieval_packet_file: $retrieval_packet_file,
      ranking_packet_file: $ranking_packet_file,
      prompt_file: $prompt_file,
      raw_response_file: $raw_response_file,
      judge_model: $model,
      mapping: {
        A: $packet_a_kind,
        B: $packet_b_kind
      },
      preferred_variant: $preferred_variant,
      continuation_readiness: $continuation_readiness,
      evidence_relevance: $evidence_relevance,
      judge: $judge
    }' >"$comparison_file"

  jq -nc \
    --arg recorded_at "$(python3 -c 'import datetime; print(datetime.datetime.now(datetime.UTC).isoformat())')" \
    --arg run_id "$run_id" \
    --arg fixture_id "$fixture_id" \
    --arg project "$project" \
    --arg result_file "$RESULT_FILE" \
    --arg comparison_file "$comparison_file" \
    --arg judge_model "$MODEL" \
    --arg preferred_variant "$preferred_variant" \
    --argjson continuation_readiness "$continuation_readiness" \
    --argjson evidence_relevance "$evidence_relevance" \
    '{
      recorded_at: $recorded_at,
      event: "retrieval-baseline-comparison",
      run_id: $run_id,
      fixture_id: $fixture_id,
      project: $project,
      result_file: $result_file,
      comparison_file: $comparison_file,
      judge_model: $judge_model,
      preferred_variant: $preferred_variant,
      continuation_readiness: $continuation_readiness,
      evidence_relevance: $evidence_relevance
    }' >>"$ledger_file"

  printf 'retrieval_packet: %s\nranking_packet: %s\nprompt: %s\nraw_response: %s\ncomparison: %s\nledger: %s\n' \
    "$retrieval_packet_file" "$ranking_packet_file" "$prompt_file" "$raw_response_file" "$comparison_file" "$ledger_file"
}

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
  main "$@"
fi
