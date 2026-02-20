#!/usr/bin/env python3
"""
inference-traffic.py — Sustained high-volume traffic to vLLM inference pool via NGF gateway

Sends concurrent OpenAI-compatible chat completion requests through the gateway's
inference-route to stress-test the vLLM inference pool and populate observability dashboards.

Uses only Python stdlib (asyncio + urllib.request) — no pip dependencies required.

Usage:
    ./inference-traffic.py
    ./inference-traffic.py --duration 60 --concurrency 12
    ./inference-traffic.py --elb-host my-elb.example.com --model my-model
"""

import argparse
import asyncio
import json
import os
import sys
import time
import urllib.request
import urllib.error


def parse_args():
    parser = argparse.ArgumentParser(description="Inference pool load test via NGF gateway")
    parser.add_argument(
        "--elb-host",
        default=os.environ.get(
            "ELB_HOST",
            "k8s-nginxgat-llmgatew-bda66d5efd-8abb027b3d9f5051.elb.us-east-1.amazonaws.com",
        ),
        help="Gateway ELB hostname (default: env ELB_HOST or the kph-demo ELB)",
    )
    parser.add_argument(
        "--virtual-host",
        default=os.environ.get("VIRTUAL_HOST", "inference.llm.local"),
        help="Host header for the inference route (default: inference.llm.local)",
    )
    parser.add_argument(
        "--duration",
        type=int,
        default=int(os.environ.get("DURATION", "300")),
        help="Test duration in seconds (default: 300)",
    )
    parser.add_argument(
        "--concurrency",
        type=int,
        default=int(os.environ.get("CONCURRENCY", "24")),
        help="Number of concurrent workers (default: 24)",
    )
    parser.add_argument(
        "--model",
        default=os.environ.get("MODEL", "TinyLlama/TinyLlama-1.1B-Chat-v1.0"),
        help="Model name for the request payload",
    )
    parser.add_argument(
        "--max-tokens",
        type=int,
        default=int(os.environ.get("MAX_TOKENS", "50")),
        help="Max tokens per response (default: 50)",
    )
    parser.add_argument(
        "--prompt",
        default="In one sentence, what is machine learning?",
        help="Prompt to send in each request",
    )
    return parser.parse_args()


def make_payload(model: str, prompt: str, max_tokens: int) -> bytes:
    return json.dumps({
        "model": model,
        "messages": [{"role": "user", "content": prompt}],
        "max_tokens": max_tokens,
        "stream": False,
    }).encode("utf-8")


class Stats:
    def __init__(self):
        self.ok = 0
        self.err = 0
        self.total_latency = 0.0
        self.min_latency = float("inf")
        self.max_latency = 0.0
        self.start_time = time.time()

    def record_success(self, latency: float):
        self.ok += 1
        self.total_latency += latency
        self.min_latency = min(self.min_latency, latency)
        self.max_latency = max(self.max_latency, latency)

    def record_error(self):
        self.err += 1

    @property
    def total(self):
        return self.ok + self.err

    @property
    def elapsed(self):
        return time.time() - self.start_time

    @property
    def rps(self):
        e = self.elapsed
        return self.total / e if e > 0 else 0

    @property
    def avg_latency(self):
        return (self.total_latency / self.ok * 1000) if self.ok > 0 else 0

    def summary(self):
        return (
            f"{self.ok} ok, {self.err} err, "
            f"{self.rps:.1f} req/s, "
            f"avg {self.avg_latency:.0f}ms"
        )


async def send_request(endpoint: str, payload: bytes, headers: dict, timeout: int = 30):
    """Send a single HTTP POST request using urllib (blocking, run in executor)."""
    loop = asyncio.get_event_loop()

    def do_request():
        req = urllib.request.Request(endpoint, data=payload, headers=headers, method="POST")
        t0 = time.time()
        try:
            with urllib.request.urlopen(req, timeout=timeout) as resp:
                resp.read()
                return resp.status, time.time() - t0
        except urllib.error.HTTPError as e:
            return e.code, time.time() - t0
        except Exception:
            return 0, time.time() - t0

    return await loop.run_in_executor(None, do_request)


async def worker(
    worker_id: int,
    endpoint: str,
    payload: bytes,
    headers: dict,
    duration: int,
    stats: Stats,
):
    """Continuously send requests until duration expires."""
    end_time = stats.start_time + duration
    while time.time() < end_time:
        status, latency = await send_request(endpoint, payload, headers)
        if 200 <= status < 300:
            stats.record_success(latency)
        else:
            stats.record_error()

        # Progress output every 50 requests
        if stats.total % 50 == 0 and stats.total > 0:
            elapsed = int(stats.elapsed)
            remaining = max(0, duration - elapsed)
            print(
                f"  [{elapsed:>3}s] {stats.summary()} | {remaining}s remaining",
                flush=True,
            )


async def run_load_test(args):
    endpoint = f"http://{args.elb_host}/v1/chat/completions"
    payload = make_payload(args.model, args.prompt, args.max_tokens)
    headers = {
        "Host": args.virtual_host,
        "Content-Type": "application/json",
    }

    print("=" * 52)
    print("  Inference Pool Traffic Generator")
    print("=" * 52)
    print(f"  ELB:         {args.elb_host}")
    print(f"  Host Header: {args.virtual_host}")
    print(f"  Endpoint:    POST /v1/chat/completions")
    print(f"  Model:       {args.model}")
    print(f"  Max Tokens:  {args.max_tokens}")
    print(f"  Duration:    {args.duration}s")
    print(f"  Concurrency: {args.concurrency} workers")
    print("=" * 52)
    print()

    # Warm-up: single request to verify connectivity
    print("  Warm-up request...", end=" ", flush=True)
    status, latency = await send_request(endpoint, payload, headers, timeout=60)
    if 200 <= status < 300:
        print(f"OK ({status}, {latency*1000:.0f}ms)")
    else:
        print(f"FAILED (status={status}, {latency*1000:.0f}ms)")
        print("  WARNING: Warm-up request failed. Proceeding anyway...")
    print()

    stats = Stats()

    print(f"  [{0:>3}s] Starting {args.concurrency} workers for {args.duration}s...")
    tasks = [
        asyncio.create_task(
            worker(i, endpoint, payload, headers, args.duration, stats)
        )
        for i in range(args.concurrency)
    ]
    await asyncio.gather(*tasks)

    print()
    print("=" * 52)
    print("  Inference Traffic — Final Summary")
    print("=" * 52)
    print(f"  Duration:    {stats.elapsed:.0f}s")
    print(f"  Concurrency: {args.concurrency}")
    print(f"  Total:       {stats.total}")
    print(f"  Success:     {stats.ok}")
    print(f"  Errors:      {stats.err}")
    if stats.ok > 0:
        print(f"  Avg RPS:     {stats.rps:.2f}")
        print(f"  Avg Latency: {stats.avg_latency:.0f}ms")
        print(f"  Min Latency: {stats.min_latency*1000:.0f}ms")
        print(f"  Max Latency: {stats.max_latency*1000:.0f}ms")
    if stats.total > 0:
        success_rate = stats.ok / stats.total * 100
        print(f"  Success Rate:{success_rate:.1f}%")
    print("=" * 52)

    return 0 if stats.err < stats.total * 0.1 else 1  # fail if >10% errors


def main():
    args = parse_args()
    exit_code = asyncio.run(run_load_test(args))
    sys.exit(exit_code)


if __name__ == "__main__":
    main()
