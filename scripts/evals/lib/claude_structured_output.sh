#!/usr/bin/env bash

run_claude_structured_output() {
  local model="$1"
  local schema_json="$2"
  local prompt_file="$3"
  local raw_response_file="$4"
  local label="${5:-claude structured output}"
  local max_attempts="${RELAY_EVAL_CLAUDE_STRUCTURED_MAX_ATTEMPTS:-3}"
  local retry_sleep_seconds="${RELAY_EVAL_CLAUDE_STRUCTURED_RETRY_SLEEP_SECONDS:-2}"

  if ! [[ "$max_attempts" =~ ^[0-9]+$ ]] || (( max_attempts == 0 )); then
    echo "RELAY_EVAL_CLAUDE_STRUCTURED_MAX_ATTEMPTS must be a positive integer" >&2
    return 1
  fi

  local attempt attempt_response_file
  for (( attempt = 1; attempt <= max_attempts; attempt++ )); do
    attempt_response_file="$raw_response_file"
    if (( attempt > 1 )); then
      attempt_response_file="${raw_response_file%.json}.attempt-${attempt}.json"
    fi

    if claude \
      -p \
      --output-format json \
      --model "$model" \
      --tools "" \
      --json-schema "$schema_json" \
      "$(cat "$prompt_file")" >"$attempt_response_file" \
      && jq -e '.structured_output' "$attempt_response_file" >/dev/null; then
      if [[ "$attempt_response_file" != "$raw_response_file" ]]; then
        cp "$attempt_response_file" "$raw_response_file"
      fi
      if (( attempt > 1 )); then
        echo "${label}: structured output succeeded on attempt ${attempt}/${max_attempts}" >&2
      fi
      return 0
    fi

    cp "$attempt_response_file" "$raw_response_file" 2>/dev/null || true
    if (( attempt < max_attempts )); then
      local failure_reason
      failure_reason="$(jq -r '.subtype // (.errors[0]? // empty) // .type // "unknown structured-output failure"' "$attempt_response_file" 2>/dev/null || printf 'invalid claude response')"
      echo "${label}: attempt ${attempt}/${max_attempts} failed (${failure_reason}); retrying" >&2
      sleep "$retry_sleep_seconds"
    fi
  done

  echo "${label}: failed to produce valid structured output after ${max_attempts} attempts" >&2
  return 1
}
