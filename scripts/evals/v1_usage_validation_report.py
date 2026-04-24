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
    retrieval_comparison_path = record.get("retrieval_comparison_file")
    retrieval_comparison = None
    if retrieval_comparison_path:
        retrieval_comparison = load_json(Path(retrieval_comparison_path).resolve())
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
        "retrieval_comparison_file": retrieval_comparison_path,
        "retrieval_preferred_variant": retrieval_comparison.get("preferred_variant") if retrieval_comparison else None,
        "retrieval_continuation_readiness": retrieval_comparison.get("continuation_readiness") if retrieval_comparison else None,
        "retrieval_evidence_relevance": retrieval_comparison.get("evidence_relevance") if retrieval_comparison else None,
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
        f"- retrieval paired runs: `{summary['retrieval_paired_runs']}`",
        f"- retrieval-aware wins: `{summary['retrieval_aware_wins']}`",
        f"- retrieval-aware win rate: `{summary['retrieval_aware_win_rate']:.2%}`" if summary["retrieval_aware_win_rate"] is not None else "- retrieval-aware win rate: `n/a`",
        f"- avg retrieval continuation readiness: `{summary['avg_retrieval_continuation_readiness']:.2f}`" if summary["avg_retrieval_continuation_readiness"] is not None else "- avg retrieval continuation readiness: `n/a`",
        f"- avg retrieval evidence relevance: `{summary['avg_retrieval_evidence_relevance']:.2f}`" if summary["avg_retrieval_evidence_relevance"] is not None else "- avg retrieval evidence relevance: `n/a`",
        f"- avg handoff duration: `{summary['avg_total_handoff_duration_ms']:.1f} ms`" if summary["avg_total_handoff_duration_ms"] is not None else "- avg handoff duration: `n/a`",
        f"- avg artifact count: `{summary['avg_artifact_count']:.2f}`" if summary["avg_artifact_count"] is not None else "- avg artifact count: `n/a`",
        f"- gate pass: `{str(gate['pass']).lower()}`",
        "",
        "## Release Gate",
        "",
        f"- min style-aware win rate: `{gate['thresholds']['min_style_aware_win_rate']}`",
        f"- min avg style_match: `{gate['thresholds']['min_avg_style_match']}`",
        f"- min budget pass rate: `{gate['thresholds']['min_budget_pass_rate']}`",
        f"- min retrieval-aware win rate: `{gate['thresholds']['min_retrieval_aware_win_rate']}`",
        f"- min avg retrieval continuation readiness: `{gate['thresholds']['min_avg_retrieval_continuation_readiness']}`",
        f"- min avg retrieval evidence relevance: `{gate['thresholds']['min_avg_retrieval_evidence_relevance']}`",
        f"- observed win rate: `{summary['style_aware_win_rate']:.2%}`" if summary["style_aware_win_rate"] is not None else "- observed win rate: `n/a`",
        f"- observed avg style_match: `{summary['avg_style_match']:.2f}`" if summary["avg_style_match"] is not None else "- observed avg style_match: `n/a`",
        f"- observed budget pass rate: `{summary['budget_pass_rate']:.2%}`" if summary["budget_pass_rate"] is not None else "- observed budget pass rate: `n/a`",
        f"- observed retrieval-aware win rate: `{summary['retrieval_aware_win_rate']:.2%}`" if summary["retrieval_aware_win_rate"] is not None else "- observed retrieval-aware win rate: `n/a`",
        f"- observed avg retrieval continuation readiness: `{summary['avg_retrieval_continuation_readiness']:.2f}`" if summary["avg_retrieval_continuation_readiness"] is not None else "- observed avg retrieval continuation readiness: `n/a`",
        f"- observed avg retrieval evidence relevance: `{summary['avg_retrieval_evidence_relevance']:.2f}`" if summary["avg_retrieval_evidence_relevance"] is not None else "- observed avg retrieval evidence relevance: `n/a`",
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
        "| scenario | style preferred | style_match | retrieval preferred | retrieval readiness | retrieval relevance | artifacts | handoff_ms | budget |",
        "| --- | --- | --- | --- | --- | --- | --- | --- | --- |",
    ])
    for run in summary["runs"]:
        preferred = run["preferred_continuation"] or "unscored"
        style_match = "n/a" if run["style_match"] is None else run["style_match"]
        retrieval_preferred = run["retrieval_preferred_variant"] or "unscored"
        retrieval_readiness = "n/a" if run["retrieval_continuation_readiness"] is None else run["retrieval_continuation_readiness"]
        retrieval_relevance = "n/a" if run["retrieval_evidence_relevance"] is None else run["retrieval_evidence_relevance"]
        handoff_ms = "n/a" if run["total_handoff_duration_ms"] is None else run["total_handoff_duration_ms"]
        lines.append(
            f"| {run['scenario_label']} | {preferred} | {style_match} | {retrieval_preferred} | {retrieval_readiness} | {retrieval_relevance} | {run['artifact_count']} | {handoff_ms} | {str(run['budget_pass']).lower()} |"
        )
    return "\n".join(lines) + "\n"


def main():
    parser = argparse.ArgumentParser(description="Aggregate Relay V1 usage-validation batch results.")
    parser.add_argument("--batch-runs-file", required=True, help="Path to batch-runs.jsonl")
    parser.add_argument("--min-style-aware-win-rate", required=True, type=float, help="Minimum style-aware win rate gate")
    parser.add_argument("--min-avg-style-match", required=True, type=float, help="Minimum average style_match gate")
    parser.add_argument("--min-budget-pass-rate", required=True, type=float, help="Minimum budget-pass rate gate")
    parser.add_argument("--min-retrieval-aware-win-rate", required=True, type=float, help="Minimum retrieval-aware win rate gate")
    parser.add_argument("--min-avg-retrieval-continuation-readiness", required=True, type=float, help="Minimum average retrieval continuation readiness gate")
    parser.add_argument("--min-avg-retrieval-evidence-relevance", required=True, type=float, help="Minimum average retrieval evidence relevance gate")
    parser.add_argument("--output-json", required=True, help="Path to write aggregate JSON")
    parser.add_argument("--output-md", required=True, help="Path to write aggregate Markdown")
    args = parser.parse_args()

    batch_runs_path = Path(args.batch_runs_file).resolve()
    records = load_jsonl(batch_runs_path)
    run_summaries = [build_run_summary(record) for record in records]

    paired_runs = [run for run in run_summaries if run["preferred_continuation"] is not None]
    retrieval_runs = [run for run in run_summaries if run["retrieval_preferred_variant"] is not None]
    style_matches = [run["style_match"] for run in paired_runs if run["style_match"] is not None]
    retrieval_continuation_readiness = [run["retrieval_continuation_readiness"] for run in retrieval_runs if run["retrieval_continuation_readiness"] is not None]
    retrieval_evidence_relevance = [run["retrieval_evidence_relevance"] for run in retrieval_runs if run["retrieval_evidence_relevance"] is not None]
    durations = [run["total_handoff_duration_ms"] for run in run_summaries if run["total_handoff_duration_ms"] is not None]
    artifact_counts = [run["artifact_count"] for run in run_summaries]
    style_aware_wins = sum(1 for run in paired_runs if run["preferred_continuation"] == "style-aware")
    retrieval_aware_wins = sum(1 for run in retrieval_runs if run["retrieval_preferred_variant"] == "retrieval-aware")
    budget_pass_runs = sum(1 for run in run_summaries if run["budget_pass"])
    style_aware_win_rate = None if not paired_runs else style_aware_wins / len(paired_runs)
    retrieval_aware_win_rate = None if not retrieval_runs else retrieval_aware_wins / len(retrieval_runs)
    avg_style_match = mean_or_none(style_matches)
    avg_retrieval_continuation_readiness = mean_or_none(retrieval_continuation_readiness)
    avg_retrieval_evidence_relevance = mean_or_none(retrieval_evidence_relevance)
    budget_pass_rate = None if not run_summaries else budget_pass_runs / len(run_summaries)
    gate_reasons = []
    if style_aware_win_rate is None or style_aware_win_rate < args.min_style_aware_win_rate:
        gate_reasons.append("style-aware win rate is below threshold")
    if avg_style_match is None or avg_style_match < args.min_avg_style_match:
        gate_reasons.append("average style_match is below threshold")
    if budget_pass_rate is None or budget_pass_rate < args.min_budget_pass_rate:
        gate_reasons.append("budget pass rate is below threshold")
    if retrieval_aware_win_rate is None or retrieval_aware_win_rate < args.min_retrieval_aware_win_rate:
        gate_reasons.append("retrieval-aware win rate is below threshold")
    if avg_retrieval_continuation_readiness is None or avg_retrieval_continuation_readiness < args.min_avg_retrieval_continuation_readiness:
        gate_reasons.append("average retrieval continuation readiness is below threshold")
    if avg_retrieval_evidence_relevance is None or avg_retrieval_evidence_relevance < args.min_avg_retrieval_evidence_relevance:
        gate_reasons.append("average retrieval evidence relevance is below threshold")

    summary = {
        "batch_id": records[0]["batch_id"] if records else "empty-batch",
        "total_runs": len(run_summaries),
        "paired_runs": len(paired_runs),
        "retrieval_paired_runs": len(retrieval_runs),
        "style_aware_wins": style_aware_wins,
        "style_aware_win_rate": style_aware_win_rate,
        "avg_style_match": avg_style_match,
        "retrieval_aware_wins": retrieval_aware_wins,
        "retrieval_aware_win_rate": retrieval_aware_win_rate,
        "avg_retrieval_continuation_readiness": avg_retrieval_continuation_readiness,
        "avg_retrieval_evidence_relevance": avg_retrieval_evidence_relevance,
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
                "min_retrieval_aware_win_rate": args.min_retrieval_aware_win_rate,
                "min_avg_retrieval_continuation_readiness": args.min_avg_retrieval_continuation_readiness,
                "min_avg_retrieval_evidence_relevance": args.min_avg_retrieval_evidence_relevance,
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
