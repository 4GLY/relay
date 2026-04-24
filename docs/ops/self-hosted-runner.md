# Relay Self-Hosted Eval Runner

Relay keeps the GitHub-hosted checks light and runs Claude-heavy usage
validation on a self-hosted Linux runner.

## Current Runner

- Host alias: `jump`
- Hostname: `hoon-ch.orca-fir.ts.net`
- Runner name: `jump-relay-evals`
- Labels: `self-hosted`, `Linux`, `X64`, `relay-evals`
- Service: `actions.runner.4GLY-relay.jump-relay-evals.service`
- Runner directory: `~/actions-runner`

The required usage-validation workflow targets:

```yaml
runs-on: [self-hosted, Linux, X64, relay-evals]
```

## Health Checks

Check GitHub runner registration:

```bash
gh api repos/4GLY/relay/actions/runners \
  --jq '.runners[] | select(.name=="jump-relay-evals")'
```

Check the systemd service:

```bash
ssh jump 'sudo systemctl status actions.runner.4GLY-relay.jump-relay-evals.service --no-pager'
```

Restart the runner when it is offline or stuck:

```bash
ssh jump 'sudo systemctl restart actions.runner.4GLY-relay.jump-relay-evals.service'
```

Inspect recent runner logs:

```bash
ssh jump 'sudo journalctl -u actions.runner.4GLY-relay.jump-relay-evals.service -n 200 --no-pager'
```

## Docker Maintenance

The usage-validation gate starts disposable services and can leave image/cache
pressure on the runner.

Check disk usage:

```bash
ssh jump 'docker system df'
ssh jump 'df -h'
```

Clean Docker state only when no Actions job is running:

```bash
ssh jump 'docker system prune -af --volumes'
```

## Claude Auth Rotation

The runner uses repository secret `CLAUDE_CODE_OAUTH_TOKEN` for headless
`claude` CLI evaluation. `ANTHROPIC_API_KEY` is not required for the current
gate.

Rotate the token from a trusted local machine:

```bash
claude setup-token
gh secret set CLAUDE_CODE_OAUTH_TOKEN --repo 4GLY/relay --body "$CLAUDE_CODE_OAUTH_TOKEN"
```

Then rerun the `usage-validation` workflow on a PR.

## Codex Auth

The manual `consumer-stability` workflow also runs Codex as a packet consumer.
It installs `@openai/codex` during the job, but authentication is owned by the
trusted jump runner account instead of a repository `OPENAI_API_KEY` secret.

Current setup:

```bash
ssh jump 'test -f /home/hoon-ch/.codex/auth.json'
```

The workflow sets `CODEX_HOME=/home/hoon-ch/.codex` and fails fast when the
runner user's Codex login is missing or expired. The workflow installs the
Codex CLI binary after `actions/setup-node`, so the host shell does not need a
global `codex` binary for normal operation. Rotate Codex auth on `jump` with
the normal interactive Codex login flow for the `hoon-ch` account.

Then run the workflow manually:

```bash
gh workflow run consumer-stability.yml -f runs=3 -f fixture_limit=1 -f judge_model=opus
```

## Model Limits

`consumer-stability` uses real Claude and Codex consumer calls, so it can be
blocked by provider usage limits. Treat usage limits as eval capacity failures,
not Relay product failures.

Fallback order:

1. Lower cost first: use fewer `runs`, keep `fixture_limit=1`, and rerun after
   the provider limit resets.
2. If Codex is limited, keep the PR/release decision on `usage-validation-gate`
   and postpone the Codex consumer comparison.
3. If Claude/Opus is limited, either wait for reset or explicitly choose a
   cheaper `judge_model`; mark that run exploratory.
4. If both are limited, run only deterministic packet/API checks and do not
   update consumer-stability thresholds from that run.

Do not silently swap providers or accounts for canonical stability numbers.
If a substitute model is used, make it explicit in workflow inputs and artifact
metadata.

The Claude structured-output helper detects common limit messages such as
`You've hit your limit` and stops retrying immediately, because short retries
cannot fix provider quota exhaustion. It returns exit code `75` for this
capacity failure. The required PR workflow catches that code, marks the run as
degraded in the step summary, runs deterministic `go test ./...` with CI
service env cleared, and exits successfully so product changes are not blocked
by temporary model capacity. Those degraded runs are not canonical benchmark
evidence.

The manual `consumer-stability` workflow also catches exit code `75`, writes
`stability/<prefix>/run-status.json` with `status=blocked_by_model_limit`, and
exits successfully. That keeps the workflow useful for verifying runner setup
and Codex auth even when model capacity prevents a canonical stability result.

## Failure Modes

- `jump-relay-evals` offline: PRs block because the required check cannot be
  scheduled. Restart the service, then confirm the GitHub runner status is
  `online`.
- Docker disk pressure: jobs fail during service startup or image pulls. Check
  `docker system df` and prune only after confirming the runner is idle.
- Claude auth failure: `scripts/ci/run_usage_validation_gate.sh` fails before
  the batch judge starts. Rotate `CLAUDE_CODE_OAUTH_TOKEN` and rerun.
- Codex auth failure: `consumer-stability` fails before the stability run
  starts. Refresh the `hoon-ch` Codex login on `jump` and confirm
  `/home/hoon-ch/.codex/auth.json` exists. If a Codex binary is available in
  the shell, also run `CODEX_HOME=/home/hoon-ch/.codex codex login status`.
- Model usage limit: reduce `runs`/`fixture_limit`, wait for reset, or mark the
  substituted model run as exploratory. Do not mix it into canonical stability
  thresholds.
- Long busy state: confirm whether a job is actually running in GitHub Actions
  before restarting; restarting during a live job abandons the job.
