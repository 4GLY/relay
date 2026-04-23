#!/usr/bin/env python3
import argparse
import json
from pathlib import Path


def load_json(path: Path):
    return json.loads(path.read_text())


def load_jsonl(path: Path):
    records = []
    for line in path.read_text().splitlines():
        line = line.strip()
        if not line:
            continue
        records.append(json.loads(line))
    return records


def mean_or_none(values):
    return None if not values else sum(values) / len(values)


def build_run_summary(record):
    result_path = Path(record["result_file"]).resolve()
    result = load_json(result_path)
    comparison_path = record.get("comparison_file")
    comparison = None
    if comparison_path:
        comparison = load_json(Path(comparison_path).resolve())
    style_packet = load_json(Path(result["style_packet_file"]).resolve())
    rubric = result.get("rubric_scores", {})

    return {
        "run_id": result.get("run_id"),
        "fixture_id": result.get("fixture_id"),
        "scenario_label": result.get("scenario_label") or record.get("scenario_label") or result.get("fixture_id"),
        "project": result.get("project"),
        "budget_pass": bool(result.get("budget", {}).get("pass")),
        "total_handoff_duration_ms": result.get("total_handoff_duration_ms"),
        "preferred_continuation": comparison.get("preferred_continuation") if comparison else None,
        "style_match": comparison.get("style_match") if comparison else None,
        "artifact_count": len(style_packet.get("supporting_artifacts", [])),
        "manual_corrections": rubric.get("manual_corrections"),
        "comparison_file": comparison_path,
        "result_file": str(result_path),
    }


def render_markdown(summary):
    gate = summary["gate"]
    lines = [
        "# Relay V1 Usage Validation Batch",
        "",
        "## Overview",
        "",
        f"- batch_id: `{summary['batch_id']}`",
        f"- fixtures: `{summary['total_runs']}`",
        f"- paired runs: `{summary['paired_runs']}`",
        f"- style-aware wins: `{summary['style_aware_wins']}`",
        f"- style-aware win rate: `{summary['style_aware_win_rate']:.2%}`" if summary["style_aware_win_rate"] is not None else "- style-aware win rate: `n/a`",
        f"- avg style_match: `{summary['avg_style_match']:.2f}`" if summary["avg_style_match"] is not None else "- avg style_match: `n/a`",
        f"- avg handoff duration: `{summary['avg_total_handoff_duration_ms']:.1f} ms`" if summary["avg_total_handoff_duration_ms"] is not None else "- avg handoff duration: `n/a`",
        f"- avg artifact count: `{summary['avg_artifact_count']:.2f}`" if summary["avg_artifact_count"] is not None else "- avg artifact count: `n/a`",
        f"- gate pass: `{str(gate['pass']).lower()}`",
        "",
        "## Release Gate",
        "",
        f"- min style-aware win rate: `{gate['thresholds']['min_style_aware_win_rate']}`",
        f"- min avg style_match: `{gate['thresholds']['min_avg_style_match']}`",
        f"- min budget pass rate: `{gate['thresholds']['min_budget_pass_rate']}`",
        f"- observed win rate: `{summary['style_aware_win_rate']:.2%}`" if summary["style_aware_win_rate"] is not None else "- observed win rate: `n/a`",
        f"- observed avg style_match: `{summary['avg_style_match']:.2f}`" if summary["avg_style_match"] is not None else "- observed avg style_match: `n/a`",
        f"- observed budget pass rate: `{summary['budget_pass_rate']:.2%}`" if summary["budget_pass_rate"] is not None else "- observed budget pass rate: `n/a`",
    ]
    if gate["reasons"]:
        lines.extend([
            "",
            "Gate blockers:",
        ])
        for reason in gate["reasons"]:
            lines.append(f"- {reason}")
    lines.extend([
        "",
        "## Per Scenario",
        "",
        "| scenario | preferred | style_match | artifacts | handoff_ms | budget |",
        "| --- | --- | --- | --- | --- | --- |",
    ])
    for run in summary["runs"]:
        preferred = run["preferred_continuation"] or "unscored"
        style_match = "n/a" if run["style_match"] is None else run["style_match"]
        handoff_ms = "n/a" if run["total_handoff_duration_ms"] is None else run["total_handoff_duration_ms"]
        lines.append(
            f"| {run['scenario_label']} | {preferred} | {style_match} | {run['artifact_count']} | {handoff_ms} | {str(run['budget_pass']).lower()} |"
        )
    return "\n".join(lines) + "\n"


def main():
    parser = argparse.ArgumentParser(description="Aggregate Relay V1 usage-validation batch results.")
    parser.add_argument("--batch-runs-file", required=True, help="Path to batch-runs.jsonl")
    parser.add_argument("--min-style-aware-win-rate", required=True, type=float, help="Minimum style-aware win rate gate")
    parser.add_argument("--min-avg-style-match", required=True, type=float, help="Minimum average style_match gate")
    parser.add_argument("--min-budget-pass-rate", required=True, type=float, help="Minimum budget-pass rate gate")
    parser.add_argument("--output-json", required=True, help="Path to write aggregate JSON")
    parser.add_argument("--output-md", required=True, help="Path to write aggregate Markdown")
    args = parser.parse_args()

    batch_runs_path = Path(args.batch_runs_file).resolve()
    records = load_jsonl(batch_runs_path)
    run_summaries = [build_run_summary(record) for record in records]

    paired_runs = [run for run in run_summaries if run["preferred_continuation"] is not None]
    style_matches = [run["style_match"] for run in paired_runs if run["style_match"] is not None]
    durations = [run["total_handoff_duration_ms"] for run in run_summaries if run["total_handoff_duration_ms"] is not None]
    artifact_counts = [run["artifact_count"] for run in run_summaries]
    style_aware_wins = sum(1 for run in paired_runs if run["preferred_continuation"] == "style-aware")
    budget_pass_runs = sum(1 for run in run_summaries if run["budget_pass"])
    style_aware_win_rate = None if not paired_runs else style_aware_wins / len(paired_runs)
    avg_style_match = mean_or_none(style_matches)
    budget_pass_rate = None if not run_summaries else budget_pass_runs / len(run_summaries)
    gate_reasons = []
    if style_aware_win_rate is None or style_aware_win_rate < args.min_style_aware_win_rate:
        gate_reasons.append("style-aware win rate is below threshold")
    if avg_style_match is None or avg_style_match < args.min_avg_style_match:
        gate_reasons.append("average style_match is below threshold")
    if budget_pass_rate is None or budget_pass_rate < args.min_budget_pass_rate:
        gate_reasons.append("budget pass rate is below threshold")

    summary = {
        "batch_id": records[0]["batch_id"] if records else "empty-batch",
        "total_runs": len(run_summaries),
        "paired_runs": len(paired_runs),
        "style_aware_wins": style_aware_wins,
        "style_aware_win_rate": style_aware_win_rate,
        "avg_style_match": avg_style_match,
        "avg_total_handoff_duration_ms": mean_or_none(durations),
        "max_total_handoff_duration_ms": None if not durations else max(durations),
        "avg_artifact_count": mean_or_none(artifact_counts),
        "budget_pass_runs": budget_pass_runs,
        "budget_pass_rate": budget_pass_rate,
        "gate": {
            "pass": len(gate_reasons) == 0,
            "thresholds": {
                "min_style_aware_win_rate": args.min_style_aware_win_rate,
                "min_avg_style_match": args.min_avg_style_match,
                "min_budget_pass_rate": args.min_budget_pass_rate,
            },
            "reasons": gate_reasons,
        },
        "runs": run_summaries,
    }

    output_json = Path(args.output_json).resolve()
    output_md = Path(args.output_md).resolve()
    output_json.write_text(json.dumps(summary, indent=2) + "\n")
    output_md.write_text(render_markdown(summary))


if __name__ == "__main__":
    main()
