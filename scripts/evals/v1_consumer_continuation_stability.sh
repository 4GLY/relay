#!/usr/bin/env bash
set -euo pipefail

BASE_URL="${RELAY_BASE_URL:-https://relay.4gly.dev}"
MCP_URL="${RELAY_MCP_URL:-${BASE_URL%/}/mcp}"
CLIENT_TOKEN="${RELAY_CLIENT_TOKEN:-${RELAY_MCP_TOKEN:-}}"
ADMIN_TOKEN="${RELAY_ADMIN_TOKEN:-${RELAY_API_TOKEN:-}}"
FIXTURES_FILE="${RELAY_EVAL_FIXTURES_FILE:-scripts/evals/fixtures/v1_usage_validation.json}"
OUTPUT_ROOT="${RELAY_ACCEPTANCE_OUTPUT_ROOT:-.gstack/projects/relay}"
MODEL="${RELAY_EVAL_JUDGE_MODEL:-opus}"
CODEX_CONSUMER_MODEL="${RELAY_EVAL_CODEX_CONSUMER_MODEL:-}"
RUNS="${RELAY_EVAL_CONSUMER_STABILITY_RUNS:-3}"
FIXTURE_LIMIT="${RELAY_EVAL_CONSUMER_STABILITY_FIXTURE_LIMIT:-1}"
BATCH_PREFIX="${RELAY_EVAL_CONSUMER_STABILITY_PREFIX:-consumer-continuation-stability-$(date -u +%Y%m%dT%H%M%SZ)}"

usage() {
  cat <<EOF
Usage:
  v1_consumer_continuation_stability.sh [options]

Runs the usage-validation batch with --consumer-continuation repeatedly, then
aggregates consumer continuation score stability.

Options:
  --fixtures-file PATH       Fixture JSON file. Default: ${FIXTURES_FILE}
  --fixture-limit N          Use the first N fixtures. Use 0 for all. Default: ${FIXTURE_LIMIT}
  --runs N                   Number of repeated batches. Default: ${RUNS}
  --base-url URL             Relay API base URL. Default: ${BASE_URL}
  --mcp-url URL              Relay MCP URL. Default: \$base_url/mcp
  --client-token TOKEN       Issued client token for normal /v1 and /mcp calls
  --admin-token TOKEN        Bootstrap admin token for issuing temporary keys
  --model MODEL              Claude consumer and judge model. Default: ${MODEL}
  --codex-consumer-model M   Optional Codex consumer model. Default: Codex CLI config
  --batch-prefix PREFIX      Prefix for generated batch IDs. Default: ${BATCH_PREFIX}
  --output-root DIR          Output root. Default: ${OUTPUT_ROOT}
EOF
}

parse_args() {
  while [[ $# -gt 0 ]]; do
    case "$1" in
      --fixtures-file)
        FIXTURES_FILE="${2:?fixtures file required}"
        shift 2
        ;;
      --fixture-limit)
        FIXTURE_LIMIT="${2:?fixture limit required}"
        shift 2
        ;;
      --runs)
        RUNS="${2:?runs required}"
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
      --admin-token)
        ADMIN_TOKEN="${2:?admin token required}"
        shift 2
        ;;
      --model)
        MODEL="${2:?model required}"
        shift 2
        ;;
      --codex-consumer-model)
        CODEX_CONSUMER_MODEL="${2:?Codex consumer model required}"
        shift 2
        ;;
      --batch-prefix)
        BATCH_PREFIX="${2:?batch prefix required}"
        shift 2
        ;;
      --output-root)
        OUTPUT_ROOT="${2:?output root required}"
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

validate_positive_integer() {
  local name="$1"
  local value="$2"
  if ! [[ "$value" =~ ^[0-9]+$ ]]; then
    echo "${name} must be a non-negative integer: ${value}" >&2
    exit 1
  fi
}

main() {
  parse_args "$@"
  require_command jq
  require_command python3
  require_command claude
  require_command codex

  validate_positive_integer "--runs" "$RUNS"
  validate_positive_integer "--fixture-limit" "$FIXTURE_LIMIT"
  if (( RUNS == 0 )); then
    echo "--runs must be greater than 0" >&2
    exit 1
  fi
  if [[ ! -f "$FIXTURES_FILE" ]]; then
    echo "fixtures file not found: $FIXTURES_FILE" >&2
    exit 1
  fi

  local stability_dir fixture_subset summary_json summary_md
  stability_dir="${OUTPUT_ROOT%/}/stability/${BATCH_PREFIX}"
  fixture_subset="${stability_dir}/fixtures.json"
  summary_json="${stability_dir}/consumer-stability-summary.json"
  summary_md="${stability_dir}/consumer-stability-summary.md"
  mkdir -p "$stability_dir"

  if (( FIXTURE_LIMIT == 0 )); then
    cp "$FIXTURES_FILE" "$fixture_subset"
  else
    jq --argjson limit "$FIXTURE_LIMIT" '.[0:$limit]' "$FIXTURES_FILE" >"$fixture_subset"
  fi
  if [[ "$(jq 'length' "$fixture_subset")" -eq 0 ]]; then
    echo "fixture subset is empty" >&2
    exit 1
  fi

  local -a batch_summary_args=()
  local index batch_id batch_summary
  for (( index = 1; index <= RUNS; index++ )); do
    batch_id="${BATCH_PREFIX}-run-${index}"
    echo "running consumer continuation stability batch ${index}/${RUNS}: ${batch_id}"
    local -a batch_cmd=(
      ./scripts/evals/v1_usage_validation_batch.sh
      --fixtures-file "$fixture_subset"
      --base-url "$BASE_URL"
      --mcp-url "$MCP_URL"
      --model "$MODEL"
      --batch-id "$batch_id"
      --output-root "$OUTPUT_ROOT"
      --consumer-continuation
    )
    if [[ -n "$CLIENT_TOKEN" ]]; then
      batch_cmd+=(--client-token "$CLIENT_TOKEN")
    fi
    if [[ -n "$ADMIN_TOKEN" ]]; then
      batch_cmd+=(--admin-token "$ADMIN_TOKEN")
    fi
    if [[ -n "$CODEX_CONSUMER_MODEL" ]]; then
      batch_cmd+=(--codex-consumer-model "$CODEX_CONSUMER_MODEL")
    fi
    "${batch_cmd[@]}"

    batch_summary="${OUTPUT_ROOT%/}/batches/${batch_id}/batch-summary.json"
    if [[ ! -f "$batch_summary" ]]; then
      echo "missing batch summary: $batch_summary" >&2
      exit 1
    fi
    batch_summary_args+=(--batch-summary "$batch_summary")
  done

  ./scripts/evals/v1_consumer_stability_report.py \
    "${batch_summary_args[@]}" \
    --output-json "$summary_json" \
    --output-md "$summary_md"

  printf 'stability_dir: %s\nsummary_json: %s\nsummary_md: %s\n' \
    "$stability_dir" \
    "$summary_json" \
    "$summary_md"
}

main "$@"
