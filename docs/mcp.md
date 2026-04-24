# Relay MCP

Relay exposes a narrow MCP surface for agents that need shared memory tools.

This page is for MCP consumers.
Use it when you want to connect an agent to Relay over `stdio` or remote `HTTP`.

## Base Endpoint

- production: `https://relay.4gly.dev/mcp`
- local: `http://127.0.0.1:8080/mcp`

## Example Consumers

- raw HTTP `tools/list`: [`examples/mcp/http/tools-list.sh`](../examples/mcp/http/tools-list.sh)
- raw HTTP `tools/call`: [`examples/mcp/http/call-tool.sh`](../examples/mcp/http/call-tool.sh)
- Go client with official `go-sdk`: [`examples/mcp/go/main.go`](../examples/mcp/go/main.go)

## End-to-End Session

This is the shortest useful remote MCP session:

1. `relay_capture` stores raw memory and returns `project_id` plus any created note ids.
2. `relay_promote` turns that raw note into a durable decision or question.
3. `relay_build_packet` produces an agent-ready `resume` packet.
4. `relay_show_project` checks aggregate state using the canonical `project_id`.
5. `relay_retrieve_project` surfaces query-conditioned context before the next packet or action.

Example with the bundled raw HTTP helper:

```bash
set -euo pipefail

export RELAY_CLIENT_TOKEN=...
PROJECT="relay-mcp-e2e-docs"

CAPTURE_JSON="$(
  ./examples/mcp/http/call-tool.sh relay_capture "$(
    jq -nc \
      --arg project "$PROJECT" \
      --arg source "chat" \
      --arg body "Document an end-to-end MCP session for Relay." \
      '{
        project: $project,
        source: $source,
        body: $body,
        idempotency_key: "docs-e2e-capture-001"
      }'
  )"
)"

PROJECT_ID="$(printf '%s' "$CAPTURE_JSON" | jq -r '.result.structuredContent.project_id')"
NOTE_ID="$(printf '%s' "$CAPTURE_JSON" | jq -r '.result.structuredContent.created_note_ids[0]')"

PROMOTE_JSON="$(
  ./examples/mcp/http/call-tool.sh relay_promote "$(
    jq -nc \
      --arg project "$PROJECT" \
      --arg note_id "$NOTE_ID" \
      '{
        project: $project,
        kind: "decision",
        summary: "Relay exposes a narrow public MCP surface.",
        reason: "Consumers should use a small stable tool set.",
        source_note_ids: [$note_id],
        idempotency_key: "docs-e2e-promote-002"
      }'
  )"
)"

PACKET_JSON="$(
  ./examples/mcp/http/call-tool.sh relay_build_packet "$(
    jq -nc \
      --arg project "$PROJECT" \
      '{project: $project}'
  )"
)"

SHOW_JSON="$(
  ./examples/mcp/http/call-tool.sh relay_show_project "$(
    jq -nc \
      --arg project_id "$PROJECT_ID" \
      '{project_id: $project_id}'
  )"
)"

printf '%s\n' "$CAPTURE_JSON" | jq '.result.structuredContent'
printf '%s\n' "$PROMOTE_JSON" | jq '.result.structuredContent'
printf '%s\n' "$PACKET_JSON" | jq '.result.structuredContent'
printf '%s\n' "$SHOW_JSON" | jq '.result.structuredContent'
```

Expected shape:

- `relay_capture`
  - `project_id`
  - `created_note_ids`
  - `created_artifact_ids`
- `relay_promote`
  - `kind`
  - `object_id`
  - `project_id`
- `relay_build_packet`
  - `packet_id`
  - `snapshot_id` when `persist_snapshot` is true
  - `project_id`
  - `schema_version`
  - `type`
  - `target`
  - `task_summary`
  - `body`
  - `rendered_body`
  - `style_cues`
  - `supporting_notes`
  - `supporting_decisions`
  - `supporting_questions`
  - `supporting_artifacts`
  - `why_included`
  - `decision_ids`
  - `open_question_ids`
  - `source_artifact_ids`
  - `approved_heuristic_ids`
  - `missing_context`
- `relay_show_project`
  - `project_id`
  - `note_count`
  - `artifact_count`
  - `decision_count`
  - `open_question_count`
  - `latest_packet_id`

## Auth

- remote `/mcp` requires `Authorization: Bearer <token>`
- remote `/mcp` accepts the same bearer policy as `/v1/*`
- use an issued API key for normal remote agent access
- `RELAY_CLIENT_TOKEN` is the issued client token for normal MCP access
- `RELAY_MCP_TOKEN` remains a compatibility alias, but it must also be an issued client token

## Public Tool Surface

`tools/list` on the deployed endpoint is intentionally small.

- `relay_health`
- `relay_capture`
- `relay_promote`
- `relay_build_packet`
- `relay_show_project`
- `relay_retrieve_project`

API key issue, list, and revoke are not part of the public MCP surface.
Use the HTTP API or the local skill for those operator tasks.

Style-memory mutation is also intentionally not part of the public MCP surface.
Agents consume approved style memory through `relay_build_packet`; proposal creation, approval, and heuristic updates stay on the HTTP API/operator path.

## Tool Guide

### `relay_health`

Use:
- verify reachability
- confirm the resolved Relay base URL

Input:
- none

Output:
- `status`
- `base_url`
- `admin_enabled`

### `relay_capture`

Use:
- store raw working memory
- attach optional repo or document artifacts

Minimum input:

```json
{
  "note": "The user wants one Relay shared across remote environments."
}
```

Optional fields:
- `project`
- `repo_path`
- `handoff_path`
- `design_path`
- `extra_artifacts`
- `source`
- `body`
- `idempotency_key`

Notes:
- always send `idempotency_key` on retries or automated writes
- `project` is optional; if omitted, the server may infer it from `repo_path`
- `repo_path`, `handoff_path`, and `design_path` are convenience aliases for common trusted artifacts
- `extra_artifacts` is the general evidence list for items such as `code_path`, `changed_files`, and `pr_diff`
- `source` defaults to `manual`
- `body` is optional; `note` is accepted as an alias for the stored text

### `relay_promote`

Use:
- turn raw memory into a durable decision
- record an open question that still blocks work

Decision example:

```json
{
  "project": "relay",
  "kind": "decision",
  "summary": "Relay serves remote MCP from the main API process.",
  "reason": "Deployment stays simple while implementation remains layered.",
  "idempotency_key": "promote-remote-mcp-001"
}
```

Question example:

```json
{
  "project": "relay",
  "kind": "question",
  "summary": "Should packet formatting become target-specific in v1?"
}
```

Optional fields:
- `reason`
- `source_note_ids`
- `source_artifact_ids`
- `idempotency_key`

Notes:
- `kind` must be `decision` or `question`
- `reason` is required for `decision`

### `relay_build_packet`

Use:
- generate an agent-ready summary packet from stored Relay memory

Minimum input:

```json
{
  "project": "relay"
}
```

Optional fields:
- `type`
- `target`
- `workflow`
- `artifact_type`
- `task_summary`
- `disable_style_cues`
- `disable_retrieval`
- `persist_snapshot`
- `idempotency_key`

Defaults:
- `type`: `resume`
- `target`: `codex`

Style-memory behavior:
- approved heuristics can be returned as `style_cues`
- `workflow` and `artifact_type` narrow style-cue selection
- `task_summary` helps rank which supporting artifacts are most relevant for the packet
- `disable_style_cues` builds a normal packet without style cues
- `disable_retrieval` builds a ranking-only packet without query-conditioned retrieval
- `persist_snapshot` returns a `snapshot_id` for deterministic replay
- each `style_cue` now includes `canonical_text` and `why_included`
- packet evidence is exposed through `supporting_notes`, `supporting_decisions`, `supporting_questions`, `supporting_artifacts`, and `why_included`
- packet output now includes `retrieval_mode` so callers can tell whether the packet was `query-conditioned` or `ranking-only`

Output:
- `packet_id`
- `snapshot_id`
- `project_id`
- `schema_version`
- `type`
- `target`
- `task_summary`
- `body`
- `rendered_body`
- `style_cues`
- `supporting_notes`
- `supporting_decisions`
- `supporting_questions`
- `supporting_artifacts`
- `why_included`
- `decision_ids`
- `open_question_ids`
- `source_artifact_ids`
- `approved_heuristic_ids`
- `missing_context`

### `relay_show_project`

Use:
- inspect aggregate project state
- fetch the canonical `project_id` counts after capture or promotion

Input:

```json
{
  "project_id": "proj_xxx"
}
```

Output:
- `project_id`
- `note_count`
- `artifact_count`
- `decision_count`
- `open_question_count`
- `latest_packet_id`

### `relay_retrieve_project`

Use:
- retrieve query-conditioned context before planning the next step
- inspect which notes, artifacts, decisions, and open questions currently match a task

Input:

```json
{
  "project_id": "proj_xxx",
  "query": "continue api packet boundary work",
  "limit": 8
}
```

Output:
- `project_id`
- `query`
- `hits`

Notes:
- this is the first public retrieval layer for agents
- `decision` and `open_question` hits can surface because their linked notes or artifacts matched the query
- `decision` and `open_question` hits can also surface with weaker inferred-support boosts when there is no canonical link yet
- `why_included` explains the ranking signal for each hit

## Transport Notes

### Remote HTTP

Use:
- shared Relay across multiple remote environments
- public endpoint protected by bearer auth

Behavior:
- `POST /mcp`
- streamable HTTP
- stateless

Quick raw example:

```bash
RELAY_CLIENT_TOKEN=... \
./examples/mcp/http/tools-list.sh
```

Quick Go example:

```bash
RELAY_CLIENT_TOKEN=... \
go run ./examples/mcp/go
```

### Local stdio

Use:
- local debugging
- local operator workflows

Entrypoint:

```bash
go run ./cmd/relay-mcp
```

The local stdio entrypoint may expose extra admin tools when the local environment has admin credentials.
