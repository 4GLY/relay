# Relay

Relay is an API-first second-brain backend for long-running AI-assisted work.

The product surface is the HTTP API.

- `POST /v1/capture`
- `POST /v1/promote`
- `POST /v1/packets/build`
- `GET /v1/projects/{project_id}`

The local CLI still exists, but only as a dev/debug wrapper around the same service logic.

## Agent Skill

For local agent workflows, use the repo-owned skill wrapper:

```bash
./skills/relay-api-agent/scripts/setup.sh
./skills/relay-api-agent/scripts/relay-api.sh doctor
```

The skill keeps `docs/openapi.yaml` as the canonical contract and gives agents a stable wrapper for key issuance, capture, promote, packet build, and project inspection.

## MCP

Relay also exposes an MCP surface above the same API contract.

- `stdio`: local agent process integration through `cmd/relay-mcp`
- `http`: remote MCP integration served by the main `relay-api` process at `/mcp`

Run stdio MCP:

```bash
go run ./cmd/relay-mcp
```

Run the API with `/mcp` enabled:

```bash
go run ./cmd/relay-api
```

Notes:

- HTTP MCP uses streamable HTTP at `POST /mcp`.
- The deployed `/mcp` endpoint is stateless, which fits Relay's request-response tools.
- `RELAY_MCP_TOKEN` protects the remote MCP endpoint for multi-environment use.
- If `RELAY_MCP_TOKEN` is unset, Relay falls back to `RELAY_API_TOKEN`.
- The public `/mcp` surface is intentionally narrow:
  - `relay_health`
  - `relay_capture`
  - `relay_promote`
  - `relay_build_packet`
  - `relay_show_project`
- API key issue/list/revoke stays on the HTTP API and local skill, not the public MCP surface.
- Local `cmd/relay-mcp` can still expose admin tools for operator/debug workflows.

## Status

This repo currently includes:

- Go API
- PostgreSQL schema and embedded migrations
- Neon-backed deployment
- Bearer auth on all `/v1/*` routes
- OpenAPI spec and API contract docs

## API Contract

Start here:

- Contract guide: [docs/api.md](docs/api.md)
- MCP guide: [docs/mcp.md](docs/mcp.md)
- OpenAPI spec: [docs/openapi.yaml](docs/openapi.yaml)

Current production base URL:

- `https://relay.4gly.dev`

Auth model:

- `/healthz` is public
- every `/v1/*` route requires `Authorization: Bearer <token>`
- `RELAY_API_TOKEN` is the bootstrap admin token
- issued API keys can be minted through `POST /v1/api-keys/issue`
- issued API keys can be listed through `GET /v1/api-keys`
- issued API keys can be revoked through `POST /v1/api-keys/revoke`

## Environment

Create a local `.env` from `.env.example`.

```bash
cp .env.example .env
```

Fill in:

```bash
RELAY_ADDR=:8080
RELAY_BASE_URL='https://relay.4gly.dev'
RELAY_DATABASE_URL='postgresql://user:password@host/neondb?sslmode=require'
RELAY_API_TOKEN='replace-with-a-long-random-token'
RELAY_MCP_TOKEN='replace-with-a-separate-mcp-token'
```

`.env` is ignored by git.

## Bootstrap

Load env vars:

```bash
set -a; source .env; set +a
```

Apply migrations explicitly:

```bash
go run ./cmd/relay migrate
```

Start the API:

```bash
go run ./cmd/relay-api
```

The API also applies migrations automatically on startup and serves `/mcp`.

## API Smoke Test

Set a shell helper:

```bash
export RELAY_BASE_URL="${RELAY_BASE_URL:-http://127.0.0.1:8080}"
export RELAY_API_TOKEN="${RELAY_API_TOKEN:?missing RELAY_API_TOKEN}"
```

Health check:

```bash
curl -sS "$RELAY_BASE_URL/healthz"
```

Capture:

```bash
curl -sS -X POST "$RELAY_BASE_URL/v1/capture" \
  -H "Authorization: Bearer $RELAY_API_TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"project":"relay-api-smoke","source":"chat","body":"api smoke test","idempotency_key":"api-capture-1"}'
```

Promote:

```bash
curl -sS -X POST "$RELAY_BASE_URL/v1/promote" \
  -H "Authorization: Bearer $RELAY_API_TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"project":"relay-api-smoke","kind":"decision","summary":"Keep Neon as initial PG provider","reason":"Fastest path for Relay validation","idempotency_key":"api-promote-1"}'
```

Issue an API key:

```bash
curl -sS -X POST "$RELAY_BASE_URL/v1/api-keys/issue" \
  -H "Authorization: Bearer $RELAY_API_TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"name":"agent-smoke"}'
```

List API keys:

```bash
curl -sS "$RELAY_BASE_URL/v1/api-keys" \
  -H "Authorization: Bearer $RELAY_API_TOKEN"
```

Revoke an API key:

```bash
curl -sS -X POST "$RELAY_BASE_URL/v1/api-keys/revoke" \
  -H "Authorization: Bearer $RELAY_API_TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"key_id":"key_xxx"}'
```

Build packet:

```bash
curl -sS -X POST "$RELAY_BASE_URL/v1/packets/build" \
  -H "Authorization: Bearer $RELAY_API_TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"project":"relay-api-smoke","type":"resume","target":"codex"}'
```

Show by project id:

```bash
curl -sS \
  -H "Authorization: Bearer $RELAY_API_TOKEN" \
  "$RELAY_BASE_URL/v1/projects/<project_id>"
```

## Verification

Format and tests:

```bash
gofmt -w $(find . -name '*.go' -type f)
go test ./...
```

## Dev CLI

The CLI is not the primary product surface.

Keep it only for:

- local debugging
- migration bootstrap
- manual service smoke tests

Examples:

```bash
go run ./cmd/relay migrate
go run ./cmd/relay capture --stdin-json <<'EOF'
{"project":"relay-smoke","source":"chat","body":"hello from relay","idempotency_key":"smoke-capture-1"}
EOF
```
