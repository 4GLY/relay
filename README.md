# Relay

Relay is an agent-driven second-brain backend for long-running AI-assisted work.

Current shape:

- `agent CLI -> Relay API -> PostgreSQL`
- Postgres canonical store
- public CLI surface:
  - `relay capture`
  - `relay promote`
  - `relay packet build`
  - `relay show`

## Status

This repo currently includes:

- Go CLI scaffold
- Go API scaffold
- PostgreSQL schema and embedded migrations
- Neon-backed local verification for:
  - `capture`
  - `promote`
  - `packet build`
  - `show`

## Requirements

- Go `1.25.5`
- PostgreSQL-compatible database
- a `RELAY_DATABASE_URL`

Neon works out of the box. For Neon, prefer the direct Postgres connection string with `sslmode=require`.

## Environment

Create a local `.env` from `.env.example`.

```bash
cp .env.example .env
```

Fill in:

```bash
RELAY_ADDR=:8080
RELAY_DATABASE_URL='postgresql://user:password@host/neondb?sslmode=require'
RELAY_API_TOKEN='replace-with-a-long-random-token'
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

The API also applies migrations automatically on startup.

## CLI Smoke Test

Capture a note:

```bash
go run ./cmd/relay capture --stdin-json <<'EOF'
{"project":"relay-smoke","source":"chat","body":"hello from relay","idempotency_key":"smoke-capture-1"}
EOF
```

Promote a decision:

```bash
go run ./cmd/relay promote --stdin-json <<'EOF'
{"project":"relay-smoke","kind":"decision","summary":"Use Neon first","reason":"Fastest path for validation","idempotency_key":"smoke-promote-1"}
EOF
```

Build a resume packet:

```bash
go run ./cmd/relay packet build --stdin-json <<'EOF'
{"project":"relay-smoke","type":"resume","target":"codex"}
EOF
```

Show project state by name:

```bash
go run ./cmd/relay show --stdin-json <<'EOF'
{"project":"relay-smoke"}
EOF
```

## API Smoke Test

Health check:

```bash
curl -sS http://127.0.0.1:8080/healthz
```

Set a local shell helper for protected routes:

```bash
export RELAY_API_TOKEN="${RELAY_API_TOKEN:?missing RELAY_API_TOKEN}"
```

Capture:

```bash
curl -sS -X POST http://127.0.0.1:8080/v1/capture \
  -H "Authorization: Bearer $RELAY_API_TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"project":"relay-api-smoke","source":"chat","body":"api smoke test","idempotency_key":"api-capture-1"}'
```

Promote:

```bash
curl -sS -X POST http://127.0.0.1:8080/v1/promote \
  -H "Authorization: Bearer $RELAY_API_TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"project":"relay-api-smoke","kind":"decision","summary":"Keep Neon as initial PG provider","reason":"Fastest path for Relay validation","idempotency_key":"api-promote-1"}'
```

Build packet:

```bash
curl -sS -X POST http://127.0.0.1:8080/v1/packets/build \
  -H "Authorization: Bearer $RELAY_API_TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"project":"relay-api-smoke","type":"resume","target":"codex"}'
```

Show by project id:

```bash
curl -sS \
  -H "Authorization: Bearer $RELAY_API_TOKEN" \
  http://127.0.0.1:8080/v1/projects/<project_id>
```

Note:

- `/healthz` stays open
- all `/v1/*` routes require `Authorization: Bearer <token>` when `RELAY_API_TOKEN` is set
- CLI `show` is name-based
- API `GET /v1/projects/{project_id}` is id-based

## Current Schema

Current tables:

- `projects`
- `notes`
- `artifacts`
- `decisions`
- `open_questions`
- `packets`
- `schema_migrations`

## Verification

Format and tests:

```bash
gofmt -w $(find . -name '*.go' -type f)
go test ./...
```

## Next

Likely next steps:

- improve packet formatting and provenance detail
- document or implement deployment shape for Neon + API hosting
