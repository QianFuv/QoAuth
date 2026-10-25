/** Source-bound release contracts, tested assets, notes and explicit publication. */
import assert from "node:assert/strict";
import { createHash } from "node:crypto";
import { execFileSync } from "node:child_process";
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";
import { randomBytes } from "node:crypto";
import os from "node:os";
import net from "node:net";
import { spawn } from "node:child_process";

const ROOT = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const GROUPS = new Map([["feat", "Features"], ["fix", "Bug fixes"], ["perf", "Performance"], ["refactor", "Refactoring"], ["test", "Tests"], ["build", "Build"], ["ci", "CI"], ["docs", "Documentation"], ["style", "Style"], ["chore", "Maintenance"], ["revert", "Reverts"], ["other", "Other commits"]]);

/** Run a bounded tool without shell interpolation or credential output. */
function run(command, argumentsList, options = {}) {
  return execFileSync(command, argumentsList, { cwd: ROOT, encoding: "utf8", timeout: 180000, maxBuffer: 32 * 1024 * 1024, ...options }).trim();
}

/** Validate the stable numeric version shared by tags, manifests and archives. */
export function parseVersion(value) {
  assert.match(value, /^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)$/);
  return value;
}

/** Compare stable versions without lexical ordering or integer overflow. */
export function compareVersions(first, second) {
  const parts = parseVersion(first).split(".").map(BigInt);
  const previous = parseVersion(second).split(".").map(BigInt);
  const index = parts.findIndex((part, position) => part !== previous[position]);
  return index < 0 ? 0 : parts[index] > previous[index] ? 1 : -1;
}

/** Resolve a first push, ordinary check or explicit same-version publication. */
export function resolveContext({ event, ref, source, version, requestedVersion = "", sourceRun = "", previousVersion }) {
  parseVersion(version);
  assert.match(source, /^[a-f0-9]{40}$/);
  assert(["push", "pull_request", "workflow_dispatch"].includes(event));
  if (previousVersion) assert(compareVersions(version, previousVersion) >= 0, "VERSION cannot decrease");
  if (requestedVersion) assert.equal(requestedVersion, version, "Requested version must match source");
  if (sourceRun) {
    assert.equal(event, "workflow_dispatch");
    assert.equal(ref, "refs/heads/main");
    assert.match(sourceRun, /^[1-9]\d*$/);
  }
  return { source, version, mode: sourceRun ? "publish" : event === "pull_request" ? "quality" : "build", source_run: sourceRun };
}

/** Return the only platform archives allowed on a stable Release. */
export function assetNames(version) {
  parseVersion(version);
  return [`qoauth_${version}_linux_amd64.tar.gz`, `qoauth_${version}_windows_amd64.zip`];
}

/** Hash bytes with ordinary SHA-256. */
export function sha256(bytes) {
  return createHash("sha256").update(bytes).digest("hex");
}

/** Inventory every file, including hidden license metadata, rejecting symlinks. */
export function inventory(directory, prefix = "") {
  const files = {};
  for (const entry of fs.readdirSync(directory, { withFileTypes: true }).sort((first, second) => first.name.localeCompare(second.name))) {
    assert(!entry.isSymbolicLink(), "Artifact symlinks are forbidden");
    const relative = prefix + entry.name;
    const absolute = path.join(directory, entry.name);
    if (entry.isDirectory()) Object.assign(files, inventory(absolute, relative + "/"));
    else {
      assert(entry.isFile());
      files[relative] = sha256(fs.readFileSync(absolute));
    }
  }
  return files;
}

/** Reject missing, extra or altered bytes against an exact artifact inventory. */
export function verifyInventory(directory, expected) {
  assert(Object.keys(expected).length > 0, "Empty artifacts are forbidden");
  assert.deepEqual(inventory(directory), expected, "Artifact membership or hashes changed");
}

/** Escape subjects so Markdown or HTML cannot hide commit entries. */
function escapeSubject(subject) {
  return subject.replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;").replace(/([\\`*_{}\[\]()#!|~])/g, "\\$1");
}

/** Render all source-ancestry commits using LitRadar's categorized release format. */
export function renderNotes({ version, repository, commits, previous }) {
  parseVersion(version);
  assert.match(repository, /^[\w.-]+\/[\w.-]+$/);
  const groups = new Map([...GROUPS.keys()].map(type => [type, []]));
  for (const { hash, subject } of commits) {
    assert.match(hash, /^[a-f0-9]{40}$/);
    const type = /^([a-z]+)(?:\([^\r\n)]+\))?!?:\s+/.exec(subject)?.[1];
    groups.get(GROUPS.has(type) ? type : "other").push(`- ${escapeSubject(subject)} ([${hash.slice(0, 7)}](https://github.com/${repository}/commit/${hash}))`);
  }
  const sections = [`## Commits (${commits.length})`, previous ? `Changes since ${previous}.` : "All commits in this first stable release."];
  for (const [type, entries] of groups) if (entries.length) sections.push(`### ${GROUPS.get(type)} (${entries.length})\n\n${entries.join("\n")}`);
  const comparison = previous ? `compare/${previous}...v${version}` : `commits/v${version}`;
  sections.push(`**Full changelog:** [${previous ? `${previous}...v${version}` : `v${version}`}](https://github.com/${repository}/${comparison})`);
  const notes = sections.join("\n\n") + "\n";
  assert(Buffer.byteLength(notes) <= 125000, "Complete notes exceed GitHub's body limit");
  return notes;
}

/** Read the canonical local version and exact Git revision. */
function sourceContext() {
  const version = parseVersion(fs.readFileSync(path.join(ROOT, "VERSION"), "utf8").trim());
  assert.equal(JSON.parse(fs.readFileSync(path.join(ROOT, "web/package.json"))).version, version);
  return { version, source: run("git", ["rev-parse", "HEAD"]) };
}

/** Write structured JSON without environment or credential dumps. */
function writeJson(filename, value) {
  fs.mkdirSync(path.dirname(filename), { recursive: true });
  fs.writeFileSync(filename, JSON.stringify(value, null, 2) + "\n");
}

/** Expose validated job outputs without executing untrusted strings. */
function outputs(value) {
  if (process.env.GITHUB_OUTPUT) fs.appendFileSync(process.env.GITHUB_OUTPUT, Object.entries(value).map(([key, item]) => `${key}=${item}\n`).join(""));
  console.log(JSON.stringify(value));
}

/** Validate exact archive names, checksums and their internal provenance. */
export function verifyAssets(directory, source, version) {
  const expected = [...assetNames(version), "SHA256SUMS"].sort();
  assert.deepEqual(fs.readdirSync(directory).sort(), expected, "Release asset membership differs");
  const checksums = parseChecksums(fs.readFileSync(path.join(directory, "SHA256SUMS"), "utf8"), version);
  for (const [name, checksum] of Object.entries(checksums)) assert.equal(sha256(fs.readFileSync(path.join(directory, name))), checksum, "Archive digest mismatch");
  run(process.env.PYTHON || (process.platform === "win32" ? "python" : "python3"), ["scripts/package-release.test.py", "--verify", path.resolve(directory), source, version]);
  return inventory(directory);
}

/** Parse both platform digests, accepting Windows line endings but no extra members. */
export function parseChecksums(text, version) {
  const lines = text.trim().split(/\r?\n/);
  assert.equal(lines.length, 2);
  const seen = new Set();
  for (const line of lines) {
    const match = /^([a-f0-9]{64})  ([A-Za-z0-9_.-]+)$/.exec(line);
    assert(match && assetNames(version).includes(match[2]) && !seen.has(match[2]), "Invalid checksum membership");
    seen.add(match[2]);
  }
  return Object.fromEntries(lines.map(line => { const [checksum, name] = line.split("  "); return [name, checksum]; }));
}

/** Read authenticated GitHub JSON, permitting only an explicitly handled 404. */
function github(endpoint, argumentsList = [], allowMissing = false) {
  try { return JSON.parse(run("gh", ["api", endpoint, ...argumentsList], { stdio: ["ignore", "pipe", "pipe"] })); }
  catch (error) {
    if (allowMissing && String(error.stderr).includes("HTTP 404")) return null;
    throw new Error(`GitHub request failed for ${endpoint}: exit ${error.status}`);
  }
}

/** Require successful same-SHA source CI and both native acceptance jobs. */
export function validateRun(runInfo, jobs, source, repository) {
  assert.equal(runInfo.repository.full_name, repository);
  assert.equal(runInfo.head_sha, source);
  assert.equal(runInfo.head_branch, "main");
  assert.equal(runInfo.path, ".github/workflows/ci.yaml");
  assert.equal(runInfo.status, "completed");
  assert.equal(runInfo.conclusion, "success");
  assert(["push", "workflow_dispatch"].includes(runInfo.event));
  for (const name of ["Linux archive and image", "Windows archive"]) assert(jobs.some(job => job.name.endsWith(name) && job.conclusion === "success"), `Missing successful ${name}`);
}

/** Smoke the native packaged executable using only fresh private synthetic state. */
async function smoke(directory, reportPath) {
  const { source, version } = sourceContext();
  const manifest = JSON.parse(fs.readFileSync(path.join(directory, "manifest.json")));
  assert.equal(manifest.revision, source);
  assert.equal(manifest.version, version);
  const binary = path.resolve(directory, manifest.executable);
  assert.equal(sha256(fs.readFileSync(binary)), manifest.sha256);
  const temporary = fs.mkdtempSync(path.join(os.tmpdir(), "qoauth-release-"));
  const state = path.join(temporary, "state");
  const password = randomBytes(32).toString("base64url");
  let service;
  try {
    run(binary, ["init", "--data-dir", state, "--username", "owner"], { input: password + "\n" });
    const listener = net.createServer();
    await new Promise(resolve => listener.listen(0, "127.0.0.1", resolve));
    const port = listener.address().port;
    await new Promise(resolve => listener.close(resolve));
    const issuer = `http://127.0.0.1:${port}`;
    service = spawn(binary, ["serve", "--data-dir", state, "--issuer", issuer, "--listen", `127.0.0.1:${port}`, "--development-http"], { stdio: "ignore", windowsHide: true });
    let startError;
    service.on("error", error => { startError = error; });
    const deadline = Date.now() + 20000;
    let isReady = false;
    while (Date.now() < deadline) {
      assert(!startError && service.exitCode === null, "Native service exited during startup");
      try { if ((await fetch(issuer + "/readyz", { signal: AbortSignal.timeout(1000) })).ok) { isReady = true; break; } } catch { /* wait within the startup deadline */ }
      await new Promise(resolve => setTimeout(resolve, 200));
    }
    assert(isReady, "Native archive did not become ready");
    const home = await fetch(issuer);
    assert.equal(home.status, 200);
    assert((await home.text()).includes("/assets/"));
    assert(home.headers.get("content-security-policy").includes("font-src 'self'"));
    assert.equal((await (await fetch(issuer + "/.well-known/openid-configuration")).json()).issuer, issuer);
    await stopProcess(service);
    service = null;
    run(binary, ["backup", "--data-dir", state, "--snapshot", path.join(temporary, "backup")]);
    run(binary, ["cleanup", "--data-dir", state, "--batch-size", "16"]);
    writeJson(reportPath, { source, version, platform: process.platform, binary_sha256: manifest.sha256, archive_sha256: process.env.ARCHIVE_SHA256 || null, checks: ["init", "readiness", "embedded UI and CSP", "discovery", "shutdown", "backup", "cleanup"], result: "pass" });
  } finally {
    if (service) await stopProcess(service);
    assert(path.resolve(temporary).startsWith(path.resolve(os.tmpdir()) + path.sep));
    fs.rmSync(temporary, { recursive: true, force: true });
  }
}

/** Stop a task-owned child within a deadline, including already-exited children. */
export async function stopProcess(child) {
  if (child.exitCode !== null || child.signalCode !== null || !child.pid) return;
  await new Promise((resolve, reject) => {
    const timer = setTimeout(() => { child.kill("SIGKILL"); reject(new Error("Task child shutdown timed out")); }, 10000);
    child.once("exit", () => { clearTimeout(timer); resolve(); });
    child.once("error", () => { clearTimeout(timer); resolve(); });
    child.kill();
  });
}

/** Execute one explicit release operation; publication is never inferred from a push. */
async function main() {
  const [command, argument, secondArgument] = process.argv.slice(2);
  const { source, version } = sourceContext();
  const repository = process.env.GITHUB_REPOSITORY || "QianFuv/QoAuth";
  assert.equal(repository, "QianFuv/QoAuth", "Unexpected publication repository");
  if (command === "context") {
    let previousVersion;
    const before = process.env.BEFORE_SHA;
    if (before && /^[a-f0-9]{40}$/.test(before) && !/^0+$/.test(before) && run("git", ["ls-tree", "--name-only", before, "--", "VERSION"]) === "VERSION") previousVersion = run("git", ["show", `${before}:VERSION`]);
    outputs(resolveContext({ event: process.env.GITHUB_EVENT_NAME, ref: process.env.GITHUB_REF, source, version, previousVersion, requestedVersion: process.env.REQUESTED_VERSION || "", sourceRun: process.env.SOURCE_RUN || "" }));
  } else if (command === "export-web") {
    assert(!fs.existsSync(argument));
    fs.mkdirSync(argument, { recursive: true });
    fs.cpSync(path.join(ROOT, "internal/webassets/dist"), path.join(argument, "web"), { recursive: true });
    writeJson(path.join(argument, "frontend-manifest.json"), { source, version, files: inventory(path.join(argument, "web")) });
  } else if (command === "import-web") {
    const manifest = JSON.parse(fs.readFileSync(path.join(argument, "frontend-manifest.json")));
    assert.equal(manifest.source, source);
    assert.equal(manifest.version, version);
    verifyInventory(path.join(argument, "web"), manifest.files);
    const destination = path.join(ROOT, "internal/webassets/dist");
    assert(!fs.existsSync(destination), "Import requires a fresh generated-asset directory");
    fs.cpSync(path.join(argument, "web"), destination, { recursive: true });
  } else if (command === "stage") {
    assert(!fs.existsSync(secondArgument));
    const files = verifyAssets(argument, source, version);
    fs.mkdirSync(secondArgument, { recursive: true });
    fs.cpSync(argument, path.join(secondArgument, "assets"), { recursive: true });
    writeJson(path.join(secondArgument, "acceptance.json"), { source, version, files });
  } else if (command === "verify-assets") {
    outputs({ source, version, files: verifyAssets(argument, source, version) });
  } else if (command === "smoke") {
    await smoke(argument, secondArgument);
  } else if (command === "prepublish") {
    assert.match(argument, /^[1-9]\d*$/);
    const runInfo = github(`repos/${repository}/actions/runs/${argument}`);
    const jobs = github(`repos/${repository}/actions/runs/${argument}/jobs?per_page=100`).jobs;
    validateRun(runInfo, jobs, source, repository);
    const artifacts = github(`repos/${repository}/actions/runs/${argument}/artifacts?per_page=100`).artifacts;
    const selected = {};
    for (const [key, name] of [["accepted_id", `accepted-release-${source}`], ["windows_id", `windows-acceptance-${source}`]]) {
      const matches = artifacts.filter(item => item.name === name && !item.expired);
      assert.equal(matches.length, 1, `Expected exactly one ${name}`);
      selected[key] = String(matches[0].id);
    }
    outputs({ ...selected, source_run: argument, source, version });
  } else if (command === "publish") {
    const accepted = JSON.parse(fs.readFileSync(path.join(argument, "acceptance.json")));
    assert.equal(accepted.source, source);
    assert.equal(accepted.version, version);
    const assets = path.join(argument, "assets");
    assert.deepEqual(verifyAssets(assets, source, version), accepted.files);
    const windows = JSON.parse(fs.readFileSync(secondArgument));
    assert.equal(windows.source, source);
    assert.equal(windows.result, "pass");
    assert.equal(windows.archive_sha256, accepted.files[`qoauth_${version}_windows_amd64.zip`]);
    const tag = `v${version}`;
    assert.equal(github(`repos/${repository}/releases/tags/${tag}`, [], true), null, "Never overwrite a Release");
    assert.equal(github(`repos/${repository}/git/ref/tags/${tag}`, [], true), null, "Never overwrite a tag");
    assert.equal(run("git", ["rev-parse", "--is-shallow-repository"]), "false");
    const reachableTags = new Set(run("git", ["tag", "--merged", source]).split(/\r?\n/));
    const releases = [];
    for (let page = 1; ; page++) {
      const batch = github(`repos/${repository}/releases?per_page=100&page=${page}`);
      assert(Array.isArray(batch));
      releases.push(...batch);
      if (batch.length < 100) break;
    }
    const previous = releases.filter(release => !release.draft && !release.prerelease && /^v(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)$/.test(release.tag_name) && reachableTags.has(release.tag_name) && compareVersions(release.tag_name.slice(1), version) < 0).sort((first, second) => compareVersions(second.tag_name.slice(1), first.tag_name.slice(1)))[0]?.tag_name;
    const log = run("git", ["log", "--reverse", "--format=%H%x00%s", previous ? `${previous}..${source}` : source, "--"]);
    const commits = log.split(/\r?\n/).map(line => { const separator = line.indexOf("\0"); return { hash: line.slice(0, separator), subject: line.slice(separator + 1) }; });
    const notes = path.join(argument, "notes.md");
    fs.writeFileSync(notes, renderNotes({ version, repository, commits, previous }));
    github(`repos/${repository}/git/refs`, ["--method", "POST", "-f", `ref=refs/tags/${tag}`, "-f", `sha=${source}`]);
    run("gh", ["release", "create", tag, "--repo", repository, "--verify-tag", "--title", `QoAuth ${tag}`, "--notes-file", notes, "--draft", ...[...assetNames(version), "SHA256SUMS"].map(name => path.join(assets, name))]);
    run("gh", ["release", "edit", tag, "--repo", repository, "--draft=false", "--latest"]);
  } else if (command === "verify-published") {
    const release = github(`repos/${repository}/releases/tags/v${version}`);
    assert.equal(release.draft, false);
    assert.equal(release.prerelease, false);
    assert.equal(release.name, `QoAuth v${version}`);
    assert.equal(github(`repos/${repository}/git/ref/tags/v${version}`).object.sha, source);
    assert.deepEqual(release.assets.map(asset => asset.name).sort(), [...assetNames(version), "SHA256SUMS"].sort());
    assert(!fs.existsSync(argument));
    fs.mkdirSync(argument, { recursive: true });
    for (const asset of release.assets) {
      const response = await fetch(asset.browser_download_url, { signal: AbortSignal.timeout(180000) });
      assert(response.ok, "Public asset download failed");
      const bytes = Buffer.from(await response.arrayBuffer());
      assert.equal(`sha256:${sha256(bytes)}`, asset.digest);
      fs.writeFileSync(path.join(argument, asset.name), bytes);
    }
    outputs({ source, version, url: release.html_url, files: verifyAssets(argument, source, version) });
  } else throw new Error("Expected context, export-web, import-web, stage, verify-assets, smoke, prepublish, publish or verify-published");
}

if (process.argv[1] && import.meta.url === pathToFileURL(path.resolve(process.argv[1])).href) await main();
