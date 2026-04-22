#!/usr/bin/env bash
set -euo pipefail

BASE_URL="${RELAY_BASE_URL:-https://relay.4gly.dev}"
MCP_URL="${RELAY_MCP_URL:-${BASE_URL%/}/mcp}"
CLIENT_TOKEN="${RELAY_CLIENT_TOKEN:-${RELAY_MCP_TOKEN:-${RELAY_TOKEN:-}}}"
ADMIN_TOKEN="${RELAY_ADMIN_TOKEN:-${RELAY_API_TOKEN:-}}"
KEEP_ISSUED_KEY=0
TEMP_KEY_ID=""
TEMP_KEY_TOKEN=""

usage() {
  cat <<EOF
Usage:
  canary.sh [--base-url URL] [--mcp-url URL] [--client-token TOKEN] [--admin-token TOKEN] [--keep-issued-key]

Checks Relay production canaries:
  1. GET /healthz
  2. issued key lookup or temporary key issuance
  3. POST /mcp initialize
  4. POST /mcp tools/list
  5. POST /mcp tools/call relay_health

Environment:
  RELAY_BASE_URL                       HTTP API base URL
  RELAY_MCP_URL                        MCP endpoint URL (default: \$RELAY_BASE_URL/mcp)
  RELAY_CLIENT_TOKEN / RELAY_MCP_TOKEN / RELAY_TOKEN
                                       Normal issued API key
  RELAY_ADMIN_TOKEN / RELAY_API_TOKEN  Bootstrap admin token used only if a temporary key must be issued
EOF
}

json_get() {
  local path="$1"
  python3 -c 'import json,sys
data=json.load(sys.stdin)
cur=data
for part in sys.argv[1].split("."):
    if part.isdigit():
        cur=cur[int(part)]
    else:
        cur=cur[part]
print(cur)' "$path"
}

curl_json() {
  local token="$1"
  local method="$2"
  local url="$3"
  local body="${4:-}"
  local -a cmd=(curl --fail --silent --show-error -X "$method" "$url")
  if [[ -n "$token" ]]; then
    cmd+=(-H "Authorization: Bearer ${token}")
  fi
  if [[ -n "$body" ]]; then
    cmd+=(-H "Content-Type: application/json" --data "$body")
  fi
  "${cmd[@]}"
}

issue_temp_key() {
  if [[ -z "${ADMIN_TOKEN}" ]]; then
    echo "No client token found, and admin token is missing so canary cannot issue a temporary key." >&2
    exit 1
  fi

  local name="canary-$(hostname -s 2>/dev/null || echo relay)-$(date +%s)"
  local response
  response="$(curl_json "${ADMIN_TOKEN}" POST "${BASE_URL%/}/v1/api-keys/issue" "{\"name\":\"${name}\"}")"
  TEMP_KEY_ID="$(printf '%s' "$response" | json_get "data.key_id")"
  TEMP_KEY_TOKEN="$(printf '%s' "$response" | json_get "data.token")"
  CLIENT_TOKEN="${TEMP_KEY_TOKEN}"
  echo "issued temporary client key: ${TEMP_KEY_ID}"
}

cleanup() {
  if [[ -n "${TEMP_KEY_ID}" && "${KEEP_ISSUED_KEY}" -eq 0 && -n "${ADMIN_TOKEN}" ]]; then
    curl_json "${ADMIN_TOKEN}" POST "${BASE_URL%/}/v1/api-keys/revoke" "{\"key_id\":\"${TEMP_KEY_ID}\"}" >/dev/null || true
    echo "revoked temporary client key: ${TEMP_KEY_ID}"
  fi
}

parse_args() {
  while [[ $# -gt 0 ]]; do
    case "$1" in
      --base-url)
        BASE_URL="${2:?base URL value required}"
        shift 2
        ;;
      --mcp-url)
        MCP_URL="${2:?MCP URL value required}"
        shift 2
        ;;
      --client-token)
        CLIENT_TOKEN="${2:?client token value required}"
        shift 2
        ;;
      --admin-token)
        ADMIN_TOKEN="${2:?admin token value required}"
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

main() {
  parse_args "$@"
  trap cleanup EXIT

  local health
  health="$(curl_json "" GET "${BASE_URL%/}/healthz")"
  printf 'healthz ok: %s\n' "$(printf '%s' "$health" | json_get "data.status")"

  if [[ -z "${CLIENT_TOKEN}" ]]; then
    issue_temp_key
  else
    echo "using provided client token"
  fi

  local initialize
  initialize="$(curl --fail --silent --show-error -X POST "${MCP_URL}" \
    -H "Authorization: Bearer ${CLIENT_TOKEN}" \
    -H "Content-Type: application/json" \
    -H "Accept: application/json, text/event-stream" \
    --data '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-05","capabilities":{},"clientInfo":{"name":"relay-canary","version":"0.0.1"}}}')"
  printf 'mcp initialize ok: %s\n' "$(printf '%s' "$initialize" | json_get "result.serverInfo.name")"

  local tools
  tools="$(curl_json "${CLIENT_TOKEN}" POST "${MCP_URL}" '{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}')"
  local tool_count
  tool_count="$(printf '%s' "$tools" | python3 -c 'import json,sys; print(len(json.load(sys.stdin)["result"]["tools"]))')"
  printf 'tools/list ok: %s tools\n' "${tool_count}"

  local health_tool
  health_tool="$(curl_json "${CLIENT_TOKEN}" POST "${MCP_URL}" '{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"relay_health","arguments":{}}}')"
  printf 'relay_health ok: %s\n' "$(printf '%s' "$health_tool" | json_get "result.structuredContent.status")"
}

main "$@"
