#!/usr/bin/env python3
# Hero3 本地 API 压测脚本；只依赖 Python 标准库。

from __future__ import annotations

import argparse
import json
import random
import statistics
import string
import sys
import threading
import time
import urllib.error
import urllib.parse
import urllib.request
import socket
from concurrent.futures import ThreadPoolExecutor, as_completed
from dataclasses import dataclass
from typing import Any


@dataclass
class Session:
    account_id: str
    player_id: str
    token: str


@dataclass
class Result:
    name: str
    status: int
    elapsed_ms: float
    ok: bool


def request(base_url: str, method: str, path: str, token: str | None = None, body: Any = None) -> tuple[int, Any, float]:
    url = base_url.rstrip("/") + path
    data = None
    headers = {"Content-Type": "application/json"}
    if token:
        headers["Authorization"] = "Bearer " + token
    if body is not None:
        data = json.dumps(body, ensure_ascii=False).encode("utf-8")
    req = urllib.request.Request(url, data=data, headers=headers, method=method)
    started = time.perf_counter()
    try:
        with urllib.request.urlopen(req, timeout=10) as resp:
            raw = resp.read()
            elapsed = (time.perf_counter() - started) * 1000
            return resp.status, json.loads(raw.decode("utf-8") or "{}"), elapsed
    except urllib.error.HTTPError as exc:
        raw = exc.read()
        elapsed = (time.perf_counter() - started) * 1000
        try:
            parsed = json.loads(raw.decode("utf-8") or "{}")
        except json.JSONDecodeError:
            parsed = {"error": raw.decode("utf-8", errors="replace")}
        return exc.code, parsed, elapsed
    except (TimeoutError, socket.timeout, urllib.error.URLError, OSError) as exc:
        elapsed = (time.perf_counter() - started) * 1000
        return 0, {"error": str(exc)}, elapsed


def random_suffix(length: int = 8) -> str:
    alphabet = string.ascii_lowercase + string.digits
    return "".join(random.choice(alphabet) for _ in range(length))


def create_session(base_url: str, prefix: str, index: int) -> Session:
    username = f"{prefix}_{index}_{random_suffix()}"
    password = "Loadtest123!"
    status, account, _ = request(base_url, "POST", "/api/v1/accounts/register", body={
        "username": username,
        "password": password,
    })
    if status != 201:
        raise RuntimeError(f"register failed status={status} body={account}")
    token = account["token"]
    account_id = account["accountId"]
    status, player, _ = request(base_url, "POST", "/api/v1/players/create", token=token, body={
        "accountId": account_id,
        "nickname": f"压测{index}{random_suffix(4)}",
        "faction": "wei",
        "generalId": "caocao",
    })
    if status != 201:
        raise RuntimeError(f"create player failed status={status} body={player}")
    return Session(account_id=account_id, player_id=player["playerId"], token=token)


def seed_resources(base_url: str, session: Session) -> None:
    request(base_url, "POST", "/api/v1/city/resources/fill", token=session.token, body={"playerId": session.player_id})


def pick_building(index: int) -> str:
    candidates = [
        "wood_camp-1",
        "wood_camp-2",
        "stone_quarry-1",
        "iron_mine-1",
        "farm-1",
        "warehouse-1",
    ]
    return candidates[index % len(candidates)]


def get_view(base_url: str, session: Session, name: str, path: str) -> Result:
    status, _, elapsed = request(base_url, "GET", path + "?" + urllib.parse.urlencode({"playerId": session.player_id}), token=session.token)
    return Result(name=name, status=status, elapsed_ms=elapsed, ok=200 <= status < 300)


def post_upgrade(base_url: str, session: Session, op_index: int) -> Result:
    status, _, elapsed = request(base_url, "POST", "/api/v1/city/buildings/upgrade", token=session.token, body={
        "playerId": session.player_id,
        "buildingId": pick_building(op_index),
    })
    return Result(name="upgrade", status=status, elapsed_ms=elapsed, ok=200 <= status < 300)


def post_recruit(base_url: str, session: Session) -> Result:
    status, _, elapsed = request(base_url, "POST", "/api/v1/military/recruit", token=session.token, body={
        "playerId": session.player_id,
        "unitId": "qingZhouArmy",
        "amount": 1,
    })
    return Result(name="recruit", status=status, elapsed_ms=elapsed, ok=200 <= status < 300)


def run_operation(base_url: str, session: Session, op_index: int, mode: str) -> Result:
    roll = random.random()
    if mode == "frontend":
        if roll < 0.25:
            return get_view(base_url, session, "resource_view", "/api/v1/resources/view")
        if roll < 0.45:
            return get_view(base_url, session, "city_view", "/api/v1/city/view")
        if roll < 0.60:
            return get_view(base_url, session, "military_view", "/api/v1/military/view")
        if roll < 0.75:
            return get_view(base_url, session, "npc_list", "/api/v1/map/npc-cities")
        if roll < 0.85:
            return post_upgrade(base_url, session, op_index)
        if roll < 0.95:
            return post_recruit(base_url, session)
        return get_view(base_url, session, "summary_view", "/api/v1/game/summary")

    if mode == "readonly":
        if roll < 0.25:
            return get_view(base_url, session, "resource_view", "/api/v1/resources/view")
        if roll < 0.45:
            return get_view(base_url, session, "city_view", "/api/v1/city/view")
        if roll < 0.60:
            return get_view(base_url, session, "military_view", "/api/v1/military/view")
        if roll < 0.72:
            return get_view(base_url, session, "inventory_view", "/api/v1/inventory/view")
        if roll < 0.84:
            return get_view(base_url, session, "generals_view", "/api/v1/generals/view")
        if roll < 0.94:
            return get_view(base_url, session, "npc_list", "/api/v1/map/npc-cities")
        return get_view(base_url, session, "summary_view", "/api/v1/game/summary")

    if mode == "write-local":
        if roll < 0.45:
            return post_upgrade(base_url, session, op_index)
        if roll < 0.85:
            return post_recruit(base_url, session)
        return get_view(base_url, session, "resource_view", "/api/v1/resources/view")

    if roll < 0.55:
        name = "get_state"
        path = "/api/v1/game/state?" + urllib.parse.urlencode({"playerId": session.player_id})
        status, _, elapsed = request(base_url, "GET", path, token=session.token)
    elif roll < 0.75:
        name = "npc_list"
        path = "/api/v1/map/npc-cities?" + urllib.parse.urlencode({"playerId": session.player_id})
        status, _, elapsed = request(base_url, "GET", path, token=session.token)
    elif roll < 0.90:
        return post_upgrade(base_url, session, op_index)
    else:
        return post_recruit(base_url, session)
    # 409/422 属于业务拒绝，不算 HTTP 系统错误，但会在结果里单独看到成功率下降。
    ok = 200 <= status < 300
    return Result(name=name, status=status, elapsed_ms=elapsed, ok=ok)


def percentile(values: list[float], pct: float) -> float:
    if not values:
        return 0.0
    ordered = sorted(values)
    idx = min(len(ordered) - 1, int((pct / 100.0) * (len(ordered) - 1)))
    return ordered[idx]


def summarize(label: str, results: list[Result], wall_seconds: float) -> dict[str, Any]:
    latencies = [r.elapsed_ms for r in results]
    ok_count = sum(1 for r in results if r.ok)
    by_status: dict[int, int] = {}
    by_name: dict[str, int] = {}
    for result in results:
        by_status[result.status] = by_status.get(result.status, 0) + 1
        by_name[result.name] = by_name.get(result.name, 0) + 1
    return {
        "label": label,
        "requests": len(results),
        "ok": ok_count,
        "success_rate": ok_count / len(results) if results else 0,
        "rps": len(results) / wall_seconds if wall_seconds > 0 else 0,
        "avg_ms": statistics.mean(latencies) if latencies else 0,
        "p50_ms": percentile(latencies, 50),
        "p95_ms": percentile(latencies, 95),
        "p99_ms": percentile(latencies, 99),
        "max_ms": max(latencies) if latencies else 0,
        "by_status": dict(sorted(by_status.items())),
        "by_name": dict(sorted(by_name.items())),
    }


def run_stage(base_url: str, sessions: list[Session], concurrency: int, requests: int, mode: str) -> dict[str, Any]:
    results: list[Result] = []
    lock = threading.Lock()
    started = time.perf_counter()
    with ThreadPoolExecutor(max_workers=concurrency) as pool:
        futures = []
        for i in range(requests):
            session = sessions[i % len(sessions)]
            futures.append(pool.submit(run_operation, base_url, session, i, mode))
        for future in as_completed(futures):
            with lock:
                results.append(future.result())
    wall_seconds = time.perf_counter() - started
    return summarize(f"{mode}_c{concurrency}_n{requests}", results, wall_seconds)


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--base-url", default="http://localhost:8080")
    parser.add_argument("--users", type=int, default=80)
    parser.add_argument("--stages", default="10:200,25:500,50:1000,100:2000")
    parser.add_argument("--mode", choices=["legacy", "frontend", "readonly", "write-local"], default="frontend")
    parser.add_argument("--prefix", default="lt")
    args = parser.parse_args()

    health_status, health, _ = request(args.base_url, "GET", "/healthz")
    if health_status != 200:
        print(f"health check failed: {health_status} {health}", file=sys.stderr)
        return 1

    print(f"Preparing {args.users} users against {args.base_url} ...", flush=True)
    sessions: list[Session] = []
    with ThreadPoolExecutor(max_workers=min(16, args.users)) as pool:
        futures = [pool.submit(create_session, args.base_url, args.prefix, i) for i in range(args.users)]
        for future in as_completed(futures):
            sessions.append(future.result())

    print("Seeding resources ...", flush=True)
    with ThreadPoolExecutor(max_workers=min(16, len(sessions))) as pool:
        list(pool.map(lambda s: seed_resources(args.base_url, s), sessions))

    summaries = []
    for raw_stage in args.stages.split(","):
        concurrency_raw, requests_raw = raw_stage.split(":", 1)
        concurrency = int(concurrency_raw)
        requests = int(requests_raw)
        print(f"Running stage mode={args.mode} concurrency={concurrency} requests={requests} ...", flush=True)
        summary = run_stage(args.base_url, sessions, concurrency, requests, args.mode)
        summaries.append(summary)
        print(json.dumps(summary, ensure_ascii=False), flush=True)

    print("SUMMARY_JSON=" + json.dumps(summaries, ensure_ascii=False))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
