#!/usr/bin/env bash
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd -P)"
BASE_URL="${RELAY_BASE_URL:-http://127.0.0.1:8080}"
MCP_URL="${RELAY_MCP_URL:-${BASE_URL%/}/mcp}"
OUTPUT_ROOT="${RELAY_ACCEPTANCE_OUTPUT_ROOT:-.gstack/projects/relay-consumer-stability}"
BATCH_PREFIX="${RELAY_EVAL_CONSUMER_STABILITY_PREFIX:-consumer-stability-${GITHUB_RUN_ID:-local}-$(date -u +%Y%m%dT%H%M%SZ)}"
FIXTURES_FILE="${RELAY_EVAL_FIXTURES_FILE:-scripts/evals/fixtures/v1_usage_validation.json}"
MODEL="${RELAY_EVAL_JUDGE_MODEL:-opus}"
RUNS="${RELAY_EVAL_CONSUMER_STABILITY_RUNS:-3}"
FIXTURE_LIMIT="${RELAY_EVAL_CONSUMER_STABILITY_FIXTURE_LIMIT:-1}"
CODEX_CONSUMER_MODEL="${RELAY_EVAL_CODEX_CONSUMER_MODEL:-}"
API_PID=""

usage() {
  cat <<EOF
Usage:
  run_consumer_stability.sh

Starts relay-api against the configured Postgres database, runs repeated
consumer continuation stability checks, and writes the soft-gate summary.

Required environment:
  RELAY_DATABASE_URL
  RELAY_ADMIN_TOKEN
  RELAY_DATA_ENCRYPTION_KEY

Optional environment:
  RELAY_BASE_URL
  RELAY_MCP_URL
  RELAY_ADDR
  RELAY_ACCEPTANCE_OUTPUT_ROOT
  RELAY_EVAL_CONSUMER_STABILITY_PREFIX
  RELAY_EVAL_CONSUMER_STABILITY_RUNS
  RELAY_EVAL_CONSUMER_STABILITY_FIXTURE_LIMIT
  RELAY_EVAL_FIXTURES_FILE
  RELAY_EVAL_JUDGE_MODEL
  RELAY_EVAL_CODEX_CONSUMER_MODEL
  RELAY_EVAL_MIN_CONSUMER_RUNS
  RELAY_EVAL_MIN_CONSUMER_PACKET_ONLY_PASS_RATE
  RELAY_EVAL_MIN_CONSUMER_CODEX_STYLE_MATCH
  RELAY_EVAL_MIN_CONSUMER_CLAUDE_STYLE_MATCH
  RELAY_EVAL_MIN_CONSUMER_CODEX_CONTINUATION_READINESS
  RELAY_EVAL_MIN_CONSUMER_CLAUDE_CONTINUATION_READINESS
  CODEX_HOME
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

verify_codex_auth() {
  if codex login status >/dev/null 2>&1; then
    return 0
  fi
  echo "Codex auth is required. On the jump self-hosted runner, ensure CODEX_HOME points at the logged-in runner user's ~/.codex." >&2
  exit 1
}

cleanup() {
  if [[ -n "$API_PID" ]] && kill -0 "$API_PID" >/dev/null 2>&1; then
    kill "$API_PID" >/dev/null 2>&1 || true
    wait "$API_PID" >/dev/null 2>&1 || true
  fi
}

append_evidence_status_summary() {
  if [[ -z "${GITHUB_STEP_SUMMARY:-}" ]]; then
    return 0
  fi
  {
    echo
    echo "## Relay Evidence Status"
    echo
    ./scripts/evals/relay_evidence_status.py --root "$OUTPUT_ROOT"
  } >>"$GITHUB_STEP_SUMMARY"
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
  require_command codex

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
  export RELAY_EVAL_CONSUMER_STABILITY_PREFIX="$BATCH_PREFIX"
  export RELAY_EVAL_FIXTURES_FILE="$FIXTURES_FILE"
  export RELAY_EVAL_JUDGE_MODEL="$MODEL"
  export RELAY_EVAL_CONSUMER_STABILITY_RUNS="$RUNS"
  export RELAY_EVAL_CONSUMER_STABILITY_FIXTURE_LIMIT="$FIXTURE_LIMIT"
  export RELAY_EVAL_CODEX_CONSUMER_MODEL="$CODEX_CONSUMER_MODEL"

  verify_claude_auth
  verify_codex_auth
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

  (
    cd "$REPO_ROOT"
    ./scripts/evals/v1_consumer_continuation_stability.sh \
      --fixtures-file "$FIXTURES_FILE" \
      --fixture-limit "$FIXTURE_LIMIT" \
      --runs "$RUNS" \
      --base-url "$BASE_URL" \
      --mcp-url "$MCP_URL" \
      --model "$MODEL" \
      --batch-prefix "$BATCH_PREFIX" \
      --output-root "$OUTPUT_ROOT"
  )

  local stability_dir summary_json summary_md
  stability_dir="${OUTPUT_ROOT%/}/stability/${BATCH_PREFIX}"
  summary_json="${stability_dir}/consumer-stability-summary.json"
  summary_md="${stability_dir}/consumer-stability-summary.md"
  if [[ ! -f "$summary_json" ]]; then
    echo "missing consumer stability summary: $summary_json" >&2
    exit 1
  fi

  printf 'stability_dir=%s\nsummary_json=%s\nsummary_md=%s\napi_log=%s\n' \
    "$stability_dir" \
    "$summary_json" \
    "$summary_md" \
    "$api_log"

  if [[ -n "${GITHUB_STEP_SUMMARY:-}" && -f "$summary_md" ]]; then
    cat "$summary_md" >>"$GITHUB_STEP_SUMMARY"
  fi
  append_evidence_status_summary
}

main "$@"
