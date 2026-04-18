import argparse
import asyncio
import csv
import json
import time
import urllib.error
import urllib.request
from pathlib import Path
from urllib.parse import urlsplit, urlunsplit


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="InferFlow starter load generator")
    parser.add_argument("--url", default="http://localhost:8080/v1/chat/completions")
    parser.add_argument("--requests", type=int, default=10)
    parser.add_argument("--model", default="mock-llm")
    parser.add_argument("--output", default="results/sample.csv")
    parser.add_argument("--strategy", default="round_robin")
    parser.add_argument("--strategies", default="")
    # repeat-factor N: every N-th request reuses a fixed prompt (drives kv_aware cache hits)
    parser.add_argument("--repeat-factor", type=int, default=0,
                        help="Reuse the same prompt every N requests (0 = all unique)")
    return parser.parse_args()


REPEATED_PROMPT = "Tell me about distributed systems and load balancing"

def build_payload(model: str, request_id: int, repeat_factor: int = 0) -> bytes:
    if repeat_factor > 0 and request_id % repeat_factor == 0:
        content = REPEATED_PROMPT
    else:
        content = f"Starter request {request_id} from InferFlow loadgen"
    payload = {
        "model": model,
        "messages": [{"role": "user", "content": content}],
        "stream": False,
    }
    return json.dumps(payload).encode("utf-8")


def strategy_url(chat_url: str) -> str:
    parts = urlsplit(chat_url)
    base_path = parts.path.removesuffix("/v1/chat/completions")
    return urlunsplit((parts.scheme, parts.netloc, f"{base_path}/strategy", "", ""))


def normalize_strategy_list(args: argparse.Namespace) -> list[str]:
    if args.strategies.strip():
        return [part.strip() for part in args.strategies.split(",") if part.strip()]
    return [args.strategy.strip()]


async def switch_strategy(url: str, strategy: str) -> None:
    payload = json.dumps({"strategy": strategy}).encode("utf-8")

    def _send() -> None:
        req = urllib.request.Request(
            url,
            data=payload,
            headers={"Content-Type": "application/json"},
            method="PUT",
        )
        with urllib.request.urlopen(req, timeout=10):
            return None

    await asyncio.to_thread(_send)


async def issue_request(url: str, model: str, request_id: int, strategy: str, repeat_factor: int = 0) -> dict:
    started = time.time()
    is_repeat = repeat_factor > 0 and request_id % repeat_factor == 0
    prompt = REPEATED_PROMPT if is_repeat else f"Starter request {request_id} from InferFlow loadgen"

    def _send() -> dict:
        req = urllib.request.Request(
            url,
            data=build_payload(model, request_id, repeat_factor),
            headers={"Content-Type": "application/json"},
            method="POST",
        )
        with urllib.request.urlopen(req, timeout=10) as resp:
            body = json.loads(resp.read().decode("utf-8"))
            return {
                "status": resp.status,
                "body": body,
                "backend": resp.headers.get("X-Inferflow-Backend", ""),
                "strategy": resp.headers.get("X-Inferflow-Strategy", strategy),
            }

    try:
        result = await asyncio.to_thread(_send)
        total_ms = int((time.time() - started) * 1000)
        text = result["body"]["choices"][0]["message"]["content"]
        return {
            "timestamp": int(started),
            "prompt_length": len(prompt),
            "strategy": result["strategy"],
            "backend": result["backend"],
            "ttft_ms": total_ms,
            "total_ms": total_ms,
            "tokens_generated": len(text.split()),
            "cache_hit": is_repeat,
            "error": "",
        }
    except urllib.error.HTTPError as exc:
        total_ms = int((time.time() - started) * 1000)
        return {
            "timestamp": int(started),
            "prompt_length": len(prompt),
            "strategy": strategy,
            "backend": "",
            "ttft_ms": total_ms,
            "total_ms": total_ms,
            "tokens_generated": 0,
            "cache_hit": False,
            "error": f"http_{exc.code}",
        }
    except Exception as exc:  # noqa: BLE001
        total_ms = int((time.time() - started) * 1000)
        return {
            "timestamp": int(started),
            "prompt_length": len(prompt),
            "strategy": strategy,
            "backend": "",
            "ttft_ms": total_ms,
            "total_ms": total_ms,
            "tokens_generated": 0,
            "cache_hit": False,
            "error": str(exc),
        }


async def main() -> None:
    args = parse_args()
    rows = []
    for strategy in normalize_strategy_list(args):
        await switch_strategy(strategy_url(args.url), strategy)
        rows.extend(
            await asyncio.gather(
                *(issue_request(args.url, args.model, idx, strategy, args.repeat_factor)
                  for idx in range(args.requests))
            )
        )

    output = Path(args.output)
    output.parent.mkdir(parents=True, exist_ok=True)

    with output.open("w", newline="", encoding="utf-8") as handle:
        writer = csv.DictWriter(
            handle,
            fieldnames=[
                "timestamp",
                "prompt_length",
                "strategy",
                "backend",
                "ttft_ms",
                "total_ms",
                "tokens_generated",
                "cache_hit",
                "error",
            ],
        )
        writer.writeheader()
        writer.writerows(rows)

    print(f"Wrote {len(rows)} rows to {output}")
    _print_summary(rows)


def _print_summary(rows: list[dict]) -> None:
    import statistics
    strategies = sorted({r["strategy"] for r in rows})
    print("\n--- Summary ---")
    for strat in strategies:
        strat_rows = [r for r in rows if r["strategy"] == strat and not r["error"]]
        if not strat_rows:
            print(f"{strat}: no successful requests")
            continue
        latencies = sorted(r["total_ms"] for r in strat_rows)
        backends = {}
        for r in strat_rows:
            backends[r["backend"]] = backends.get(r["backend"], 0) + 1
        hits = sum(1 for r in strat_rows if r.get("cache_hit"))
        p50 = statistics.median(latencies)
        p95 = latencies[int(len(latencies) * 0.95)]
        print(f"\n{strat} ({len(strat_rows)} ok / {len(rows)//len(strategies)} total)")
        print(f"  latency  p50={p50}ms  p95={p95}ms  min={latencies[0]}ms  max={latencies[-1]}ms")
        print(f"  backends {backends}")
        if hits:
            print(f"  cache_hit_requests {hits}/{len(strat_rows)}")


if __name__ == "__main__":
    asyncio.run(main())
