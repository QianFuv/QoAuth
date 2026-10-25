/** Embedded production portal with an isolated legacy test-fixture presentation. */
import React from "react";
import { createRoot } from "react-dom/client";
import "@douyinfe/semi-ui/react19-adapter";
import Button from "@douyinfe/semi-ui/lib/es/button";
import "@douyinfe/semi-ui/lib/es/_base/base.css";
import "./style.css";
import { Portal } from "./portal";

/** Keep the M1 fixture presentation separate from the production portal. */
function App() {
  const requestId = new URLSearchParams(window.location.search).get("request");
  const isFixtureLogin = window.location.pathname === "/fixture/login";
  if (!isFixtureLogin) return <Portal />;
  return (
    <main>
      <h1>QoAuth</h1>
      <p>Lightweight self-hosted authentication</p>
      {isFixtureLogin && requestId ? (
        <section aria-labelledby="fixture-title">
          <h2 id="fixture-title">Synthetic local test</h2>
          <p>This test identity has no real account or password.</p>
          <form method="post" action="/fixture/login">
            <input type="hidden" name="request" value={requestId} />
            <Button htmlType="submit" theme="solid">Continue as fixture user</Button>
          </form>
        </section>
      ) : (
        <section>
          <h2>Local feasibility build</h2>
          <p>Production account setup is not available in this build.</p>
        </section>
      )}
    </main>
  );
}

const root = document.getElementById("root");
if (!root) throw new Error("Missing application root");
createRoot(root).render(<App />);
