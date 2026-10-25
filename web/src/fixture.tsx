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
      <p className="eyebrow">测试夹具</p>
      {requestId ? <>
        <h1 className="auth-title">本地合成测试</h1>
        <p className="auth-lead">这个测试身份没有真实账户或密码。</p>
        <form method="post" action="/fixture/login" className="logout-form">
          <input type="hidden" name="request" value={requestId} />
          <Button htmlType="submit" theme="solid" size="large" block>以夹具用户继续</Button>
        </form>
      </> : <>
        <h1 className="auth-title">本地可行性构建</h1>
        <p className="auth-lead">此构建不提供正式账户初始化。</p>
      </>}
    </div>
  </AuthLayout>;
}
