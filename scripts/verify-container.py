"""Exercise one immutable image with disposable storage and loopback-only trusted TLS."""
import argparse
import base64
import hashlib
import hmac
import http.client
import http.cookiejar
import http.server
import io
import json
import os
import pathlib
import re
import shutil
import socket
import ssl
import struct
import subprocess
import tarfile
import tempfile
import threading
import time
import urllib.error
import urllib.parse
import urllib.request
import uuid


class NoRedirect(urllib.request.HTTPRedirectHandler):
    """Leave protocol redirects observable to the test driver."""

    def redirect_request(self, request, response, code, message, headers, target):
        """Prevent automatic navigation from hiding intermediate authorization results."""
        return None


class Browser:
    """Keep synthetic cookies and certificate trust inside this test client."""

    def __init__(self, certificate):
        """Trust only the temporary fixture certificate without changing host trust."""
        self.client = urllib.request.build_opener(
            urllib.request.HTTPSHandler(context=ssl.create_default_context(cafile=certificate)),
            urllib.request.HTTPCookieProcessor(http.cookiejar.CookieJar()), NoRedirect(),
        )

    def request(self, target, expected=200, data=None, headers=None):
        """Return a bounded response without logging request bodies, cookies or tokens."""
        request = urllib.request.Request(target, data=data, headers=headers or {})
        try:
            response = self.client.open(request, timeout=8)
        except urllib.error.HTTPError as error:
            response = error
        with response:
            payload = response.read(1024 * 1024)
            assert response.status == expected, f"Unexpected HTTP status {response.status}, wanted {expected}"
            return payload, response.headers

    def state(self, origin):
        """Read current CSRF/session state from the real service."""
        return json.loads(self.request(origin + "/api/session")[0])

    def post(self, origin, path, data, expected=200):
        """Supply current browser-bound CSRF and the canonical Origin."""
        headers = {"Content-Type": "application/json", "Origin": origin,
                   "X-CSRF-Token": self.state(origin)["csrf"]}
        payload, response_headers = self.request(origin + path, expected, json.dumps(data).encode(), headers)
        return json.loads(payload) if expected == 200 else response_headers


class ProxyHandler(http.server.BaseHTTPRequestHandler):
    """Forward to one task-owned backend while replacing untrusted address headers."""

    def log_message(self, format_string, *arguments):
        """Suppress URLs and bodies that may contain synthetic protocol credentials."""
        return

    def forward(self):
        """Preserve canonical Host/Origin and replace X-Real-IP with the socket peer."""
        length = int(self.headers.get("Content-Length", "0"))
        if length > 65536:
            self.send_error(413)
            return
        headers = {name: value for name, value in self.headers.items()
                   if name.lower() not in ("host", "connection", "x-real-ip", "forwarded")
                   and not name.lower().startswith("x-forwarded-")}
        headers["Host"] = self.server.canonical_host
        headers["X-Real-IP"] = self.client_address[0]
        connection = http.client.HTTPConnection("127.0.0.1", self.server.backend_port, timeout=8)
        try:
            connection.request(self.command, self.path, self.rfile.read(length), headers)
            response = connection.getresponse()
            payload = response.read(1024 * 1024)
            self.send_response(response.status)
            for name, value in response.getheaders():
                if name.lower() not in ("connection", "transfer-encoding", "content-length", "server", "date"):
                    self.send_header(name, value)
            self.send_header("Content-Length", str(len(payload)))
            self.end_headers()
            self.wfile.write(payload)
        except OSError:
            self.send_error(503, "Temporary test backend unavailable")
        finally:
            connection.close()

    do_GET = forward
    do_POST = forward


def proxy(certificate, key):
    """Start a temporary loopback HTTPS proxy using an isolated synthetic certificate."""
    server = http.server.ThreadingHTTPServer(("127.0.0.1", 0), ProxyHandler)
    server.canonical_host = f"127.0.0.1:{server.server_port}"
    server.backend_port = 0
    context = ssl.SSLContext(ssl.PROTOCOL_TLS_SERVER)
    context.load_cert_chain(certificate, key)
    server.socket = context.wrap_socket(server.socket, server_side=True)
    thread = threading.Thread(target=server.serve_forever, daemon=True)
    thread.start()
    return server, thread, "https://" + server.canonical_host


def main():
    """Create only uniquely labeled test resources, then stop and remove exactly those."""
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--image", required=True)
    parser.add_argument("--rp-binary", type=pathlib.Path, required=True)
    parser.add_argument("--output", type=pathlib.Path, required=True)
    args = parser.parse_args()
    assert re.fullmatch(r"sha256:[0-9a-f]{64}", args.image), "An immutable local image ID is required"
    assert os.sys.platform.startswith("linux") and os.geteuid() != 0
    args.output.mkdir(parents=True, exist_ok=False)
    os.umask(0o077)
    os.nice(10)
    identity = "qoauth-a6-" + uuid.uuid4().hex[:12]
    volume = identity + "-data"
    containers, transcripts, secrets, proxies = [], [], [], []
    rp_process = None
    has_volume = False

    def docker(*arguments, secret=None, succeeds=True, binary=False):
        """Run a bounded existing Docker command without shell interpolation."""
        result = subprocess.run(["docker", *map(str, arguments)], input=secret,
                                capture_output=True, text=not binary, timeout=120)
        if not binary:
            transcripts.append(result.stdout + result.stderr)
        if (result.returncode == 0) != succeeds:
            raise RuntimeError(f"Docker {arguments[0]} returned {result.returncode}")
        return result.stdout if binary else result.stdout + result.stderr

    def common(name):
        """Bound resources and retain image UID, immutable rootfs and dropped capabilities."""
        containers.append(name)
        return ["--name", name, "--label", "qoauth.validation=" + identity, "--read-only",
                "--cap-drop", "ALL", "--security-opt", "no-new-privileges:true",
                "--cpus", "1", "--memory", "512m", "--pids-limit", "128",
                "--mount", f"type=volume,source={volume},target=/data"]

    def command(*arguments, secret=None, succeeds=True):
        """Run offline maintenance without network access or a persistent container."""
        name = identity + "-cli-" + uuid.uuid4().hex[:6]
        return docker("run", "--rm", "-i", *common(name), "--network", "none",
                      args.image, *arguments, secret=secret, succeeds=succeeds)

    def files(path):
        """Inspect synthetic volume tar metadata in memory without extracting credentials."""
        name = identity + "-inspect-" + uuid.uuid4().hex[:6]
        docker("create", *common(name), "--network", "none", args.image)
        payload = docker("cp", name + ":" + path, "-", binary=True)
        assert len(payload) < 16 * 1024 * 1024
        with tarfile.open(fileobj=io.BytesIO(payload)) as archive:
            members = {member.name: {"uid": member.uid, "gid": member.gid, "mode": member.mode}
                       for member in archive.getmembers()}
            content = {member.name: archive.extractfile(member).read()
                       for member in archive.getmembers() if member.isfile() and member.name.endswith("keys.json")}
        docker("rm", name)
        return members, content

    def start(directory, service_proxy, origin, gateway):
        """Publish the backend only on host loopback and verify its observed proxy peer."""
        name = identity + "-serve-" + uuid.uuid4().hex[:6]
        docker("run", "-d", *common(name), "-p", "127.0.0.1::8080", args.image,
               "serve", "--data-dir", directory, "--listen", "0.0.0.0:8080",
               "--issuer", origin, "--trusted-proxy", gateway)
        inspection = json.loads(docker("inspect", name))[0]
        assert inspection["Config"]["User"] == "65532:65532"
        assert inspection["HostConfig"]["ReadonlyRootfs"]
        assert inspection["HostConfig"]["CapDrop"] == ["ALL"]
        assert "no-new-privileges:true" in inspection["HostConfig"]["SecurityOpt"]
        published = inspection["NetworkSettings"]["Ports"]["8080/tcp"]
        assert len(published) == 1 and published[0]["HostIp"] == "127.0.0.1"
        service_proxy.backend_port = int(published[0]["HostPort"])
        client = Browser(certificate)
        deadline = time.monotonic() + 30
        while time.monotonic() < deadline:
            try:
                payload, headers = client.request(origin + "/readyz")
                assert json.loads(payload) == {"ready": True} and headers.get("Set-Cookie") is None
                break
            except (OSError, AssertionError):
                time.sleep(0.2)
        else:
            raise RuntimeError("Container readiness deadline exceeded")
        connection = http.client.HTTPConnection("127.0.0.1", service_proxy.backend_port, timeout=3)
        connection.request("GET", "/readyz", headers={"Host": service_proxy.canonical_host})
        response = connection.getresponse()
        assert response.status == 400, "Configured proxy IP did not match the actual peer"
        response.read(); connection.close()
        connection = http.client.HTTPConnection("127.0.0.1", service_proxy.backend_port, timeout=3)
        connection.request("GET", "/readyz", headers={"Host": "wrong.invalid", "X-Real-IP": "127.0.0.1"})
        response = connection.getresponse()
        assert response.status == 400
        response.read(); connection.close()
        return name

    def stop(name):
        """Verify real Docker SIGTERM shutdown before releasing the container resource."""
        docker("stop", "--time", "10", name)
        state = json.loads(docker("inspect", name))[0]["State"]
        assert not state["Running"] and state["ExitCode"] == 0 and not state["OOMKilled"]
        docker("logs", name)
        docker("rm", name)

    try:
        with tempfile.TemporaryDirectory(prefix="qoauth-container-validation-") as directory:
            temporary = pathlib.Path(directory)
            certificate, key = temporary / "certificate.pem", temporary / "key.pem"
            generated = subprocess.run(["openssl", "req", "-x509", "-newkey", "rsa:2048", "-nodes",
                                        "-keyout", str(key), "-out", str(certificate), "-days", "1",
                                        "-subj", "/CN=127.0.0.1", "-addext", "subjectAltName=IP:127.0.0.1"],
                                       capture_output=True, timeout=20)
            assert generated.returncode == 0
            service_proxy, service_thread, origin = proxy(certificate, key)
            rp_proxy, rp_thread, rp_origin = proxy(certificate, key)
            proxies.extend(((service_proxy, service_thread), (rp_proxy, rp_thread)))
            docker("volume", "create", "--label", "qoauth.validation=" + identity, volume)
            has_volume = True
            gateway = json.loads(docker("network", "inspect", "bridge"))[0]["IPAM"]["Config"][0]["Gateway"]
            password = "isolated container fixture credential canary"
            secrets.append(password)
            command("init", "--data-dir", "/data/state", "--username", "owner", secret=password)
            command("init", "--data-dir", "/data/state", "--username", "other", secret=password, succeeds=False)
            metadata, contents = files("/data/state")
            assert metadata["state"]["uid"] == 65532 and metadata["state"]["mode"] == 0o700
            assert metadata["state/keys.json"]["uid"] == 65532 and metadata["state/keys.json"]["mode"] == 0o600
            original_keys = json.loads(contents["state/keys.json"])
            secrets.extend((original_keys["Encryption"], original_keys["Signing"]))
            active = start("/data/state", service_proxy, origin, gateway)
            owner = Browser(certificate)
            payload, headers = owner.request(origin + "/api/session")
            cookie = headers.get("Set-Cookie", "")
            assert cookie.startswith("__Host-qoauth=") and all(flag in cookie for flag in ("Secure", "HttpOnly", "Path=/", "SameSite=Lax"))
            owner.post(origin, "/api/login", {"username": "owner", "password": password})
            owner.request(origin + "/api/users", 403)
            factor = owner.post(origin, "/api/mfa/start", {})["secret"]
            secrets.append(factor)
            counter = int(time.time()) // 30 - 1
            digest = hmac.new(base64.b32decode(factor + "=" * (-len(factor) % 8)), struct.pack(">Q", counter), hashlib.sha1).digest()
            offset = digest[-1] & 15
            code = f"{(struct.unpack('>I', digest[offset:offset + 4])[0] & 0x7fffffff) % 1000000:06d}"
            recovery = owner.post(origin, "/api/mfa/confirm", {"code": code})["recovery_codes"]
            secrets.extend(recovery)
            member = owner.post(origin, "/api/users/create", {"username": "member", "password": password})["id"]
            application = {"id": "container-rp", "name": "Container fixture RP", "redirects": [rp_origin + "/callback"],
                           "logout_redirects": [rp_origin + "/signed-out"], "scopes": ["openid", "profile", "offline_access"], "audiences": []}
            client_secret = owner.post(origin, "/api/applications/create", {"application": application})["client_secret"]
            secrets.append(client_secret)
            owner.post(origin, "/api/assignments", {"user": member, "client": application["id"], "allow": True})
            secret_file = temporary / "client-secret"
            secret_file.write_text(client_secret)
            rp_binary = temporary / "go-rp"
            shutil.copyfile(args.rp_binary, rp_binary); rp_binary.chmod(0o700)
            with socket.socket() as listener:
                listener.bind(("127.0.0.1", 0)); rp_port = listener.getsockname()[1]
            rp_proxy.backend_port = rp_port
            rp_process = subprocess.Popen([str(rp_binary), "--issuer", origin, "--client-id", application["id"],
                                            "--client-secret-file", str(secret_file), "--callback", rp_origin + "/callback",
                                            "--listen", f"127.0.0.1:{rp_port}"], stdout=subprocess.PIPE, stderr=subprocess.PIPE,
                                           text=True, env=dict(os.environ, SSL_CERT_FILE=str(certificate), GOMAXPROCS="2"))
            browser = Browser(certificate)
            deadline = time.monotonic() + 10
            while time.monotonic() < deadline:
                if rp_process.poll() is not None:
                    raise RuntimeError("Standard RP exited before startup")
                try:
                    browser.request(rp_origin + "/"); break
                except (OSError, AssertionError):
                    time.sleep(0.1)
            else:
                raise RuntimeError("Standard RP startup deadline exceeded")
            browser.post(origin, "/api/login", {"username": "member", "password": password})
            target = browser.request(rp_origin + "/login", 302)[1]["Location"]
            interaction = urllib.parse.urljoin(origin, browser.request(target, 302)[1]["Location"])
            request_id = urllib.parse.parse_qs(urllib.parse.urlparse(interaction).query)["request"][0]
            assert b"/assets/" in browser.request(interaction)[0]
            callback = urllib.parse.urljoin(origin, browser.post(origin, "/api/consent", {"request": request_id, "approve": True})["redirect"])
            rp_callback = browser.request(callback, 302)[1]["Location"]
            browser.request(rp_callback, 303)
            assert member.encode() in browser.request(rp_origin + "/profile")[0]
            discovery = json.loads(browser.request(origin + "/.well-known/openid-configuration")[0])
            first_keys = json.loads(browser.request(discovery["jwks_uri"])[0])
            assert "busy" in command("backup", "--data-dir", "/data/state", "--snapshot", "/data/forbidden", succeeds=False)
            stop(active)
            active = start("/data/state", service_proxy, origin, gateway)
            assert first_keys == json.loads(browser.request(discovery["jwks_uri"])[0])
            assert owner.state(origin)["assurance"]
            assert member.encode() in browser.request(rp_origin + "/profile")[0]
            page = browser.request(rp_origin + "/")[0].decode()
            csrf = re.search(r'name="csrf" value="([^"]+)"', page).group(1)
            target = browser.request(rp_origin + "/logout", 303, urllib.parse.urlencode({"csrf": csrf}).encode(),
                                     {"Content-Type": "application/x-www-form-urlencoded", "Origin": rp_origin})[1]["Location"]
            browser.request(target)
            csrf = browser.state(origin)["csrf"]
            browser.request(target, 302, urllib.parse.urlencode({"csrf": csrf}).encode(),
                            {"Content-Type": "application/x-www-form-urlencoded", "Origin": origin})
            browser.request(rp_origin + "/profile", 401)
            assert not browser.state(origin)["authenticated"]
            stop(active)
            command("backup", "--data-dir", "/data/state", "--snapshot", "/data/snapshot")
            command("rotate-signing", "--data-dir", "/data/state", "--confirm")
            rotated_keys = json.loads(next(iter(files("/data/state/keys.json")[1].values())))
            assert rotated_keys["Encryption"] == original_keys["Encryption"] and rotated_keys["Signing"] != original_keys["Signing"]
            active = start("/data/state", service_proxy, origin, gateway)
            assert len(json.loads(owner.request(discovery["jwks_uri"])[0])["keys"]) == 2
            stop(active)
            command("restore", "--snapshot", "/data/snapshot", "--data-dir", "/data/restored", "--confirm")
            assert "recover-owner" in command("serve", "--data-dir", "/data/restored", "--issuer", origin, succeeds=False)
            command("recover-owner", "--data-dir", "/data/restored", "--confirm", "--restore-reviewed", secret=password + " renewed")
            assert "quarantine" in command("serve", "--data-dir", "/data/restored", "--issuer", origin, succeeds=False)
            counts = json.loads(command("cleanup", "--data-dir", "/data/state", "--batch-size", "64"))
            assert len(counts) == 7
            assert all(secret not in entry for secret in secrets for entry in transcripts)
            report = {"result": "pass", "image": args.image, "runtime_user": "65532:65532",
                      "proxy_peer_confirmed": gateway, "host_publish": "127.0.0.1", "read_only_rootfs": True,
                      "checks": ["fresh volume ownership", "single initialization", "trusted local TLS and secure cookie",
                                 "owner MFA", "standard coreos RP code/PKCE/JWT/UserInfo", "restart/session/JWKS persistence",
                                 "confirmed logout", "SIGTERM", "maintenance lock", "backup", "signing/AES continuity",
                                 "restricted restore/recovery/quarantine", "bounded cleanup", "no credential logging"],
                      "limits": ["no physical power-loss or large-database temporary-file test",
                                 "restore quarantine startup rejection tested; real ten-minute expiry covered separately"]}
    finally:
        if rp_process is not None:
            rp_process.terminate()
            rp_process.communicate(timeout=10)
        for server, thread in proxies:
            server.shutdown(); server.server_close(); thread.join(timeout=3)
        for name in containers:
            result = subprocess.run(["docker", "inspect", name], capture_output=True, text=True, timeout=15)
            if result.returncode == 0:
                data = json.loads(result.stdout)[0]
                assert data["Config"]["Labels"].get("qoauth.validation") == identity
                if data["State"]["Running"]:
                    docker("stop", "--time", "10", name)
                docker("rm", name)
        if has_volume:
            data = json.loads(docker("volume", "inspect", volume))[0]
            assert data["Labels"].get("qoauth.validation") == identity
            docker("volume", "rm", volume)
    report["temporary_resources_removed"] = True
    (args.output / "report.json").write_text(json.dumps(report, indent=2) + "\n", encoding="utf-8")
    print(json.dumps(report))


if __name__ == "__main__":
    main()
