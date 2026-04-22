---
name: relay-api-agent
description: Use when an agent needs direct, scriptable access to the Relay API for capturing memory, promoting decisions or questions, building packets, showing project state, or managing Relay API keys through the repo-owned shell wrapper.
---

# Relay API Agent

Use this skill when you want predictable raw access to Relay from an agent workflow.

Relay is API-first. This skill is the fast agent-facing wrapper around the canonical HTTP API in `docs/openapi.yaml`.

## Safe Start

Run setup once:

```bash
/Users/hoon-ch/repos/relay/skills/relay-api-agent/scripts/setup.sh
```

Then validate:

```bash
/Users/hoon-ch/repos/relay/skills/relay-api-agent/scripts/relay-api.sh doctor
```

`setup.sh` and `issue-key --store-client` write to macOS Keychain.
If the current session cannot access the login Keychain, use env vars instead and rerun setup locally.

The helper resolves settings in this order.

Base URL:
1. `RELAY_BASE_URL`
2. macOS Keychain entry `codex.relay-api/base-url`
3. default `https://relay.4gly.dev`

Admin token:
1. `RELAY_ADMIN_TOKEN`
2. `RELAY_API_TOKEN`
3. macOS Keychain entry `codex.relay-api/admin-token`

Client token:
1. `RELAY_CLIENT_TOKEN`
2. `RELAY_TOKEN`
3. macOS Keychain entry `codex.relay-api/client-token`
4. admin token fallback

## When To Use

- Capture chat or working notes into Relay
- Promote durable decisions or open questions
- Build `resume` packets for another agent
- Show aggregate project state by `project_id`
- Issue, list, or revoke Relay API keys
- Debug Relay HTTP responses without hand-writing curl every time

## Quick Reference

| Operation | Command |
| --- | --- |
| Health check | `relay-api.sh doctor` |
| Issue API key | `relay-api.sh issue-key <name>` |
| List API keys | `relay-api.sh list-keys` |
| Revoke API key | `relay-api.sh revoke-key <key-id>` |
| Capture memory | `relay-api.sh capture <json-file|->` |
| Promote | `relay-api.sh promote <json-file|->` |
| Build packet | `relay-api.sh build-packet <json-file|->` |
| Show project | `relay-api.sh show <project-id>` |
| Raw request | `relay-api.sh raw <METHOD> <path> [json-file|-]` |

## Core Workflows

Validate health and auth:

```bash
/Users/hoon-ch/repos/relay/skills/relay-api-agent/scripts/relay-api.sh doctor
```

Issue a new client token and store it in Keychain:

```bash
/Users/hoon-ch/repos/relay/skills/relay-api-agent/scripts/relay-api.sh issue-key codex-agent --store-client
```

Capture a note:

```bash
cat <<'JSON' >/tmp/relay-capture.json
{
  "project": "relay",
  "source": "chat",
  "body": "user said API-first is the product surface",
  "idempotency_key": "capture-001"
}
JSON

/Users/hoon-ch/repos/relay/skills/relay-api-agent/scripts/relay-api.sh capture /tmp/relay-capture.json
```

Promote a decision:

```bash
cat <<'JSON' >/tmp/relay-promote.json
{
  "project": "relay",
  "kind": "decision",
  "summary": "Relay is API-first",
  "reason": "CLI is dev-only",
  "idempotency_key": "promote-001"
}
JSON

/Users/hoon-ch/repos/relay/skills/relay-api-agent/scripts/relay-api.sh promote /tmp/relay-promote.json
```

Build a packet:

```bash
cat <<'JSON' >/tmp/relay-packet.json
{
  "project": "relay",
  "type": "resume",
  "target": "codex"
}
JSON

/Users/hoon-ch/repos/relay/skills/relay-api-agent/scripts/relay-api.sh build-packet /tmp/relay-packet.json
```

Show a project:

```bash
/Users/hoon-ch/repos/relay/skills/relay-api-agent/scripts/relay-api.sh show proj_xxx
```

## Guidelines

- Treat `docs/openapi.yaml` as the canonical wire contract.
- Use admin token only for `issue-key`, `list-keys`, and `revoke-key`.
- Prefer client tokens for normal agent operations.
- Always send write payloads as JSON files or stdin, not ad-hoc shell quoting.
- Keep `idempotency_key` on write operations.
- Do not store plaintext tokens in repo files.

## Files

- [scripts/relay-api.sh](/Users/hoon-ch/repos/relay/skills/relay-api-agent/scripts/relay-api.sh)
- [scripts/setup.sh](/Users/hoon-ch/repos/relay/skills/relay-api-agent/scripts/setup.sh)
- [docs/api.md](/Users/hoon-ch/repos/relay/docs/api.md)
- [docs/openapi.yaml](/Users/hoon-ch/repos/relay/docs/openapi.yaml)
