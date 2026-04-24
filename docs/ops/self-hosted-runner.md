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

## Failure Modes

- `jump-relay-evals` offline: PRs block because the required check cannot be
  scheduled. Restart the service, then confirm the GitHub runner status is
  `online`.
- Docker disk pressure: jobs fail during service startup or image pulls. Check
  `docker system df` and prune only after confirming the runner is idle.
- Claude auth failure: `scripts/ci/run_usage_validation_gate.sh` fails before
  the batch judge starts. Rotate `CLAUDE_CODE_OAUTH_TOKEN` and rerun.
- Long busy state: confirm whether a job is actually running in GitHub Actions
  before restarting; restarting during a live job abandons the job.
