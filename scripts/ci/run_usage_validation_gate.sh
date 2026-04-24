#!/usr/bin/env bash
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd -P)"
BASE_URL="${RELAY_BASE_URL:-http://127.0.0.1:8080}"
MCP_URL="${RELAY_MCP_URL:-${BASE_URL%/}/mcp}"
OUTPUT_ROOT="${RELAY_ACCEPTANCE_OUTPUT_ROOT:-.gstack/projects/relay-ci}"
BATCH_ID="${RELAY_EVAL_BATCH_ID:-v1-usage-validation-ci-$(date -u +%Y%m%dT%H%M%SZ)}"
FIXTURES_FILE="${RELAY_EVAL_FIXTURES_FILE:-scripts/evals/fixtures/v1_usage_validation.json}"
MODEL="${RELAY_EVAL_JUDGE_MODEL:-opus}"
API_PID=""

usage() {
  cat <<EOF
Usage:
  run_usage_validation_gate.sh

Starts relay-api against the configured Postgres database, runs the repeated
usage-validation benchmark, and writes the batch summary to the configured
output root.

Required environment:
  RELAY_DATABASE_URL
  RELAY_ADMIN_TOKEN

Optional environment:
  RELAY_BASE_URL
  RELAY_MCP_URL
  RELAY_ADDR
  RELAY_ACCEPTANCE_OUTPUT_ROOT
  RELAY_EVAL_BATCH_ID
  RELAY_EVAL_FIXTURES_FILE
  RELAY_EVAL_JUDGE_MODEL
  CLAUDE_CODE_OAUTH_TOKEN
  ANTHROPIC_API_KEY
EOF
}

require_command() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "$1 is required" >&2
    exit 1
  fi
}

wait_for_healthz() {
  local url="$1"
  local attempt
  for attempt in $(seq 1 60); do
    if curl --fail --silent --show-error "${url%/}/healthz" >/dev/null; then
      return 0
    fi
    sleep 1
  done
  echo "relay-api did not become healthy at ${url%/}/healthz" >&2
  return 1
}

verify_claude_auth() {
  if [[ -n "${CLAUDE_CODE_OAUTH_TOKEN:-}" ]]; then
    return 0
  fi

  if [[ -n "${ANTHROPIC_API_KEY:-}" ]]; then
    return 0
  fi

  if claude auth status >/dev/null 2>&1; then
    return 0
  fi

  echo "Claude auth is required. Set CLAUDE_CODE_OAUTH_TOKEN, ANTHROPIC_API_KEY, or log in with claude auth login." >&2
  exit 1
}

cleanup() {
  if [[ -n "$API_PID" ]] && kill -0 "$API_PID" >/dev/null 2>&1; then
    kill "$API_PID" >/dev/null 2>&1 || true
    wait "$API_PID" >/dev/null 2>&1 || true
  fi
}

main() {
  if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
    usage
    exit 0
  fi

  require_command go
  require_command curl
  require_command jq
  require_command python3
  require_command claude

  if [[ -z "${RELAY_DATABASE_URL:-}" ]]; then
    echo "RELAY_DATABASE_URL is required" >&2
    exit 1
  fi
  if [[ -z "${RELAY_ADMIN_TOKEN:-}" ]]; then
    echo "RELAY_ADMIN_TOKEN is required" >&2
    exit 1
  fi

  export REPO_ROOT
  export RELAY_API_TOKEN="${RELAY_API_TOKEN:-$RELAY_ADMIN_TOKEN}"
  export RELAY_BASE_URL="$BASE_URL"
  export RELAY_MCP_URL="$MCP_URL"
  export RELAY_ACCEPTANCE_OUTPUT_ROOT="$OUTPUT_ROOT"
  export RELAY_EVAL_BATCH_ID="$BATCH_ID"
  export RELAY_EVAL_FIXTURES_FILE="$FIXTURES_FILE"
  export RELAY_EVAL_JUDGE_MODEL="$MODEL"

  verify_claude_auth
  trap cleanup EXIT

  mkdir -p "${OUTPUT_ROOT%/}"
  go run ./cmd/relay migrate

  local api_log
  api_log="${OUTPUT_ROOT%/}/relay-api.log"
  (
    cd "$REPO_ROOT"
    go run ./cmd/relay-api
  ) >"$api_log" 2>&1 &
  API_PID="$!"

  wait_for_healthz "$BASE_URL"

  local batch_dir summary_json summary_md run_status_json
  batch_dir="${OUTPUT_ROOT%/}/batches/${BATCH_ID}"
  summary_json="${batch_dir}/batch-summary.json"
  summary_md="${batch_dir}/batch-summary.md"
  run_status_json="${batch_dir}/run-status.json"

  local batch_status
  set +e
  (
    cd "$REPO_ROOT"
    ./scripts/evals/v1_usage_validation_batch.sh \
      --fixtures-file "$FIXTURES_FILE" \
      --base-url "$BASE_URL" \
      --mcp-url "$MCP_URL" \
      --model "$MODEL" \
      --batch-id "$BATCH_ID" \
      --output-root "$OUTPUT_ROOT"
  )
  batch_status="$?"
  set -e

  if [[ "$batch_status" -eq 75 ]]; then
    mkdir -p "$batch_dir"
    jq -n \
      --arg status "blocked_by_model_limit" \
      --arg evidence_kind "usage_validation_gate" \
      --arg batch_id "$BATCH_ID" \
      --arg judge_model "$MODEL" \
      --arg github_run_id "${GITHUB_RUN_ID:-}" \
      --arg github_run_attempt "${GITHUB_RUN_ATTEMPT:-}" \
      '{
        status: $status,
        evidence_kind: $evidence_kind,
        canonical_benchmark_evidence: false,
        batch_id: $batch_id,
        judge_model: $judge_model,
        github_run_id: $github_run_id,
        github_run_attempt: $github_run_attempt,
        reason: "provider usage limit reached before usage validation could complete"
      }' >"$run_status_json"
    printf 'batch_dir=%s\nrun_status_json=%s\napi_log=%s\n' \
      "$batch_dir" \
      "$run_status_json" \
      "$api_log"
    exit 75
  fi
  if [[ "$batch_status" -ne 0 ]]; then
    exit "$batch_status"
  fi

  if [[ ! -f "$summary_json" ]]; then
    echo "missing batch summary: $summary_json" >&2
    exit 1
  fi
  if [[ ! -f "$run_status_json" ]]; then
    echo "missing run status: $run_status_json" >&2
    exit 1
  fi

  printf 'batch_dir=%s\nsummary_json=%s\nsummary_md=%s\nrun_status_json=%s\napi_log=%s\n' \
    "$batch_dir" \
    "$summary_json" \
    "$summary_md" \
    "$run_status_json" \
    "$api_log"

  if [[ -n "${GITHUB_STEP_SUMMARY:-}" && -f "$summary_md" ]]; then
    cat "$summary_md" >>"$GITHUB_STEP_SUMMARY"
  fi
}

main "$@"
