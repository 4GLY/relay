#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
BASE_URL="${RELAY_WEB_BASE_URL:-https://relay.4gly.dev}"
RUN_ID="qa$(date -u +%Y%m%d%H%M%S)$(openssl rand -hex 3)"
PROJECT_ID="proj_${RUN_ID}"
PROJECT_NAME="relay-e2e-${RUN_ID}"
USER_ID="usr_${RUN_ID}"
SESSION_ID="usess_${RUN_ID}"
SNAPSHOT_ID="psnap_${RUN_ID}"
PUBLIC_TOKEN="psnap_${RUN_ID}_token"
SESSION_TOKEN="rsess_$(openssl rand -hex 24)"
SESSION_HASH="$(printf '%s' "${SESSION_TOKEN}" | shasum -a 256 | awk '{print $1}')"

PSQL="${PSQL:-/opt/homebrew/opt/libpq/bin/psql}"
if [[ ! -x "${PSQL}" ]]; then
  PSQL="psql"
fi

DB_URL="${RELAY_DATABASE_URL:-}"
if [[ -z "${DB_URL}" ]]; then
  DB_URL="$(kubectl get secret relay-secrets -n relay -o jsonpath='{.data.database_url}' | base64 -d)"
fi

cleanup() {
  set +e
  "${PSQL}" "${DB_URL}" -v ON_ERROR_STOP=1 -q <<SQL >/dev/null 2>&1
DELETE FROM idempotency_records WHERE scope_project_id = '${PROJECT_ID}';
DELETE FROM projects WHERE id = '${PROJECT_ID}';
DELETE FROM user_provider_credentials WHERE user_id = '${USER_ID}';
DELETE FROM api_keys WHERE owner_user_id = '${USER_ID}';
DELETE FROM user_sessions WHERE user_id = '${USER_ID}';
DELETE FROM users WHERE id = '${USER_ID}';
SQL
}
trap cleanup EXIT

"${PSQL}" "${DB_URL}" -v ON_ERROR_STOP=1 -q <<SQL >/dev/null
INSERT INTO users (id, email, display_name)
VALUES ('${USER_ID}', '${RUN_ID}@example.invalid', 'Relay E2E');

INSERT INTO projects (id, name, root_path, status, owner_user_id)
VALUES ('${PROJECT_ID}', '${PROJECT_NAME}', '/tmp/${PROJECT_NAME}', 'active', '${USER_ID}');

INSERT INTO user_sessions (id, user_id, token_hash, expires_at)
VALUES ('${SESSION_ID}', '${USER_ID}', '${SESSION_HASH}', NOW() + INTERVAL '30 days');

INSERT INTO user_onboarding (
  user_id,
  default_project_id,
  onboarding_completed_at,
  last_validated_at
)
VALUES ('${USER_ID}', '${PROJECT_ID}', NOW(), NOW());

INSERT INTO judgment_traces (
  id,
  project_id,
  task_id,
  agent_id,
  workflow,
  artifact_type,
  decision,
  alternatives,
  rationale,
  constraints,
  source_refs,
  language
)
VALUES (
  'trace_${RUN_ID}',
  '${PROJECT_ID}',
  'live-e2e-${RUN_ID}',
  'codex',
  'qa_live',
  'style_memory',
  'Prefer explicit recovery actions over generic error states.',
  '["Generic error state"]'::jsonb,
  'Authenticated live E2E needs a deterministic pending proposal.',
  '["Temporary QA fixture"]'::jsonb,
  '["qa:live:e2e"]'::jsonb,
  'en'
);

INSERT INTO heuristic_proposals (
  id,
  project_id,
  origin_trace_id,
  workflow,
  artifact_type,
  heuristic_key,
  canonical_text,
  normalized_text,
  state,
  source_trace_ids,
  source_refs,
  proposed_by
)
VALUES (
  'hprop_${RUN_ID}',
  '${PROJECT_ID}',
  'trace_${RUN_ID}',
  'qa_live',
  'style_memory',
  'qa_live_authenticated_e2e_${RUN_ID}',
  'When a workflow can fail, show a specific recovery action instead of a generic error message.',
  'specific recovery action over generic error',
  'pending',
  '["trace_${RUN_ID}"]'::jsonb,
  '["qa:live:e2e"]'::jsonb,
  'codex-live-e2e'
);

INSERT INTO packet_snapshots (
  id,
  project_id,
  packet_kind,
  target,
  schema_version,
  task_summary,
  rendered_body,
  style_cues,
  supporting_notes,
  supporting_decisions,
  supporting_questions,
  supporting_artifacts,
  why_included,
  approved_heuristic_ids,
  decision_ids,
  open_question_ids,
  source_artifact_ids,
  missing_context,
  public_readable,
  public_token,
  og_image_path
)
VALUES (
  '${SNAPSHOT_ID}',
  '${PROJECT_ID}',
  'handoff',
  'codex',
  'relay.packet.v1',
  'Live E2E public snapshot fixture',
  'Project: ${PROJECT_NAME}
Current goal: verify public snapshot positive route automation.',
  '[]'::jsonb,
  '[]'::jsonb,
  '[]'::jsonb,
  '[]'::jsonb,
  '[]'::jsonb,
  '[]'::jsonb,
  '[]'::jsonb,
  '[]'::jsonb,
  '[]'::jsonb,
  '[]'::jsonb,
  '[]'::jsonb,
  TRUE,
  '${PUBLIC_TOKEN}',
  ''
);
SQL

(
  cd "${ROOT_DIR}/web"
  RELAY_WEB_BASE_URL="${BASE_URL}" \
  RELAY_QA_SESSION_COOKIE="${SESSION_TOKEN}" \
  RELAY_QA_AUTH_PROJECT_ID="${PROJECT_ID}" \
  RELAY_QA_PROJECT_ID="${PROJECT_ID}" \
  RELAY_QA_PUBLIC_SNAPSHOT_TOKEN="${PUBLIC_TOKEN}" \
  npm run qa:e2e
)

echo "live_e2e_run_id=${RUN_ID}"
echo "live_e2e_project=${PROJECT_ID}"
echo "live_e2e_public_snapshot=/p/${PUBLIC_TOKEN}"
