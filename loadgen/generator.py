import argparse
import json
import threading
import time
import urllib.error
import urllib.request
from concurrent.futures import FIRST_COMPLETED, Future, ThreadPoolExecutor, wait
from datetime import datetime, timezone
from pathlib import Path
from typing import Optional


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="InferFlow load generator")
    parser.add_argument("--url", default="http://localhost:8080/v1/chat/completions")
    parser.add_argument("--requests", type=int, default=10)
    parser.add_argument("--model", default="mock-llm")
    parser.add_argument("--concurrency", type=int, default=1)
    parser.add_argument("--strategy", default="round_robin")
    parser.add_argument("--strategy-url", default="http://localhost:8080/strategy")
    parser.add_argument("--warmup", type=int, default=3)
    parser.add_argument("--output", default="results/run.jsonl")
    parser.add_argument("--duration", type=int, default=30)
    return parser.parse_args()


def utc_timestamp() -> str:
    return datetime.now(timezone.utc).isoformat().replace("+00:00", "Z")


def build_payload(model: str, request_id: int) -> bytes:
    payload = {
        "model": model,
        "messages": [
            {
                "role": "user",
                "content": "Name 2 colors.",
            }
        ],
        "stream": False,
    }
    return json.dumps(payload).encode("utf-8")


def switch_strategy(strategy_url: str, strategy: str) -> None:
    req = urllib.request.Request(
        strategy_url,
        data=json.dumps({"strategy": strategy}).encode("utf-8"),
        headers={"Content-Type": "application/json"},
        method="PUT",
    )
    try:
        with urllib.request.urlopen(req, timeout=10) as resp:
            if resp.status != 200:
                raise RuntimeError(f"failed to switch strategy, status={resp.status}")
    except urllib.error.HTTPError as exc:
        raise RuntimeError(f"failed to switch strategy, status={exc.code}") from exc
    except urllib.error.URLError as exc:
        raise RuntimeError(f"failed to switch strategy: {exc}") from exc


def issue_request(
    *,
    url: str,
    model: str,
    strategy: str,
    concurrency: int,
    request_id: int,
) -> dict:
    started_mono = time.monotonic()
    started_ts = utc_timestamp()
    req = urllib.request.Request(
        url,
        data=build_payload(model, request_id),
        headers={"Content-Type": "application/json"},
        method="POST",
    )

    status_code: Optional[int] = None
    error: Optional[str] = None
    try:
        with urllib.request.urlopen(req, timeout=60) as resp:
            status_code = resp.status
            _ = resp.read()
    except urllib.error.HTTPError as exc:
        status_code = exc.code
        error = f"http_{exc.code}"
    except Exception as exc:  # noqa: BLE001
        error = str(exc)

    latency_ms = (time.monotonic() - started_mono) * 1000.0
    return {
        "strategy": strategy,
        "concurrency": concurrency,
        "request_id": request_id,
        "status_code": status_code,
        "latency_ms": round(latency_ms, 3),
        "timestamp": started_ts,
        "error": error,
    }


def run_warmup(args: argparse.Namespace) -> None:
    if args.warmup <= 0:
        return
    for idx in range(1, args.warmup + 1):
        _ = issue_request(
            url=args.url,
            model=args.model,
            strategy=args.strategy,
            concurrency=args.concurrency,
            request_id=idx,
        )
    print(f"Warmup complete ({args.warmup} requests discarded).")


def print_live_summary(
    start_mono: float, total: int, ok: int, err: int, total_latency_ms: float
) -> None:
    elapsed = int(time.monotonic() - start_mono)
    avg_ms = (total_latency_ms / total) if total > 0 else 0.0
    print(f"[{elapsed}s] requests={total} ok={ok} err={err} avg_ms={avg_ms:.1f}")


def main() -> None:
    args = parse_args()
    if args.concurrency < 1:
        raise SystemExit("--concurrency must be >= 1")
    if args.warmup < 0:
        raise SystemExit("--warmup must be >= 0")
    if args.requests < 1:
        raise SystemExit("--requests must be >= 1")
    if args.duration < 0:
        raise SystemExit("--duration must be >= 0")

    switch_strategy(args.strategy_url, args.strategy)
    print(f"Strategy set to {args.strategy}.")

    run_warmup(args)

    output = Path(args.output)
    output.parent.mkdir(parents=True, exist_ok=True)

    lock = threading.Lock()
    total = 0
    ok = 0
    err = 0
    total_latency_ms = 0.0
    next_request_id = 1

    start_mono = time.monotonic()
    deadline = start_mono + args.duration
    next_summary = start_mono + 5.0

    def should_schedule() -> bool:
        if args.duration > 0:
            return time.monotonic() < deadline
        return next_request_id <= args.requests

    with output.open("w", encoding="utf-8") as out, ThreadPoolExecutor(
        max_workers=args.concurrency
    ) as pool:
        in_flight: set[Future] = set()

        def submit_one() -> None:
            nonlocal next_request_id
            rid = next_request_id
            next_request_id += 1
            in_flight.add(
                pool.submit(
                    issue_request,
                    url=args.url,
                    model=args.model,
                    strategy=args.strategy,
                    concurrency=args.concurrency,
                    request_id=rid,
                )
            )

        while len(in_flight) < args.concurrency and should_schedule():
            submit_one()

        while in_flight:
            done, _ = wait(in_flight, timeout=1.0, return_when=FIRST_COMPLETED)
            if not done:
                now = time.monotonic()
                if now >= next_summary:
                    with lock:
                        print_live_summary(start_mono, total, ok, err, total_latency_ms)
                    next_summary += 5.0
                continue

            for fut in done:
                in_flight.remove(fut)
                record = fut.result()
                out.write(json.dumps(record) + "\n")
                out.flush()

                with lock:
                    total += 1
                    total_latency_ms += record["latency_ms"]
                    if record["status_code"] == 200 and record["error"] is None:
                        ok += 1
                    else:
                        err += 1

            while len(in_flight) < args.concurrency and should_schedule():
                submit_one()

            now = time.monotonic()
            if now >= next_summary:
                with lock:
                    print_live_summary(start_mono, total, ok, err, total_latency_ms)
                next_summary += 5.0

    print_live_summary(start_mono, total, ok, err, total_latency_ms)
    print(f"Wrote {total} records to {output}")


if __name__ == "__main__":
    main()
