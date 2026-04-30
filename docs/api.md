# Relay API Contract

Relay is API-first.

The HTTP API is the product contract.
The CLI is only a local dev/debug wrapper.

## Base URLs

- local: `http://127.0.0.1:8080`
- production: `https://relay.4gly.dev`

## Public Snapshot Surface

`GET /p/{token}` and `GET /p/{token}/og.png` are served by the Go API and are the
canonical S7 public snapshot surface. They own public-token lookup, revocation,
OG metadata, and OG PNG bytes. The Next `web/app/p/[snapshotId]` route is only a
redirect fallback when a request reaches the web app instead of the API gateway.

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

## Curator Worker

Relay's V1 curator is a separate proposal-only worker process.

Run:

```bash
go run ./cmd/relay-worker
```

Environment:
- `RELAY_DATABASE_URL`: required
- `RELAY_CURATOR_WORKER_ID`: optional worker lease owner, defaults to `relay-curator`
- `RELAY_CURATOR_PROVIDER`: optional provider, currently `rule-based`
- `RELAY_CURATOR_BATCH_SIZE`: optional claim batch size, defaults to `5`
- `RELAY_CURATOR_POLL_INTERVAL`: optional poll interval, defaults to `5s`
- `RELAY_CURATOR_LEASE_DURATION`: optional queue lease duration, defaults to `30s`
- `RELAY_CURATOR_RETRY_BACKOFF`: optional retry base backoff, defaults to `30s`
- `RELAY_CURATOR_MAX_ATTEMPTS`: optional max attempts before `failed`, defaults to `5`

Contract notes:
- writing a `judgment_trace` enqueues curator work when the queue store is available
- the worker can only emit `heuristic_proposals`
- the worker never creates or mutates `approved_heuristics`
- API and MCP packet building remain usable when the worker is down

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
  "extra_artifacts": [
    {
      "type": "code_path",
      "source_path": "internal/services/packets.go",
      "trust_level": "trusted"
    },
    {
      "type": "pr_diff",
      "source_path": "scripts/evals/fixtures/artifacts/pr-diffs/session-without-summary.md"
    }
  ],
  "note": "user said offline matters",
  "idempotency_key": "capture-001"
}
```

Contract notes:
- `project` is optional; when omitted, capture can infer from `repo_path` for normal flows or use the bound project when it safely matches a project-scoped key
- `repo_path`, `handoff_path`, and `design_path` remain convenience aliases for common trusted artifacts
- `extra_artifacts` is the general path for attaching additional evidence such as changed-files manifests, code paths, and PR diffs
- `body` is optional; `note` is accepted as an alias for the stored memory text
- `source` is optional and defaults to `manual`
- each `extra_artifacts[]` item requires `type` and `source_path`; `trust_level` defaults to `trusted`
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

### `POST /v1/judgment-traces`

Purpose:
- record a concrete user/agent judgment that can later become a reusable style heuristic
- preserve why a decision was made, not only what was decided

Request body:

```json
{
  "project": "relay",
  "task_id": "task-001",
  "agent_id": "codex",
  "workflow": "design_handoff",
  "artifact_type": "design_doc",
  "decision": "Prefer explicit contracts over implicit inference.",
  "rationale": "Keeps model-to-model handoff deterministic.",
  "alternatives": ["Let agents infer the contract from chat history"],
  "constraints": ["Same-project V1 proof first"],
  "source_refs": ["docs/research/context-graph-and-semantic-retrieval.md"],
  "language": "en",
  "idempotency_key": "trace-001"
}
```

Response fields:
- `trace_id`
- `project_id`
- `curator_job_id`

Contract notes:
- `task_id`, `agent_id`, `decision`, and `rationale` are required
- `workflow` defaults to `design_handoff`
- `artifact_type` defaults to `design_doc`
- `idempotency_key` should be supplied by agents on writes

### `POST /v1/heuristic-proposals`

Purpose:
- propose a reusable style heuristic from one or more judgment traces
- keep proposal creation separate from approval

Request body:

```json
{
  "project": "relay",
  "origin_trace_id": "trace_xxx",
  "workflow": "design_handoff",
  "artifact_type": "design_doc",
  "heuristic_key": "explicit_contracts_over_magic",
  "canonical_text": "Prefer explicit contracts over magic inference.",
  "source_trace_ids": ["trace_xxx"],
  "source_refs": ["docs/design.md"],
  "proposed_by": "manual",
  "idempotency_key": "proposal-001"
}
```

Response fields:
- `proposal_id`
- `project_id`
- `state`

Contract notes:
- new proposals start as `pending`
- approval is a separate review step
- `origin_trace_id` is added to `source_trace_ids` when omitted there

### `POST /v1/heuristic-proposals/review`

Purpose:
- approve, reject, or archive a heuristic proposal
- create an approved heuristic only on `approve`

Request body:

```json
{
  "project": "relay",
  "proposal_id": "hprop_xxx",
  "action": "approve",
  "review_notes": "Matches the V1 handoff contract."
}
```

Contract notes:
- bootstrap admin token is required
- `action` is `approve`, `reject`, or `archive`
- `approve` returns `approved_heuristic_id`
- `reject` and `archive` update only proposal state

### `POST /v1/approved-heuristics/update`

Purpose:
- disable, archive, re-enable, or re-approve an approved heuristic

Request body:

```json
{
  "project": "relay",
  "heuristic_id": "heur_xxx",
  "action": "disable"
}
```

Contract notes:
- bootstrap admin token is required
- `action` is `disable`, `archive`, `approve`, or `enable`
- issued client keys cannot approve or mutate approved heuristics

### `POST /v1/packets/build`

Purpose:
- generate an agent-ready packet from stored memory

Request body:

```json
{
  "project": "relay",
  "type": "resume",
  "target": "codex",
  "workflow": "design_handoff",
  "artifact_type": "design_doc",
  "task_summary": "continue the same-project model-to-model handoff proof",
  "persist_snapshot": true,
  "idempotency_key": "packet-001"
}
```

Contract notes:
- current `type` is effectively `resume`
- current `target` is free-form, but `codex` is the primary path
- `task_summary` is used both in the rendered packet body and to rank which supporting artifacts are most relevant to include
- `disable_retrieval` turns off query-conditioned retrieval and keeps a ranking-only baseline for notes, decisions, questions, and artifacts
- approved style heuristics are returned as `style_cues` unless `disable_style_cues` is true
- packet output now includes `supporting_notes`, `supporting_decisions`, `supporting_questions`, `supporting_artifacts`, and `why_included`
- each `style_cue` now carries the approved heuristic `canonical_text` and `why_included`
- `persist_snapshot` writes an immutable packet snapshot and returns `snapshot_id`
- packet output includes `schema_version`, `rendered_body`, `approved_heuristic_ids`, `missing_context`, and `retrieval_mode`

### `GET /v1/projects/{project_id}/packet-snapshots/latest`

Purpose:
- read the latest immutable packet snapshot for the project workspace
- power the Packet Builder WYSIWYG first slice without exposing packet source panels by default

Query params:
- `type`: optional packet kind; defaults to `resume`
- `target`: optional packet target; defaults to `codex`

Contract notes:
- requires project access through a session cookie or bearer auth
- response includes the rendered packet body, supporting evidence arrays, missing context, and immutable snapshot ids
- `public_readable` and `public_token` are included so UI can link to an already published snapshot without calling admin-only publish/revoke routes

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

### `GET /v1/projects/{project_id}/graph`

Purpose:
- project-scoped canonical graph projection for retrieval and packet-composer planning

Path params:
- `project_id`: canonical project id, not project name

Response fields:
- `project_id`
- `nodes[]`
- `edges[]`

Contract notes:
- this is a read-only projection over the existing relational stores; no separate graph database is involved
- current node kinds are `project`, `note`, `artifact`, `decision`, `open_question`, `judgment_trace`, `heuristic_proposal`, `approved_heuristic`, and latest `packet_snapshot`
- current canonical edge types are `includes` and `derived_from`
- inferred candidate edge types are `possible_support` and `possible_answer`
- project containment is emitted as `includes`
- Style Memory and packet evidence edges use `derived_from` to point back to traces, proposals, heuristics, decisions, questions, and artifacts when those source nodes are present in the projection
- inferred edges carry `status=candidate`, `score`, and `why_included`
- full packet history listing is still outside this graph slice; the graph includes only the latest packet snapshot evidence node

### `GET /v1/projects/{project_id}/explorer`

Purpose:
- return the first V2.5 Project Explorer read model for an authenticated user

Auth:
- bootstrap admin bearer, issued bearer token, or `relay_session` cookie
- session callers must own the project
- project-scoped keys must be bound to the same project

Path params:
- `project_id`: canonical project id, not project name

Response fields:
- `project`: `project_id`, `name`, `status`
- `counts`: notes, artifacts, decisions, open questions, judgment traces, pending proposals, approved heuristics, rejected proposals, packet snapshots
- `latest_snapshot`: latest packet snapshot summary when present; includes `public_token` only when the snapshot is public-readable
- `style_memory`: preview of the next pending proposal when present
- `recent_activity`: compact recent judgment trace / approved heuristic activity

Contract notes:
- this is a read-only aggregation endpoint; mutation stays on Style Memory and packet APIs
- full packet content stays on `GET /v1/projects/{project_id}/packet-snapshots/latest`
- proposal counts currently use the Style Memory stores and are capped by the first read-model slice, not cross-project analytics

### `GET /v1/projects/{project_id}/judgment-traces`

Purpose:
- list compact judgment trace cards for Project Explorer and the later Trace Browser surface

Auth:
- bootstrap admin bearer, issued bearer token, or `relay_session` cookie
- session callers must own the project
- project-scoped keys must be bound to the same project

Path params:
- `project_id`: canonical project id, not project name

Query params:
- `limit`: optional max item count, defaults to `20`, max `100`
- `cursor`: accepted for forward compatibility; pagination cursor emission is not implemented in this first slice

Response fields:
- `items[]`
- `next_cursor`

Item fields:
- `trace_id`
- `project_id`
- `task_id`
- `agent_id`
- `workflow`
- `artifact_type`
- `decision`
- `rationale`
- `source_refs`
- `created_at`

### `GET /v1/projects/{project_id}/retrieve`

Purpose:
- query-conditioned retrieval across project notes, artifacts, decisions, and open questions

Path params:
- `project_id`: canonical project id, not project name

Query params:
- `query`: required task or question text
- `limit`: optional max hit count, defaults to `12`

Response fields:
- `project_id`
- `query`
- `hits[]`

Contract notes:
- this is the first semantic-retrieval layer, implemented as a graph-complement ranking pass over existing project memory
- current hits are lexical and provenance-aware, not embedding-backed yet
- `decision` and `open_question` hits can receive extra score when their linked notes or artifacts also match the query
- `decision` and `open_question` hits can also receive a smaller boost from inferred candidate edges when relevant notes or artifacts match the query
- `kind` is one of `note`, `artifact`, `decision`, `open_question`
- `why_included` explains why a hit surfaced for the current query

## Stable Error Codes

Current known codes:

- `INVALID_JSON`
- `INVALID_LIMIT`
- `UNAUTHORIZED`
- `FORBIDDEN`
- `PROJECT_ID_REQUIRED`
- `PROJECT_NOT_FOUND`
- `MISSING_REQUIRED_FIELDS`
- `IDEMPOTENCY_CONFLICT`
- `INVALID_HEURISTIC_REVIEW_ACTION`
- `INVALID_HEURISTIC_ACTION`
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
