/**
 * Minimal Chrome DevTools Protocol driver for local synthetic UI acceptance.
 *
 * It launches the installed Chrome headless with a throwaway profile, talks only
 * to loopback, and exposes navigation, input, evaluation and screenshot helpers.
 */
import { spawn } from "node:child_process";
import { mkdirSync, writeFileSync, readFileSync, existsSync } from "node:fs";
import path from "node:path";
import { stopProcess } from "./release.mjs";

const CHROME = process.env.CHROME_PATH || (process.platform === "win32" ? "C:/Program Files/Google/Chrome/Application/chrome.exe" : "/usr/bin/google-chrome");

/** Sleep for a number of milliseconds. */
export const sleep = milliseconds => new Promise(resolve => setTimeout(resolve, milliseconds));

/** Launch Chrome and return a connected browser handle. */
export async function launch(profileDirectory) {
  if (existsSync(profileDirectory)) throw new Error("Browser profile must be new");
  mkdirSync(profileDirectory, { recursive: true });
  const child = spawn(CHROME, ["--headless=new", "--remote-debugging-port=0", `--user-data-dir=${profileDirectory}`, "--no-first-run", "--no-default-browser-check", "--disable-extensions", "--disable-sync", "--disable-background-networking", "--hide-scrollbars", "--force-color-profile=srgb", "--lang=en-US", "about:blank"], { stdio: "ignore", windowsHide: true });
  let launchError;
  child.on("error", error => { launchError = error; });
  try {
  let version;
  let port;
  for (let attempt = 0; attempt < 50; attempt++) {
    if (launchError) throw new Error("Chromium could not start");
    try { port = Number(readFileSync(path.join(profileDirectory, "DevToolsActivePort"), "utf8").split("\n")[0]); version = await (await fetch(`http://127.0.0.1:${port}/json/version`)).json(); break; } catch { await sleep(200); }
  }
  if (!version) { await stopProcess(child); throw new Error("Chrome did not start"); }
  const socket = new WebSocket(version.webSocketDebuggerUrl);
  await new Promise((resolve, reject) => { socket.onopen = resolve; socket.onerror = reject; });
  let nextId = 1;
  const pending = new Map();
  const listeners = new Set();
  socket.onmessage = event => {
    const message = JSON.parse(event.data);
    if (message.id && pending.has(message.id)) {
      const { resolve, reject } = pending.get(message.id);
      pending.delete(message.id);
      if (message.error) reject(new Error(message.error.message)); else resolve(message.result);
    } else for (const listener of listeners) listener(message);
  };
  /** Send one CDP command. */
  const send = (method, params = {}, sessionId) => new Promise((resolve, reject) => {
    const id = nextId++;
    const timer = setTimeout(() => { pending.delete(id); reject(new Error("CDP timeout: " + method)); }, 15000);
    pending.set(id, { resolve: value => { clearTimeout(timer); resolve(value); }, reject: error => { clearTimeout(timer); reject(error); } });
    socket.send(JSON.stringify({ id, method, params, sessionId }));
  });
  return {
    send, listeners,
    /** Close only this task browser and release the protocol socket. */
    async close() { try { await send("Browser.close"); } catch { /* already closed */ } socket.close(); await stopProcess(child); },
    /** Open a page in a fresh isolated browser context (separate cookies). */
    async newPage() {
      const { browserContextId } = await send("Target.createBrowserContext", { disposeOnDetach: true });
      const { targetId } = await send("Target.createTarget", { url: "about:blank", browserContextId });
      const { sessionId } = await send("Target.attachToTarget", { targetId, flatten: true });
      return createPage(send, listeners, sessionId);
    },
  };
  } catch (error) {
    await stopProcess(child);
    throw error;
  }
}

/** Wrap one attached target with convenience helpers. */
async function createPage(send, listeners, sessionId) {
  const call = (method, params) => send(method, params, sessionId);
  const problems = [];
  const requests = [];
  const responses = [];
  const failures = [];
  const requestUrls = new Map();
  const loadedDocuments = new Set();
  let viewport = { width: 1440, height: 900, mobile: false };
  listeners.add(message => {
    if (message.sessionId !== sessionId) return;
    if (message.method === "Page.lifecycleEvent" && message.params.name === "load") loadedDocuments.add(message.params.loaderId);
    if (message.method === "Network.requestWillBeSent") requestUrls.set(message.params.requestId, message.params.request.url);
    if (message.method === "Network.loadingFailed") failures.push({ url: requestUrls.get(message.params.requestId), blockedReason: message.params.blockedReason, errorText: message.params.errorText });
    if (message.method === "Network.responseReceived") responses.push(message.params.response);
    if (message.method === "Network.requestWillBeSent") requests.push(message.params.request.method + " " + message.params.request.url);
    if (message.method === "Runtime.exceptionThrown") problems.push("exception: " + (message.params.exceptionDetails.exception?.description ?? message.params.exceptionDetails.text));
    if (message.method === "Runtime.consoleAPICalled" && message.params.type === "error") problems.push("console.error: " + message.params.args.map(arg => arg.value ?? arg.description ?? "").join(" ").slice(0, 300));
    if (message.method === "Log.entryAdded" && !message.params.entry.url?.endsWith("/favicon.ico") && (message.params.entry.level === "error" || /Content Security Policy|Refused to/.test(message.params.entry.text))) problems.push("log: " + message.params.entry.text.slice(0, 300));
  });
  await call("Page.enable"); await call("Runtime.enable"); await call("Log.enable"); await call("Network.enable");
  await call("Page.setLifecycleEventsEnabled", { enabled: true });
  const page = {
    cdp: call,
    problems,
    requests,
    responses,
    failures,
    /** Return the current emulated viewport. */
    get viewport() { return viewport; },
    /** Emulate one explicit viewport and its touch capability. */
    async setViewport(width, height, mobile) {
      viewport = { width, height, mobile };
      await call("Emulation.setDeviceMetricsOverride", { width, height, deviceScaleFactor: 1, mobile });
      await call("Emulation.setTouchEmulationEnabled", mobile ? { enabled: true, maxTouchPoints: 5 } : { enabled: false });
    },
    /** Emulate the requested reduced-motion preference. */
    async setReducedMotion(isReduced) {
      await call("Emulation.setEmulatedMedia", { features: [{ name: "prefers-reduced-motion", value: isReduced ? "reduce" : "no-preference" }] });
    },
    /** Evaluate a bounded test expression and return its serialized value. */
    async evaluate(expression) {
      const result = await call("Runtime.evaluate", { expression, awaitPromise: true, returnByValue: true });
      if (result.exceptionDetails) throw new Error("evaluate failed: " + (result.exceptionDetails.exception?.description ?? result.exceptionDetails.text) + " :: " + expression.slice(0, 120));
      return result.result.value;
    },
    /** Navigate and wait for the target document to finish loading. */
    async goto(url) {
      const navigation = await call("Page.navigate", { url });
      if (navigation.errorText) throw new Error("Page navigation failed: " + navigation.errorText);
      if (navigation.loaderId) {
        const deadline = Date.now() + 15000;
        while (!loadedDocuments.has(navigation.loaderId) && Date.now() < deadline) await sleep(100);
        if (!loadedDocuments.has(navigation.loaderId)) throw new Error("Target document load deadline exceeded");
      } else await page.waitFor(`location.href === ${JSON.stringify(url)}`, 15000);
      await page.waitFor("document.readyState === 'complete'", 15000);
      await sleep(250);
    },
    /** Read the current page URL in memory without logging it. */
    async url() { return page.evaluate("location.href"); },
    /** Await a test condition within its explicit deadline. */
    async waitFor(expression, timeout = 10000) {
      const deadline = Date.now() + timeout;
      while (Date.now() < deadline) {
        try { if (await page.evaluate(`Boolean(${expression})`)) return; } catch { /* navigating */ }
        await sleep(120);
      }
      throw new Error("timeout waiting for: " + expression);
    },
    /** Await visible page text within a deadline. */
    async waitText(text, timeout = 10000) { await page.waitFor(`document.body && document.body.innerText.includes(${JSON.stringify(text)})`, timeout); },
    /** Await a known fragment without persisting the complete URL. */
    async waitUrl(fragment, timeout = 10000) { await page.waitFor(`location.href.includes(${JSON.stringify(fragment)})`, timeout); },
    /** Click the center of the first visible element matching a CSS selector, optionally filtered by text. */
    async click(selector, text) {
      const point = await page.evaluate(`(() => {
        const candidates = [...document.querySelectorAll(${JSON.stringify(selector)})].filter(element => {
          const box = element.getBoundingClientRect();
          return box.width > 0 && box.height > 0 && (${JSON.stringify(text ?? null)} === null || element.innerText.trim().includes(${JSON.stringify(text ?? "")}));
        });
        const element = candidates[0];
        if (!element) return null;
        element.setAttribute("data-cdp-target", "1");
        element.scrollIntoView({ block: "center", inline: "nearest", behavior: "instant" });
        return true;
      })()`) && await (async () => { await sleep(120); return page.evaluate(`(() => { const element = document.querySelector("[data-cdp-target]"); element.removeAttribute("data-cdp-target"); const box = element.getBoundingClientRect(); return { x: box.left + box.width / 2, y: box.top + box.height / 2 }; })()`); })();
      if (!point) throw new Error(`no clickable element: ${selector} ${text ?? ""}`);
      await sleep(80);
      for (const type of ["mouseMoved", "mousePressed", "mouseReleased"]) await call("Input.dispatchMouseEvent", { type, x: point.x, y: point.y, button: "left", clickCount: 1 });
      await sleep(250);
    },
    /** Focus an input and insert text as real keyboard input. */
    async type(selector, text) {
      const isFound = await page.evaluate(`(() => { const element = document.querySelector(${JSON.stringify(selector)}); if (!element) return false; element.scrollIntoView({ block: "center", behavior: "instant" }); element.focus(); return document.activeElement === element; })()`);
      if (!isFound) throw new Error("no input: " + selector);
      await call("Input.insertText", { text });
      await sleep(80);
    },
    /** Dispatch one real keyboard key sequence. */
    async press(key) {
      const codes = { Enter: 13, Escape: 27, Tab: 9 };
      await call("Input.dispatchKeyEvent", { type: "keyDown", key, windowsVirtualKeyCode: codes[key], code: key, text: key === "Enter" ? "\r" : undefined });
      await call("Input.dispatchKeyEvent", { type: "keyUp", key, windowsVirtualKeyCode: codes[key], code: key });
      await sleep(200);
    },
    /** Report horizontal overflow of the document and the widest offending elements. */
    async overflow() {
      return page.evaluate(`(() => {
        const width = document.documentElement.clientWidth;
        const offenders = [];
        for (const element of document.querySelectorAll("body *")) {
          const box = element.getBoundingClientRect();
          if (box.width === 0 || box.height === 0) continue;
          const style = getComputedStyle(element);
          if (style.visibility === "hidden" || element.getAttribute("aria-hidden") === "true" && element.tagName === "TEXTAREA") continue;
          let clipped = false;
          for (let parent = element.parentElement; parent; parent = parent.parentElement) {
            const parentStyle = getComputedStyle(parent);
            if (/(auto|scroll|hidden|clip)/.test(parentStyle.overflowX) && parent !== document.body && parent !== document.documentElement) { const parentBox = parent.getBoundingClientRect(); if (parentBox.right <= width + 1) { clipped = true; break; } }
          }
          if (!clipped && (box.right > width + 1 || box.left < -1)) offenders.push((element.className && typeof element.className === "string" ? element.tagName.toLowerCase() + "." + element.className.split(" ")[0] : element.tagName.toLowerCase()) + " [" + Math.round(box.left) + "," + Math.round(box.right) + "]");
        }
        return { scrollWidth: document.documentElement.scrollWidth, clientWidth: width, offenders: offenders.slice(0, 8) };
      })()`);
    },
    /** Capture a PNG; full page when requested. */
    async screenshot(file, isFullPage = false) {
      await sleep(450);
      let clip;
      if (isFullPage) {
        const height = await page.evaluate("Math.max(document.documentElement.scrollHeight, document.body.scrollHeight)");
        clip = { x: 0, y: 0, width: viewport.width, height: Math.min(height, 12000), scale: 1 };
      }
      const { data } = await call("Page.captureScreenshot", { format: "png", captureBeyondViewport: isFullPage, ...(clip ? { clip } : {}) });
      mkdirSync(path.dirname(file), { recursive: true });
      writeFileSync(file, Buffer.from(data, "base64"));
      return file;
    },
    /** Intercept matching requests and fail them with a status, for error-state coverage. */
    async failRequests(urlPattern, status, body) {
      await call("Fetch.enable", { patterns: [{ urlPattern }] });
      listeners.add(async message => {
        if (message.sessionId !== sessionId || message.method !== "Fetch.requestPaused") return;
        await call("Fetch.fulfillRequest", { requestId: message.params.requestId, responseCode: status, responseHeaders: [{ name: "Content-Type", value: "text/plain" }], body: Buffer.from(body).toString("base64") });
      });
    },
    /** Remove this page request interception. */
    async stopFailing() { await call("Fetch.disable"); },
  };
  return page;
}
