#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

actual="$(
  RELAY_CLIENT_TOKEN=client-token bash -c '
    set -euo pipefail
    source "'"${SCRIPT_DIR}/canary.sh"'"
    resolve_client_token
    printf "%s" "$CLIENT_TOKEN"
  '
)"
if [[ "$actual" != "client-token" ]]; then
  echo "expected RELAY_CLIENT_TOKEN to win, got: $actual" >&2
  exit 1
fi

actual="$(
  RELAY_MCP_TOKEN=mcp-token bash -c '
    set -euo pipefail
    source "'"${SCRIPT_DIR}/canary.sh"'"
    resolve_client_token
    printf "%s" "$CLIENT_TOKEN"
  '
)"
if [[ "$actual" != "mcp-token" ]]; then
  echo "expected RELAY_MCP_TOKEN to resolve, got: $actual" >&2
  exit 1
fi

actual="$(
  RELAY_TOKEN=legacy-token bash -c '
    set -euo pipefail
    source "'"${SCRIPT_DIR}/canary.sh"'"
    if resolve_client_token; then
      printf "%s" "$CLIENT_TOKEN"
    else
      printf "missing"
    fi
  '
)"
if [[ "$actual" != "missing" ]]; then
  echo "expected RELAY_TOKEN to be ignored, got: $actual" >&2
  exit 1
fi
