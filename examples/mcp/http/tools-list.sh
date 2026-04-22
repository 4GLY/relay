#!/usr/bin/env bash
set -euo pipefail

MCP_URL="${RELAY_MCP_URL:-https://relay.4gly.dev/mcp}"
MCP_TOKEN="${RELAY_CLIENT_TOKEN:-${RELAY_MCP_TOKEN:-}}"

if [[ -z "${MCP_TOKEN}" ]]; then
  echo "RELAY_CLIENT_TOKEN or RELAY_MCP_TOKEN is required" >&2
  exit 1
fi

curl -sS -X POST "${MCP_URL}" \
  -H "Authorization: Bearer ${MCP_TOKEN}" \
  -H 'Content-Type: application/json' \
  -d '{
    "jsonrpc": "2.0",
    "id": 1,
    "method": "tools/list",
    "params": {}
  }'
