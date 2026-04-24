#!/usr/bin/env bash
set -euo pipefail

BASE_URL="${RELAY_BASE_URL:-https://relay.4gly.dev}"
MCP_URL="${RELAY_MCP_URL:-${BASE_URL%/}/mcp}"
CLIENT_TOKEN="${RELAY_CLIENT_TOKEN:-${RELAY_MCP_TOKEN:-}}"
ADMIN_TOKEN="${RELAY_ADMIN_TOKEN:-${RELAY_API_TOKEN:-}}"
FIXTURES_FILE="${RELAY_EVAL_FIXTURES_FILE:-scripts/evals/fixtures/v1_usage_validation.json}"
OUTPUT_ROOT="${RELAY_ACCEPTANCE_OUTPUT_ROOT:-.gstack/projects/relay}"
MODEL="${RELAY_EVAL_JUDGE_MODEL:-claude-opus-4.7}"
BATCH_ID="${RELAY_EVAL_BATCH_ID:-v1-usage-validation-$(date -u +%Y%m%dT%H%M%SZ)}"
MIN_STYLE_AWARE_WIN_RATE="${RELAY_EVAL_MIN_STYLE_AWARE_WIN_RATE:-0.8}"
MIN_AVG_STYLE_MATCH="${RELAY_EVAL_MIN_AVG_STYLE_MATCH:-4.0}"
MIN_BUDGET_PASS_RATE="${RELAY_EVAL_MIN_BUDGET_PASS_RATE:-1.0}"
KEEP_ISSUED_KEY=0
TEMP_KEY_ID=""
TEMP_KEY_TOKEN=""
REPO_ROOT="$(git rev-parse --show-toplevel 2>/dev/null || pwd -P)"

usage() {
  cat <<EOF
Usage:
  v1_usage_validation_batch.sh [options]

Runs a repeated Relay V1 usage-validation benchmark:
  fixture -> acceptance -> paired judge -> aggregate report

Options:
  --fixtures-file PATH  Fixture JSON file. Default: ${FIXTURES_FILE}
  --base-url URL        Relay API base URL. Default: ${BASE_URL}
  --mcp-url URL         Relay MCP URL. Default: \$base_url/mcp
  --client-token TOKEN  Issued client token for normal /v1 and /mcp calls
  --admin-token TOKEN   Bootstrap admin token for issuing or revoking temp keys
  --model MODEL         Judge model for copilot CLI. Default: ${MODEL}
  --batch-id ID         Batch id for output grouping
  --output-root DIR     Output root. Default: ${OUTPUT_ROOT}
  --min-win-rate FLOAT  Minimum style-aware win rate gate. Default: ${MIN_STYLE_AWARE_WIN_RATE}
  --min-style-match N   Minimum average style_match gate. Default: ${MIN_AVG_STYLE_MATCH}
  --min-budget-rate F   Minimum budget-pass rate gate. Default: ${MIN_BUDGET_PASS_RATE}
  --keep-issued-key     Keep an issued temporary key instead of revoking it
EOF
}

parse_args() {
  while [[ $# -gt 0 ]]; do
    case "$1" in
      --fixtures-file)
        FIXTURES_FILE="${2:?fixtures file required}"
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
      --batch-id)
        BATCH_ID="${2:?batch id required}"
        shift 2
        ;;
      --output-root)
        OUTPUT_ROOT="${2:?output root required}"
        shift 2
        ;;
      --min-win-rate)
        MIN_STYLE_AWARE_WIN_RATE="${2:?minimum win rate required}"
        shift 2
        ;;
      --min-style-match)
        MIN_AVG_STYLE_MATCH="${2:?minimum style match required}"
        shift 2
        ;;
      --min-budget-rate)
        MIN_BUDGET_PASS_RATE="${2:?minimum budget pass rate required}"
        shift 2
        ;;
      --keep-issued-key)
        KEEP_ISSUED_KEY=1
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

resolve_client_token() {
  if [[ -n "${CLIENT_TOKEN}" ]]; then
    return 0
  fi
  if [[ -n "${RELAY_CLIENT_TOKEN:-}" ]]; then
    CLIENT_TOKEN="${RELAY_CLIENT_TOKEN}"
    return 0
  fi
  if [[ -n "${RELAY_MCP_TOKEN:-}" ]]; then
    CLIENT_TOKEN="${RELAY_MCP_TOKEN}"
    return 0
  fi
  return 1
}

issue_temp_key() {
  if [[ -z "${ADMIN_TOKEN}" ]]; then
    echo "No client token found, and admin token is missing so the benchmark cannot issue a temporary key." >&2
    exit 1
  fi

  local name="usage-validation-$(date -u +%Y%m%dT%H%M%SZ)"
  local response
  response="$(curl_json "${ADMIN_TOKEN}" POST "${BASE_URL%/}/v1/api-keys/issue" "{\"name\":\"${name}\"}")"
  TEMP_KEY_ID="$(jq -r '.data.key_id' <<<"$response")"
  TEMP_KEY_TOKEN="$(jq -r '.data.token' <<<"$response")"
  CLIENT_TOKEN="${TEMP_KEY_TOKEN}"
  echo "issued temporary client key: ${TEMP_KEY_ID}"
}

cleanup() {
  if [[ -n "${TEMP_KEY_ID}" && "${KEEP_ISSUED_KEY}" -eq 0 && -n "${ADMIN_TOKEN}" ]]; then
    curl_json "${ADMIN_TOKEN}" POST "${BASE_URL%/}/v1/api-keys/revoke" "{\"key_id\":\"${TEMP_KEY_ID}\"}" >/dev/null || true
    echo "revoked temporary client key: ${TEMP_KEY_ID}"
  fi
}

json_string() {
  local query="$1"
  local payload="$2"
  jq -r "${query} // empty" <<<"$payload"
}

json_array() {
  local query="$1"
  local payload="$2"
  jq -c "${query} // []" <<<"$payload"
}

load_json_array_into_paths() {
  local json_payload="$1"
  local path
  GENERATED_PATHS=()
  while IFS= read -r path; do
    if [[ -n "$path" ]]; then
      GENERATED_PATHS+=("$path")
    fi
  done < <(jq -r '.[]' <<<"$json_payload")
}

current_diff_names() {
  local -a paths=("$@")
  if [[ ${#paths[@]} -eq 0 ]]; then
    return 0
  fi

  git -C "$REPO_ROOT" diff --name-only HEAD -- "${paths[@]}" 2>/dev/null || true
  git -C "$REPO_ROOT" ls-files --others --exclude-standard -- "${paths[@]}" 2>/dev/null || true
}

generate_changed_files_artifact() {
  local target_path="$1"
  shift
  local -a paths=("$@")
  local changed_names=""

  changed_names="$(current_diff_names "${paths[@]}" | sed '/^$/d' | sort -u)"
  if [[ -z "${changed_names}" ]]; then
    changed_names="$(printf '%s\n' "${paths[@]}" | sed '/^$/d' | sort -u)"
  fi

  printf '%s\n' "$changed_names" >"$target_path"
}

generate_pr_diff_artifact() {
  local target_path="$1"
  local fixture_slug="$2"
  local scenario_label="$3"
  local task_summary="$4"
  shift 4
  local -a paths=("$@")
  local diff_source="working-tree diff against HEAD"
  local diff_body=""

  if [[ ${#paths[@]} -gt 0 ]]; then
    diff_body="$(git -C "$REPO_ROOT" diff --no-color --stat --patch --unified=1 HEAD -- "${paths[@]}" 2>/dev/null || true)"
    if [[ -z "${diff_body//[$' \t\r\n']/}" ]]; then
      diff_source="latest commit touching selected paths"
      diff_body="$(git -C "$REPO_ROOT" log -1 --no-color --stat --patch --format=fuller -- "${paths[@]}" 2>/dev/null || true)"
    fi
  fi

  if [[ -z "${diff_body//[$' \t\r\n']/}" ]]; then
    diff_source="no matching git diff or history"
    diff_body="No git diff or commit history was found for the selected evidence paths at generation time."
  fi

  {
    printf '# Generated PR Diff\n\n'
    printf -- '- fixture: `%s`\n' "$fixture_slug"
    printf -- '- scenario: `%s`\n' "$scenario_label"
    printf -- '- source: %s\n' "$diff_source"
    printf -- '- repo_root: `%s`\n' "$REPO_ROOT"
    printf -- '- task_summary: %s\n' "$task_summary"
    printf -- '- evidence_paths:\n'
    for path in "${paths[@]}"; do
      printf '  - `%s`\n' "$path"
    done
    printf '\n```diff\n'
    printf '%s\n' "$diff_body" | sed -n '1,160p'
    printf '```\n'
  } >"$target_path"
}

build_fixture_extra_artifacts() {
  local fixture_json="$1"
  local batch_dir="$2"
  local fixture_slug="$3"
  local scenario_label="$4"
  local task_summary="$5"
  local base_extra_artifacts_json evidence_paths_json scenario_dir changed_files_path pr_diff_path

  base_extra_artifacts_json="$(json_array '.extra_artifacts | map(select(.type != "changed_files" and .type != "pr_diff"))' "$fixture_json")"
  evidence_paths_json="$(json_array '.evidence_paths' "$fixture_json")"
  if [[ "$(jq 'length' <<<"$evidence_paths_json")" -eq 0 ]]; then
    printf '%s\n' "$base_extra_artifacts_json"
    return 0
  fi

  load_json_array_into_paths "$evidence_paths_json"
  scenario_dir="${batch_dir}/generated-artifacts/${fixture_slug}"
  changed_files_path="${scenario_dir}/changed-files.txt"
  pr_diff_path="${scenario_dir}/pr-diff.md"
  mkdir -p "$scenario_dir"

  generate_changed_files_artifact "$changed_files_path" "${GENERATED_PATHS[@]}"
  generate_pr_diff_artifact "$pr_diff_path" "$fixture_slug" "$scenario_label" "$task_summary" "${GENERATED_PATHS[@]}"

  jq -nc \
    --argjson base "$base_extra_artifacts_json" \
    --arg changed_files_path "$changed_files_path" \
    --arg pr_diff_path "$pr_diff_path" \
    '$base + [
      {type: "changed_files", source_path: $changed_files_path, trust_level: "trusted"},
      {type: "pr_diff", source_path: $pr_diff_path, trust_level: "trusted"}
    ]'
}

run_fixture() {
  local fixture_json="$1"
  local batch_runs_file="$2"
  local batch_dir="$3"

  local fixture_slug scenario_label run_stamp fixture_id run_id project result_file comparison_file
  fixture_slug="$(json_string '.id' "$fixture_json")"
  scenario_label="$(json_string '.scenario_label // .id' "$fixture_json")"
  run_stamp="$(date -u +%Y%m%dT%H%M%SZ)"
  fixture_id="v1-batch-${fixture_slug}-${run_stamp}"
  run_id="${run_stamp}-${fixture_id}"
  project="relay-${fixture_id}"
  result_file="${OUTPUT_ROOT%/}/${run_id}/result.json"
  comparison_file="${OUTPUT_ROOT%/}/${run_id}/paired-comparison.json"

  local repo_path handoff_path design_path task_summary capture_body decision_summary decision_reason question_summary
  local workflow artifact_type packet_type packet_target trace_decision trace_alternatives_json trace_rationale
  local trace_constraints_json trace_source_refs_json heuristic_key heuristic_canonical_text heuristic_normalized_text
  local proposal_source_refs_json extra_artifacts_json

  repo_path="$(json_string '.repo_path' "$fixture_json")"
  handoff_path="$(json_string '.handoff_path' "$fixture_json")"
  design_path="$(json_string '.design_path' "$fixture_json")"
  task_summary="$(json_string '.task_summary' "$fixture_json")"
  extra_artifacts_json="$(build_fixture_extra_artifacts "$fixture_json" "$batch_dir" "$fixture_slug" "$scenario_label" "$task_summary")"
  capture_body="$(json_string '.capture_body' "$fixture_json")"
  decision_summary="$(json_string '.decision_summary' "$fixture_json")"
  decision_reason="$(json_string '.decision_reason' "$fixture_json")"
  question_summary="$(json_string '.question_summary' "$fixture_json")"
  workflow="$(json_string '.workflow' "$fixture_json")"
  artifact_type="$(json_string '.artifact_type' "$fixture_json")"
  packet_type="$(json_string '.packet_type' "$fixture_json")"
  packet_target="$(json_string '.packet_target' "$fixture_json")"
  trace_decision="$(json_string '.trace_decision' "$fixture_json")"
  trace_alternatives_json="$(json_array '.trace_alternatives' "$fixture_json")"
  trace_rationale="$(json_string '.trace_rationale' "$fixture_json")"
  trace_constraints_json="$(json_array '.trace_constraints' "$fixture_json")"
  trace_source_refs_json="$(json_array '.trace_source_refs' "$fixture_json")"
  heuristic_key="$(json_string '.heuristic_key' "$fixture_json")"
  heuristic_canonical_text="$(json_string '.heuristic_canonical_text' "$fixture_json")"
  heuristic_normalized_text="$(json_string '.heuristic_normalized_text' "$fixture_json")"
  proposal_source_refs_json="$(json_array '.proposal_source_refs' "$fixture_json")"

  echo "running fixture: ${scenario_label} (${fixture_id})"
  RELAY_BASE_URL="$BASE_URL" \
  RELAY_MCP_URL="$MCP_URL" \
  RELAY_CLIENT_TOKEN="$CLIENT_TOKEN" \
  RELAY_ADMIN_TOKEN="$ADMIN_TOKEN" \
  RELAY_ACCEPTANCE_FIXTURE_ID="$fixture_id" \
  RELAY_ACCEPTANCE_RUN_ID="$run_id" \
  RELAY_ACCEPTANCE_PROJECT="$project" \
  RELAY_ACCEPTANCE_SCENARIO_LABEL="$scenario_label" \
  RELAY_ACCEPTANCE_REPO_PATH="$repo_path" \
  RELAY_ACCEPTANCE_HANDOFF_PATH="$handoff_path" \
  RELAY_ACCEPTANCE_DESIGN_PATH="$design_path" \
  RELAY_ACCEPTANCE_EXTRA_ARTIFACTS_JSON="$extra_artifacts_json" \
  RELAY_ACCEPTANCE_TASK_SUMMARY="$task_summary" \
  RELAY_ACCEPTANCE_CAPTURE_BODY="$capture_body" \
  RELAY_ACCEPTANCE_DECISION_SUMMARY="$decision_summary" \
  RELAY_ACCEPTANCE_DECISION_REASON="$decision_reason" \
  RELAY_ACCEPTANCE_QUESTION_SUMMARY="$question_summary" \
  RELAY_ACCEPTANCE_WORKFLOW="$workflow" \
  RELAY_ACCEPTANCE_ARTIFACT_TYPE="$artifact_type" \
  RELAY_ACCEPTANCE_PACKET_TYPE="$packet_type" \
  RELAY_ACCEPTANCE_PACKET_TARGET="$packet_target" \
  RELAY_ACCEPTANCE_TRACE_DECISION="$trace_decision" \
  RELAY_ACCEPTANCE_TRACE_ALTERNATIVES_JSON="$trace_alternatives_json" \
  RELAY_ACCEPTANCE_TRACE_RATIONALE="$trace_rationale" \
  RELAY_ACCEPTANCE_TRACE_CONSTRAINTS_JSON="$trace_constraints_json" \
  RELAY_ACCEPTANCE_TRACE_SOURCE_REFS_JSON="$trace_source_refs_json" \
  RELAY_ACCEPTANCE_HEURISTIC_KEY="$heuristic_key" \
  RELAY_ACCEPTANCE_HEURISTIC_CANONICAL_TEXT="$heuristic_canonical_text" \
  RELAY_ACCEPTANCE_HEURISTIC_NORMALIZED_TEXT="$heuristic_normalized_text" \
  RELAY_ACCEPTANCE_PROPOSAL_SOURCE_REFS_JSON="$proposal_source_refs_json" \
    ./scripts/acceptance/v1_canonical_handoff.sh --base-url "$BASE_URL" --mcp-url "$MCP_URL"

  ./scripts/evals/v1_copilot_paired_judge.sh --result-file "$result_file" --model "$MODEL"

  jq -nc \
    --arg recorded_at "$(python3 -c 'import datetime; print(datetime.datetime.now(datetime.UTC).isoformat())')" \
    --arg batch_id "$BATCH_ID" \
    --arg scenario_label "$scenario_label" \
    --arg fixture_id "$fixture_id" \
    --arg run_id "$run_id" \
    --arg project "$project" \
    --arg result_file "$result_file" \
    --arg comparison_file "$comparison_file" \
    --arg judge_model "$MODEL" \
    '{
      recorded_at: $recorded_at,
      batch_id: $batch_id,
      scenario_label: $scenario_label,
      fixture_id: $fixture_id,
      run_id: $run_id,
      project: $project,
      result_file: $result_file,
      comparison_file: $comparison_file,
      judge_model: $judge_model
    }' >>"$batch_runs_file"
}

main() {
  parse_args "$@"
  require_command curl
  require_command jq
  require_command python3
  require_command copilot

  if [[ ! -f "$FIXTURES_FILE" ]]; then
    echo "fixtures file not found: $FIXTURES_FILE" >&2
    exit 1
  fi

  local batch_dir batch_runs_file batch_summary_json batch_summary_md fixture_count
  batch_dir="${OUTPUT_ROOT%/}/batches/${BATCH_ID}"
  batch_runs_file="${batch_dir}/batch-runs.jsonl"
  batch_summary_json="${batch_dir}/batch-summary.json"
  batch_summary_md="${batch_dir}/batch-summary.md"
  mkdir -p "$batch_dir"
  cp "$FIXTURES_FILE" "${batch_dir}/fixtures.json"

  trap cleanup EXIT
  if ! resolve_client_token; then
    issue_temp_key
  fi

  : >"$batch_runs_file"
  fixture_count="$(jq 'length' "$FIXTURES_FILE")"
  if (( fixture_count == 0 )); then
    echo "fixtures file is empty: $FIXTURES_FILE" >&2
    exit 1
  fi

  local index fixture_json
  for (( index = 0; index < fixture_count; index++ )); do
    fixture_json="$(jq -c ".[$index]" "$FIXTURES_FILE")"
    run_fixture "$fixture_json" "$batch_runs_file" "$batch_dir"
  done

  ./scripts/evals/v1_usage_validation_report.py \
    --batch-runs-file "$batch_runs_file" \
    --min-style-aware-win-rate "$MIN_STYLE_AWARE_WIN_RATE" \
    --min-avg-style-match "$MIN_AVG_STYLE_MATCH" \
    --min-budget-pass-rate "$MIN_BUDGET_PASS_RATE" \
    --output-json "$batch_summary_json" \
    --output-md "$batch_summary_md"

  printf 'batch_runs: %s\nsummary_json: %s\nsummary_md: %s\n' \
    "$batch_runs_file" \
    "$batch_summary_json" \
    "$batch_summary_md"
}

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
  main "$@"
fi
