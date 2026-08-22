#!/usr/bin/env python3
"""Mock-mode API smoke. Cost: ¥0. No metered providers."""

from __future__ import annotations

import json
import os
import sys
import urllib.error
import urllib.request

BASE = os.environ.get("API_BASE", "http://127.0.0.1:27332").rstrip("/")


def req(method: str, path: str, body=None, token: str | None = None):
    data = None if body is None else json.dumps(body).encode("utf-8")
    headers = {"Content-Type": "application/json"}
    if token:
        headers["Authorization"] = f"Bearer {token}"
    r = urllib.request.Request(BASE + path, data=data, headers=headers, method=method)
    try:
        with urllib.request.urlopen(r, timeout=20) as resp:
            raw = resp.read().decode("utf-8")
            return resp.status, json.loads(raw) if raw else {}
    except urllib.error.HTTPError as e:
        raw = e.read().decode("utf-8")
        try:
            payload = json.loads(raw) if raw else {}
        except json.JSONDecodeError:
            payload = {"raw": raw}
        return e.code, payload


def check(name: str, cond: bool, detail: str = "") -> None:
    status = "PASS" if cond else "FAIL"
    print(f"[{status}] {name} {detail}".rstrip())
    if not cond:
        raise SystemExit(1)


def main() -> None:
    code, body = req("GET", "/api/v1/health")
    check("Health Check", code == 200 and body.get("code") == 0, str(body)[:200])

    code, body = req("GET", "/api/v1/rules")
    check("Auth required", code == 401, str(code))

    code, body = req("POST", "/api/v1/auth/login", {"username": "admin", "password": "bad"})
    check("Bad password", code == 401, str(body)[:200])

    code, body = req("POST", "/api/v1/auth/login", {"username": "admin", "password": "Admin@12345"})
    token = (body.get("data") or {}).get("token")
    check("Login", code == 200 and bool(token))

    code, body = req("GET", "/api/v1/rules", token=token)
    check("List rules", code == 200 and body.get("code") == 0, str(body)[:240])

    code, body = req(
        "POST",
        "/api/v1/tasks",
        {
            "name": "smoke-task",
            "rule_id": 1,
            "seed_urls": ["http://mock-target/list.html"],
            "max_depth": 1,
            "concurrency": 2,
        },
        token=token,
    )
    check("Create task", code in (200, 201, 202) and body.get("code") == 0, str(body)[:240])

    code, body = req("GET", "/api/v1/cluster/nodes", token=token)
    check("Cluster nodes", code == 200, str(body)[:200])

    code, body = req("GET", "/api/v1/proxies", token=token)
    check("Proxy pool", code == 200, str(body)[:200])

    print("[PASS] Mock API Response")
    print("Cost ¥0")


if __name__ == "__main__":
    try:
        main()
    except SystemExit:
        raise
    except Exception as exc:  # noqa: BLE001
        print(f"[FAIL] smoke crashed: {exc}")
        sys.exit(1)
