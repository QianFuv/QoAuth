"""Run existing Linux test binaries and a disposable native executable lifecycle."""
import argparse
import hashlib
import json
import os
import pathlib
import re
import shutil
import socket
import subprocess
import tempfile
import time
import urllib.request


PACKAGES = (
    "cmd-qoauth", "internal-fixture", "internal-identity", "internal-maintenance",
    "internal-oidc", "internal-store", "internal-webassets", "examples-go-rp",
)


def run_suites(directory, temporary, output):
    """Execute native tests serially, retaining logs outside disposable credential state."""
    environment = dict(os.environ, GOMAXPROCS="2", TMPDIR=str(temporary))
    for name in list(environment):
        if name.startswith("QOAUTH_"):
            del environment[name]
    results = []
    for package in PACKAGES:
        binary = temporary / (package + ".test")
        shutil.copyfile(directory / binary.name, binary)
        binary.chmod(0o700)
        result = subprocess.run(
            [str(binary), "-test.v", "-test.timeout=180s", "-test.parallel=2"],
            cwd=temporary, env=environment, capture_output=True, text=True, timeout=195,
        )
        transcript = result.stdout + result.stderr
        (output / (package + ".txt")).write_text(transcript, encoding="utf-8")
        results.append({"package": package, "exit": result.returncode,
                        "passed": len(re.findall(r"^\s*--- PASS:", transcript, re.M)),
                        "skipped": len(re.findall(r"^\s*--- SKIP:", transcript, re.M))})
        if result.returncode:
            raise RuntimeError(f"Native test failed: {package}; inspect its retained log")
    return results


def smoke(source_binary, temporary):
    """Check CLI wiring, ext4 privacy, graceful shutdown and maintenance without real users."""
    binary = temporary / "qoauth"
    shutil.copyfile(source_binary, binary)
    binary.chmod(0o700)
    password = "isolated Linux lifecycle credential canary"
    processes, transcript = [], []

    def command(*arguments, secret=None, succeeds=True):
        """Invoke only the copied executable with bounded stdin and no secret arguments."""
        result = subprocess.run([str(binary), *map(str, arguments)], input=secret,
                                capture_output=True, text=True, timeout=30)
        transcript.append(result.stdout + result.stderr)
        if (result.returncode == 0) != succeeds:
            raise RuntimeError(f"Unexpected native command result: {arguments[0]} ({result.returncode})")
        return result.stdout + result.stderr

    def start(directory, origin, port):
        """Start a task-owned loopback listener and require actual embedded UI/readiness."""
        process = subprocess.Popen([str(binary), "serve", "--data-dir", str(directory),
                                    "--issuer", origin, "--listen", f"127.0.0.1:{port}",
                                    "--development-http"], stdout=subprocess.PIPE,
                                   stderr=subprocess.PIPE, text=True)
        processes.append(process)
        deadline = time.monotonic() + 12
        while time.monotonic() < deadline:
            if process.poll() is not None:
                raise RuntimeError("Native server exited before readiness")
            try:
                with urllib.request.urlopen(origin + "/readyz", timeout=1) as response:
                    assert json.load(response) == {"ready": True}
                    assert response.headers.get("Set-Cookie") is None
                with urllib.request.urlopen(origin + "/.well-known/openid-configuration", timeout=1) as response:
                    metadata = json.load(response)
                assert metadata["issuer"] == origin
                with urllib.request.urlopen(metadata["jwks_uri"], timeout=1) as response:
                    keys = json.load(response)
                with urllib.request.urlopen(origin + "/login", timeout=1) as response:
                    assert "<script" in response.read().decode()
                return process, keys
            except OSError:
                time.sleep(0.1)
        raise RuntimeError("Native readiness deadline exceeded")

    def stop(process):
        """Send SIGTERM, await graceful exit and retain only the task process's output."""
        if process.poll() is None:
            process.terminate()
        stdout, stderr = process.communicate(timeout=15)
        transcript.append(stdout + stderr)
        assert process.returncode == 0, "Native graceful shutdown failed"

    try:
        state, snapshot, restored = temporary / "state", temporary / "snapshot", temporary / "restored"
        command("init", "--data-dir", state, "--username", "owner", secret=password)
        assert state.stat().st_uid == os.geteuid() and state.stat().st_mode & 0o777 == 0o700
        for name in ("keys.json", "qoauth.db", "installation.lock"):
            info = (state / name).stat()
            assert info.st_uid == os.geteuid() and info.st_mode & 0o777 == 0o600
        with socket.socket() as listener:
            listener.bind(("127.0.0.1", 0))
            port = listener.getsockname()[1]
        origin = f"http://127.0.0.1:{port}"
        process, first = start(state, origin, port)
        assert "busy" in command("backup", "--data-dir", state, "--snapshot", snapshot, succeeds=False)
        assert not snapshot.exists()
        stop(process)
        process, second = start(state, origin, port)
        assert first == second
        stop(process)
        command("backup", "--data-dir", state, "--snapshot", snapshot)
        command("backup", "--data-dir", state, "--snapshot", snapshot, succeeds=False)
        original_keys = json.loads((state / "keys.json").read_text())
        command("rotate-signing", "--data-dir", state, "--confirm")
        rotated_keys = json.loads((state / "keys.json").read_text())
        assert original_keys["Encryption"] == rotated_keys["Encryption"]
        assert original_keys["Signing"] != rotated_keys["Signing"]
        process, rotated = start(state, origin, port)
        assert len(rotated["keys"]) == 2
        assert first["keys"][0]["kid"] in [key["kid"] for key in rotated["keys"]]
        stop(process)
        command("rotate-signing", "--data-dir", state, "--confirm", succeeds=False)
        command("restore", "--snapshot", snapshot, "--data-dir", restored, "--confirm")
        restored_keys = json.loads((restored / "keys.json").read_text())
        assert original_keys["Encryption"] == restored_keys["Encryption"]
        assert original_keys["Signing"] != restored_keys["Signing"] and "Previous" not in restored_keys
        assert "recover-owner" in command("serve", "--data-dir", restored, "--issuer", origin,
                                          "--development-http", succeeds=False)
        command("recover-owner", "--data-dir", restored, "--confirm", "--restore-reviewed",
                secret=password + " renewed")
        assert "quarantine" in command("serve", "--data-dir", restored, "--issuer", origin,
                                        "--development-http", succeeds=False)
        counts = json.loads(command("cleanup", "--data-dir", state))
        assert len(counts) == 7 and all(isinstance(value, int) for value in counts.values())
        command("cleanup", "--data-dir", state, "--batch-size", "0", succeeds=False)
        assert all(password not in entry and original_keys["Encryption"] not in entry for entry in transcript)
        return ["private ext4 state", "embedded UI/discovery/readiness", "SIGTERM and restart/JWKS",
                "installation lock", "consistent snapshot", "rotation/AES continuity",
                "isolated restore/recovery/quarantine", "bounded cleanup", "secret-free output"]
    finally:
        for process in processes:
            if process.poll() is None:
                process.terminate()
                try:
                    process.communicate(timeout=15)
                except subprocess.TimeoutExpired:
                    process.kill()
                    process.communicate(timeout=5)


def main():
    """Require non-root Linux, create one private test tree and remove only that tree."""
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--tests-dir", type=pathlib.Path, required=True)
    parser.add_argument("--binary", type=pathlib.Path, required=True)
    parser.add_argument("--output", type=pathlib.Path, required=True)
    args = parser.parse_args()
    if not os.sys.platform.startswith("linux") or os.geteuid() == 0:
        raise RuntimeError("Run as an existing unprivileged Linux user")
    args.output.mkdir(parents=True, exist_ok=False)
    os.nice(10)
    os.environ["GOMAXPROCS"] = "2"
    with tempfile.TemporaryDirectory(prefix="qoauth-linux-validation-") as directory:
        temporary = pathlib.Path(directory)
        suites = run_suites(args.tests_dir.resolve(), temporary, args.output.resolve())
        checks = smoke(args.binary.resolve(), temporary)
    report = {"result": "pass", "kernel": os.uname().release, "uid": os.geteuid(),
              "binary_sha256": hashlib.sha256(args.binary.read_bytes()).hexdigest(),
              "packages": suites, "checks": checks, "temporary_state_removed": not temporary.exists(),
              "limit": "Native Linux non-race tests; Windows race evidence is separate; no power-cut test"}
    (args.output / "report.json").write_text(json.dumps(report, indent=2) + "\n", encoding="utf-8")
    print(json.dumps(report))


if __name__ == "__main__":
    main()
