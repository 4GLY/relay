#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
KEYCHAIN_SERVICE="${RELAY_KEYCHAIN_SERVICE:-codex.relay-api}"
KEYCHAIN_ADMIN_ACCOUNT="${RELAY_KEYCHAIN_ADMIN_ACCOUNT:-admin-token}"
KEYCHAIN_CLIENT_ACCOUNT="${RELAY_KEYCHAIN_CLIENT_ACCOUNT:-client-token}"
KEYCHAIN_BASE_URL_ACCOUNT="${RELAY_KEYCHAIN_BASE_URL_ACCOUNT:-base-url}"
DEFAULT_BASE_URL="https://relay.4gly.dev"

BASE_URL=""
BASE_URL_SOURCE=""
ADMIN_TOKEN=""
ADMIN_TOKEN_SOURCE=""
CLIENT_TOKEN=""
CLIENT_TOKEN_SOURCE=""

usage() {
  cat <<'EOF'
Usage:
  relay-api.sh doctor
  relay-api.sh issue-key <name> [--scope global|project] [--project <name>] [--project-id <id>] [--store-client]
  relay-api.sh list-keys
  relay-api.sh revoke-key <key-id>
  relay-api.sh capture <json-file|->
  relay-api.sh promote <json-file|->
  relay-api.sh build-packet <json-file|->
  relay-api.sh show <project-id>
  relay-api.sh raw <METHOD> <path> [json-file|-] [--admin]

Environment:
  RELAY_BASE_URL                          Override base URL
  RELAY_ADMIN_TOKEN / RELAY_API_TOKEN     Bootstrap admin token
  RELAY_CLIENT_TOKEN / RELAY_MCP_TOKEN    Issued client token
  RELAY_KEYCHAIN_SERVICE                  macOS Keychain service name
  RELAY_KEYCHAIN_ADMIN_ACCOUNT            Keychain account for admin token
  RELAY_KEYCHAIN_CLIENT_ACCOUNT           Keychain account for client token
  RELAY_KEYCHAIN_BASE_URL_ACCOUNT         Keychain account for base URL

Examples:
  relay-api.sh doctor
  relay-api.sh issue-key codex-agent --store-client
  relay-api.sh issue-key codex-agent --scope project --project relay --store-client
  relay-api.sh capture payload.json
  cat payload.json | relay-api.sh promote -
  relay-api.sh raw GET /v1/projects/proj_xxx
EOF
}

read_keychain_secret() {
  local account="$1"
  if [[ "$(uname -s)" != "Darwin" ]]; then
    return 1
  fi
  if ! command -v security >/dev/null 2>&1; then
    return 1
  fi
  security find-generic-password -w -s "${KEYCHAIN_SERVICE}" -a "${account}" 2>/dev/null || return 1
}

store_keychain_secret() {
  local account="$1"
  local value="$2"
  if [[ "$(uname -s)" != "Darwin" ]]; then
    echo "Keychain storage requires macOS." >&2
    exit 1
  fi
  if ! security add-generic-password -U -s "${KEYCHAIN_SERVICE}" -a "${account}" -w "${value}" >/dev/null 2>&1; then
    echo "Failed to store Relay secret in Keychain (${KEYCHAIN_SERVICE}/${account})." >&2
    echo "This usually means the current session cannot access the login Keychain. Re-run locally in an interactive macOS session or use env vars instead." >&2
    exit 1
  fi
}

resolve_base_url() {
  if [[ -n "${BASE_URL}" ]]; then
    return 0
  fi
  if [[ -n "${RELAY_BASE_URL:-}" ]]; then
    BASE_URL="${RELAY_BASE_URL}"
    BASE_URL_SOURCE="env:RELAY_BASE_URL"
    return 0
  fi
  local keychain_base_url=""
  if keychain_base_url="$(read_keychain_secret "${KEYCHAIN_BASE_URL_ACCOUNT}")"; then
    BASE_URL="${keychain_base_url}"
    BASE_URL_SOURCE="keychain:${KEYCHAIN_SERVICE}/${KEYCHAIN_BASE_URL_ACCOUNT}"
    return 0
  fi
  BASE_URL="${DEFAULT_BASE_URL}"
  BASE_URL_SOURCE="default:${DEFAULT_BASE_URL}"
}

resolve_admin_token() {
  if [[ -n "${ADMIN_TOKEN}" ]]; then
    return 0
  fi
  if [[ -n "${RELAY_ADMIN_TOKEN:-}" ]]; then
    ADMIN_TOKEN="${RELAY_ADMIN_TOKEN}"
    ADMIN_TOKEN_SOURCE="env:RELAY_ADMIN_TOKEN"
    return 0
  fi
  if [[ -n "${RELAY_API_TOKEN:-}" ]]; then
    ADMIN_TOKEN="${RELAY_API_TOKEN}"
    ADMIN_TOKEN_SOURCE="env:RELAY_API_TOKEN"
    return 0
  fi
  local keychain_token=""
  if keychain_token="$(read_keychain_secret "${KEYCHAIN_ADMIN_ACCOUNT}")"; then
    ADMIN_TOKEN="${keychain_token}"
    ADMIN_TOKEN_SOURCE="keychain:${KEYCHAIN_SERVICE}/${KEYCHAIN_ADMIN_ACCOUNT}"
    return 0
  fi
  return 1
}

resolve_client_token() {
  if [[ -n "${CLIENT_TOKEN}" ]]; then
    return 0
  fi
  if [[ -n "${RELAY_CLIENT_TOKEN:-}" ]]; then
    CLIENT_TOKEN="${RELAY_CLIENT_TOKEN}"
    CLIENT_TOKEN_SOURCE="env:RELAY_CLIENT_TOKEN"
    return 0
  fi
  if [[ -n "${RELAY_MCP_TOKEN:-}" ]]; then
    CLIENT_TOKEN="${RELAY_MCP_TOKEN}"
    CLIENT_TOKEN_SOURCE="env:RELAY_MCP_TOKEN"
    return 0
  fi
  local keychain_token=""
  if keychain_token="$(read_keychain_secret "${KEYCHAIN_CLIENT_ACCOUNT}")"; then
    CLIENT_TOKEN="${keychain_token}"
    CLIENT_TOKEN_SOURCE="keychain:${KEYCHAIN_SERVICE}/${KEYCHAIN_CLIENT_ACCOUNT}"
    return 0
  fi
  return 1
}

require_admin_token() {
  if ! resolve_admin_token; then
    echo "Relay admin token not found. Set RELAY_ADMIN_TOKEN/RELAY_API_TOKEN or store it with scripts/setup.sh." >&2
    exit 1
  fi
}

require_client_token() {
  if ! resolve_client_token; then
    echo "Relay client token not found. Set RELAY_CLIENT_TOKEN/RELAY_MCP_TOKEN or store it with scripts/setup.sh or issue-key --store-client." >&2
    exit 1
  fi
}

json_data_arg() {
  local input="$1"
  if [[ "$input" == "-" ]]; then
    local tmp
    tmp="$(mktemp "${TMPDIR:-/tmp}/relay-api.XXXXXX.json")"
    cat >"$tmp"
    printf '%s\n' "$tmp"
    return 0
  fi
  printf '%s\n' "$input"
}

cleanup_temp() {
  local file="$1"
  if [[ -n "$file" && -f "$file" && "$(basename "$file")" == relay-api.*.json ]]; then
    rm -f "$file"
  fi
}

curl_request() {
  local token="$1"
  shift
  resolve_base_url
  local -a cmd=(curl --fail --silent --show-error)
  if [[ -n "$token" ]]; then
    cmd+=(-H "Authorization: Bearer ${token}")
  fi
  cmd+=("$@")
  "${cmd[@]}"
}

call_json() {
  local token="$1"
  local method="$2"
  local path="$3"
  local payload="${4:-}"
  resolve_base_url
  if [[ -n "$payload" ]]; then
    local payload_file
    local status=0
    payload_file="$(json_data_arg "$payload")"
    curl_request "$token" \
      -X "$method" \
      -H "Content-Type: application/json" \
      --data "@${payload_file}" \
      "${BASE_URL}${path}" || status=$?
    cleanup_temp "$payload_file"
    return "$status"
  fi
  curl_request "$token" -X "$method" "${BASE_URL}${path}"
}

json_get_field() {
  local field="$1"
  python3 -c 'import json,sys; data=json.load(sys.stdin); parts=sys.argv[1].split("."); cur=data
for part in parts:
    cur=cur[part]
print(cur)' "$field"
}

doctor() {
  resolve_base_url
  local health_body
  health_body="$(curl_request "" "${BASE_URL}/healthz")"
  echo "healthz ok (${BASE_URL_SOURCE})"
  echo "$health_body"

  if resolve_client_token; then
    local code body
    body="$(mktemp "${TMPDIR:-/tmp}/relay-api-doctor.XXXXXX.json")"
    code="$(curl --silent --show-error --output "$body" --write-out '%{http_code}' \
      -H "Authorization: Bearer ${CLIENT_TOKEN}" \
      "${BASE_URL}/v1/projects/proj_doctor_missing")"
    if [[ "$code" == "200" || "$code" == "404" ]]; then
      echo "client token usable (${CLIENT_TOKEN_SOURCE}, status=${code})"
    else
      echo "client token check failed (${CLIENT_TOKEN_SOURCE}, status=${code})" >&2
      cat "$body" >&2
      rm -f "$body"
      return 1
    fi
    rm -f "$body"

    body="$(mktemp "${TMPDIR:-/tmp}/relay-api-doctor.XXXXXX.json")"
    code="$(curl --silent --show-error --output "$body" --write-out '%{http_code}' \
      -X POST \
      -H "Authorization: Bearer ${CLIENT_TOKEN}" \
      -H "Content-Type: application/json" \
      -H "Accept: application/json, text/event-stream" \
      --data '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-05","capabilities":{},"clientInfo":{"name":"relay-api-doctor","version":"0.0.1"}}}' \
      "${BASE_URL}/mcp")"
    if [[ "$code" == "200" ]]; then
      echo "mcp initialize ok (${CLIENT_TOKEN_SOURCE}, status=${code})"
    else
      echo "mcp initialize failed (${CLIENT_TOKEN_SOURCE}, status=${code})" >&2
      cat "$body" >&2
      rm -f "$body"
      return 1
    fi
    rm -f "$body"
  else
    echo "client token not configured"
  fi

  if resolve_admin_token; then
    local code body
    body="$(mktemp "${TMPDIR:-/tmp}/relay-api-doctor.XXXXXX.json")"
    code="$(curl --silent --show-error --output "$body" --write-out '%{http_code}' \
      -H "Authorization: Bearer ${ADMIN_TOKEN}" \
      "${BASE_URL}/v1/api-keys")"
    if [[ "$code" == "200" ]]; then
      echo "admin token usable (${ADMIN_TOKEN_SOURCE}, status=${code})"
    else
      echo "admin token check failed (${ADMIN_TOKEN_SOURCE}, status=${code})" >&2
      cat "$body" >&2
      rm -f "$body"
      return 1
    fi
    rm -f "$body"
  else
    echo "admin token not configured"
  fi
}

issue_key() {
  local name="$1"
  local scope="${2:-}"
  local project="${3:-}"
  local project_id="${4:-}"
  local store_client="${5:-0}"
  require_admin_token
  if [[ -n "${project}" || -n "${project_id}" ]]; then
    if [[ "${scope}" != "project" ]]; then
      echo "project and project-id require --scope project" >&2
      exit 1
    fi
  fi
  if [[ "${scope}" == "project" && -z "${project}" && -z "${project_id}" ]]; then
    echo "--scope project requires --project or --project-id" >&2
    exit 1
  fi
  local payload
  payload="$(python3 - "$name" "$scope" "$project" "$project_id" <<'PY'
import json
import sys

name, scope, project, project_id = sys.argv[1:]
data = {"name": name}
if scope:
    data["scope"] = scope
if project:
    data["project"] = project
if project_id:
    data["project_id"] = project_id
print(json.dumps(data))
PY
)"
  local response
  response="$(call_json "${ADMIN_TOKEN}" POST /v1/api-keys/issue - <<<"${payload}")"
  if [[ "$store_client" == "1" ]]; then
    local token
    token="$(printf '%s' "$response" | json_get_field "data.token")"
    store_keychain_secret "${KEYCHAIN_CLIENT_ACCOUNT}" "${token}"
    echo "Stored Relay client token in Keychain (${KEYCHAIN_SERVICE}/${KEYCHAIN_CLIENT_ACCOUNT})." >&2
  fi
  printf '%s\n' "$response"
}

list_keys() {
  require_admin_token
  call_json "${ADMIN_TOKEN}" GET /v1/api-keys
}

revoke_key() {
  local key_id="$1"
  require_admin_token
  call_json "${ADMIN_TOKEN}" POST /v1/api-keys/revoke - <<EOF
{"key_id":"${key_id}"}
EOF
}

capture() {
  require_client_token
  call_json "${CLIENT_TOKEN}" POST /v1/capture "${1}"
}

promote() {
  require_client_token
  call_json "${CLIENT_TOKEN}" POST /v1/promote "${1}"
}

build_packet() {
  require_client_token
  call_json "${CLIENT_TOKEN}" POST /v1/packets/build "${1}"
}

show_project() {
  local project_id="$1"
  require_client_token
  call_json "${CLIENT_TOKEN}" GET "/v1/projects/${project_id}"
}

raw_request() {
  local method="$1"
  local path="$2"
  local payload="${3:-}"
  local use_admin="${4:-0}"
  if [[ "$use_admin" == "1" ]]; then
    require_admin_token
    call_json "${ADMIN_TOKEN}" "$method" "$path" "$payload"
    return 0
  fi
  require_client_token
  call_json "${CLIENT_TOKEN}" "$method" "$path" "$payload"
}

main() {
  local command="${1:-}"
  if [[ -z "$command" || "$command" == "-h" || "$command" == "--help" ]]; then
    usage
    exit 0
  fi

  case "$command" in
    doctor)
      doctor
      ;;
    issue-key)
      shift
      local store_client=0
      local scope=""
      local project=""
      local project_id=""
      local name="${1:-}"
      if [[ -z "$name" ]]; then
        echo "issue-key requires <name>" >&2
        exit 1
      fi
      shift
      while [[ $# -gt 0 ]]; do
        case "$1" in
          --store-client)
            store_client=1
            ;;
          --scope)
            shift
            scope="${1:?--scope requires global or project}"
            ;;
          --project)
            shift
            project="${1:?--project requires a project name}"
            ;;
          --project-id)
            shift
            project_id="${1:?--project-id requires a project id}"
            ;;
          --scope=*)
            scope="${1#*=}"
            ;;
          --project=*)
            project="${1#*=}"
            ;;
          --project-id=*)
            project_id="${1#*=}"
            ;;
          *)
            echo "Unknown issue-key option: $1" >&2
            exit 1
            ;;
        esac
        shift
      done
      issue_key "$name" "$scope" "$project" "$project_id" "$store_client"
      ;;
    list-keys)
      list_keys
      ;;
    revoke-key)
      revoke_key "${2:?revoke-key requires <key-id>}"
      ;;
    capture)
      capture "${2:?capture requires <json-file|->}"
      ;;
    promote)
      promote "${2:?promote requires <json-file|->}"
      ;;
    build-packet)
      build_packet "${2:?build-packet requires <json-file|->}"
      ;;
    show)
      show_project "${2:?show requires <project-id>}"
      ;;
    raw)
      shift
      local method="${1:?raw requires <METHOD>}"
      local path="${2:?raw requires <path>}"
      local payload=""
      local use_admin=0
      shift 2
      if [[ $# -gt 0 && "$1" != "--admin" ]]; then
        payload="$1"
        shift
      fi
      if [[ "${1:-}" == "--admin" ]]; then
        use_admin=1
      fi
      raw_request "$method" "$path" "$payload" "$use_admin"
      ;;
    *)
      echo "Unknown command: ${command}" >&2
      usage >&2
      exit 1
      ;;
  esac
}

main "$@"
