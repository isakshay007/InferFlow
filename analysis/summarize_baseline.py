import argparse
import json
from dataclasses import dataclass
from datetime import datetime
from pathlib import Path
from typing import Optional

import numpy as np


@dataclass
class RunSummary:
    backend: Optional[str]
    strategy: str
    concurrency: int
    total_req: int
    success_count: int
    error_count: int
    error_rate: float
    p50_ms: float
    p95_ms: float
    p99_ms: float
    throughput_rps: float


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Summarize InferFlow baseline runs")
    parser.add_argument("--input-dir", default="results/baseline")
    parser.add_argument("--output", default="results/baseline/summary.md")
    parser.add_argument("--compare", default="")
    parser.add_argument("--plot", action="store_true")
    return parser.parse_args()


def parse_timestamp(value: str) -> datetime:
    return datetime.fromisoformat(value.replace("Z", "+00:00"))


def parse_meta(path: Path, first_record: Optional[dict]) -> tuple[str, int]:
    if first_record is not None:
        strategy = str(first_record.get("strategy", "")).strip()
        concurrency = int(first_record.get("concurrency", 0) or 0)
        if strategy and concurrency > 0:
            return strategy, concurrency

    stem = path.stem
    if "_c" in stem:
        strategy, concurrency = stem.rsplit("_c", 1)
        if concurrency.isdigit():
            return strategy, int(concurrency)
    return stem, 0


def summarize_file(path: Path, backend: Optional[str] = None) -> Optional[RunSummary]:
    records = []
    with path.open("r", encoding="utf-8") as handle:
        for line in handle:
            line = line.strip()
            if not line:
                continue
            records.append(json.loads(line))

    if not records:
        return None

    strategy, concurrency = parse_meta(path, records[0])
    total_req = len(records)
    success = [r for r in records if r.get("status_code") == 200 and r.get("error") is None]
    error_count = total_req - len(success)

    latency_values = np.array(
        [float(r.get("latency_ms", 0.0)) for r in success], dtype=np.float64
    )
    if latency_values.size == 0:
        p50_ms = 0.0
        p95_ms = 0.0
        p99_ms = 0.0
    else:
        p50_ms = float(np.percentile(latency_values, 50))
        p95_ms = float(np.percentile(latency_values, 95))
        p99_ms = float(np.percentile(latency_values, 99))

    start_times = [parse_timestamp(r["timestamp"]).timestamp() for r in records]
    end_times = [
        parse_timestamp(r["timestamp"]).timestamp() + (float(r.get("latency_ms", 0.0)) / 1000.0)
        for r in records
    ]
    duration_s = max(max(end_times) - min(start_times), 1e-9)
    throughput_rps = total_req / duration_s

    return RunSummary(
        backend=backend,
        strategy=strategy,
        concurrency=concurrency,
        total_req=total_req,
        success_count=len(success),
        error_count=error_count,
        error_rate=(error_count / total_req) * 100.0,
        p50_ms=p50_ms,
        p95_ms=p95_ms,
        p99_ms=p99_ms,
        throughput_rps=throughput_rps,
    )


def build_markdown(rows: list[RunSummary], include_backend: bool = False) -> str:
    lines = []
    lines.append("# Baseline Summary")
    lines.append("")
    if include_backend:
        lines.append(
            "| backend | strategy | concurrency | total_req | error_rate | p50_ms | p95_ms | p99_ms | throughput_rps |"
        )
        lines.append("|---|---|---:|---:|---:|---:|---:|---:|---:|")
    else:
        lines.append(
            "| strategy | concurrency | total_req | error_rate | p50_ms | p95_ms | p99_ms | throughput_rps |"
        )
        lines.append("|---|---:|---:|---:|---:|---:|---:|---:|")

    for row in rows:
        if include_backend:
            lines.append(
                f"| {row.backend or ''} | {row.strategy} | {row.concurrency} | {row.total_req} | "
                f"{row.error_rate:.2f}% | {row.p50_ms:.1f} | {row.p95_ms:.1f} | "
                f"{row.p99_ms:.1f} | {row.throughput_rps:.2f} |"
            )
        else:
            lines.append(
                f"| {row.strategy} | {row.concurrency} | {row.total_req} | "
                f"{row.error_rate:.2f}% | {row.p50_ms:.1f} | {row.p95_ms:.1f} | "
                f"{row.p99_ms:.1f} | {row.throughput_rps:.2f} |"
            )
    lines.append("")
    return "\n".join(lines)


def maybe_plot(rows: list[RunSummary], plot_dir: Path) -> None:
    import matplotlib.pyplot as plt

    strategies = ["round_robin", "least_pending", "cost_aware"]
    conc_levels = sorted({row.concurrency for row in rows})
    width = 0.22
    x = np.arange(len(conc_levels))

    def value_for(metric: str, strategy: str, concurrency: int) -> float:
        for row in rows:
            if row.strategy == strategy and row.concurrency == concurrency:
                return float(getattr(row, metric))
        return 0.0

    plot_dir.mkdir(parents=True, exist_ok=True)

    def grouped_bar(metric: str, ylabel: str, title: str, filename: str) -> None:
        plt.figure(figsize=(10, 6))
        for idx, strategy in enumerate(strategies):
            offsets = x + (idx - 1) * width
            values = [value_for(metric, strategy, c) for c in conc_levels]
            plt.bar(offsets, values, width=width, label=strategy)

        plt.xticks(x, [str(c) for c in conc_levels])
        plt.xlabel("concurrency")
        plt.ylabel(ylabel)
        plt.title(title)
        plt.legend()
        plt.tight_layout()
        plt.savefig(plot_dir / filename)
        plt.close()

    grouped_bar(
        "p95_ms",
        "p95 latency (ms)",
        "P95 Latency by Strategy",
        "latency_p95_by_strategy.png",
    )
    grouped_bar(
        "throughput_rps",
        "throughput (requests/sec)",
        "Throughput by Strategy",
        "throughput_by_strategy.png",
    )
    grouped_bar(
        "error_rate",
        "error rate (%)",
        "Error Rate by Strategy",
        "error_rate_by_strategy.png",
    )


def collect_summaries(input_dir: Path, backend: Optional[str] = None) -> list[RunSummary]:
    files = sorted(input_dir.glob("*.jsonl"))
    summaries: list[RunSummary] = []
    for file_path in files:
        summary = summarize_file(file_path, backend=backend)
        if summary is not None:
            summaries.append(summary)
    return summaries


def sort_summaries(rows: list[RunSummary]) -> None:
    backend_order = {"mock": 0, "real": 1}
    strategy_order = {"round_robin": 0, "least_pending": 1, "cost_aware": 2}
    rows.sort(
        key=lambda row: (
            backend_order.get(row.backend or "", 99),
            row.concurrency,
            strategy_order.get(row.strategy, 99),
            row.strategy,
        )
    )


def maybe_plot_compare_latency(rows: list[RunSummary]) -> None:
    import matplotlib.pyplot as plt

    strategies = ["round_robin", "least_pending", "cost_aware"]
    target_concurrency = 4
    backends = ["mock", "real"]
    width = 0.35
    x = np.arange(len(strategies))

    def p95_for(backend: str, strategy: str) -> float:
        for row in rows:
            if (
                row.backend == backend
                and row.strategy == strategy
                and row.concurrency == target_concurrency
            ):
                return row.p95_ms
        return 0.0

    plt.figure(figsize=(10, 6))
    for idx, backend in enumerate(backends):
        offset = x + (idx - 0.5) * width
        values = [p95_for(backend, strategy) for strategy in strategies]
        plt.bar(offset, values, width=width, label=backend)

    plt.xticks(x, strategies)
    plt.xlabel("strategy")
    plt.ylabel("p95 latency (ms)")
    plt.title("P95 Latency Comparison (mock vs real) at concurrency=4")
    plt.legend()
    plt.tight_layout()
    out = Path("results/latency_comparison_mock_vs_real.png")
    out.parent.mkdir(parents=True, exist_ok=True)
    plt.savefig(out)
    plt.close()


def run_single_mode(args: argparse.Namespace) -> None:
    input_dir = Path(args.input_dir)
    summaries = collect_summaries(input_dir)
    sort_summaries(summaries)

    output_path = Path(args.output)
    output_path.parent.mkdir(parents=True, exist_ok=True)
    output_path.write_text(build_markdown(summaries), encoding="utf-8")
    print(f"Wrote summary to {output_path}")

    if args.plot:
        plot_dir = input_dir / "plots"
        maybe_plot(summaries, plot_dir)
        print(f"Wrote plots to {plot_dir}/")


def run_compare_mode(args: argparse.Namespace) -> None:
    raw_dirs = [part.strip() for part in args.compare.split(",") if part.strip()]
    if len(raw_dirs) != 2:
        raise SystemExit("--compare must contain exactly two directories separated by a comma")

    mock_dir = Path(raw_dirs[0])
    real_dir = Path(raw_dirs[1])

    summaries = []
    summaries.extend(collect_summaries(mock_dir, backend="mock"))
    summaries.extend(collect_summaries(real_dir, backend="real"))
    sort_summaries(summaries)

    output_path = Path("results/comparison.md")
    output_path.parent.mkdir(parents=True, exist_ok=True)
    output_path.write_text(build_markdown(summaries, include_backend=True), encoding="utf-8")
    print(f"Wrote comparison summary to {output_path}")

    if args.plot:
        maybe_plot_compare_latency(summaries)
        print("Wrote comparison plot to results/latency_comparison_mock_vs_real.png")


def main() -> None:
    args = parse_args()
    if args.compare:
        run_compare_mode(args)
        return
    run_single_mode(args)


if __name__ == "__main__":
    main()
