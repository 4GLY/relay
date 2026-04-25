#!/usr/bin/env bash
set -euo pipefail

BASE_URL="${RELAY_BASE_URL:-http://127.0.0.1:8080}"
MCP_URL="${RELAY_MCP_URL:-${BASE_URL%/}/mcp}"
CLIENT_TOKEN="${RELAY_CLIENT_TOKEN:-${RELAY_MCP_TOKEN:-}}"
ADMIN_TOKEN="${RELAY_ADMIN_TOKEN:-${RELAY_API_TOKEN:-}}"
OUTPUT_ROOT="${RELAY_ACCEPTANCE_OUTPUT_ROOT:-.gstack/projects/relay}"
FIXTURE_ID="${RELAY_ACCEPTANCE_FIXTURE_ID:-v1-canonical-$(date -u +%Y%m%dT%H%M%SZ)}"
PROJECT="${RELAY_ACCEPTANCE_PROJECT:-relay-${FIXTURE_ID}}"
RUN_ID="${RELAY_ACCEPTANCE_RUN_ID:-$(date -u +%Y%m%dT%H%M%SZ)-${FIXTURE_ID}}"
REPO_PATH="${RELAY_ACCEPTANCE_REPO_PATH:-$(pwd -P)}"
HANDOFF_PATH="${RELAY_ACCEPTANCE_HANDOFF_PATH:-scripts/acceptance/v1_canonical_handoff.sh}"
DESIGN_PATH="${RELAY_ACCEPTANCE_DESIGN_PATH:-docs/evals/v1-canonical-handoff.md}"
EXTRA_ARTIFACTS_JSON="${RELAY_ACCEPTANCE_EXTRA_ARTIFACTS_JSON:-[]}"
SCENARIO_LABEL="${RELAY_ACCEPTANCE_SCENARIO_LABEL:-${FIXTURE_ID}}"

STYLE_MATCH="${RELAY_ACCEPTANCE_STYLE_MATCH:-0}"
HEURISTIC_RELEVANCE="${RELAY_ACCEPTANCE_HEURISTIC_RELEVANCE:-yes}"
MANUAL_CORRECTIONS="${RELAY_ACCEPTANCE_MANUAL_CORRECTIONS:-0}"
CONTINUATION_WITHOUT_SUMMARY="${RELAY_ACCEPTANCE_CONTINUATION_WITHOUT_SUMMARY:-yes}"
PREFERRED_CONTINUATION="${RELAY_ACCEPTANCE_PREFERRED_CONTINUATION:-unscored}"

PACKET_BUILD_BUDGET_MS="${RELAY_ACCEPTANCE_PACKET_BUILD_BUDGET_MS:-5000}"
MCP_RESUME_BUDGET_MS="${RELAY_ACCEPTANCE_MCP_RESUME_BUDGET_MS:-10000}"
FIRST_RESPONSE_BUDGET_MS="${RELAY_ACCEPTANCE_FIRST_RESPONSE_BUDGET_MS:-45000}"
TOTAL_BUDGET_MS="${RELAY_ACCEPTANCE_TOTAL_BUDGET_MS:-60000}"

default_capture_body() {
  cat <<EOF
Relay V1 canonical handoff fixture ${FIXTURE_ID}
User style seed:
- Prefer explicit contracts over implicit inference across model and session handoff.
- Keep Relay API-first and keep public MCP packet-centric.
- Human approval controls durable heuristics.
- Same-project proof comes before broader implicit-learning claims.
EOF
}

default_trace_alternatives_json() {
  printf '%s\n' '["Let the next model infer the product contract from chat history."]'
}

default_trace_constraints_json() {
  printf '%s\n' '["Same-project V1 proof first","Public MCP remains packet-centric"]'
}

default_trace_source_refs_json() {
  printf '%s\n' '["scripts/acceptance/v1_canonical_handoff.sh","docs/evals/v1-canonical-handoff.md"]'
}

default_proposal_source_refs_json() {
  printf '%s\n' '["docs/evals/v1-canonical-handoff.md"]'
}

usage() {
  cat <<EOF
Usage:
  v1_canonical_handoff.sh [options]

Runs the Relay V1 canonical handoff acceptance flow:
  capture -> judgment_trace -> heuristic_proposal -> admin approval
  -> public MCP style-aware packet -> public MCP control packet

Options:
  --base-url URL        Relay API base URL. Default: ${BASE_URL}
  --mcp-url URL         Relay MCP URL. Default: \$base_url/mcp
  --client-token TOKEN  Issued client token for normal /v1 and /mcp calls
  --admin-token TOKEN   Bootstrap admin token for approval
  --project NAME        Project name. Default: relay-\$fixture_id
  --fixture-id ID       Fixture id. Default: v1-canonical-<utc timestamp>
  --run-id ID           Output run id. Default: <utc timestamp>-\$fixture_id
  --output-root DIR     Output root. Default: .gstack/projects/relay

Environment rubric overrides:
  RELAY_ACCEPTANCE_STYLE_MATCH=1..5
  RELAY_ACCEPTANCE_HEURISTIC_RELEVANCE=yes|no
  RELAY_ACCEPTANCE_MANUAL_CORRECTIONS=0
  RELAY_ACCEPTANCE_CONTINUATION_WITHOUT_SUMMARY=yes|no
  RELAY_ACCEPTANCE_PREFERRED_CONTINUATION=style-aware|control|unscored
EOF
}

parse_args() {
  while [[ $# -gt 0 ]]; do
    case "$1" in
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
      --project)
        PROJECT="${2:?project required}"
        shift 2
        ;;
      --fixture-id)
        FIXTURE_ID="${2:?fixture id required}"
        shift 2
        ;;
      --run-id)
        RUN_ID="${2:?run id required}"
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

epoch_ms() {
  python3 -c 'import time; print(int(time.time() * 1000))'
}

iso_now() {
  python3 -c 'import datetime; print(datetime.datetime.now(datetime.UTC).isoformat())'
}

ms_to_iso() {
  python3 -c 'import datetime,sys; print(datetime.datetime.fromtimestamp(int(sys.argv[1]) / 1000, datetime.UTC).isoformat())' "$1"
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

api_json() {
  local token="$1"
  local method="$2"
  local path="$3"
  local body="${4:-}"
  curl_json "$token" "$method" "${BASE_URL%/}${path}" "$body"
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

excerpt_from_packet() {
  jq -r '.rendered_body // .body // ""' | python3 -c 'import sys; print(sys.stdin.read().replace("\n", " ")[:500])'
}

write_summary() {
  local summary_file="$1"
  cat >"$summary_file" <<EOF
# Relay V1 Canonical Handoff Acceptance

Run purpose:
- Prove the same-project style-memory path can run from seed trace to public MCP packet without manual chat summary.

Fixture:
- fixture_id: \`${FIXTURE_ID}\`
- scenario: \`${SCENARIO_LABEL}\`
- project: \`${PROJECT}\`
- project_id: \`${PROJECT_ID}\`

Run type:
- seed: judgment trace plus heuristic proposal
- style-aware: public MCP packet with approved style cues
- control: public MCP packet with style cues disabled

Timing:
- style packet MCP duration: ${STYLE_PACKET_DURATION_MS} ms
- control packet MCP duration: ${CONTROL_PACKET_DURATION_MS} ms
- total handoff duration: ${TOTAL_HANDOFF_DURATION_MS} ms

Artifacts:
- trace_id: \`${TRACE_ID}\`
- proposal_id: \`${PROPOSAL_ID}\`
- approved_heuristic_id: \`${APPROVED_HEURISTIC_ID}\`
- style_snapshot_id: \`${STYLE_SNAPSHOT_ID}\`
- latest_snapshot_id: \`${LATEST_SNAPSHOT_ID}\`
- control_snapshot_id: \`${CONTROL_SNAPSHOT_ID}\`

Notable packet contents:
- schema_version: \`${SCHEMA_VERSION}\`
- approved_heuristic_ids: \`${APPROVED_HEURISTIC_IDS}\`
- style excerpt: ${STYLE_EXCERPT}
- latest snapshot excerpt: ${LATEST_EXCERPT}
- control excerpt: ${CONTROL_EXCERPT}

Snapshot continuation smoke:
- latest snapshot MCP duration: ${LATEST_SNAPSHOT_MCP_DURATION_MS} ms
- latest snapshot API duration: ${LATEST_SNAPSHOT_API_DURATION_MS} ms
- latest snapshot contract pass: ${LATEST_SNAPSHOT_CONTRACT_PASS}

Rubric:
- style_match: ${STYLE_MATCH}
- heuristic_relevance: ${HEURISTIC_RELEVANCE}
- manual_corrections: ${MANUAL_CORRECTIONS}
- continuation_without_summary: ${CONTINUATION_WITHOUT_SUMMARY}
- preferred_continuation: ${PREFERRED_CONTINUATION}
EOF
}

main() {
  parse_args "$@"
  require_command curl
  require_command jq
  require_command python3

  if [[ -z "${CLIENT_TOKEN}" ]]; then
    echo "RELAY_CLIENT_TOKEN or --client-token is required" >&2
    exit 1
  fi
  if [[ -z "${ADMIN_TOKEN}" ]]; then
    echo "RELAY_ADMIN_TOKEN or --admin-token is required" >&2
    exit 1
  fi

  local workflow="${RELAY_ACCEPTANCE_WORKFLOW:-design_handoff}"
  local artifact_type="${RELAY_ACCEPTANCE_ARTIFACT_TYPE:-design_doc}"
  local packet_type="${RELAY_ACCEPTANCE_PACKET_TYPE:-style_handoff}"
  local packet_target="${RELAY_ACCEPTANCE_PACKET_TARGET:-codex}"
  local task_summary="${RELAY_ACCEPTANCE_TASK_SUMMARY:-Resume Relay V1 implementation from the canonical same-project handoff fixture.}"
  local capture_body_text="${RELAY_ACCEPTANCE_CAPTURE_BODY:-$(default_capture_body)}"
  local decision_summary="${RELAY_ACCEPTANCE_DECISION_SUMMARY:-Keep Relay API-first and keep public MCP packet-centric for V1 handoff.}"
  local decision_reason="${RELAY_ACCEPTANCE_DECISION_REASON:-The next agent should continue from explicit packet contracts instead of hidden chat state.}"
  local question_summary="${RELAY_ACCEPTANCE_QUESTION_SUMMARY:-What additional packet evidence is still needed before semantic retrieval is introduced?}"
  local trace_decision="${RELAY_ACCEPTANCE_TRACE_DECISION:-Prefer explicit contracts over implicit inference for model-to-model handoff.}"
  local trace_alternatives_json="${RELAY_ACCEPTANCE_TRACE_ALTERNATIVES_JSON:-$(default_trace_alternatives_json)}"
  local trace_rationale="${RELAY_ACCEPTANCE_TRACE_RATIONALE:-The next model should preserve user decision style without re-reading the whole conversation.}"
  local trace_constraints_json="${RELAY_ACCEPTANCE_TRACE_CONSTRAINTS_JSON:-$(default_trace_constraints_json)}"
  local trace_source_refs_json="${RELAY_ACCEPTANCE_TRACE_SOURCE_REFS_JSON:-$(default_trace_source_refs_json)}"
  local heuristic_key="${RELAY_ACCEPTANCE_HEURISTIC_KEY:-explicit_contracts_over_magic}"
  local heuristic_canonical_text="${RELAY_ACCEPTANCE_HEURISTIC_CANONICAL_TEXT:-Prefer explicit contracts over magic inference when handing work from one model or session to another.}"
  local heuristic_normalized_text="${RELAY_ACCEPTANCE_HEURISTIC_NORMALIZED_TEXT:-prefer explicit contracts over magic inference for model-to-model handoff}"
  local proposal_source_refs_json="${RELAY_ACCEPTANCE_PROPOSAL_SOURCE_REFS_JSON:-$(default_proposal_source_refs_json)}"

  local output_dir="${OUTPUT_ROOT%/}/${RUN_ID}"
  local result_file="${output_dir}/result.json"
  local summary_file="${output_dir}/summary.md"
  local ledger_file="${OUTPUT_ROOT%/}/usage-validation.jsonl"
  local style_packet_file="${output_dir}/style-packet.json"
  local latest_snapshot_file="${output_dir}/latest-snapshot.json"
  local latest_snapshot_api_file="${output_dir}/latest-snapshot-api.json"
  local control_packet_file="${output_dir}/control-packet.json"
  mkdir -p "$output_dir" "${OUTPUT_ROOT%/}"

  api_json "" GET "/healthz" >/dev/null

  local capture_body capture_response
  capture_body="$(jq -nc \
    --arg project "$PROJECT" \
    --arg body "$capture_body_text" \
    --arg repo_path "$REPO_PATH" \
    --arg handoff_path "$HANDOFF_PATH" \
    --arg design_path "$DESIGN_PATH" \
    --argjson extra_artifacts "$EXTRA_ARTIFACTS_JSON" \
    '{
    project: $project,
    source: "acceptance",
    body: $body,
    repo_path: $repo_path,
    handoff_path: $handoff_path,
    design_path: $design_path,
    extra_artifacts: $extra_artifacts,
    idempotency_key: ($project + "-capture")
  }')"
  capture_response="$(api_json "$CLIENT_TOKEN" POST "/v1/capture" "$capture_body")"
  PROJECT_ID="$(jq -r '.data.project_id' <<<"$capture_response")"
  local note_id
  note_id="$(jq -r '.data.created_note_ids[0] // ""' <<<"$capture_response")"
  local artifact_ids_json
  artifact_ids_json="$(jq -c '.data.created_artifact_ids // []' <<<"$capture_response")"

  local seed_decision_body seed_question_body
  seed_decision_body="$(jq -nc \
    --arg project "$PROJECT" \
    --arg note_id "$note_id" \
    --arg fixture "$FIXTURE_ID" \
    --arg summary "$decision_summary" \
    --arg reason "$decision_reason" \
    --argjson artifact_ids "$artifact_ids_json" \
    '{
    project: $project,
    kind: "decision",
    summary: $summary,
    reason: $reason,
    source_note_ids: (if $note_id == "" then [] else [$note_id] end),
    source_artifact_ids: $artifact_ids,
    idempotency_key: ($fixture + "-seed-decision")
  }')"
  api_json "$CLIENT_TOKEN" POST "/v1/promote" "$seed_decision_body" >/dev/null

  seed_question_body="$(jq -nc \
    --arg project "$PROJECT" \
    --arg note_id "$note_id" \
    --arg fixture "$FIXTURE_ID" \
    --arg summary "$question_summary" \
    --argjson artifact_ids "$artifact_ids_json" \
    '{
    project: $project,
    kind: "question",
    summary: $summary,
    source_note_ids: (if $note_id == "" then [] else [$note_id] end),
    source_artifact_ids: $artifact_ids,
    idempotency_key: ($fixture + "-seed-question")
  }')"
  api_json "$CLIENT_TOKEN" POST "/v1/promote" "$seed_question_body" >/dev/null

  local trace_body trace_response
  trace_body="$(jq -nc \
    --arg project "$PROJECT" \
    --arg fixture "$FIXTURE_ID" \
    --arg workflow "$workflow" \
    --arg artifact_type "$artifact_type" \
    --arg decision "$trace_decision" \
    --arg rationale "$trace_rationale" \
    --argjson alternatives "$trace_alternatives_json" \
    --argjson constraints "$trace_constraints_json" \
    --argjson source_refs "$trace_source_refs_json" \
    '{
    project: $project,
    task_id: ($fixture + "-task"),
    agent_id: "acceptance-seed-agent",
    workflow: $workflow,
    artifact_type: $artifact_type,
    decision: $decision,
    alternatives: $alternatives,
    rationale: $rationale,
    constraints: $constraints,
    source_refs: $source_refs,
    language: "en",
    idempotency_key: ($fixture + "-trace")
  }')"
  trace_response="$(api_json "$CLIENT_TOKEN" POST "/v1/judgment-traces" "$trace_body")"
  TRACE_ID="$(jq -r '.data.trace_id' <<<"$trace_response")"
  local curator_job_id
  curator_job_id="$(jq -r '.data.curator_job_id // ""' <<<"$trace_response")"

  local proposal_body proposal_response
  proposal_body="$(jq -nc \
    --arg project "$PROJECT" \
    --arg trace_id "$TRACE_ID" \
    --arg note_id "$note_id" \
    --arg fixture "$FIXTURE_ID" \
    --arg workflow "$workflow" \
    --arg artifact_type "$artifact_type" \
    --arg heuristic_key "$heuristic_key" \
    --arg canonical_text "$heuristic_canonical_text" \
    --arg normalized_text "$heuristic_normalized_text" \
    --argjson proposal_source_refs "$proposal_source_refs_json" \
    '{
    project: $project,
    origin_trace_id: $trace_id,
    workflow: $workflow,
    artifact_type: $artifact_type,
    heuristic_key: $heuristic_key,
    canonical_text: $canonical_text,
    normalized_text: $normalized_text,
    source_trace_ids: [$trace_id],
    source_refs: ($proposal_source_refs + (if $note_id == "" then [] else [$note_id] end)),
    proposed_by: "acceptance-runner",
    idempotency_key: ($fixture + "-proposal")
  }')"
  proposal_response="$(api_json "$CLIENT_TOKEN" POST "/v1/heuristic-proposals" "$proposal_body")"
  PROPOSAL_ID="$(jq -r '.data.proposal_id' <<<"$proposal_response")"

  local review_body review_response
  review_body="$(jq -nc --arg project "$PROJECT" --arg proposal_id "$PROPOSAL_ID" '{
    project: $project,
    proposal_id: $proposal_id,
    action: "approve",
    review_notes: "Acceptance seed approval for V1 canonical handoff."
  }')"
  review_response="$(api_json "$ADMIN_TOKEN" POST "/v1/heuristic-proposals/review" "$review_body")"
  APPROVED_HEURISTIC_ID="$(jq -r '.data.approved_heuristic_id' <<<"$review_response")"

  local handoff_start_ms style_start_ms style_end_ms control_start_ms control_end_ms
  handoff_start_ms="$(epoch_ms)"

  local style_args style_mcp_response style_packet
  style_args="$(jq -nc \
    --arg project "$PROJECT" \
    --arg fixture "$FIXTURE_ID" \
    --arg packet_type "$packet_type" \
    --arg packet_target "$packet_target" \
    --arg workflow "$workflow" \
    --arg artifact_type "$artifact_type" \
    --arg task_summary "$task_summary" \
    '{
    project: $project,
    type: $packet_type,
    target: $packet_target,
    workflow: $workflow,
    artifact_type: $artifact_type,
    task_summary: $task_summary,
    persist_snapshot: true,
    idempotency_key: ($fixture + "-style-packet")
  }')"
  style_start_ms="$(epoch_ms)"
  style_mcp_response="$(mcp_call relay_build_packet "$style_args")"
  style_end_ms="$(epoch_ms)"
  style_packet="$(structured_content "$style_mcp_response")"
  STYLE_SNAPSHOT_ID="$(jq -r '.snapshot_id // ""' <<<"$style_packet")"

  local latest_args latest_snapshot_mcp_response latest_snapshot latest_snapshot_api_response latest_snapshot_api
  local latest_mcp_start_ms latest_mcp_end_ms latest_api_start_ms latest_api_end_ms
  latest_args="$(jq -nc \
    --arg project "$PROJECT" \
    --arg packet_type "$packet_type" \
    --arg packet_target "$packet_target" \
    '{
    project: $project,
    type: $packet_type,
    target: $packet_target
  }')"
  latest_mcp_start_ms="$(epoch_ms)"
  latest_snapshot_mcp_response="$(mcp_call relay_latest_packet_snapshot "$latest_args")"
  latest_mcp_end_ms="$(epoch_ms)"
  latest_snapshot="$(structured_content "$latest_snapshot_mcp_response")"

  latest_api_start_ms="$(epoch_ms)"
  latest_snapshot_api_response="$(api_json "$CLIENT_TOKEN" GET "/v1/projects/${PROJECT_ID}/packet-snapshots/latest?type=${packet_type}&target=${packet_target}")"
  latest_api_end_ms="$(epoch_ms)"
  latest_snapshot_api="$(jq -c '.data' <<<"$latest_snapshot_api_response")"

  local control_args control_mcp_response control_packet
  control_args="$(jq -nc \
    --arg project "$PROJECT" \
    --arg fixture "$FIXTURE_ID" \
    --arg packet_type "$packet_type" \
    --arg packet_target "$packet_target" \
    --arg workflow "$workflow" \
    --arg artifact_type "$artifact_type" \
    --arg task_summary "$task_summary" \
    '{
    project: $project,
    type: $packet_type,
    target: $packet_target,
    workflow: $workflow,
    artifact_type: $artifact_type,
    task_summary: $task_summary,
    disable_style_cues: true,
    persist_snapshot: true,
    idempotency_key: ($fixture + "-control-packet")
  }')"
  control_start_ms="$(epoch_ms)"
  control_mcp_response="$(mcp_call relay_build_packet "$control_args")"
  control_end_ms="$(epoch_ms)"
  control_packet="$(structured_content "$control_mcp_response")"

  STYLE_PACKET_DURATION_MS=$((style_end_ms - style_start_ms))
  LATEST_SNAPSHOT_MCP_DURATION_MS=$((latest_mcp_end_ms - latest_mcp_start_ms))
  LATEST_SNAPSHOT_API_DURATION_MS=$((latest_api_end_ms - latest_api_start_ms))
  CONTROL_PACKET_DURATION_MS=$((control_end_ms - control_start_ms))
  TOTAL_HANDOFF_DURATION_MS=$((control_end_ms - handoff_start_ms))
  SCHEMA_VERSION="$(jq -r '.schema_version // ""' <<<"$style_packet")"
  LATEST_SNAPSHOT_ID="$(jq -r '.snapshot_id // ""' <<<"$latest_snapshot")"
  LATEST_SNAPSHOT_API_ID="$(jq -r '.snapshot_id // ""' <<<"$latest_snapshot_api")"
  CONTROL_SNAPSHOT_ID="$(jq -r '.snapshot_id // ""' <<<"$control_packet")"
  APPROVED_HEURISTIC_IDS="$(jq -c '.approved_heuristic_ids // []' <<<"$style_packet")"
  STYLE_EXCERPT="$(excerpt_from_packet <<<"$style_packet")"
  LATEST_EXCERPT="$(excerpt_from_packet <<<"$latest_snapshot")"
  CONTROL_EXCERPT="$(excerpt_from_packet <<<"$control_packet")"
  jq . <<<"$style_packet" >"$style_packet_file"
  jq . <<<"$latest_snapshot" >"$latest_snapshot_file"
  jq . <<<"$latest_snapshot_api" >"$latest_snapshot_api_file"
  jq . <<<"$control_packet" >"$control_packet_file"

  local first_response_duration_ms budget_pass latest_snapshot_contract_pass heuristic_relevance_json result_json ledger_json
  if [[ "$STYLE_SNAPSHOT_ID" != "" &&
        "$LATEST_SNAPSHOT_ID" == "$STYLE_SNAPSHOT_ID" &&
        "$LATEST_SNAPSHOT_API_ID" == "$STYLE_SNAPSHOT_ID" ]] &&
     jq -e --arg body "$(jq -r '.rendered_body // ""' <<<"$style_packet")" '.rendered_body == $body' <<<"$latest_snapshot" >/dev/null &&
     jq -e --arg body "$(jq -r '.rendered_body // ""' <<<"$style_packet")" '.rendered_body == $body' <<<"$latest_snapshot_api" >/dev/null; then
    latest_snapshot_contract_pass=true
  else
    latest_snapshot_contract_pass=false
  fi
  LATEST_SNAPSHOT_CONTRACT_PASS="$latest_snapshot_contract_pass"
  first_response_duration_ms="$STYLE_PACKET_DURATION_MS"
  if (( STYLE_PACKET_DURATION_MS <= PACKET_BUILD_BUDGET_MS &&
        STYLE_PACKET_DURATION_MS <= MCP_RESUME_BUDGET_MS &&
        first_response_duration_ms <= FIRST_RESPONSE_BUDGET_MS &&
        TOTAL_HANDOFF_DURATION_MS <= TOTAL_BUDGET_MS )) &&
     [[ "$latest_snapshot_contract_pass" == true ]]; then
    budget_pass=true
  else
    budget_pass=false
  fi

  heuristic_relevance_json="$(jq -c --arg relevant "$HEURISTIC_RELEVANCE" '[.[] | {heuristic_id: ., relevant: $relevant}]' <<<"$APPROVED_HEURISTIC_IDS")"

  result_json="$(jq -n \
    --arg run_id "$RUN_ID" \
    --arg fixture_id "$FIXTURE_ID" \
    --arg scenario_label "$SCENARIO_LABEL" \
    --arg project "$PROJECT" \
    --arg project_id "$PROJECT_ID" \
    --arg trace_id "$TRACE_ID" \
    --arg curator_job_id "$curator_job_id" \
    --arg proposal_id "$PROPOSAL_ID" \
    --arg approved_heuristic_id "$APPROVED_HEURISTIC_ID" \
    --arg schema_version "$SCHEMA_VERSION" \
    --arg style_snapshot_id "$STYLE_SNAPSHOT_ID" \
    --arg latest_snapshot_id "$LATEST_SNAPSHOT_ID" \
    --arg latest_snapshot_api_id "$LATEST_SNAPSHOT_API_ID" \
    --arg control_snapshot_id "$CONTROL_SNAPSHOT_ID" \
    --argjson approved_heuristic_ids "$APPROVED_HEURISTIC_IDS" \
    --arg handoff_start_time "$(ms_to_iso "$handoff_start_ms")" \
    --arg packet_built_time "$(ms_to_iso "$style_end_ms")" \
    --arg mcp_resume_start_time "$(ms_to_iso "$style_start_ms")" \
    --arg first_usable_response_time "$(ms_to_iso "$style_end_ms")" \
    --arg style_packet_file "$style_packet_file" \
    --arg latest_snapshot_file "$latest_snapshot_file" \
    --arg latest_snapshot_api_file "$latest_snapshot_api_file" \
    --arg control_packet_file "$control_packet_file" \
    --argjson style_packet_duration_ms "$STYLE_PACKET_DURATION_MS" \
    --argjson latest_snapshot_mcp_duration_ms "$LATEST_SNAPSHOT_MCP_DURATION_MS" \
    --argjson latest_snapshot_api_duration_ms "$LATEST_SNAPSHOT_API_DURATION_MS" \
    --argjson control_packet_duration_ms "$CONTROL_PACKET_DURATION_MS" \
    --argjson first_response_duration_ms "$first_response_duration_ms" \
    --argjson total_handoff_duration_ms "$TOTAL_HANDOFF_DURATION_MS" \
    --arg first_continuation_excerpt "$STYLE_EXCERPT" \
    --arg latest_snapshot_excerpt "$LATEST_EXCERPT" \
    --arg control_continuation_excerpt "$CONTROL_EXCERPT" \
    --argjson style_match "$STYLE_MATCH" \
    --argjson heuristic_relevance "$heuristic_relevance_json" \
    --argjson manual_corrections "$MANUAL_CORRECTIONS" \
    --arg continuation_without_summary "$CONTINUATION_WITHOUT_SUMMARY" \
    --arg preferred_continuation "$PREFERRED_CONTINUATION" \
    --argjson budget_pass "$budget_pass" \
    --argjson latest_snapshot_contract_pass "$latest_snapshot_contract_pass" \
    --argjson packet_build_budget_ms "$PACKET_BUILD_BUDGET_MS" \
    --argjson mcp_resume_budget_ms "$MCP_RESUME_BUDGET_MS" \
    --argjson first_response_budget_ms "$FIRST_RESPONSE_BUDGET_MS" \
    --argjson total_budget_ms "$TOTAL_BUDGET_MS" \
    '{
      run_id: $run_id,
      fixture_id: $fixture_id,
      scenario_label: $scenario_label,
      project: $project,
      project_id: $project_id,
      trace_id: $trace_id,
      curator_job_id: $curator_job_id,
      proposal_id: $proposal_id,
      approved_heuristic_id: $approved_heuristic_id,
      packet_schema_version: $schema_version,
      style_snapshot_id: $style_snapshot_id,
      packet_snapshot_id: $style_snapshot_id,
      latest_snapshot_id: $latest_snapshot_id,
      latest_snapshot_api_id: $latest_snapshot_api_id,
      control_snapshot_id: $control_snapshot_id,
      approved_heuristic_ids: $approved_heuristic_ids,
      handoff_start_time: $handoff_start_time,
      packet_built_time: $packet_built_time,
      mcp_resume_start_time: $mcp_resume_start_time,
      first_usable_response_time: $first_usable_response_time,
      style_packet_file: $style_packet_file,
      latest_snapshot_file: $latest_snapshot_file,
      latest_snapshot_api_file: $latest_snapshot_api_file,
      control_packet_file: $control_packet_file,
      style_packet_duration_ms: $style_packet_duration_ms,
      latest_snapshot_mcp_duration_ms: $latest_snapshot_mcp_duration_ms,
      latest_snapshot_api_duration_ms: $latest_snapshot_api_duration_ms,
      control_packet_duration_ms: $control_packet_duration_ms,
      first_response_duration_ms: $first_response_duration_ms,
      total_handoff_duration_ms: $total_handoff_duration_ms,
      first_continuation_excerpt: $first_continuation_excerpt,
      latest_snapshot_excerpt: $latest_snapshot_excerpt,
      control_continuation_excerpt: $control_continuation_excerpt,
      latest_snapshot_contract: {
        pass: $latest_snapshot_contract_pass,
        mcp_snapshot_id: $latest_snapshot_id,
        api_snapshot_id: $latest_snapshot_api_id
      },
      rubric_scores: {
        style_match: $style_match,
        heuristic_relevance: $heuristic_relevance,
        manual_corrections: $manual_corrections,
        continuation_without_summary: $continuation_without_summary,
        preferred_continuation: $preferred_continuation
      },
      budget: {
        pass: $budget_pass,
        packet_build_ms: $packet_build_budget_ms,
        mcp_resume_ms: $mcp_resume_budget_ms,
        first_response_ms: $first_response_budget_ms,
        total_ms: $total_budget_ms
      }
    }')"
  printf '%s\n' "$result_json" >"$result_file"
  write_summary "$summary_file"

  ledger_json="$(jq -nc \
    --arg recorded_at "$(iso_now)" \
    --arg run_id "$RUN_ID" \
    --arg fixture_id "$FIXTURE_ID" \
    --arg project "$PROJECT" \
    --arg result_file "$result_file" \
    --arg summary_file "$summary_file" \
    --argjson total_handoff_duration_ms "$TOTAL_HANDOFF_DURATION_MS" \
    --arg preferred_continuation "$PREFERRED_CONTINUATION" \
    --argjson budget_pass "$budget_pass" \
    '{
      recorded_at: $recorded_at,
      run_id: $run_id,
      fixture_id: $fixture_id,
      project: $project,
      result_file: $result_file,
      summary_file: $summary_file,
      total_handoff_duration_ms: $total_handoff_duration_ms,
      preferred_continuation: $preferred_continuation,
      budget_pass: $budget_pass
    }')"
  printf '%s\n' "$ledger_json" >>"$ledger_file"

  printf 'result: %s\nsummary: %s\nledger: %s\n' "$result_file" "$summary_file" "$ledger_file"

  if [[ "$budget_pass" != true ]]; then
    echo "acceptance budget failed" >&2
    exit 1
  fi
}

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
  main "$@"
fi
