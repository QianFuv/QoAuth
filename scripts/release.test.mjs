/** Regression tests for release intent, artifact identity and categorized notes. */
import assert from "node:assert/strict";
import { test } from "node:test";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import { assetNames, compareVersions, inventory, parseChecksums, parseVersion, renderNotes, resolveContext, stopProcess, validateRun, verifyAssets, verifyInventory } from "./release.mjs";
import { spawn } from "node:child_process";

const SOURCE = "a".repeat(40);

test("first push builds without publishing; only explicit main dispatch publishes", () => {
  assert.equal(resolveContext({ event: "push", ref: "refs/heads/main", source: SOURCE, version: "0.1.0" }).mode, "build");
  assert.equal(resolveContext({ event: "pull_request", ref: "refs/pull/1/merge", source: SOURCE, version: "0.1.0" }).mode, "quality");
  assert.equal(resolveContext({ event: "workflow_dispatch", ref: "refs/heads/main", source: SOURCE, version: "0.1.0", requestedVersion: "0.1.0", sourceRun: "123" }).mode, "publish");
  assert.throws(() => resolveContext({ event: "push", ref: "refs/heads/main", source: SOURCE, version: "0.1.0", sourceRun: "123" }));
  assert.throws(() => resolveContext({ event: "workflow_dispatch", ref: "refs/heads/other", source: SOURCE, version: "0.1.0", sourceRun: "123" }));
  assert.throws(() => resolveContext({ event: "workflow_dispatch", ref: "refs/heads/main", source: SOURCE, version: "0.1.0", requestedVersion: "0.2.0" }));
  assert.throws(() => resolveContext({ event: "push", source: SOURCE, version: "0.1.0", previousVersion: "0.2.0" }));
});

test("stable versions and archive names agree without lexical ordering", () => {
  assert.equal(compareVersions("0.10.0", "0.9.0"), 1);
  for (const version of ["v0.1.0", "01.0.0", "0.1.0-dev", "0.1.0\n"]) assert.throws(() => parseVersion(version));
  assert.deepEqual(assetNames("0.1.0"), ["qoauth_0.1.0_linux_amd64.tar.gz", "qoauth_0.1.0_windows_amd64.zip"]);
});

test("notes preserve every categorized commit and escape hostile subjects", () => {
  const notes = renderNotes({ version: "0.1.0", repository: "QianFuv/QoAuth", commits: [{ hash: SOURCE, subject: "feat(ui): add [home] <script>" }, { hash: "b".repeat(40), subject: "fix(auth): reject replay" }, { hash: "c".repeat(40), subject: "Initial source" }] });
  assert(notes.startsWith("## Commits (3)\n\nAll commits in this first stable release."));
  assert(notes.includes("### Features (1)"));
  assert(notes.includes("### Bug fixes (1)"));
  assert(notes.includes("### Other commits (1)"));
  assert(notes.includes("\\[home\\] &lt;script&gt;"));
  assert.equal((notes.match(/\/commit\//g) || []).length, 3);
  assert(notes.includes("**Full changelog:** [v0.1.0]"));
});

test("tested frontend identity includes hidden metadata and rejects changed or extra files", () => {
  const temporary = fs.mkdtempSync(path.join(os.tmpdir(), "qoauth-inventory-"));
  try {
    fs.mkdirSync(path.join(temporary, ".vite"));
    fs.writeFileSync(path.join(temporary, ".vite/license.json"), "[]");
    fs.writeFileSync(path.join(temporary, "index.html"), "tested bytes");
    const expected = inventory(temporary);
    assert(expected[".vite/license.json"]);
    verifyInventory(temporary, expected);
    fs.writeFileSync(path.join(temporary, "index.html"), "changed bytes");
    assert.throws(() => verifyInventory(temporary, expected));
    fs.writeFileSync(path.join(temporary, "index.html"), "tested bytes");
    fs.writeFileSync(path.join(temporary, "keys.json"), "private fixture");
    assert.throws(() => verifyInventory(temporary, expected));
  } finally {
    assert(path.resolve(temporary).startsWith(path.resolve(os.tmpdir()) + path.sep));
    fs.rmSync(temporary, { recursive: true, force: true });
  }
});

test("publication requires successful same-source CI with both acceptance jobs", () => {
  const run = { repository: { full_name: "QianFuv/QoAuth" }, head_sha: SOURCE, head_branch: "main", path: ".github/workflows/ci.yaml", status: "completed", conclusion: "success", event: "push" };
  const jobs = ["Linux archive and image", "Windows archive"].map(name => ({ name: "Release / " + name, conclusion: "success" }));
  validateRun(run, jobs, SOURCE, "QianFuv/QoAuth");
  assert.throws(() => validateRun({ ...run, head_sha: "b".repeat(40) }, jobs, SOURCE, "QianFuv/QoAuth"));
  assert.throws(() => validateRun({ ...run, conclusion: "failure" }, jobs, SOURCE, "QianFuv/QoAuth"));
  assert.throws(() => validateRun(run, jobs.slice(0, 1), SOURCE, "QianFuv/QoAuth"));
  assert.throws(() => validateRun(run, [...jobs.slice(0, 1), { ...jobs[1], conclusion: "skipped" }], SOURCE, "QianFuv/QoAuth"));
});

test("checksums cannot admit extra assets or repeated archive names", () => {
  const temporary = fs.mkdtempSync(path.join(os.tmpdir(), "qoauth-assets-"));
  try {
    fs.writeFileSync(path.join(temporary, "unexpected.db"), "private fixture");
    assert.throws(() => verifyAssets(temporary, SOURCE, "0.1.0"));
  } finally {
    assert(path.resolve(temporary).startsWith(path.resolve(os.tmpdir()) + path.sep));
    fs.rmSync(temporary, { recursive: true, force: true });
  }
});

test("Windows checksum lines identify each platform exactly once", () => {
  const names = assetNames("0.1.0");
  const lines = names.map((name, index) => `${String(index).repeat(64)}  ${name}`);
  assert.deepEqual(parseChecksums(lines.join("\r\n") + "\r\n", "0.1.0"), { [names[0]]: "0".repeat(64), [names[1]]: "1".repeat(64) });
  assert.throws(() => parseChecksums([lines[0], lines[0]].join("\n"), "0.1.0"));
  assert.throws(() => parseChecksums(lines[0], "0.1.0"));
  assert.throws(() => parseChecksums(lines.join("\n") + "\n" + "0".repeat(64) + "  keys.json", "0.1.0"));
});

test("already-exited task children stop without waiting for a past exit event", { timeout: 3000 }, async () => {
  const child = spawn(process.execPath, ["-e", "process.exit(0)"], { stdio: "ignore", windowsHide: true });
  await new Promise((resolve, reject) => { child.once("exit", resolve); child.once("error", reject); });
  await stopProcess(child);
});

test("read-only reusable build acceptance cannot request publication permissions", () => {
  const ci = fs.readFileSync(new URL("../.github/workflows/ci.yaml", import.meta.url), "utf8").replaceAll("\r\n", "\n");
  const release = fs.readFileSync(new URL("../.github/workflows/release.yaml", import.meta.url), "utf8");
  assert(!/contents:\s*write/.test(release), "A read-only caller cannot invoke a callee containing a write-permission job, even if that job is conditional");
  const jobs = [...ci.matchAll(/^  ([a-z][\w-]*):\n([\s\S]*?)(?=^  [a-z][\w-]*:\n|$(?![\s\S]))/gm)];
  assert.deepEqual(jobs.filter(([, , body]) => /contents:\s*write/.test(body)).map(([, name]) => name), ["release-publish"]);
  const publication = jobs.find(([, name]) => name === "release-publish")?.[2];
  assert(publication?.includes("mode == 'publish'"));
  assert(publication?.includes("node scripts/release.mjs prepublish"));
  assert(publication?.includes("node scripts/release.mjs verify-published"));
  assert(ci.includes("uses: ./.github/workflows/release.yaml"));
});
