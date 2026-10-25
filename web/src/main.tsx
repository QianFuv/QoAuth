/** Browser entry: public homepage, production portal and the isolated test-fixture page. */
import React, { Suspense, lazy } from "react";
import { createRoot } from "react-dom/client";
import "@douyinfe/semi-ui/react19-adapter";
import LocaleProvider from "@douyinfe/semi-ui/lib/es/locale/localeProvider";
import englishLocale from "@douyinfe/semi-ui/lib/es/locale/source/en_US";
import "@douyinfe/semi-ui/lib/es/_base/base.css";
import "./style.css";

void import("./fonts");

const Hero = lazy(() => import("./hero").then(module => ({ default: module.Hero })));
const Portal = lazy(() => import("./portal").then(module => ({ default: module.Portal })));
const FixtureLogin = lazy(() => import("./fixture").then(module => ({ default: module.FixtureLogin })));

/**
 * Pick the top-level page by path; every other path is server-authorized portal content.
 *
 * Semi built-in strings (pagination, empty states) follow the product's English copy.
 *
 * @returns The lazily loaded page for the current path.
 */
function App() {
  const path = window.location.pathname;
  return <LocaleProvider locale={englishLocale}>
    <Suspense fallback={<div className="boot" role="status"><span className="boot-mark" aria-hidden="true" /><span>Loading QoAuth…</span></div>}>
      {path === "/" ? <Hero /> : path === "/fixture/login" ? <FixtureLogin /> : <Portal />}
    </Suspense>
  </LocaleProvider>;
}

const root = document.getElementById("root");
if (!root) throw new Error("Missing application root");
createRoot(root).render(<App />);
