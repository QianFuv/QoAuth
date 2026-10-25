/** Exercise real synthetic authentication, responsive pages and network font delivery. */
import assert from "node:assert/strict";
import { createHash, randomBytes } from "node:crypto";
import { spawn, execFileSync } from "node:child_process";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import { launch, sleep } from "./browser-cdp.mjs";
import { stopProcess } from "./release.mjs";

const OUTPUT = path.resolve(process.argv[2] || ".cache/browser-release");
assert(!fs.existsSync(OUTPUT), "Browser output must be a new directory");
fs.mkdirSync(OUTPUT, { recursive: true });
const TEMPORARY = fs.mkdtempSync(path.join(os.tmpdir(), "qoauth-browser-"));
const STATE = path.join(TEMPORARY, "state.json");
const RP = path.join(TEMPORARY, process.platform === "win32" ? "go-rp.exe" : "go-rp");
const REPORT = { result: "pending", checks: [], fonts: [], viewports: [] };
let harness;
let browser;
let harnessError;

/** Record only safe check names, never synthetic credentials or token-bearing URLs. */
function check(name, condition) {
  assert(condition, name);
  REPORT.checks.push(name);
}

/** Read the latest private harness values without copying them into reports. */
async function state() {
  for (let attempt = 0; attempt < 20; attempt++) {
    try { return JSON.parse(fs.readFileSync(STATE, "utf8")); } catch { await sleep(50); }
  }
  throw new Error("Private harness state could not be read within its deadline");
}

/** Submit a real password login and optional current owner factor using the UI. */
async function signIn(page, username, password, isOwner = false) {
  await page.waitFor("document.querySelector('input[name=username]')");
  await page.type("input[name=username]", username);
  await page.type("input[name=password]", password);
  await page.click("button[type=submit]");
  if (isOwner) {
    await page.waitText("验证本次登录", 15000);
    await responsiveCurrent(page, "owner MFA");
    await page.type("input[name=code]", (await state()).owner_code);
    await page.click("button[type=submit]");
  }
}

/** Disable local font lookup and cache before navigation, so network evidence is mandatory. */
async function fontPage() {
  const page = await browser.newPage();
  await page.cdp("DOM.enable");
  await page.cdp("CSS.enable");
  await page.cdp("CSS.setLocalFontsEnabled", { enabled: false });
  await page.cdp("Network.setCacheDisabled", { cacheDisabled: true });
  return page;
}

/** Check a visible node's actual platform font usage rather than its CSS family string. */
async function fontUsage(page, selector) {
  const { root } = await page.cdp("DOM.getDocument");
  const { nodeId } = await page.cdp("DOM.querySelector", { nodeId: root.nodeId, selector });
  assert(nodeId, "Typography probe node missing");
  return (await page.cdp("CSS.getPlatformFontsForNode", { nodeId })).fonts;
}

/** Require usable layouts at every contract viewport without persisting private page state. */
async function responsive(page, issuer, route, text) {
  for (const width of [360, 390, 768, 1440]) {
    await page.setViewport(width, 900, width < 768);
    await page.goto(issuer + route);
    await page.waitText(text, 15000);
    await sleep(350);
    const overflow = await page.overflow();
    check(`${route} fits ${width}px`, overflow.scrollWidth <= overflow.clientWidth + 1 && overflow.offenders.length === 0);
    REPORT.viewports.push({ route, width, scrollWidth: overflow.scrollWidth, clientWidth: overflow.clientWidth });
  }
}

/** Verify a transient MFA or consent state without navigating away from it. */
async function responsiveCurrent(page, label) {
  for (const width of [360, 390, 768, 1440]) {
    await page.setViewport(width, 900, width < 768);
    await sleep(200);
    const overflow = await page.overflow();
    check(`${label} fits ${width}px`, overflow.scrollWidth <= overflow.clientWidth + 1 && overflow.offenders.length === 0);
    REPORT.viewports.push({ route: label, width, scrollWidth: overflow.scrollWidth, clientWidth: overflow.clientWidth });
  }
}

/** Make one same-origin CSRF-protected administrative request in the owner's browser. */
async function ownerPost(page, endpoint, payload) {
  return page.evaluate(`(async () => {
    const session = await (await fetch('/api/session')).json();
    const response = await fetch(${JSON.stringify(endpoint)}, {method:'POST',headers:{'Content-Type':'application/json','X-CSRF-Token':session.csrf},body:JSON.stringify(${JSON.stringify(payload)})});
    if (!response.ok) throw new Error('Synthetic administrative request failed');
    return response.json();
  })()`);
}

try {
  execFileSync("go", ["build", "-o", RP, "./examples/go-rp"], { timeout: 180000, stdio: "pipe", windowsHide: true });
  harness = spawn("go", ["test", "-count=1", "-run", "^TestBrowserHarness$", "-timeout", "10m", "./internal/oidc"], { env: { ...process.env, QOAUTH_BROWSER_STATE: STATE, QOAUTH_BROWSER_RP: RP, GOMAXPROCS: "2" }, stdio: "ignore", windowsHide: true });
  harness.on("error", error => { harnessError = error; });
  const deadline = Date.now() + 90000;
  while (!fs.existsSync(STATE) && Date.now() < deadline) {
    assert(!harnessError && harness.exitCode === null, "Synthetic browser harness failed to start");
    await sleep(200);
  }
  assert(fs.existsSync(STATE), "Synthetic browser harness startup deadline");
  const { issuer, rp, username, password } = await state();
  browser = await launch(path.join(TEMPORARY, "profile"));
  for (const [route, text] of [["/", "一次登录"], ["/login", "登录 QoAuth"]]) {
    const page = await fontPage();
    await responsive(page, issuer, route, text);
    await page.evaluate("document.fonts.ready");
    const responses = page.responses.filter(response => new URL(response.url).pathname.endsWith(".woff2"));
    const paths = [...new Set(responses.map(response => new URL(response.url).pathname))];
    check(`${route} fetches a proper subset of same-origin WOFF2`, paths.length > 0 && paths.length < 668 && responses.every(response => response.status === 200 && new URL(response.url).origin === issuer));
    check(`${route} has no font preloads`, await page.evaluate("!document.querySelector('link[rel=preload][as=font]')"));
    const weights = await page.evaluate("[...new Set([...document.fonts].filter(font=>font.family.replaceAll('\"','')==='JetBrainsLxgwNerdMono' && font.status==='loaded').map(font=>font.weight))]");
    check(`${route} loads both Regular and Bold faces`, weights.includes("400") && weights.includes("700"));
    const heading = await fontUsage(page, "h1");
    const paragraph = await fontUsage(page, route === "/" ? ".hero-lead" : ".auth-lead").catch(() => fontUsage(page, "p"));
    check(`${route} renders downloaded custom fonts for heading and regular text`, heading.some(font => font.isCustomFont && font.glyphCount > 0 && /JetBrainsLxgwNerdMono/i.test(font.familyName + font.postScriptName)) && paragraph.some(font => font.isCustomFont && font.glyphCount > 0 && /JetBrainsLxgwNerdMono/i.test(font.familyName + font.postScriptName)));
    REPORT.fonts.push({ route, localFontsEnabled: false, requestedSlices: paths.length, statuses: [...new Set(responses.map(response => response.status))], weights, heading, paragraph });
    if (route === "/" && page.problems.length) REPORT.publicPageProblems = page.problems.slice(0, 4).map(problem => problem.replace(/https?:\/\/\S+/g, "[URL redacted]").replace(/[A-Za-z0-9_-]{32,}/g, "[value]").slice(0, 400));
    check(`${route} has no unexpected CSP or runtime errors`, page.problems.filter(problem => !problem.includes("favicon")).length === 0);
  }

  const owner = await fontPage();
  await owner.goto(issuer + "/login");
  await signIn(owner, "owner", password, true);
  await owner.waitText("活动会话", 15000);
  check("owner password and TOTP login", true);
  await responsive(owner, issuer, "/account", "活动会话");
  await responsive(owner, issuer, "/admin", "管理员 MFA 已验证");
  for (const [route, text] of [["/admin#users", "用户"], ["/admin#applications", "已注册应用"], ["/admin#access", "分配应用访问"], ["/admin#activity", "安全事件"]]) await responsive(owner, issuer, route, text);
  await owner.goto(issuer + "/admin#users");
  await owner.waitText("创建用户");
  await owner.click("button", "创建用户");
  await owner.waitText("创建本地用户");
  await owner.type(".form-sheet input[name=username]", "release-member");
  await owner.type(".form-sheet input[type=password]", randomBytes(24).toString("base64url"));
  await owner.click(".form-sheet .sheet-footer button", "创建用户");
  await owner.waitText("用户已创建", 15000);
  check("administrator creates a synthetic local account through the UI", true);
  const member = await owner.evaluate("(async () => (await (await fetch('/api/users')).json()).find(user=>user.username==='member').id)()");
  const callback = issuer + "/synthetic-callback";
  await ownerPost(owner, "/api/applications/create", { application: { id: "release-public", name: "Release acceptance", public: true, redirects: [callback], logout_redirects: [issuer + "/signed-out"], scopes: ["openid", "profile", "offline_access"], audiences: [] } });
  await ownerPost(owner, "/api/assignments", { user: member, client: "release-public", allow: true });
  check("administrator registers a public client and explicit assignment", true);

  const memberPage = await fontPage();
  const verifier = randomBytes(32).toString("base64url");
  const challenge = createHash("sha256").update(verifier).digest("base64url");
  const query = new URLSearchParams({ client_id: "release-public", response_type: "code", redirect_uri: callback, scope: "openid profile offline_access", state: "release-state", nonce: "release-nonce", code_challenge: challenge, code_challenge_method: "S256" });
  await memberPage.goto(issuer + "/authorize?" + query);
  await signIn(memberPage, username, password);
  await memberPage.waitText("请确认该应用请求的权限", 15000);
  await responsiveCurrent(memberPage, "OIDC consent");
  await memberPage.click("button", "同意并继续");
  await memberPage.waitUrl("code=", 15000);
  const location = new URL(await memberPage.url());
  check("consent redirects to the exact callback with state", location.origin + location.pathname === callback && location.searchParams.get("state") === "release-state");
  const tokenResponse = await fetch(issuer + "/oauth/token", { method: "POST", body: new URLSearchParams({ grant_type: "authorization_code", client_id: "release-public", redirect_uri: callback, code: location.searchParams.get("code"), code_verifier: verifier }) });
  const tokens = await tokenResponse.json();
  check("PKCE authorization code exchange returns ID and refresh tokens", tokenResponse.ok && Boolean(tokens.id_token) && Boolean(tokens.refresh_token));
  const refresh = await fetch(issuer + "/oauth/token", { method: "POST", body: new URLSearchParams({ grant_type: "refresh_token", client_id: "release-public", refresh_token: tokens.refresh_token }) });
  check("real refresh-token exchange succeeds", refresh.ok && Boolean((await refresh.json()).access_token));
  await responsive(memberPage, issuer, "/account", "活动会话");
  await memberPage.goto(issuer + "/admin");
  await memberPage.waitText("需要管理员权限");
  check("member cannot enter administration", true);
  await memberPage.goto(`${issuer}/end_session?${new URLSearchParams({ id_token_hint: tokens.id_token, client_id: "release-public", post_logout_redirect_uri: issuer + "/signed-out", state: "signed-out" })}`);
  await memberPage.waitText("退出 QoAuth？");
  await responsiveCurrent(memberPage, "logout confirmation");
  await memberPage.click("button[type=submit]", "确认退出");
  await memberPage.waitUrl("state=signed-out", 15000);
  check("RP-initiated logout clears QoAuth authentication", await memberPage.evaluate("(async () => !(await (await fetch('/api/session')).json()).authenticated)()"));

  const standard = await fontPage();
  await standard.goto(rp + "/login");
  await signIn(standard, username, password);
  await standard.waitText("请确认该应用请求的权限", 15000);
  await standard.click("button", "同意并继续");
  await standard.waitText("Signed in as", 15000);
  check("standard Go RP verifies the real callback and ID token", true);
  await standard.goto(rp + "/profile");
  await standard.waitText("Verified subject:");
  check("standard Go RP reads authenticated UserInfo", await standard.evaluate(`location.pathname === '/profile' && document.body.innerText.trim() === ${JSON.stringify("Verified subject: " + member)}`));
  await standard.goto(rp);
  await standard.click("form[action='/logout'] button");
  await standard.waitText("退出 QoAuth？");
  await standard.click("button[type=submit]", "确认退出");
  await standard.waitUrl("/signed-out", 15000);
  check("standard Go RP and QoAuth logout complete", new URL(await standard.url()).origin === rp);

  const fallback = await fontPage();
  await fallback.cdp("Network.setBlockedURLs", { urls: ["*.woff2"] });
  await responsive(fallback, issuer, "/login", "登录 QoAuth");
  const fontDeadline = Date.now() + 5000;
  while (!fallback.failures.some(failure => failure.url?.endsWith(".woff2")) && Date.now() < fontDeadline) await sleep(100);
  const blockedFonts = fallback.failures.filter(failure => failure.url?.endsWith(".woff2") && (failure.blockedReason === "inspector" || failure.errorText === "net::ERR_BLOCKED_BY_CLIENT"));
  check("font-failure scenario actually blocks WOFF2 requests", blockedFonts.length > 0);
  REPORT.fonts.push({ scenario: "blocked WOFF2", blockedRequests: blockedFonts.length, reasons: [...new Set(blockedFonts.map(failure => failure.blockedReason || failure.errorText))] });
  await fallback.evaluate("document.fonts.ready");
  const fonts = await fontUsage(fallback, "h1");
  check("blocked WOFF2 uses a rendered system fallback", fonts.some(font => !font.isCustomFont && font.glyphCount > 0) && !fonts.some(font => font.isCustomFont));
  await signIn(fallback, username, password);
  await fallback.waitText("活动会话", 15000);
  check("font network failure preserves usable authentication", true);
  check("owner pages have no unexpected CSP or runtime errors", owner.problems.filter(problem => !problem.includes("favicon")).length === 0);
  REPORT.result = "pass";
} catch (error) {
  REPORT.result = "fail";
  REPORT.failure = String(error.message).replace(/https?:\/\/\S+/g, "[URL redacted]");
  process.exitCode = 1;
} finally {
  try { if (browser) await browser.close(); }
  catch { REPORT.result = "fail"; REPORT.failure = "Task browser shutdown failed"; }
  try {
    if (harness) {
      fs.writeFileSync(STATE + ".stop", "stop");
      const deadline = Date.now() + 8000;
      while (harness.exitCode === null && harness.signalCode === null && Date.now() < deadline) await sleep(100);
      await stopProcess(harness);
      assert.equal(harness.exitCode, 0, "Synthetic Go harness did not exit successfully");
    }
  } catch { REPORT.result = "fail"; REPORT.failure = "Synthetic Go harness did not stop successfully"; }
  try {
    assert(path.resolve(TEMPORARY).startsWith(path.resolve(os.tmpdir()) + path.sep));
    fs.rmSync(TEMPORARY, { recursive: true, force: true, maxRetries: 10, retryDelay: 100 });
  } catch { REPORT.result = "fail"; REPORT.failure = "Private browser fixture cleanup failed"; }
  if (REPORT.result !== "pass") process.exitCode = 1;
  fs.writeFileSync(path.join(OUTPUT, "report.json"), JSON.stringify(REPORT, null, 2) + "\n");
  console.log(JSON.stringify({ result: REPORT.result, checks: REPORT.checks.length, failure: REPORT.failure }));
}
