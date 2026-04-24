#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo "v1_copilot_paired_judge.sh is deprecated; forwarding to v1_claude_paired_judge.sh" >&2
exec "${SCRIPT_DIR}/v1_claude_paired_judge.sh" "$@"
