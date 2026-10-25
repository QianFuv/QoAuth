"""Run a bounded prior-binary snapshot/current-binary restore using temporary synthetic data."""
import argparse
import base64
from contextlib import closing
import hashlib
import hmac
import http.cookiejar
import json
import pathlib
import shutil
import socket
import sqlite3
import struct
import subprocess
import tempfile
import time
import urllib.error
import urllib.parse
import urllib.request


class SyntheticBrowser:
    """Drive existing same-origin endpoints with a disposable cookie jar."""

    def __init__(self, origin):
        """Create a fresh isolated browser state."""
        self.origin = origin
        self.client = urllib.request.build_opener(urllib.request.HTTPCookieProcessor(http.cookiejar.CookieJar()))
        self.csrf = self.get("/api/session")["csrf"]

    def get(self, path):
        """Read one bounded JSON response."""
        with self.client.open(self.origin + path, timeout=3) as response:
            return json.load(response)

    def post(self, path, data):
        """Submit JSON with canonical origin and current CSRF proof."""
        request = urllib.request.Request(self.origin + path, json.dumps(data).encode(), headers={"Content-Type": "application/json", "Origin": self.origin, "X-CSRF-Token": self.csrf})
        with self.client.open(request, timeout=3) as response:
            result = json.load(response)
        self.csrf = self.get("/api/session")["csrf"]
        return result


def main():
    """Keep both installations private, preserve the source and enforce real quarantine time."""
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--previous", type=pathlib.Path, required=True)
    parser.add_argument("--current", type=pathlib.Path, required=True)
    args = parser.parse_args()
    workspace = pathlib.Path(__file__).resolve().parent.parent
    cache = workspace / ".cache"
    cache.mkdir(exist_ok=True)
    root = pathlib.Path(tempfile.mkdtemp(prefix="upgrade-synthetic-", dir=cache)).resolve()
    processes, output = [], []
    password = "isolated prior version password canary"
    flags = getattr(subprocess, "CREATE_NO_WINDOW", 0)

    def command(binary, *arguments, secret=None, success=True):
        """Run a bounded maintenance command without logging secrets or request bodies."""
        result = subprocess.run([str(binary.resolve()), *map(str, arguments)], input=secret, text=True, capture_output=True, timeout=30, creationflags=flags)
        output.append(result.stdout + result.stderr)
        if (result.returncode == 0) != success:
            raise AssertionError(f"{arguments[0]} unexpected exit {result.returncode}")
        return result.stdout + result.stderr

    def start(binary, directory, origin, port):
        """Start only the local task process and require bounded discovery readiness."""
        process = subprocess.Popen([str(binary.resolve()), "serve", "--data-dir", str(directory), "--issuer", origin, "--listen", f"127.0.0.1:{port}", "--development-http"], stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True, creationflags=flags)
        processes.append(process)
        deadline = time.monotonic() + 10
        while time.monotonic() < deadline:
            if process.poll() is not None:
                raise AssertionError("temporary server exited before readiness")
            try:
                with urllib.request.urlopen(origin + "/.well-known/openid-configuration", timeout=1) as response:
                    assert json.load(response)["issuer"] == origin
                return process
            except OSError:
                time.sleep(0.1)
        raise AssertionError("temporary startup deadline exceeded")

    def stop(process):
        """Terminate and reap only this script's known child."""
        if process.poll() is None:
            process.terminate()
        stdout, stderr = process.communicate(timeout=10)
        output.append(stdout + stderr)

    try:
        source, snapshot, restored = root / "prior", root / "snapshot", root / "restored"
        with socket.socket() as listener:
            listener.bind(("127.0.0.1", 0))
            port = listener.getsockname()[1]
        origin = f"http://127.0.0.1:{port}"
        command(args.previous, "init", "--data-dir", source, "--username", "owner", secret=password)
        process = start(args.previous, source, origin, port)
        browser = SyntheticBrowser(origin)
        browser.post("/api/login", {"username": "owner", "password": password})
        factor = browser.post("/api/mfa/start", {})["secret"]
        # RFC 6238 reference computation is test-only; product TOTP uses the pinned OTP library.
        digest = hmac.new(base64.b32decode(factor + "=" * (-len(factor) % 8)), struct.pack(">Q", int(time.time()) // 30), hashlib.sha1).digest()
        offset = digest[-1] & 15
        code = f"{(struct.unpack('>I', digest[offset:offset + 4])[0] & 0x7fffffff) % 1000000:06d}"
        recovery = browser.post("/api/mfa/confirm", {"code": code})["recovery_codes"]
        browser.post("/api/users/create", {"username": "prior-member", "password": password})
        browser.post("/api/applications/create", {"application": {"id": "prior-rp", "name": "Prior version RP", "public": True, "redirects": [origin + "/callback"], "logout_redirects": [], "scopes": ["openid"], "audiences": []}})
        stop(process)
        command(args.previous, "backup", "--data-dir", source, "--snapshot", snapshot)
        source_keys = (source / "keys.json").read_bytes()
        source_db = (source / "qoauth.db").read_bytes()
        command(args.current, "restore", "--snapshot", snapshot, "--data-dir", restored, "--confirm")
        assert "recover-owner" in command(args.current, "serve", "--data-dir", restored, "--issuer", origin, "--development-http", success=False)
        command(args.current, "recover-owner", "--data-dir", restored, "--confirm", "--restore-reviewed", secret=password + " renewed")
        assert "quarantine" in command(args.current, "serve", "--data-dir", restored, "--issuer", origin, "--development-http", success=False)
        with closing(sqlite3.connect(f"file:{restored.as_posix()}/qoauth.db?mode=ro", uri=True)) as database:
            ready_after = database.execute("SELECT ready_after FROM installation").fetchone()[0]
            schema = database.execute("SELECT max(version) FROM schema_migrations").fetchone()[0]
            assert database.execute("SELECT count(*) FROM clients WHERE enabled=1").fetchone()[0] == 0
            assert database.execute("SELECT count(*) FROM recovery_codes WHERE consumed=0").fetchone()[0] == 0
        deadline = time.monotonic() + 630
        while time.time() <= ready_after:
            assert time.monotonic() < deadline, "quarantine deadline exceeded"
            time.sleep(min(2, max(0.1, ready_after + 0.1 - time.time())))
        process = start(args.current, restored, origin, port)
        with urllib.request.urlopen(origin + "/readyz", timeout=2) as response:
            assert json.load(response) == {"ready": True}
        assert browser.get("/api/session")["authenticated"] is False
        browser = SyntheticBrowser(origin)
        browser.post("/api/login", {"username": "owner", "password": password + " renewed"})
        state = browser.get("/api/session")
        assert state["assurance"] is False and state["has_factor"] is False
        try:
            browser.get("/api/users")
            raise AssertionError("restored owner bypassed new MFA enrollment")
        except urllib.error.HTTPError as error:
            assert error.code == 403
        stop(process)
        assert (source / "keys.json").read_bytes() == source_keys and (source / "qoauth.db").read_bytes() == source_db
        for secret in [password, factor, *recovery]:
            assert all(secret not in entry for entry in output), "secret appeared in process output"
        report = {"result": "pass", "prior_sha256": hashlib.sha256(args.previous.read_bytes()).hexdigest(), "current_sha256": hashlib.sha256(args.current.read_bytes()).hexdigest(), "schema": schema, "checks": ["prior binary enrollment/user/client", "locked snapshot", "new-directory restore", "unchanged source", "disabled clients", "invalid old recovery/session", "real persisted quarantine", "current readiness", "restricted recovered owner", "temporary state removed"], "limit": "M2.2 schema 7 to current schema 7; not an arbitrary older-schema migration"}
    finally:
        for process in processes:
            if process.poll() is None:
                stop(process)
        assert root.is_relative_to(cache.resolve()) and root.name.startswith("upgrade-synthetic-")
        shutil.rmtree(root)
    print(json.dumps(report))


if __name__ == "__main__":
    main()
