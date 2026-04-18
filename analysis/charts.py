"""
Generate load test charts from results CSV.
Usage: python analysis/charts.py --input results/loadtest.csv --output results/
"""
import argparse
import statistics
from pathlib import Path

import pandas as pd
import matplotlib
matplotlib.use("Agg")
import matplotlib.pyplot as plt
import matplotlib.patches as mpatches
import numpy as np


STRATEGY_COLORS = {
    "round_robin":   "#4C72B0",
    "least_pending": "#DD8452",
    "random":        "#55A868",
    "kv_aware":      "#C44E52",
}
STRATEGY_ORDER = ["round_robin", "random", "least_pending", "kv_aware"]


def load(path: str) -> pd.DataFrame:
    df = pd.read_csv(path)
    df = df[df["error"].isna() | (df["error"] == "")]
    df["strategy"] = df["strategy"].str.strip()
    return df


# ── Chart 1: p50 / p95 latency bar chart ────────────────────────────────────

def chart_latency_bars(df: pd.DataFrame, out: Path) -> None:
    strategies = [s for s in STRATEGY_ORDER if s in df["strategy"].unique()]
    p50 = [statistics.median(df[df["strategy"] == s]["total_ms"]) for s in strategies]
    p95 = [np.percentile(df[df["strategy"] == s]["total_ms"], 95) for s in strategies]

    x = np.arange(len(strategies))
    width = 0.35
    fig, ax = plt.subplots(figsize=(9, 5))
    bars50 = ax.bar(x - width / 2, p50, width, label="p50", color=[STRATEGY_COLORS[s] for s in strategies], alpha=0.9)
    bars95 = ax.bar(x + width / 2, p95, width, label="p95", color=[STRATEGY_COLORS[s] for s in strategies], alpha=0.5, hatch="//")

    ax.set_xlabel("Routing Strategy")
    ax.set_ylabel("Latency (ms)")
    ax.set_title("p50 / p95 Latency by Strategy")
    ax.set_xticks(x)
    ax.set_xticklabels([s.replace("_", "\n") for s in strategies])
    ax.legend()
    ax.bar_label(bars50, fmt="%.0f", padding=3, fontsize=8)
    ax.bar_label(bars95, fmt="%.0f", padding=3, fontsize=8)
    ax.grid(axis="y", linestyle="--", alpha=0.4)
    fig.tight_layout()
    fig.savefig(out / "latency_bars.png", dpi=150)
    plt.close(fig)
    print(f"  saved latency_bars.png")


# ── Chart 2: CDF of latency per strategy ────────────────────────────────────

def chart_latency_cdf(df: pd.DataFrame, out: Path) -> None:
    fig, ax = plt.subplots(figsize=(9, 5))
    for s in STRATEGY_ORDER:
        sub = df[df["strategy"] == s]["total_ms"].sort_values()
        if sub.empty:
            continue
        cdf = np.arange(1, len(sub) + 1) / len(sub)
        ax.plot(sub, cdf, label=s.replace("_", " "), color=STRATEGY_COLORS[s], linewidth=2)

    ax.set_xlabel("Latency (ms)")
    ax.set_ylabel("CDF")
    ax.set_title("Latency CDF by Strategy")
    ax.legend()
    ax.grid(linestyle="--", alpha=0.4)
    fig.tight_layout()
    fig.savefig(out / "latency_cdf.png", dpi=150)
    plt.close(fig)
    print(f"  saved latency_cdf.png")


# ── Chart 3: KV cache hit rate (kv_aware only) ───────────────────────────────

def chart_kv_hit_rate(df: pd.DataFrame, out: Path) -> None:
    kv = df[df["strategy"] == "kv_aware"]
    if kv.empty:
        print("  skipped kv_hit_rate (no kv_aware data)")
        return

    hits = kv["cache_hit"].sum()
    misses = len(kv) - hits
    fig, ax = plt.subplots(figsize=(5, 5))
    ax.pie(
        [hits, misses],
        labels=[f"Cache Hit\n({hits})", f"Cache Miss\n({misses})"],
        colors=["#55A868", "#C44E52"],
        autopct="%1.0f%%",
        startangle=90,
        textprops={"fontsize": 12},
    )
    ax.set_title("KV Cache Hit Rate (kv_aware strategy)")
    fig.tight_layout()
    fig.savefig(out / "kv_hit_rate.png", dpi=150)
    plt.close(fig)
    print(f"  saved kv_hit_rate.png")


# ── Chart 4: Backend distribution per strategy ───────────────────────────────

def chart_backend_distribution(df: pd.DataFrame, out: Path) -> None:
    strategies = [s for s in STRATEGY_ORDER if s in df["strategy"].unique()]
    backends = sorted(df["backend"].dropna().unique())
    backend_colors = plt.cm.Set2(np.linspace(0, 1, len(backends)))

    fig, axes = plt.subplots(1, len(strategies), figsize=(4 * len(strategies), 4), sharey=False)
    if len(strategies) == 1:
        axes = [axes]

    for ax, strat in zip(axes, strategies):
        sub = df[df["strategy"] == strat]
        counts = sub["backend"].value_counts()
        ax.bar(
            counts.index,
            counts.values,
            color=[backend_colors[backends.index(b)] if b in backends else "gray" for b in counts.index],
            alpha=0.85,
        )
        ax.set_title(strat.replace("_", "\n"), fontsize=10)
        ax.set_ylabel("Requests" if ax == axes[0] else "")
        ax.tick_params(axis="x", rotation=30, labelsize=8)
        ax.grid(axis="y", linestyle="--", alpha=0.4)

    fig.suptitle("Backend Distribution per Strategy", fontsize=13, y=1.02)
    fig.tight_layout()
    fig.savefig(out / "backend_distribution.png", dpi=150, bbox_inches="tight")
    plt.close(fig)
    print(f"  saved backend_distribution.png")


# ── Chart 5: Latency over time (request order) ──────────────────────────────

def chart_latency_over_time(df: pd.DataFrame, out: Path) -> None:
    fig, ax = plt.subplots(figsize=(11, 5))
    for s in STRATEGY_ORDER:
        sub = df[df["strategy"] == s].reset_index(drop=True)
        if sub.empty:
            continue
        ax.plot(sub.index, sub["total_ms"], label=s.replace("_", " "),
                color=STRATEGY_COLORS[s], alpha=0.7, linewidth=1.2, marker="o", markersize=3)

    ax.set_xlabel("Request Index")
    ax.set_ylabel("Latency (ms)")
    ax.set_title("Latency over Time by Strategy")
    ax.legend()
    ax.grid(linestyle="--", alpha=0.3)
    fig.tight_layout()
    fig.savefig(out / "latency_over_time.png", dpi=150)
    plt.close(fig)
    print(f"  saved latency_over_time.png")


# ── Summary table ─────────────────────────────────────────────────────────────

def print_summary(df: pd.DataFrame) -> None:
    print("\n--- Load Test Summary ---")
    print(f"{'Strategy':<16} {'N':>4} {'p50':>7} {'p95':>7} {'min':>7} {'max':>7} {'KV hits':>8}")
    print("-" * 60)
    for s in STRATEGY_ORDER:
        sub = df[df["strategy"] == s]
        if sub.empty:
            continue
        lat = sorted(sub["total_ms"])
        hits = sub["cache_hit"].sum() if "cache_hit" in sub.columns else 0
        kv_str = f"{hits}/{len(sub)}" if s == "kv_aware" else "-"
        print(f"{s:<16} {len(sub):>4} {statistics.median(lat):>7.0f} "
              f"{np.percentile(lat,95):>7.0f} {lat[0]:>7} {lat[-1]:>7} {kv_str:>8}")
    print("-" * 60)


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--input",  default="results/loadtest.csv")
    parser.add_argument("--output", default="results/")
    args = parser.parse_args()

    out = Path(args.output)
    out.mkdir(parents=True, exist_ok=True)

    df = load(args.input)
    print(f"Loaded {len(df)} successful rows from {args.input}")

    print_summary(df)

    print("\nGenerating charts...")
    chart_latency_bars(df, out)
    chart_latency_cdf(df, out)
    chart_kv_hit_rate(df, out)
    chart_backend_distribution(df, out)
    chart_latency_over_time(df, out)
    print(f"\nAll charts saved to {out}/")


if __name__ == "__main__":
    main()
