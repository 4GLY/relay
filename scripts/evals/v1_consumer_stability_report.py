#!/usr/bin/env python3
import argparse
import glob
import json
import math
from collections import Counter, defaultdict
from pathlib import Path


def load_json(path: Path):
    return json.loads(path.read_text())


def mean(values):
    return None if not values else sum(values) / len(values)


def stdev(values):
    if len(values) < 2:
        return 0.0 if values else None
    avg = mean(values)
    return math.sqrt(sum((value - avg) ** 2 for value in values) / (len(values) - 1))


def metric_stats(values):
    return {
        "count": len(values),
        "min": None if not values else min(values),
        "max": None if not values else max(values),
        "avg": mean(values),
        "stdev": stdev(values),
    }


def collect_runs(batch_summaries):
    runs = []
    for summary_path in batch_summaries:
        summary = load_json(summary_path)
        for run in summary.get("runs", []):
            if run.get("consumer_preferred_consumer") is None:
                continue
            runs.append(
                {
                    "batch_id": summary.get("batch_id"),
                    "summary_file": str(summary_path),
                    "scenario_label": run.get("scenario_label"),
                    "consumer_preferred_consumer": run.get("consumer_preferred_consumer"),
                    "consumer_codex_packet_only": run.get("consumer_codex_packet_only"),
                    "consumer_claude_packet_only": run.get("consumer_claude_packet_only"),
                    "consumer_codex_style_match": run.get("consumer_codex_style_match"),
                    "consumer_claude_style_match": run.get("consumer_claude_style_match"),
                    "consumer_codex_continuation_readiness": run.get("consumer_codex_continuation_readiness"),
                    "consumer_claude_continuation_readiness": run.get("consumer_claude_continuation_readiness"),
                    "consumer_comparison_file": run.get("consumer_comparison_file"),
                    "result_file": run.get("result_file"),
                }
            )
    return runs


def build_summary(batch_summary_paths):
    runs = collect_runs(batch_summary_paths)
    packet_only_passes = [
        run
        for run in runs
        if run["consumer_codex_packet_only"] is True and run["consumer_claude_packet_only"] is True
    ]
    preferred_counts = Counter(run["consumer_preferred_consumer"] for run in runs)

    metric_fields = {
        "codex_style_match": "consumer_codex_style_match",
        "claude_style_match": "consumer_claude_style_match",
        "codex_continuation_readiness": "consumer_codex_continuation_readiness",
        "claude_continuation_readiness": "consumer_claude_continuation_readiness",
    }
    metrics = {
        name: metric_stats([run[field] for run in runs if run.get(field) is not None])
        for name, field in metric_fields.items()
    }

    scenario_runs = defaultdict(list)
    for run in runs:
        scenario_runs[run["scenario_label"]].append(run)
    scenarios = []
    for scenario, items in sorted(scenario_runs.items()):
        scenarios.append(
            {
                "scenario_label": scenario,
                "runs": len(items),
                "packet_only_pass_rate": (
                    sum(
                        1
                        for item in items
                        if item["consumer_codex_packet_only"] is True
                        and item["consumer_claude_packet_only"] is True
                    )
                    / len(items)
                ),
                "preferred_consumer_counts": dict(Counter(item["consumer_preferred_consumer"] for item in items)),
                "codex_style_match": metric_stats(
                    [item["consumer_codex_style_match"] for item in items if item["consumer_codex_style_match"] is not None]
                ),
                "claude_style_match": metric_stats(
                    [item["consumer_claude_style_match"] for item in items if item["consumer_claude_style_match"] is not None]
                ),
                "codex_continuation_readiness": metric_stats(
                    [
                        item["consumer_codex_continuation_readiness"]
                        for item in items
                        if item["consumer_codex_continuation_readiness"] is not None
                    ]
                ),
                "claude_continuation_readiness": metric_stats(
                    [
                        item["consumer_claude_continuation_readiness"]
                        for item in items
                        if item["consumer_claude_continuation_readiness"] is not None
                    ]
                ),
            }
        )

    candidate_ready = len(runs) >= 3
    threshold_candidate = {
        "ready": candidate_ready,
        "reason": "at least 3 consumer continuation runs are required"
        if not candidate_ready
        else "candidate uses the minimum observed score from this stability sample",
        "min_packet_only_pass_rate": 1.0 if candidate_ready else None,
        "min_codex_style_match": metrics["codex_style_match"]["min"] if candidate_ready else None,
        "min_claude_style_match": metrics["claude_style_match"]["min"] if candidate_ready else None,
        "min_codex_continuation_readiness": metrics["codex_continuation_readiness"]["min"] if candidate_ready else None,
        "min_claude_continuation_readiness": metrics["claude_continuation_readiness"]["min"] if candidate_ready else None,
    }

    return {
        "batch_summary_files": [str(path) for path in batch_summary_paths],
        "batch_count": len(batch_summary_paths),
        "consumer_continuation_runs": len(runs),
        "packet_only_passes": len(packet_only_passes),
        "packet_only_pass_rate": None if not runs else len(packet_only_passes) / len(runs),
        "preferred_consumer_counts": dict(preferred_counts),
        "metrics": metrics,
        "threshold_candidate": threshold_candidate,
        "scenarios": scenarios,
        "runs": runs,
    }


def fmt(value, digits=2):
    if value is None:
        return "n/a"
    if isinstance(value, float):
        return f"{value:.{digits}f}"
    return str(value)


def render_markdown(summary):
    candidate = summary["threshold_candidate"]
    lines = [
        "# Relay Consumer Continuation Stability",
        "",
        "## Overview",
        "",
        f"- batch summaries: `{summary['batch_count']}`",
        f"- consumer continuation runs: `{summary['consumer_continuation_runs']}`",
        f"- packet-only pass rate: `{fmt(None if summary['packet_only_pass_rate'] is None else summary['packet_only_pass_rate'] * 100)}%`"
        if summary["packet_only_pass_rate"] is not None
        else "- packet-only pass rate: `n/a`",
        f"- preferred consumer counts: `{json.dumps(summary['preferred_consumer_counts'], sort_keys=True)}`",
        "",
        "## Metrics",
        "",
        "| metric | count | min | avg | max | stdev |",
        "| --- | --- | --- | --- | --- | --- |",
    ]
    for metric_name, stats in summary["metrics"].items():
        lines.append(
            f"| {metric_name} | {stats['count']} | {fmt(stats['min'])} | {fmt(stats['avg'])} | {fmt(stats['max'])} | {fmt(stats['stdev'])} |"
        )

    lines.extend(
        [
            "",
            "## Threshold Candidate",
            "",
            f"- ready: `{str(candidate['ready']).lower()}`",
            f"- reason: {candidate['reason']}",
            f"- min packet-only pass rate: `{fmt(candidate['min_packet_only_pass_rate'])}`",
            f"- min Codex style_match: `{fmt(candidate['min_codex_style_match'])}`",
            f"- min Claude style_match: `{fmt(candidate['min_claude_style_match'])}`",
            f"- min Codex continuation readiness: `{fmt(candidate['min_codex_continuation_readiness'])}`",
            f"- min Claude continuation readiness: `{fmt(candidate['min_claude_continuation_readiness'])}`",
            "",
            "## Per Scenario",
            "",
            "| scenario | runs | packet-only | preferred | codex style min/avg | claude style min/avg | codex ready min/avg | claude ready min/avg |",
            "| --- | --- | --- | --- | --- | --- | --- | --- |",
        ]
    )
    for scenario in summary["scenarios"]:
        lines.append(
            "| {scenario} | {runs} | {packet_only:.2%} | `{preferred}` | {codex_style_min}/{codex_style_avg} | {claude_style_min}/{claude_style_avg} | {codex_ready_min}/{codex_ready_avg} | {claude_ready_min}/{claude_ready_avg} |".format(
                scenario=scenario["scenario_label"],
                runs=scenario["runs"],
                packet_only=scenario["packet_only_pass_rate"],
                preferred=json.dumps(scenario["preferred_consumer_counts"], sort_keys=True),
                codex_style_min=fmt(scenario["codex_style_match"]["min"]),
                codex_style_avg=fmt(scenario["codex_style_match"]["avg"]),
                claude_style_min=fmt(scenario["claude_style_match"]["min"]),
                claude_style_avg=fmt(scenario["claude_style_match"]["avg"]),
                codex_ready_min=fmt(scenario["codex_continuation_readiness"]["min"]),
                codex_ready_avg=fmt(scenario["codex_continuation_readiness"]["avg"]),
                claude_ready_min=fmt(scenario["claude_continuation_readiness"]["min"]),
                claude_ready_avg=fmt(scenario["claude_continuation_readiness"]["avg"]),
            )
        )
    return "\n".join(lines) + "\n"


def main():
    parser = argparse.ArgumentParser(description="Aggregate Relay consumer continuation stability results.")
    parser.add_argument("--batch-summary", action="append", default=[], help="Path to a batch-summary.json file")
    parser.add_argument("--batch-summary-glob", action="append", default=[], help="Glob for batch-summary.json files")
    parser.add_argument("--output-json", required=True, help="Path to write stability JSON")
    parser.add_argument("--output-md", required=True, help="Path to write stability Markdown")
    args = parser.parse_args()

    paths = [Path(path).resolve() for path in args.batch_summary]
    for pattern in args.batch_summary_glob:
        paths.extend(Path(path).resolve() for path in sorted(glob.glob(pattern)))
    paths = sorted(set(paths))
    if not paths:
        raise SystemExit("at least one --batch-summary or --batch-summary-glob is required")

    summary = build_summary(paths)
    output_json = Path(args.output_json).resolve()
    output_md = Path(args.output_md).resolve()
    output_json.parent.mkdir(parents=True, exist_ok=True)
    output_md.parent.mkdir(parents=True, exist_ok=True)
    output_json.write_text(json.dumps(summary, indent=2) + "\n")
    output_md.write_text(render_markdown(summary))


if __name__ == "__main__":
    main()
