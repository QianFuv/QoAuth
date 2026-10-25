"""Assemble source-bound binaries, SPDX inventories and original dependency notices."""
import argparse
import base64
import datetime
import hashlib
import json
import pathlib
import re
import shutil
import subprocess
import tarfile
import urllib.parse
import uuid
import zipfile


ROOT = pathlib.Path(__file__).resolve().parent.parent
NOTICE_NAME = re.compile(r"^(?:licen[cs]e|copying|copyright|notice|third.?party)", re.I)
GO_SUPPLEMENTS = {"github.com/go-jose/go-jose/v4": ["json/LICENSE"]}
LIBC_EXCERPTS = {
    "int128.go": b"Copyright (c) 2019 Luke Champine",
    "libc_windows.go": b"Copyright (c) 1990 The Regents of the University of California.",
    "ccgo_linux_amd64.go": b"Copyright (c) 2003 Poul-Henning Kamp",
}
RELEASE_GATES = """SOURCE AND VERIFICATION LIMITS
QoAuth has no selected source license. Maintainer-authorized publication grants no additional source license; NOASSERTION is not a license grant.
Release acceptance is recorded by the exact-source CI run; packaging alone does not establish native or container execution.
This binary inventory does not describe a built container or its certificate bundle/base-image provenance.
Targeted OIDC checks and two Go RP implementations are not OIDF certification.
Basic keyboard/labels/responsive checks are not a complete accessibility audit.
Build/development-only licenses are inventoried separately; optional uninstalled packages are not claimed inspected.
Whole-host or backup disclosure exposes keys.json; this package contains no installation data or credentials.
"""


def release_version(value, revision, is_dirty, is_release):
    """Resolve a version without allowing dirty bytes to claim a formal release.

    Args:
        value: The canonical VERSION text.
        revision: The complete source revision.
        is_dirty: Whether the working tree has changes.
        is_release: Whether formal release packaging was requested.

    Returns:
        A stable release version or explicitly identified development version.
    """
    version = value.strip()
    if not re.fullmatch(r"(?:0|[1-9][0-9]*)\.(?:0|[1-9][0-9]*)\.(?:0|[1-9][0-9]*)", version):
        raise ValueError("VERSION must be a stable numeric version")
    if is_release and is_dirty:
        raise ValueError("A formal release requires a clean working tree")
    return version if is_release else version + "-dev." + revision[:12] + (".dirty" if is_dirty else "")


def validate_css_resources(payload, asset, asset_root, font_hashes):
    """Permit only reviewed local WOFF2 resources referenced by embedded CSS.

    Args:
        payload: The stylesheet bytes.
        asset: The stylesheet path.
        asset_root: The embedded asset directory.
        font_hashes: SHA-256 values of the reviewed source font shards.
    """
    for match in re.finditer(rb"url\s*\(\s*([^)]*?)\s*\)", payload, re.I):
        relative = match.group(1).strip().strip(b"\"'").decode("utf-8")
        if not re.fullmatch(r"(?:\./|/assets/)?[A-Za-z0-9_.-]+\.woff2", relative):
            raise ValueError("Unreviewed CSS resource")
        resource = ((asset_root / relative.lstrip("/")) if relative.startswith("/") else (asset.parent / relative)).resolve()
        if not resource.is_relative_to(asset_root.resolve()) or not resource.is_file() or digest(resource.read_bytes()) not in font_hashes:
            raise ValueError("CSS font resource is not covered by reviewed source hashes")


def bundled_fonts():
    """Return the reviewed source-shard hashes, rejecting an unexpected count.

    Returns:
        SHA-256 values for all 668 source font shards.
    """
    fonts = sorted((ROOT / "web/src/fonts").rglob("*.woff2"))
    if len(fonts) != 668:
        raise ValueError("Expected 668 reviewed font shards")
    hashes = sorted(digest(font.read_bytes()) for font in fonts)
    if digest(("\n".join(hashes) + "\n").encode()) != "2ef2c005f8e5c2e0d0dbc6d90aa90ea12d522779a2ed94a4503fe5943a13ed92":
        raise ValueError("Reviewed font payloads changed")
    return set(hashes)


def execute(*arguments):
    """Run an existing local tool without shell interpolation or dependency installation."""
    return subprocess.check_output(arguments, cwd=ROOT, text=True, encoding="utf-8", timeout=120)


def json_stream(text):
    """Decode concatenated JSON objects emitted by go list."""
    decoder = json.JSONDecoder()
    while text.strip():
        value, end = decoder.raw_decode(text.lstrip())
        yield value
        text = text.lstrip()[end:]


def digest(data):
    """Return an ordinary SHA-256 digest, distinct from Go module h1 integrity values."""
    return hashlib.sha256(data).hexdigest()


def component_id(name, version):
    """Create deterministic SPDX-safe component identifiers without path ambiguity."""
    return "SPDXRef-" + digest((name + "@" + version).encode())[:24]


def write_json(path, value):
    """Write deterministic readable UTF-8 inventory content."""
    path.write_text(json.dumps(value, indent=2, ensure_ascii=False) + "\n", encoding="utf-8")


def notice_files(directory):
    """Collect root notices only, avoiding unrelated nested testdata licenses."""
    return sorted(path for path in directory.iterdir() if path.is_file() and NOTICE_NAME.match(path.name))


def preserve_notices(destination, name, version, directory, source_prefix, extras=(), excerpts=None):
    """Preserve original files and explicitly reviewed inline notice blocks with source hashes."""
    target = destination / "LICENSES" / component_id(name, version)
    target.mkdir(parents=True, exist_ok=True)
    records = []
    files = notice_files(directory)
    for relative in extras:
        path = directory / relative
        if not path.is_file():
            raise ValueError(f"Required notice missing: {name}/{relative}")
        files.append(path)
    for path in sorted(set(files)):
        relative = path.relative_to(directory).as_posix()
        output = target / relative
        output.parent.mkdir(parents=True, exist_ok=True)
        payload = path.read_bytes()
        output.write_bytes(payload)
        records.append({"source": source_prefix + "/" + relative, "path": output.relative_to(destination).as_posix(), "sha256": digest(payload)})
    for relative, anchor in (excerpts or {}).items():
        original = (directory / relative).read_bytes()
        matches = [match for match in re.finditer(rb"/\*.*?\*/", original, re.S) if anchor in match.group()]
        if len(matches) != 1:
            raise ValueError(f"Inline notice changed: {name}/{relative}")
        payload = matches[0].group()
        output = target / (relative + ".notice.txt")
        output.write_bytes(payload)
        records.append({"source": source_prefix + "/" + relative, "byte_range": [matches[0].start(), matches[0].end()], "source_sha256": digest(original), "path": output.relative_to(destination).as_posix(), "sha256": digest(payload)})
    return records


def package(name, version, location="NOASSERTION", declared="NOASSERTION", comment=""):
    """Describe a component without claiming a legal conclusion or file-level scan."""
    return {"name": name, "SPDXID": component_id(name, version), "versionInfo": version, "downloadLocation": location, "filesAnalyzed": False, "licenseConcluded": "NOASSERTION", "licenseDeclared": declared, "copyrightText": "NOASSERTION", "comment": comment}


def npm_info(name, version, lock):
    """Bind bundled metadata to the exact installed and locked package."""
    directory = ROOT / "web/node_modules" / name
    installed = json.loads((directory / "package.json").read_text(encoding="utf-8"))
    locked = lock["packages"].get("node_modules/" + name)
    if installed["version"] != version or not locked or locked["version"] != version:
        raise ValueError(f"Frontend version evidence mismatch: {name}")
    return directory, installed, locked


def add_npm(destination, entry, lock, comment):
    """Collect a bundled npm package's complete root notices and archive integrity."""
    name, version = entry["name"], entry["version"]
    directory, installed, locked = npm_info(name, version, lock)
    records = preserve_notices(destination, name, version, directory, "web/node_modules/" + name)
    if not records:
        raise ValueError(f"Bundled package lacks notice evidence: {name}")
    result = package(name, version, locked.get("resolved", "NOASSERTION"), installed.get("license", "NOASSERTION"), comment)
    result["externalRefs"] = [{"referenceCategory": "PACKAGE-MANAGER", "referenceType": "purl", "referenceLocator": "pkg:npm/" + urllib.parse.quote(name, safe="/") + "@" + version}]
    integrity = locked.get("integrity", "")
    if integrity.startswith(("sha512-", "sha256-")):
        algorithm, encoded = integrity.split("-", 1)
        result["checksums"] = [{"algorithm": algorithm.upper(), "checksumValue": base64.b64decode(encoded, validate=True).hex()}]
    return result, records


def build_inventory(destination, modules, lock):
    """Keep the wider resolved graph separate from linked runtime inventories."""
    destination.mkdir()
    npm, go = [], []
    for relative, entry in sorted(lock["packages"].items()):
        if not relative:
            continue
        directory = ROOT / "web" / relative
        installed = (directory / "package.json").is_file()
        record = {"path": relative, "version": entry.get("version"), "declared_license": entry.get("license", "NOASSERTION"), "dev": entry.get("dev", False), "optional": entry.get("optional", False), "installed": installed, "archive_integrity": entry.get("integrity"), "notices": []}
        if installed:
            metadata = json.loads((directory / "package.json").read_text(encoding="utf-8"))
            if metadata["version"] != record["version"]:
                raise ValueError(f"Installed build package differs from lock: {relative}")
            record["notices"] = preserve_notices(destination, relative, record["version"], directory, "web/" + relative)
        npm.append(record)
    for name, metadata in sorted(modules.items()):
        if metadata.get("Main"):
            continue
        record = {"name": name, "version": metadata.get("Version"), "module_sum": metadata.get("Sum"), "source_available": bool(metadata.get("Dir")), "notices": []}
        if record["source_available"]:
            record["notices"] = preserve_notices(destination, name, record["version"], pathlib.Path(metadata["Dir"]), "GOMODCACHE/" + name + "@" + record["version"], GO_SUPPLEMENTS.get(name, ()))
        go.append(record)
    write_json(destination / "inventory.json", {"scope": "Build, development, tests and resolved graph; not a runtime SBOM", "npm": npm, "go": go, "gaps": ["Optional uninstalled npm packages and unavailable module sources were not inspected.", "Installed @rolldown/binding-win32-x64-msvc declares MIT but ships no root license; parent Rolldown notices are preserved.", "No build/development executables are redistributed in this directory."]})


def main():
    """Require artifact provenance and publish completion metadata only after validation."""
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--output", type=pathlib.Path)
    parser.add_argument("--allow-dirty", action="store_true", help="Local packaging verification only; marks the version dirty")
    parser.add_argument("--release", action="store_true", help="Require clean source and package the canonical VERSION")
    args = parser.parse_args()
    revision = execute("git", "rev-parse", "HEAD").strip()
    is_dirty = bool(execute("git", "status", "--porcelain").strip())
    if is_dirty and not args.allow_dirty:
        raise ValueError("Commit task changes before assembling versioned final artifacts")
    is_preview = args.allow_dirty or is_dirty
    version = release_version((ROOT / "VERSION").read_text(encoding="utf-8"), revision, is_preview, args.release)
    destination = (args.output or ROOT / "dist" / version).resolve()
    if not destination.is_relative_to(ROOT) or destination.exists():
        raise ValueError("Output must be a new directory inside the workspace")
    lock = json.loads((ROOT / "web/package-lock.json").read_text(encoding="utf-8"))
    frontend_path = ROOT / "internal/webassets/dist/.vite/license.json"
    bundled = json.loads(frontend_path.read_text(encoding="utf-8"))
    if not bundled:
        raise ValueError("Build-produced frontend inventory is required")
    modules = {item["Path"]: item for item in json_stream(execute("go", "list", "-m", "-json", "all"))}
    go_root = pathlib.Path(execute("go", "env", "GOROOT").strip())
    go_version = execute("go", "env", "GOVERSION").strip()
    destination.mkdir(parents=True)
    created = datetime.datetime.now(datetime.timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")
    artifacts = []
    font_hashes = bundled_fonts()
    assets_directory = destination / "assets"
    assets_directory.mkdir()
    for platform in ("windows", "linux"):
        name = f"qoauth-{platform}-amd64" + (".exe" if platform == "windows" else "")
        binary = ROOT / "dist" / name
        build = json.loads(execute("go", "version", "-m", "-json", str(binary)))
        settings = {entry["Key"]: entry["Value"] for entry in build["Settings"]}
        if build["GoVersion"] != go_version or settings.get("vcs.revision") != revision or settings.get("GOOS") != platform or settings.get("GOARCH") != "amd64" or settings.get("CGO_ENABLED") != "0" or (settings.get("vcs.modified") != "false" and not args.allow_dirty):
            raise ValueError(f"Artifact provenance mismatch: {name}")
        data = binary.read_bytes()
        directory = destination / f"{platform}-amd64"
        directory.mkdir()
        executable = "qoauth.exe" if platform == "windows" else "qoauth"
        shutil.copyfile(binary, directory / executable)
        main_package = package("QoAuth", version, comment="Source license undecided; no additional source license granted. Embedded frontend and Go dependencies below are artifact-specific.")
        main_package["checksums"] = [{"algorithm": "SHA256", "checksumValue": digest(data)}]
        components, notices = [main_package], []
        for dependency in build["Deps"]:
            module = modules[dependency["Path"]]
            if module.get("Replace") or dependency.get("Replace") or module["Version"] != dependency["Version"] or module.get("Sum") != dependency.get("Sum"):
                raise ValueError("Changed module provenance requires review")
            module_path, module_version = module["Path"], module["Version"]
            records = preserve_notices(directory, module_path, module_version, pathlib.Path(module["Dir"]), "GOMODCACHE/" + module_path + "@" + module_version, GO_SUPPLEMENTS.get(module_path, ()), LIBC_EXCERPTS if module_path == "modernc.org/libc" else None)
            if not records:
                raise ValueError(f"Runtime module lacks notice evidence: {module_path}")
            notices.extend(records)
            components.append(package(module_path, module_version, comment="Linked Go module (partial package retention possible). Module integrity: " + dependency["Sum"] + ". Original license/NOTICE texts are in LICENSES; no legal conclusion is inferred."))
        toolchain = package("Go runtime and standard library", go_version, "https://go.dev/dl/", "BSD-3-Clause", "The executable contains compiled runtime/standard-library portions, not the complete toolchain.")
        components.append(toolchain)
        extras = ["PATENTS", *[path.relative_to(go_root).as_posix() for path in sorted((go_root / "src/vendor").rglob("LICENSE"))]]
        notices.extend(preserve_notices(directory, toolchain["name"], go_version, go_root, "GOROOT", extras))
        entries = {entry["name"]: entry for entry in bundled}
        for supplemental, reason in {"@douyinfe/semi-theme-default": "CSS theme provenance supplement", "vite": "Generated modulepreload runtime helper only"}.items():
            metadata = json.loads((ROOT / "web/node_modules" / supplemental / "package.json").read_text(encoding="utf-8"))
            entries.setdefault(supplemental, {"name": supplemental, "version": metadata["version"], "comment": reason})
        for entry in entries.values():
            component, records = add_npm(directory, entry, lock, entry.get("comment", "Build-emitted JS chunk inventory; tree-shaken portions only."))
            components.append(component)
            notices.extend(records)
        font_directory = ROOT / "web/public/assets/licenses"
        font_notices = [path.name for path in sorted(font_directory.iterdir()) if path.is_file()]
        notices.extend(preserve_notices(directory, "JetBrainsLxgwNerdMono", "1.3", font_directory, "web/public/assets/licenses", font_notices))
        components.append(package("JetBrainsLxgwNerdMono", "1.3", "https://github.com/lvbibir/JetBrainsLxgwNerdMono/releases/tag/v1.3", "OFL-1.1", "668 unchanged WOFF2 shards; version-specific source attribution is preserved in FONTS.md."))
        assets = []
        for asset in sorted((ROOT / "internal/webassets/dist").rglob("*")):
            if not asset.is_file() or any(part.startswith(".") for part in asset.relative_to(ROOT / "internal/webassets/dist").parts):
                continue
            payload = asset.read_bytes()
            if payload not in data:
                raise ValueError("Frontend evidence does not match executable bytes")
            if asset.suffix == ".css":
                validate_css_resources(payload, asset, ROOT / "internal/webassets/dist", font_hashes)
            if asset.suffix == ".woff2" and digest(payload) not in font_hashes:
                raise ValueError("Embedded font is not covered by reviewed source hashes")
            assets.append({"path": asset.relative_to(ROOT / "internal/webassets/dist").as_posix(), "sha256": digest(payload), "bytes": len(payload), "offset": data.index(payload)})
        relations = [{"spdxElementId": "SPDXRef-DOCUMENT", "relatedSpdxElement": main_package["SPDXID"], "relationshipType": "DESCRIBES"}]
        relations.extend({"spdxElementId": main_package["SPDXID"], "relatedSpdxElement": component["SPDXID"], "relationshipType": "DEPENDS_ON"} for component in components[1:])
        document = {"spdxVersion": "SPDX-2.3", "dataLicense": "CC0-1.0", "SPDXID": "SPDXRef-DOCUMENT", "name": name + "-" + version, "documentNamespace": "https://qoauth.invalid/spdx/" + str(uuid.uuid4()), "creationInfo": {"creators": ["Tool: QoAuth-local-packager"], "created": created}, "packages": components, "relationships": relations}
        write_json(directory / "sbom.spdx.json", document)
        write_json(directory / "build-info.json", build)
        write_json(directory / "frontend-inventory.json", bundled)
        write_json(directory / "notice-index.json", notices)
        (directory / "THIRD_PARTY_NOTICES.txt").write_text("Original notices are retained byte-for-byte under LICENSES; notice-index.json maps source paths and hashes.\nModule-level aggregates may describe unused portions. Inline libc supplements are conservative, not symbol-level claims.\nGo module h1 values are recorded as module integrity, not archive SHA-256.\nSPDX NOASSERTION does not grant a license. See RELEASE-GATES.txt.\n", encoding="utf-8")
        (directory / "RELEASE-GATES.txt").write_text(RELEASE_GATES, encoding="utf-8")
        manifest = {"version": version, "revision": revision, "build_settings": settings, "executable": executable, "sha256": digest(data), "bytes": len(data), "go_dependency_modules": len(build["Deps"]), "frontend_components_with_supplements": len(entries), "embedded_assets": assets, "files": {path.relative_to(directory).as_posix(): digest(path.read_bytes()) for path in sorted(directory.rglob("*")) if path.is_file()}}
        write_json(directory / "manifest.json", manifest)
        suffix = "windows_amd64.zip" if platform == "windows" else "linux_amd64.tar.gz"
        archive = assets_directory / f"qoauth_{version}_{suffix}"
        if platform == "windows":
            with zipfile.ZipFile(archive, "w", compression=zipfile.ZIP_DEFLATED) as zipped:
                for path in sorted(directory.rglob("*")):
                    if path.is_file():
                        member = zipfile.ZipInfo(path.relative_to(directory).as_posix())
                        member.create_system = 3
                        member.external_attr = 0o100644 << 16
                        member.compress_type = zipfile.ZIP_DEFLATED
                        zipped.writestr(member, path.read_bytes())
        else:
            with tarfile.open(archive, "w:gz") as zipped:
                for path in sorted(directory.rglob("*")):
                    if path.is_file():
                        member = zipped.gettarinfo(str(path), arcname=path.relative_to(directory).as_posix())
                        member.uid = member.gid = 0
                        member.uname = member.gname = ""
                        member.mode = 0o755 if path.name == executable else 0o644
                        with path.open("rb") as source:
                            zipped.addfile(member, source)
        artifacts.append({"path": "assets/" + archive.name, "sha256": digest(archive.read_bytes()), "bytes": archive.stat().st_size})
    (assets_directory / "SHA256SUMS").write_text("".join(item["sha256"] + "  " + pathlib.Path(item["path"]).name + "\n" for item in artifacts), encoding="utf-8")
    build_inventory(destination / "build-development", modules, lock)
    source_archive = destination / f"qoauth-{version}-source.zip"
    if not is_preview and not args.release:
        execute("git", "archive", "--format=zip", "--output=" + str(source_archive), revision)
        artifacts.append({"path": source_archive.name, "sha256": digest(source_archive.read_bytes()), "bytes": source_archive.stat().st_size})
    (destination / "RELEASE-GATES.txt").write_text(RELEASE_GATES, encoding="utf-8")
    write_json(destination / "manifest.json", {"version": version, "revision": revision, "created": created, "complete": True, "artifacts": artifacts})
    print(json.dumps({"output": str(destination), "version": version, "artifacts": artifacts}))


if __name__ == "__main__":
    main()
