"""Test release boundaries and inspect formal archive contents without extraction."""

import hashlib
import importlib.util
import json
import pathlib
import re
import subprocess
import sys
import tarfile
import tempfile
import unittest
from unittest.mock import patch
import zipfile


sys.dont_write_bytecode = True
SPEC = importlib.util.spec_from_file_location("packager", pathlib.Path(__file__).with_name("package-local.py"))
PACKAGER = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(PACKAGER)
TOP_LEVEL = {"sbom.spdx.json", "build-info.json", "frontend-inventory.json", "notice-index.json", "THIRD_PARTY_NOTICES.txt", "RELEASE-GATES.txt", "manifest.json"}
FONT_NOTICES = {"FONTS.md", "JetBrainsMono-OFL.txt", "LXGW-WenKai-Screen-OFL.txt", "LXGW-ZhenKai-OFL.txt", "NerdFonts-LICENSE.txt", "NerdFonts-JetBrainsMono-OFL.txt"}


def binary_build_info(payload):
    """Read actual Go metadata from a temporary single file without executing it.

    Args:
        payload: The packaged executable bytes.

    Returns:
        The Go tool's decoded build information.
    """
    with tempfile.TemporaryDirectory(prefix="qoauth-metadata-") as directory:
        filename = pathlib.Path(directory) / "executable"
        filename.write_bytes(payload)
        return json.loads(subprocess.check_output(["go", "version", "-m", "-json", str(filename)], text=True, encoding="utf-8", timeout=30))


def verify_members(members, source, version, platform):
    """Require exact-source formal contents and a complete self-consistent inventory.

    Args:
        members: Archive member names mapped to uncompressed bytes.
        source: Expected full source revision.
        version: Expected stable version.
        platform: Expected GOOS value.
    """
    executable = "qoauth.exe" if platform == "windows" else "qoauth"
    for name in members:
        relative = pathlib.PurePosixPath(name)
        if relative.is_absolute() or ".." in relative.parts or "\\" in name:
            raise ValueError("Unsafe archive member")
        if name not in TOP_LEVEL | {executable} and not re.fullmatch(r"LICENSES/SPDXRef-[0-9a-f]{24}/[A-Za-z0-9_./-]+", name):
            raise ValueError("Unexpected archive member: " + name)
        if re.search(r"(?:^|/)(?:keys\.json|secrets?\.json|config\.json|.*\.(?:db|key|pfx|p12))$", name, re.I):
            raise ValueError("Private filename in archive")
        if name.startswith("LICENSES/") and relative.name not in FONT_NOTICES and not re.match(r"(?:licen[cs]e|copying|copyright|notice|third.?party|patents)", relative.name, re.I) and not relative.name.endswith(".notice.txt"):
            raise ValueError("Non-notice file in LICENSES")
    if {name for name in members if not name.startswith("LICENSES/")} != TOP_LEVEL | {executable}:
        raise ValueError("Missing or extra formal archive root file")
    manifest = json.loads(members["manifest.json"])
    if manifest["revision"] != source or manifest["version"] != version:
        raise ValueError("Wrong archive source or version")
    settings = manifest["build_settings"]
    if settings.get("vcs.revision") != source or settings.get("vcs.modified") != "false" or settings.get("GOOS") != platform or settings.get("GOARCH") != "amd64" or settings.get("CGO_ENABLED") != "0":
        raise ValueError("Wrong executable build identity")
    actual = {name: hashlib.sha256(payload).hexdigest() for name, payload in members.items() if name != "manifest.json"}
    if actual != manifest["files"]:
        raise ValueError("Archive inventory mismatch")
    if manifest["executable"] != executable or hashlib.sha256(members[executable]).hexdigest() != manifest["sha256"]:
        raise ValueError("Wrong executable digest")
    build = json.loads(members["build-info.json"])
    if {item["Key"]: item["Value"] for item in build["Settings"]} != settings or binary_build_info(members[executable]) != build:
        raise ValueError("Actual executable metadata does not match declared provenance")
    if len(members[executable]) != manifest["bytes"]:
        raise ValueError("Executable length mismatch")
    notices = json.loads(members["notice-index.json"])
    notice_files = {item["path"]: item["sha256"] for item in notices}
    if notice_files != {name: actual[name] for name in members if name.startswith("LICENSES/")}:
        raise ValueError("Notice index membership or digest mismatch")
    if len(manifest["embedded_assets"]) != len({asset["path"] for asset in manifest["embedded_assets"]}):
        raise ValueError("Duplicate embedded asset")
    for asset in manifest["embedded_assets"]:
        payload = members[executable][asset["offset"]:asset["offset"] + asset["bytes"]]
        if hashlib.sha256(payload).hexdigest() != asset["sha256"]:
            raise ValueError("Embedded asset does not match executable bytes")
    if len([asset for asset in manifest["embedded_assets"] if asset["path"].endswith(".woff2")]) != 668:
        raise ValueError("Font shard inventory missing")
    for marker in [b"isolated synthetic password canary", b"isolated container fixture credential canary"]:
        if marker in members[executable]:
            raise ValueError("Test credential or fixture linked into release executable")
    packages = json.loads(members["sbom.spdx.json"])["packages"]
    if packages[0]["licenseDeclared"] != "NOASSERTION":
        raise ValueError("Unexpected QoAuth source license grant")
    if not any(item["name"] == "JetBrainsLxgwNerdMono" and item["licenseDeclared"] == "OFL-1.1" for item in packages):
        raise ValueError("Font SPDX component missing")
    for notice in ["FONTS.md", "LXGW-WenKai-Screen-OFL.txt", "LXGW-ZhenKai-OFL.txt", "NerdFonts-JetBrainsMono-OFL.txt"]:
        if not any(name.endswith("/" + notice) for name in members):
            raise ValueError("Missing font notice: " + notice)


def verify_archives(directory, source, version):
    """Inspect both accepted platform archives without writing their contents.

    Args:
        directory: Directory containing the two archives and SHA256SUMS.
        source: Expected full source revision.
        version: Expected stable release version.
    """
    for platform in ["windows", "linux"]:
        suffix = "windows_amd64.zip" if platform == "windows" else "linux_amd64.tar.gz"
        filename = pathlib.Path(directory) / f"qoauth_{version}_{suffix}"
        if platform == "windows":
            with zipfile.ZipFile(filename) as archive:
                names = archive.namelist()
                if len(names) != len(set(names)):
                    raise ValueError("Duplicate ZIP member")
                if sum(item.file_size for item in archive.infolist()) > 256 * 1024 * 1024:
                    raise ValueError("Unexpected archive size")
                members = {name: archive.read(name) for name in names}
        else:
            with tarfile.open(filename, "r:gz") as archive:
                entries = archive.getmembers()
                if any(not item.isfile() for item in entries) or len(entries) != len({item.name for item in entries}):
                    raise ValueError("Non-file or duplicate TAR member")
                if sum(item.size for item in entries) > 256 * 1024 * 1024:
                    raise ValueError("Unexpected archive size")
                if next(item for item in entries if item.name == "qoauth").mode != 0o755:
                    raise ValueError("Linux executable permission missing")
                members = {item.name: archive.extractfile(item).read() for item in entries}
        verify_members(members, source, version, platform)


class ReleaseBoundaryTests(unittest.TestCase):
    """Express source, resource and privacy conditions required by formal releases."""

    def test_formal_version_requires_clean_source(self):
        """A renamed dirty preview must never become a stable release."""
        self.assertEqual(PACKAGER.release_version("0.1.0\n", "a" * 40, False, True), "0.1.0")
        with self.assertRaises(ValueError):
            PACKAGER.release_version("0.1.0", "a" * 40, True, True)
        self.assertEqual(PACKAGER.release_version("0.1.0", "a" * 40, True, False), "0.1.0-dev.aaaaaaaaaaaa.dirty")

    def test_nonstable_versions_are_rejected(self):
        """Archive and tag versions share one unambiguous numeric representation."""
        for value in ["v0.1.0", "01.0.0", "0.1.0-dev", "0.1", "0.1.0\nextra"]:
            with self.assertRaises(ValueError):
                PACKAGER.release_version(value, "a" * 40, False, True)

    def test_reviewed_font_urls_are_accepted_but_other_resources_fail(self):
        """Only a local WOFF2 with reviewed bytes may pass the CSS asset boundary."""
        with tempfile.TemporaryDirectory() as directory:
            root = pathlib.Path(directory)
            font = root / "font.woff2"
            font.write_bytes(b"reviewed font fixture")
            css = root / "fonts.css"
            hashes = {hashlib.sha256(font.read_bytes()).hexdigest()}
            PACKAGER.validate_css_resources(b'@font-face{src:url("./font.woff2")}', css, root, hashes)
            for resource in ["https://example.test/font.woff2", "data:font/woff2;base64,AA", "../font.woff2", "font.svg", "missing.woff2"]:
                with self.assertRaises(ValueError):
                    PACKAGER.validate_css_resources(("a{src:url(" + resource + ")}").encode(), css, root, hashes)
            assets = root / "assets"
            assets.mkdir()
            (assets / "font.woff2").write_bytes(font.read_bytes())
            PACKAGER.validate_css_resources(b"a{src:url(/assets/font.woff2)}", css, root, hashes)
            font.write_bytes(b"changed payload")
            with self.assertRaises(ValueError):
                PACKAGER.validate_css_resources(b"a{src:url(font.woff2)}", css, root, hashes)

    def test_private_members_are_rejected_before_manifest_processing(self):
        """Even self-consistent manifests must not admit user state or traversal."""
        for name in ["keys.json", "qoauth.db", ".cache/state.json", "config.json", "../qoauth", "/qoauth", "LICENSES/../../keys.json"]:
            with self.assertRaises(ValueError):
                verify_members({name: b"private fixture"}, "a" * 40, "0.1.0", "linux")

    def test_complete_archive_cannot_hide_extra_files_in_a_self_consistent_manifest(self):
        """A valid archive becomes invalid when private or wrong-platform bytes are added."""
        source = "a" * 40
        settings = {"vcs.revision": source, "vcs.modified": "false", "GOOS": "linux", "GOARCH": "amd64", "CGO_ENABLED": "0"}
        build = {"Settings": [{"Key": key, "Value": value} for key, value in settings.items()]}
        members = {name: b"fixture" for name in TOP_LEVEL - {"manifest.json"}}
        members["qoauth"] = b"binary fixture"
        members["build-info.json"] = json.dumps(build).encode()
        members["sbom.spdx.json"] = json.dumps({"packages": [{"name": "QoAuth", "licenseDeclared": "NOASSERTION"}, {"name": "JetBrainsLxgwNerdMono", "licenseDeclared": "OFL-1.1"}]}).encode()
        notices = []
        for notice in ["FONTS.md", "LXGW-WenKai-Screen-OFL.txt", "LXGW-ZhenKai-OFL.txt", "NerdFonts-JetBrainsMono-OFL.txt"]:
            name = "LICENSES/SPDXRef-" + "a" * 24 + "/" + notice
            members[name] = b"license fixture"
            notices.append({"path": name, "sha256": hashlib.sha256(members[name]).hexdigest()})
        members["notice-index.json"] = json.dumps(notices).encode()
        manifest = {"revision": source, "version": "0.1.0", "build_settings": settings, "executable": "qoauth", "sha256": hashlib.sha256(members["qoauth"]).hexdigest(), "bytes": len(members["qoauth"]), "embedded_assets": [{"path": str(index) + ".woff2", "offset": 0, "bytes": 1, "sha256": hashlib.sha256(b"b").hexdigest()} for index in range(668)]}
        manifest["files"] = {name: hashlib.sha256(payload).hexdigest() for name, payload in members.items()}
        members["manifest.json"] = json.dumps(manifest).encode()
        with patch(__name__ + ".binary_build_info", return_value=build):
            verify_members(members, source, "0.1.0", "linux")
            for name in ["qoauth.exe", "LICENSES/SPDXRef-" + "a" * 24 + "/settings.conf", "LICENSES/SPDXRef-" + "a" * 24 + "/keys.json"]:
                modified = dict(members, **{name: b"private fixture"})
                changed = dict(manifest, files={**manifest["files"], name: hashlib.sha256(b"private fixture").hexdigest()})
                modified["manifest.json"] = json.dumps(changed).encode()
                with self.assertRaises(ValueError):
                    verify_members(modified, source, "0.1.0", "linux")
            for name in TOP_LEVEL:
                with self.assertRaises(ValueError):
                    verify_members({key: value for key, value in members.items() if key != name}, source, "0.1.0", "linux")


if __name__ == "__main__":
    if len(sys.argv) > 1 and sys.argv[1] == "--verify":
        verify_archives(*sys.argv[2:])
        print("Both release archives have valid provenance, membership, notices and inventories")
    else:
        unittest.main()
