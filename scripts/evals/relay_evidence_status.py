#!/usr/bin/env python3
import argparse
import datetime
import json
from pathlib import Path


DEFAULT_ROOTS = [
    ".gstack/projects/relay",
    ".gstack/projects/relay-ci",
    ".gstack/projects/relay-consumer-stability",
]


def utc_now():
    return datetime.datetime.now(datetime.UTC).isoformat()


def load_status(path: Path):
    payload = json.loads(path.read_text())
    payload["_path"] = str(path)
    payload["_mtime"] = path.stat().st_mtime
    payload["_evidence_kind"] = evidence_kind(path, payload)
    return payload


def evidence_kind(path: Path, payload: dict):
    batch_id = str(payload.get("batch_id") or "")
    if batch_id.startswith("consumer-stability-"):
        return "consumer_usage_validation_batch"

    declared = payload.get("evidence_kind")
    if declared:
        return declared

    path_text = str(path)
    if "/stability/" in path_text:
        return "consumer_stability"
    if "/batches/" in path_text:
        return "usage_validation_gate"
    return "unknown"


def find_statuses(roots):
    statuses = []
    for root in roots:
        root_path = Path(root)
        if not root_path.exists():
            continue
        for path in root_path.rglob("run-status.json"):
            statuses.append(load_status(path))
    return sorted(statuses, key=lambda item: item["_mtime"], reverse=True)


def latest_by_kind(statuses):
    latest = {}
    for status in statuses:
        kind = status["_evidence_kind"]
        latest.setdefault(kind, status)
    return latest


def summarize(statuses, roots):
    latest = latest_by_kind(statuses)
    usage = latest.get("usage_validation_gate")
    consumer = latest.get("consumer_stability")

    usage_ready = bool(
        usage
        and usage.get("status") == "completed"
        and usage.get("canonical_benchmark_evidence") is True
        and usage.get("gate_pass") is True
    )
    consumer_ready = bool(
        consumer
        and consumer.get("status") == "completed"
        and consumer.get("canonical_benchmark_evidence") is True
    )

    next_actions = []
    if not usage:
        next_actions.append("Run usage-validation-gate to collect release evidence.")
    elif usage.get("status") == "blocked_by_model_limit":
        next_actions.append("Rerun usage-validation-gate after Claude judge capacity resets.")
    elif not usage_ready:
        reasons = usage.get("gate_reasons") or ["usage-validation gate did not pass"]
        next_actions.append("Fix usage-validation gate blockers: " + "; ".join(reasons))

    if not consumer:
        next_actions.append("Run consumer-stability when canonical consumer evidence is needed.")
    elif consumer.get("status") == "blocked_by_model_limit":
        next_actions.append("Rerun consumer-stability after Claude/Codex capacity resets.")
    elif not consumer_ready:
        next_actions.append("Inspect consumer-stability summary before promoting thresholds.")

    if usage_ready and consumer_ready:
        next_actions.append("Canonical release and consumer evidence are both present.")

    counts = {}
    for status in statuses:
        kind = status["_evidence_kind"]
        counts[kind] = counts.get(kind, 0) + 1

    return {
        "generated_at": utc_now(),
        "roots": roots,
        "total_status_files": len(statuses),
        "counts_by_evidence_kind": counts,
        "canonical_release_evidence": usage_ready,
        "canonical_consumer_evidence": consumer_ready,
        "latest_by_evidence_kind": {
            kind: public_status(status) for kind, status in latest.items()
        },
        "next_actions": next_actions,
    }


def public_status(status):
    hidden = {"_mtime"}
    return {key: value for key, value in status.items() if key not in hidden}


def render_markdown(summary):
    lines = [
        "# Relay Evidence Status",
        "",
        f"- generated_at: `{summary['generated_at']}`",
        f"- status files: `{summary['total_status_files']}`",
        f"- canonical release evidence: `{str(summary['canonical_release_evidence']).lower()}`",
        f"- canonical consumer evidence: `{str(summary['canonical_consumer_evidence']).lower()}`",
        "",
        "## Latest Status",
        "",
        "| evidence | status | canonical | gate | model | path |",
        "| --- | --- | --- | --- | --- | --- |",
    ]

    for kind, status in sorted(summary["latest_by_evidence_kind"].items()):
        canonical = status.get("canonical_benchmark_evidence")
        gate = status.get("gate_pass", "")
        model = status.get("judge_model") or status.get("codex_model") or ""
        lines.append(
            "| {kind} | {status} | {canonical} | {gate} | {model} | `{path}` |".format(
                kind=kind,
                status=status.get("status", "unknown"),
                canonical="" if canonical is None else str(canonical).lower(),
                gate="" if gate == "" else str(gate).lower(),
                model=model,
                path=status.get("_path", ""),
            )
        )

    lines.extend(["", "## Next Actions", ""])
    for action in summary["next_actions"]:
        lines.append(f"- {action}")
    return "\n".join(lines) + "\n"


def main():
    parser = argparse.ArgumentParser(description="Summarize Relay run-status evidence files.")
    parser.add_argument(
        "--root",
        action="append",
        dest="roots",
        help="Output root to scan. Can be passed multiple times. Defaults to known Relay roots.",
    )
    parser.add_argument("--json", action="store_true", help="Print JSON instead of Markdown.")
    parser.add_argument(
        "--strict-release",
        action="store_true",
        help="Exit non-zero unless canonical release evidence is present.",
    )
    parser.add_argument(
        "--strict-consumer",
        action="store_true",
        help="Exit non-zero unless canonical consumer evidence is present.",
    )
    args = parser.parse_args()

    roots = args.roots or DEFAULT_ROOTS
    statuses = find_statuses(roots)
    summary = summarize(statuses, roots)

    if args.json:
        print(json.dumps(summary, indent=2) + "\n", end="")
    else:
        print(render_markdown(summary), end="")

    if args.strict_release and not summary["canonical_release_evidence"]:
        raise SystemExit(1)
    if args.strict_consumer and not summary["canonical_consumer_evidence"]:
        raise SystemExit(1)


if __name__ == "__main__":
    main()
