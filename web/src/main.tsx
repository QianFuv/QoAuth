/** Browser entry: public homepage, production portal and the isolated test-fixture page. */
import React, { Suspense, lazy } from "react";
import { createRoot } from "react-dom/client";
import "@douyinfe/semi-ui/react19-adapter";
import "@douyinfe/semi-ui/lib/es/_base/base.css";
import "./style.css";

const Hero = lazy(() => import("./hero").then(module => ({ default: module.Hero })));
const Portal = lazy(() => import("./portal").then(module => ({ default: module.Portal })));
const FixtureLogin = lazy(() => import("./fixture").then(module => ({ default: module.FixtureLogin })));

/**
 * Pick the top-level page by path; every other path is server-authorized portal content.
 *
 * @returns The lazily loaded page for the current path.
 */
function App() {
  const path = window.location.pathname;
  return <Suspense fallback={<div className="boot" role="status"><span className="boot-mark" aria-hidden="true" /><span>Loading QoAuth…</span></div>}>
    {path === "/" ? <Hero /> : path === "/fixture/login" ? <FixtureLogin /> : <Portal />}
  </Suspense>;
}

const root = document.getElementById("root");
if (!root) throw new Error("Missing application root");
createRoot(root).render(<App />);
