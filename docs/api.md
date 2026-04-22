# Relay API Contract

Relay is API-first.

The HTTP API is the product contract.
The CLI is only a local dev/debug wrapper.

## Base URLs

- local: `http://127.0.0.1:8080`
- production: `https://relay.4gly.dev`

## Auth

- `GET /healthz` is public
- every `/v1/*` route requires `Authorization: Bearer <token>`
- `RELAY_ADMIN_TOKEN` is the preferred bootstrap admin token
- `RELAY_API_TOKEN` remains a legacy bootstrap fallback for admin startup and local compatibility
- `RELAY_CLIENT_TOKEN` is the issued client token for normal API use
- issued API keys are stored server-side as token hashes
- `POST /v1/api-keys/issue` accepts only the bootstrap admin token
- `GET /v1/api-keys` accepts only the bootstrap admin token
- `POST /v1/api-keys/revoke` accepts only the bootstrap admin token

Example:

```bash
curl -sS \
  -H "Authorization: Bearer $RELAY_CLIENT_TOKEN" \
  https://relay.4gly.dev/v1/projects/<project_id>
```

## Envelope Shape

Successful responses:

```json
{
  "ok": true,
  "command": "relay packet build",
  "data": {},
  "warnings": []
}
```

Failed responses:

```json
{
  "ok": false,
  "command": "relay show",
  "error": {
    "code": "PROJECT_NOT_FOUND",
    "message": "project not found",
    "retryable": false,
    "missing_fields": []
  }
}
```

## Endpoints

### `GET /healthz`

Purpose:
- liveness/readiness

Auth:
- none

Response:
- `200`

### `POST /v1/api-keys/issue`

Purpose:
- mint a new reusable API key for agents or clients
- optionally bind the key to a single project

Auth:
- bootstrap admin token only

Request body:

```json
{
  "name": "agent-smoke",
  "scope": "project",
  "project": "relay"
}
```

Response body:

```json
{
  "ok": true,
  "command": "relay api-key issue",
  "data": {
    "key_id": "key_xxx",
    "name": "agent-smoke",
    "token": "relay_live_xxx",
    "token_prefix": "relay_live_xxx",
    "scope": "project",
    "project_id": "proj_xxx"
  },
  "warnings": []
}
```

Contract notes:
- plaintext token is returned once
- only the hash is stored in Postgres
- issued keys can be used on the normal `/v1/*` routes
- `scope` defaults to `global`
- `scope: project` requires `project` or `project_id`
- `project` and `project_id` must resolve to the same project when both are present

### `GET /v1/api-keys`

Purpose:
- inspect issued API keys

Auth:
- bootstrap admin token only

Response body:

```json
{
  "ok": true,
  "command": "relay api-key list",
  "data": {
    "items": [
      {
        "key_id": "key_xxx",
        "name": "agent-smoke",
        "token_prefix": "relay_live_xxx",
        "scope": "project",
        "project_id": "proj_xxx",
        "revoked": false
      }
    ]
  },
  "warnings": []
}
```

### `POST /v1/api-keys/revoke`

Purpose:
- revoke a previously issued API key

Auth:
- bootstrap admin token only

Request body:

```json
{
  "key_id": "key_xxx"
}
```

Contract notes:
- revoked keys stop working on normal `/v1/*` routes
- bootstrap admin token is not affected
- list and revoke responses include the key scope and project binding when present

### `POST /v1/capture`

Purpose:
- store raw memory and optional artifacts
- optionally attach repo or document artifacts

Request body:

```json
{
  "repo_path": ".",
  "handoff_path": "docs/handoff.md",
  "design_path": "docs/design.md",
  "note": "user said offline matters",
  "idempotency_key": "capture-001"
}
```

Contract notes:
- `project` is optional; when omitted, capture can infer from `repo_path` for normal flows or use the bound project when it safely matches a project-scoped key
- `body` is optional; `note` is accepted as an alias for the stored memory text
- `source` is optional and defaults to `manual`
- `idempotency_key` should be supplied by agents on writes

### `POST /v1/promote`

Purpose:
- promote raw memory into a durable decision or open question

Request body:

```json
{
  "project": "relay",
  "kind": "decision",
  "summary": "Relay is API-first",
  "reason": "CLI has no product-only capability",
  "source_note_ids": ["note_123"],
  "source_artifact_ids": [],
  "idempotency_key": "promote-001"
}
```

Contract notes:
- `kind` is currently `decision` or `question`
- `summary` is the durable statement
- `reason` is why it was chosen

### `POST /v1/packets/build`

Purpose:
- generate an agent-ready packet from stored memory

Request body:

```json
{
  "project": "relay",
  "type": "resume",
  "target": "codex"
}
```

Contract notes:
- current `type` is effectively `resume`
- current `target` is free-form, but `codex` is the primary path

### `GET /v1/projects/{project_id}`

Purpose:
- inspect current aggregate project state

Path params:
- `project_id`: canonical project id, not project name

Response fields:
- `project_id`
- `note_count`
- `artifact_count`
- `decision_count`
- `open_question_count`
- `latest_packet_id`

## Stable Error Codes

Current known codes:

- `INVALID_JSON`
- `UNAUTHORIZED`
- `PROJECT_ID_REQUIRED`
- `PROJECT_NOT_FOUND`
- `MISCONFIGURED`
- `INTERNAL_ERROR`

## Idempotency Policy

Current write endpoints accept `idempotency_key`:

- `POST /v1/capture`
- `POST /v1/promote`

Expectation:
- agents should always send one on write operations
- read/build operations do not require one

## Source of Truth

If this file and implementation diverge, the finish line is:

1. implementation
2. `docs/openapi.yaml`
3. this contract doc

All three should move together.
