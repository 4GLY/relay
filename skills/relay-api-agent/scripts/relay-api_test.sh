#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RELAY_API_SH="${SCRIPT_DIR}/relay-api.sh"

fail() {
  echo "FAIL: $*" >&2
  exit 1
}

make_stubbed_path() {
  local tmpdir
  tmpdir="$(mktemp -d "${TMPDIR:-/tmp}/relay-api-test.XXXXXX")"

  cat >"${tmpdir}/curl" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail

log_file="${RELAY_API_TEST_LOG:?}"
printf '%s\n' "$*" >>"${log_file}"

url="${@: -1}"
output=""
for ((i = 1; i <= $#; i++)); do
  if [[ "${!i}" == "--output" ]]; then
    next=$((i + 1))
    output="${!next}"
    break
  fi
done

case "$url" in
  */healthz)
    printf '%s' '{"status":"ok"}'
    ;;
  */v1/projects/proj_doctor_missing)
    if [[ -n "${output}" ]]; then
      printf '%s' '{"ok":false,"command":"relay show","error":{"code":"FORBIDDEN","message":"forbidden","retryable":false,"missing_fields":[]}}' >"${output}"
    fi
    printf '403'
    ;;
  */mcp)
    if [[ -n "${output}" ]]; then
      printf '%s' '{"jsonrpc":"2.0","id":1,"result":{}}' >"${output}"
    fi
    printf '200'
    ;;
  */v1/api-keys)
    if [[ -n "${output}" ]]; then
      printf '%s' '{"ok":true,"command":"relay api-key list","data":{"items":[]},"warnings":[]}' >"${output}"
    fi
    printf '200'
    ;;
  */v1/api-keys/issue)
    if [[ -n "${output}" ]]; then
      printf '%s' '{"ok":true,"command":"relay api-key issue","data":{"token":"relay_live_scoped"},"warnings":[]}' >"${output}"
    fi
    printf '200'
    ;;
  *)
    echo "unexpected curl url: ${url}" >&2
    exit 1
    ;;
esac
EOF
  chmod +x "${tmpdir}/curl"

  cat >"${tmpdir}/security" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
echo "security helper should not be called for this test" >&2
exit 99
EOF
  chmod +x "${tmpdir}/security"

  printf '%s\n' "${tmpdir}"
}

test_doctor_accepts_scoped_token_403() {
  local tmpdir logfile out
  tmpdir="$(make_stubbed_path)"
  logfile="${tmpdir}/curl.log"
  : >"${logfile}"
  out="${tmpdir}/out"

  if ! PATH="${tmpdir}:${PATH}" \
    RELAY_BASE_URL="https://relay.example" \
    RELAY_ADMIN_TOKEN="admin-token" \
    RELAY_CLIENT_TOKEN="relay_live_project_scoped" \
    RELAY_MCP_TOKEN="" \
    RELAY_API_TEST_LOG="${logfile}" \
    "${RELAY_API_SH}" doctor >"${out}" 2>"${tmpdir}/err"; then
    cat "${tmpdir}/err" >&2
    fail "doctor should accept a scoped token that returns 403 on the fixed missing-project probe"
  fi

  if ! grep -q 'client token usable' "${out}"; then
    cat "${out}" >&2
    fail "doctor did not report the client token as usable"
  fi
  if ! grep -q 'status=403' "${out}"; then
    cat "${out}" >&2
    fail "doctor did not accept the 403 scoped-token probe"
  fi
}

test_issue_key_store_client_rejects_project_scoped_keys() {
  local tmpdir logfile out err
  tmpdir="$(make_stubbed_path)"
  logfile="${tmpdir}/curl.log"
  : >"${logfile}"
  out="${tmpdir}/out"
  err="${tmpdir}/err"

  if PATH="${tmpdir}:${PATH}" \
    RELAY_BASE_URL="https://relay.example" \
    RELAY_ADMIN_TOKEN="admin-token" \
    RELAY_API_TEST_LOG="${logfile}" \
    "${RELAY_API_SH}" issue-key scoped-agent --scope project --project relay --store-client >"${out}" 2>"${err}"; then
    fail "issue-key should reject --store-client for project-scoped keys"
  fi

  if ! grep -q -- '--store-client only applies to global client tokens' "${err}"; then
    cat "${err}" >&2
    fail "expected a clear rejection message for scoped keys"
  fi
  if [[ -s "${logfile}" ]]; then
    cat "${logfile}" >&2
    fail "issue-key should not call the API when --store-client is invalid for scoped keys"
  fi
}

main() {
  test_doctor_accepts_scoped_token_403
  test_issue_key_store_client_rejects_project_scoped_keys
  echo "relay-api skill shell tests passed"
}

main "$@"
