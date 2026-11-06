#!/usr/bin/env python3
"""Exercise an Arabica image using Docker or Podman and isolated disposable state."""

import argparse
import io
import json
import pathlib
import re
import sqlite3
import subprocess
import tarfile
import tempfile
import time
import urllib.error
import urllib.request
import uuid


def require(condition, message):
    if not condition:
        raise RuntimeError(message)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("image", nargs="?", default="arabica:local")
    parser.add_argument("--engine", choices=("docker", "podman"), default="docker")
    args = parser.parse_args()
    prefix = "arabica-smoke-" + uuid.uuid4().hex[:12]
    volume = prefix + "-data"
    containers = []
    client = urllib.request.build_opener(urllib.request.ProxyHandler({}))

    def engine(*arguments, data=None, timeout=60):
        result = subprocess.run(
            [args.engine, *arguments], input=data, stdout=subprocess.PIPE,
            stderr=subprocess.PIPE, timeout=timeout,
        )
        if result.returncode:
            raise RuntimeError(
                f"{args.engine} {' '.join(arguments)} failed: "
                + result.stderr.decode(errors="replace")
            )
        return result.stdout

    def inspect(name):
        return json.loads(engine("inspect", name))[0]

    def start(suffix, *extra):
        name = prefix + suffix
        containers.append(name)
        engine(
            "run", "--detach", "--name", name, "--read-only",
            "--cap-drop=ALL", "--security-opt=no-new-privileges:true",
            "--tmpfs", "/tmp:rw,nosuid,nodev,size=64m,mode=1777",
            *extra, args.image,
        )
        return name

    def start_app(suffix):
        name = start(
            suffix, "--mount", f"type=volume,source={volume},target=/var/lib/arabica",
            "--publish", "127.0.0.1::18910",
            "--env", "ARABICA_PUBLIC_URL=https://preview.example.com",
            "--env", "SECURE_COOKIES=true",
        )
        port = engine("port", name, "18910/tcp").decode().strip().split(":")[-1]
        base = "http://127.0.0.1:" + port
        deadline = time.monotonic() + 90
        while time.monotonic() < deadline:
            require(inspect(name)["State"]["Running"], "application exited during startup")
            try:
                with client.open(base + "/healthz", timeout=2) as response:
                    health = json.load(response)
                require(health["feed_index"]["healthy"], "SQLite health check failed")
                return name, base
            except (urllib.error.URLError, TimeoutError, ConnectionError):
                time.sleep(0.25)
        raise RuntimeError("application did not become healthy within 90 seconds")

    def stop(name):
        engine("stop", "--time", "20", name, timeout=30)
        state = inspect(name)["State"]
        require(state["ExitCode"] == 0, f"unclean shutdown: {state}")

    def download_db(name, destination):
        engine("cp", name + ":/var/lib/arabica/arabica.db", str(destination))
        with sqlite3.connect(destination) as db:
            require(db.execute("PRAGMA integrity_check").fetchone()[0] == "ok", "database integrity check failed")

    try:
        config = json.loads(engine("image", "inspect", args.image))[0]["Config"]
        require(config["User"] == "10001:10001", "image must run as uid/gid 10001")
        require("18910/tcp" in config["ExposedPorts"], "image must expose port 18910")
        engine("volume", "create", volume)
        with tempfile.TemporaryDirectory(prefix=prefix) as directory:
            directory = pathlib.Path(directory)
            name, base = start_app("-first")
            environment = dict(item.split("=", 1) for item in config["Env"])
            cert = environment.get("SSL_CERT_FILE", "/etc/ssl/certs/ca-certificates.crt")
            engine("cp", "-L", name + ":" + cert, str(directory / "ca.crt"))
            require("BEGIN CERTIFICATE" in (directory / "ca.crt").read_text(), "CA bundle is missing")

            def get(path):
                with client.open(base + path, timeout=10) as response:
                    require(response.status == 200, f"{path} did not return 200")
                    return response.headers, response.read()

            headers, shell = get("/")
            require("text/html" in headers.get("Content-Type", ""), "homepage is not HTML")
            html = shell.decode()
            require("<html" in html, "SPA shell is missing")
            assets = set(re.findall(r"[\"'](/_app/[^\"']+\.js)[\"']", html))
            require(assets, "SPA JavaScript references are missing")
            for asset in assets:
                asset_headers, body = get(asset)
                require("javascript" in asset_headers.get("Content-Type", ""), "SPA asset returned fallback HTML")
                require(body, "SPA asset is empty")
            for path in ("/explore", "/static/favicon.svg", "/static/fonts/fraunces-latin.woff2", "/static/css/output.css", "/robots.txt", "/.well-known/security.txt"):
                require(get(path)[1], f"{path} is empty")
            metadata = json.loads(get("/.well-known/oauth-client-metadata.json")[1])
            require(metadata["client_id"] == "https://preview.example.com/.well-known/oauth-client-metadata.json", "OAuth client ID mismatch")
            require(metadata["redirect_uris"] == ["https://preview.example.com/oauth/callback"], "OAuth callback mismatch")
            session = json.loads(get("/api/session")[1])
            require(isinstance(session, dict), "session API is not an object")
            print("PASS nonroot/read-only startup, SQLite health, SPA/direct loads, assets, OAuth metadata, session API, CA bundle", flush=True)

            stop(name)
            db_path = directory / "arabica.db"
            download_db(name, db_path)
            marker = uuid.uuid4().hex
            with sqlite3.connect(db_path) as db:
                db.execute("CREATE TABLE container_smoke_marker (value TEXT NOT NULL)")
                db.execute("INSERT INTO container_smoke_marker VALUES (?)", (marker,))
            archive = io.BytesIO()
            with tarfile.open(fileobj=archive, mode="w") as tar:
                info = tar.gettarinfo(str(db_path), arcname="arabica.db")
                info.uid = info.gid = 10001
                info.uname = info.gname = "arabica"
                info.mode = 0o600
                with db_path.open("rb") as source:
                    tar.addfile(info, source)
            engine("cp", "--archive", "-", name + ":/var/lib/arabica/", data=archive.getvalue())
            engine("rm", name)
            name, base = start_app("-second")
            stop(name)
            db_path.unlink()
            download_db(name, db_path)
            with sqlite3.connect(db_path) as db:
                require(db.execute("SELECT value FROM container_smoke_marker").fetchone() == (marker,), "persistent state was lost on container replacement")
            print("PASS SIGTERM shutdown, database integrity, persistent state across container replacement", flush=True)

            bad = start("-invalid", "--env", "ARABICA_AI_CONFIG=/missing-container-smoke-config.json")
            exit_code = int(engine("wait", bad, timeout=20).decode().strip())
            require(exit_code != 0, "invalid explicit AI configuration did not fail startup")
            logs = engine("logs", bad).decode(errors="replace")
            require("Invalid assistant configuration" in logs, "configuration error was not reported")
            print("PASS invalid configuration fails startup", flush=True)
        print(f"PASS container smoke tests: {args.image}", flush=True)
    except Exception:
        for name in containers:
            result = subprocess.run([args.engine, "logs", "--tail", "40", name], capture_output=True, timeout=15)
            print(result.stdout.decode(errors="replace") + result.stderr.decode(errors="replace"))
        raise
    finally:
        for name in containers:
            subprocess.run([args.engine, "rm", "--force", name], capture_output=True, timeout=30)
        subprocess.run([args.engine, "volume", "rm", volume], capture_output=True, timeout=30)


if __name__ == "__main__":
    main()
