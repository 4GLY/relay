#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
KEYCHAIN_SERVICE="${RELAY_KEYCHAIN_SERVICE:-codex.relay-api}"
KEYCHAIN_ADMIN_ACCOUNT="${RELAY_KEYCHAIN_ADMIN_ACCOUNT:-admin-token}"
KEYCHAIN_CLIENT_ACCOUNT="${RELAY_KEYCHAIN_CLIENT_ACCOUNT:-client-token}"
KEYCHAIN_BASE_URL_ACCOUNT="${RELAY_KEYCHAIN_BASE_URL_ACCOUNT:-base-url}"

ADMIN_TOKEN="${RELAY_ADMIN_TOKEN:-${RELAY_API_TOKEN:-}}"
CLIENT_TOKEN="${RELAY_CLIENT_TOKEN:-${RELAY_TOKEN:-}}"
BASE_URL="${RELAY_BASE_URL:-https://relay.4gly.dev}"
CLIENT_NAME="${RELAY_CLIENT_NAME:-codex-agent}"
SKIP_ISSUE=0
VERIFY=1

usage() {
  cat <<EOF
Usage:
  setup.sh [--admin-token TOKEN] [--client-token TOKEN] [--client-name NAME] [--base-url URL] [--skip-issue] [--no-verify]

Stores Relay credentials in macOS Keychain and issues a client key by default.

Environment:
  RELAY_KEYCHAIN_SERVICE          Keychain service name (default: ${KEYCHAIN_SERVICE})
  RELAY_KEYCHAIN_ADMIN_ACCOUNT    Keychain account name for admin token (default: ${KEYCHAIN_ADMIN_ACCOUNT})
  RELAY_KEYCHAIN_CLIENT_ACCOUNT   Keychain account name for client token (default: ${KEYCHAIN_CLIENT_ACCOUNT})
  RELAY_KEYCHAIN_BASE_URL_ACCOUNT Keychain account name for base URL (default: ${KEYCHAIN_BASE_URL_ACCOUNT})
  RELAY_CLIENT_NAME               Default issued client key name (default: ${CLIENT_NAME})

Examples:
  ${SCRIPT_DIR}/setup.sh
  ${SCRIPT_DIR}/setup.sh --admin-token "<token>"
  ${SCRIPT_DIR}/setup.sh --admin-token "<token>" --client-name "codex-macbook"
  ${SCRIPT_DIR}/setup.sh --admin-token "<token>" --client-token "<token>"
EOF
}

require_macos_keychain() {
  if [[ "$(uname -s)" != "Darwin" ]]; then
    echo "setup.sh currently supports macOS Keychain only." >&2
    exit 1
  fi
  if ! command -v security >/dev/null 2>&1; then
    echo "security command not found; macOS Keychain integration is unavailable." >&2
    exit 1
  fi
}

prompt_secret() {
  local label="$1"
  local value=""
  read -r -s -p "${label}: " value
  echo >&2
  printf '%s' "$value"
}

store_secret() {
  local account="$1"
  local value="$2"
  if ! security add-generic-password -U -s "${KEYCHAIN_SERVICE}" -a "${account}" -w "${value}" >/dev/null 2>&1; then
    echo "Failed to store Relay secret in Keychain (${KEYCHAIN_SERVICE}/${account})." >&2
    echo "This usually means the current session cannot access the login Keychain. Re-run locally in an interactive macOS session." >&2
    exit 1
  fi
}

parse_args() {
  while [[ $# -gt 0 ]]; do
    case "$1" in
      --admin-token)
        ADMIN_TOKEN="${2:?admin token value required}"
        shift 2
        ;;
      --client-token)
        CLIENT_TOKEN="${2:?client token value required}"
        shift 2
        ;;
      --client-name)
        CLIENT_NAME="${2:?client name value required}"
        shift 2
        ;;
      --base-url)
        BASE_URL="${2:?base URL value required}"
        shift 2
        ;;
      --skip-issue)
        SKIP_ISSUE=1
        shift
        ;;
      --no-verify)
        VERIFY=0
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
}

main() {
  parse_args "$@"
  require_macos_keychain

  if [[ -z "${ADMIN_TOKEN}" ]]; then
    ADMIN_TOKEN="$(prompt_secret "Relay admin token")"
  fi

  if [[ -z "${ADMIN_TOKEN}" ]]; then
    echo "Relay admin token is required." >&2
    exit 1
  fi

  store_secret "${KEYCHAIN_ADMIN_ACCOUNT}" "${ADMIN_TOKEN}"
  echo "Stored Relay admin token in Keychain (${KEYCHAIN_SERVICE}/${KEYCHAIN_ADMIN_ACCOUNT})."

  if [[ -n "${BASE_URL}" ]]; then
    store_secret "${KEYCHAIN_BASE_URL_ACCOUNT}" "${BASE_URL}"
    echo "Stored Relay base URL in Keychain (${KEYCHAIN_SERVICE}/${KEYCHAIN_BASE_URL_ACCOUNT})."
  fi

  if [[ -n "${CLIENT_TOKEN}" ]]; then
    store_secret "${KEYCHAIN_CLIENT_ACCOUNT}" "${CLIENT_TOKEN}"
    echo "Stored Relay client token in Keychain (${KEYCHAIN_SERVICE}/${KEYCHAIN_CLIENT_ACCOUNT})."
  elif [[ "${SKIP_ISSUE}" -eq 0 ]]; then
    echo "Issuing Relay client token '${CLIENT_NAME}' and storing it in Keychain..."
    RELAY_BASE_URL="${BASE_URL}" \
    RELAY_ADMIN_TOKEN="${ADMIN_TOKEN}" \
    RELAY_KEYCHAIN_SERVICE="${KEYCHAIN_SERVICE}" \
    RELAY_KEYCHAIN_ADMIN_ACCOUNT="${KEYCHAIN_ADMIN_ACCOUNT}" \
    RELAY_KEYCHAIN_CLIENT_ACCOUNT="${KEYCHAIN_CLIENT_ACCOUNT}" \
    RELAY_KEYCHAIN_BASE_URL_ACCOUNT="${KEYCHAIN_BASE_URL_ACCOUNT}" \
      "${SCRIPT_DIR}/relay-api.sh" issue-key "${CLIENT_NAME}" --store-client >/dev/null
    echo "Issued and stored Relay client token in Keychain (${KEYCHAIN_SERVICE}/${KEYCHAIN_CLIENT_ACCOUNT})."
  else
    echo "Skipped client key issuance. Run relay-api.sh issue-key <name> --store-client later."
  fi

  if [[ "${VERIFY}" -eq 1 ]]; then
    "${SCRIPT_DIR}/relay-api.sh" doctor
  fi
}

main "$@"
