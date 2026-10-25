/** Isolated M1 test-fixture page, served only by the test fixture provider. */
import React from "react";
import Button from "@douyinfe/semi-ui/lib/es/button";
import { AuthLayout } from "./layout";

/**
 * Keep the fixture presentation separate from the production portal.
 *
 * @returns The synthetic fixture sign-in page.
 */
export function FixtureLogin() {
  const requestId = new URLSearchParams(window.location.search).get("request");
  return <AuthLayout>
    <div className="auth-card">
      <p className="eyebrow">Test fixture</p>
      {requestId ? <>
        <h1 className="auth-title">Synthetic local test</h1>
        <p className="auth-lead">This test identity has no real account or password.</p>
        <form method="post" action="/fixture/login" className="logout-form">
          <input type="hidden" name="request" value={requestId} />
          <Button htmlType="submit" theme="solid" size="large" block>Continue as fixture user</Button>
        </form>
      </> : <>
        <h1 className="auth-title">Local feasibility build</h1>
        <p className="auth-lead">Production account setup is not available in this build.</p>
      </>}
    </div>
  </AuthLayout>;
}
