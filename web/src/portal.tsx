/** Sign-in, factor, consent, logout and self-service account pages. */
import React, { Suspense, lazy, useEffect, useState } from "react";
import Button from "@douyinfe/semi-ui/lib/es/button";
import Banner from "@douyinfe/semi-ui/lib/es/banner";
import Tag from "@douyinfe/semi-ui/lib/es/tag";
import Steps from "@douyinfe/semi-ui/lib/es/steps";
import Skeleton from "@douyinfe/semi-ui/lib/es/skeleton";
import Spin from "@douyinfe/semi-ui/lib/es/spin";
import Empty from "@douyinfe/semi-ui/lib/es/empty";
import Table from "@douyinfe/semi-ui/lib/es/table";
import List from "@douyinfe/semi-ui/lib/es/list";
import Popconfirm from "@douyinfe/semi-ui/lib/es/popconfirm";
import Toast from "@douyinfe/semi-ui/lib/es/toast";
import { IllustrationFailure, IllustrationNoAccess, IllustrationNoContent } from "@douyinfe/semi-illustrations";
import { IconArrowRight, IconExit, IconKey, IconRefresh, IconShield, IconTick } from "@douyinfe/semi-icons";
import { getJson, postJson, type Interaction, type SessionRecord, type SessionState } from "./api";
import { PasswordField, passwordRule } from "./password-field";
import { TextField, required, useForm } from "./form";
import { AuthLayout, COMPACT_QUERY, formatTime, navigate, useLocation, useMediaQuery, type Location } from "./layout";
import { ConsoleShell, CopyBlock, PageHeader, Panel, StatusView } from "./ui";

type Task = () => Promise<void>;

const AdminPanel = lazy(() => import("./admin").then(module => ({ default: module.AdminPanel })));

/** Shared session state and the single mutation boundary passed to every page. */
export interface Actions { session: SessionState; refresh: () => Promise<void>; run: (task: Task) => Promise<void>; busy: boolean }

/**
 * Render the authenticated product with one explicit error and mutation boundary.
 *
 * @returns The page matching the current location and server-reported session state.
 */
export function Portal() {
  const location = useLocation();
  const [session, setSession] = useState<SessionState | null>(null);
  const [isUnavailable, setIsUnavailable] = useState(false);
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  /** Refresh authority from the server after a cookie rotation or security change. */
  async function refresh() {
    const next = await getJson<SessionState>("/api/session");
    setSession(next);
    if (next.assurance && window.location.pathname === "/login" && !new URLSearchParams(window.location.search).has("request")) window.location.assign("/account");
  }
  /** Show safe server errors and prevent accidental duplicate submissions. */
  async function run(task: Task) {
    setBusy(true); setError("");
    try { await task(); } catch (failure) { setError(failure instanceof Error ? failure.message : "请求未成功。"); }
    finally { setBusy(false); }
  }
  useEffect(() => {
    let isActive = true;
    void getJson<SessionState>("/api/session").then(value => { if (isActive) setSession(value); }).catch(() => { if (isActive) setIsUnavailable(true); });
    return () => { isActive = false; };
  }, []);
  useEffect(() => { setError(""); }, [location.path, location.hash]);
  if (isUnavailable) return <AuthLayout><StatusView illustration={<IllustrationFailure />} title="QoAuth 暂不可用" description="认证服务没有响应。请确认服务器正在运行，然后重试。" action={<Button icon={<IconRefresh />} onClick={() => window.location.reload()}>重试</Button>} /></AuthLayout>;
  if (!session) return <div className="boot" role="status"><Spin size="large" /><span>正在连接 QoAuth…</span></div>;
  const requestId = new URLSearchParams(location.search).get("request");
  const notice = error ? <ErrorNotice message={error} onClose={() => setError("")} /> : null;
  return <PortalContent session={session} refresh={refresh} run={run} busy={busy} requestId={requestId} location={location} notice={notice} />;
}

/**
 * Show one dismissible error from the last failed request.
 *
 * @param props - The error message and close handler.
 * @returns An alert banner.
 */
function ErrorNotice({ message, onClose }: { message: string; onClose: () => void }) {
  return <div role="alert" className="notice-slot enter"><Banner type="danger" description={message} onClose={onClose} /></div>;
}

/** Keep login and privileged views separate while relying on server authorization. */
function PortalContent(props: Actions & { requestId: string | null; location: Location; notice: React.ReactNode }) {
  const { session, requestId, location, notice } = props;
  if (location.path === "/end_session") return <AuthLayout>{notice}<LogoutConfirmation session={session} /></AuthLayout>;
  if (!session.authenticated) return <AuthLayout>{notice}<LoginForm {...props} /></AuthLayout>;
  if (!session.assurance) return <AuthLayout>{notice}<FactorForm {...props} key={session.csrf} /></AuthLayout>;
  if (requestId) return <AuthLayout>{notice}<ConsentPage {...props} requestId={requestId} key={session.csrf} /></AuthLayout>;
  if (location.path === "/login") return <AuthLayout>{notice}<LoginForm {...props} /></AuthLayout>;
  /** End this QoAuth session and return to the public homepage. */
  const signOut = () => void props.run(async () => { await postJson("/api/logout", {}, session.csrf); window.location.assign("/"); });
  const isAdminPath = location.path === "/admin";
  const selectedKey = isAdminPath ? "admin-" + (["users", "applications", "access", "activity"].includes(location.hash) ? location.hash : "overview") : "account";
  return <ConsoleShell session={session} selectedKey={selectedKey} isBusy={props.busy} onSignOut={signOut}>
    {notice}
    {isAdminPath
      ? session.account.admin ? <Suspense fallback={<div className="console-page" role="status" aria-label="正在加载管理后台"><Spin size="large" /></div>}><AdminPanel {...props} section={location.hash} /></Suspense> : <StatusView illustration={<IllustrationNoAccess />} title="需要管理员权限" description="此账户可以管理自己的登录方式和会话，但不能管理整个安装实例。" action={<Button onClick={() => navigate("/account")}>前往我的账户</Button>} />
      : <AccountPage {...props} />}
  </ConsoleShell>;
}

/**
 * Read an authorization request's public description for login context.
 *
 * @param requestId - The pending authorization request, if any.
 * @returns The server-reported application name, or an empty string.
 */
function useApplicationName(requestId: string | null): string {
  const [name, setName] = useState("");
  useEffect(() => {
    if (!requestId) return;
    let isActive = true;
    void getJson<Interaction>("/api/interaction?request=" + encodeURIComponent(requestId)).then(value => { if (isActive) setName(value.application); }).catch(() => undefined);
    return () => { isActive = false; };
  }, [requestId]);
  return name;
}

/** Submit a password only to the same-origin login endpoint. */
function LoginForm({ session, refresh, run, busy, requestId }: Actions & { requestId?: string | null }) {
  const form = useForm({ username: "", password: "" });
  const applicationName = useApplicationName(requestId ?? null);
  /** Send the credentials, then clear the form before loading the next step. */
  function submit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!form.check({ username: required("请输入用户名。"), password: required("请输入密码。") })) return;
    const { username, password } = form.values;
    void run(async () => { await postJson("/api/login", { username, password }, session.csrf); form.reset(); await refresh(); });
  }
  return <div className="auth-card">
    <p className="eyebrow">{applicationName ? "单点登录" : "欢迎回来"}</p>
    <h1 className="auth-title">{applicationName ? <>登录以继续前往 <span className="accent-text">{applicationName}</span></> : "登录 QoAuth"}</h1>
    <p className="auth-lead">请使用管理员为你创建的本地账户。</p>
    <form className="auth-form" onSubmit={submit} noValidate>
      <TextField form={form} field="username" label="用户名" name="username" autoComplete="username" maxLength={64} autoFocus size="large" disabled={busy} />
      <TextField form={form} field="password" label="密码" name="password" mode="password" autoComplete="current-password" maxLength={1024} size="large" disabled={busy} />
      <Button htmlType="submit" theme="solid" size="large" block loading={busy} icon={<IconArrowRight />} iconPosition="right">登录</Button>
    </form>
    <p className="auth-hint">不开放公开注册。新安装实例的所有者通过本地 <code>qoauth init</code> 命令创建。</p>
  </div>;
}

/** Enroll or prove a factor; recovery always returns to restricted enrollment. */
function FactorForm({ session, refresh, run, busy, onCancel }: Actions & { onCancel?: () => void }) {
  const [secret, setSecret] = useState("");
  const [codes, setCodes] = useState<string[]>([]);
  const [isRecovering, setIsRecovering] = useState(false);
  const form = useForm({ code: "" });
  /** Verify the selected factor proof without retaining the input. */
  function verify(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const isCodeValid = form.check({ code: isRecovering ? required("请输入恢复码。") : value => value === "" ? "请输入 6 位动态码。" : /^[0-9]{6}$/.test(value) ? "" : "动态码必须是 6 位数字。" });
    if (!isCodeValid) return;
    const code = form.values.code;
    const path = isRecovering ? "/api/mfa/recover" : session.has_factor ? "/api/mfa/verify" : "/api/mfa/confirm";
    void run(async () => {
      try {
        const result = await postJson<{ recovery_codes?: string[] }>(path, { code }, session.csrf);
        if (result.recovery_codes) { setSecret(""); setCodes(result.recovery_codes); } else await refresh();
      } finally { form.reset(); }
    });
  }
  const isEnrolling = !session.has_factor;
  if (codes.length) return <div className="auth-card">
    <EnrollmentSteps current={2} />
    <h1 className="auth-title">保存你的恢复码</h1>
    <p className="auth-lead">每个恢复码只能使用一次，可用于更换丢失的验证器。继续之前请把它们保存到密码管理器中，之后不会再次显示。</p>
    <CopyBlock value={codes.join("\n")} label="恢复码" isMultiline />
    <Button theme="solid" size="large" block icon={<IconTick />} disabled={busy} onClick={() => void run(async () => { setCodes([]); await refresh(); })}>我已保存恢复码</Button>
  </div>;
  const title = isRecovering ? "恢复验证器" : session.has_factor ? "验证本次登录" : "设置验证器";
  const lead = isRecovering ? "恢复码用于重新绑定一个新的验证器，不会绕过多因素验证。" : session.has_factor ? "请输入验证器应用中的 6 位动态码。" : "使用管理员权限或已恢复的账户前，请先添加基于时间的一次性密码（TOTP）。";
  return <div className="auth-card">
    {isEnrolling ? <EnrollmentSteps current={secret ? 1 : 0} /> : <div className="auth-icon" aria-hidden="true">{isRecovering ? <IconKey size="large" /> : <IconShield size="large" />}</div>}
    <p className="eyebrow">账户安全</p>
    <h1 className="auth-title">{title}</h1>
    <p className="auth-lead">{lead}</p>
    {isEnrolling && !secret ? <Button theme="solid" size="large" block disabled={busy} loading={busy} onClick={() => void run(async () => { const result = await postJson<{ secret: string }>("/api/mfa/start", {}, session.csrf); setSecret(result.secret); })}>生成验证器设置</Button> : null}
    {secret ? <div className="setup-key">
      <p>在验证器应用中添加一个基于时间的账户，名称为 <strong>QoAuth / {session.account.username}</strong>，并输入以下设置密钥：</p>
      <CopyBlock value={secret} label="验证器设置密钥" />
      <div className="setup-key-meta"><Tag size="small">6 位数字</Tag><Tag size="small">30 秒</Tag><Tag size="small">SHA-1</Tag></div>
    </div> : null}
    {session.has_factor || secret ? <form className="auth-form" onSubmit={verify} noValidate>
      {isRecovering
        ? <TextField key="recovery" form={form} field="code" label="恢复码" name="code" autoComplete="one-time-code" maxLength={64} size="large" autoFocus disabled={busy} />
        : <TextField key="totp" form={form} field="code" label="验证器动态码" name="code" className="code-input" autoComplete="one-time-code" inputMode="numeric" maxLength={6} size="large" autoFocus placeholder="000000" disabled={busy} />}
      <Button htmlType="submit" theme="solid" size="large" block loading={busy}>{isRecovering ? "恢复并重新绑定" : "验证"}</Button>
    </form> : null}
    <div className="auth-secondary">
      {session.has_factor ? <Button theme="borderless" disabled={busy} onClick={() => { form.reset(); setIsRecovering(value => !value); }}>{isRecovering ? "改用验证器" : "使用恢复码"}</Button> : null}
      {onCancel ? <Button theme="borderless" type="tertiary" disabled={busy} onClick={onCancel}>返回我的账户</Button> : null}
    </div>
  </div>;
}

/** Show progress through the three real enrollment steps. */
function EnrollmentSteps({ current }: { current: number }) {
  return <Steps className="enroll-steps" type="basic" size="small" current={current} aria-label="验证器设置进度">
    <Steps.Step title="生成密钥" />
    <Steps.Step title="验证动态码" />
    <Steps.Step title="保存恢复码" />
  </Steps>;
}

/** Plain-language descriptions for the standard scopes QoAuth implements. */
const SCOPE_DESCRIPTIONS: Record<string, string> = {
  openid: "向该应用确认你的 QoAuth 身份。",
  profile: "读取你的基本资料，例如用户名。",
  offline_access: "通过刷新令牌保持登录，直到你撤销访问。",
};

/** Present the server's immutable requested permissions and explicit consent. */
function ConsentPage(props: Actions & { requestId: string }) {
  const [interaction, setInteraction] = useState<Interaction | null>(null);
  const [isExpired, setIsExpired] = useState(false);
  useEffect(() => {
    let isActive = true;
    void getJson<Interaction>("/api/interaction?request=" + encodeURIComponent(props.requestId)).then(value => { if (isActive) setInteraction(value); }).catch(() => { if (isActive) setIsExpired(true); });
    return () => { isActive = false; };
  }, [props.requestId, props.session.csrf]);
  if (isExpired) return <StatusView illustration={<IllustrationFailure />} title="请求已过期" description="此授权请求已失效。请返回应用重新发起登录。" />;
  if (!interaction) return <div className="auth-card" role="status" aria-label="正在加载应用请求"><Skeleton active placeholder={<><Skeleton.Title style={{ width: "60%", marginBottom: 16 }} /><Skeleton.Paragraph rows={4} /></>} loading /></div>;
  if (interaction.needs_login) return <LoginForm {...props} />;
  /** Let the server choose the validated callback after approval or denial. */
  async function decide(isApproved: boolean) { const result = await postJson<{ redirect: string }>("/api/consent", { request: props.requestId, approve: isApproved }, props.session.csrf); window.location.assign(result.redirect); }
  return <div className="auth-card consent-card">
    <div className="consent-apps" aria-hidden="true">
      <span className="consent-app-mark">{interaction.application.slice(0, 1).toUpperCase()}</span>
      <span className="consent-link" />
      <span className="consent-app-mark consent-app-mark-qoauth"><span /></span>
    </div>
    <p className="eyebrow">应用授权</p>
    <h1 className="auth-title">继续前往 <span className="accent-text">{interaction.application}</span></h1>
    <p className="auth-lead">当前登录账户为 <strong>{props.session.account.username}</strong>。请确认该应用请求的权限。</p>
    <ul className="scope-list" aria-label="请求的权限">
      {interaction.scopes.map(scope => <li key={scope}>
        <IconTick className="scope-check" aria-hidden="true" />
        <div><code>{scope}</code><span>{SCOPE_DESCRIPTIONS[scope] ?? "由本实例中已注册应用定义的权限。"}</span></div>
      </li>)}
    </ul>
    {interaction.audiences.length ? <div className="consent-resources"><span>资源 API</span>{interaction.audiences.map(audience => <Tag key={audience} size="small" color="blue">{audience}</Tag>)}</div> : null}
    <div className="consent-actions">
      <Button theme="solid" size="large" disabled={props.busy} loading={props.busy} onClick={() => void props.run(() => decide(true))}>同意并继续</Button>
      <Button size="large" type="tertiary" disabled={props.busy} onClick={() => void props.run(() => decide(false))}>拒绝</Button>
    </div>
    <p className="auth-hint">之后可以在“我的账户”中撤回此授权。</p>
  </div>;
}

/** Confirm standard RP logout through a native form with session-bound CSRF. */
function LogoutConfirmation({ session }: { session: SessionState }) {
  const query = new URLSearchParams(window.location.search);
  return <div className="auth-card">
    <div className="auth-icon" aria-hidden="true"><IconExit size="large" /></div>
    <p className="eyebrow">退出登录</p>
    <h1 className="auth-title">退出 QoAuth？</h1>
    <p className="auth-lead">这会结束当前 QoAuth 会话并撤销其授权。其他应用可能保留各自的 Cookie，直到它们让你退出登录。</p>
    <form method="post" action="/end_session" className="logout-form">
      <input type="hidden" name="csrf" value={session.csrf} />
      {["id_token_hint", "post_logout_redirect_uri", "client_id", "state"].map(name => <input key={name} type="hidden" name={name} value={query.get(name) ?? ""} />)}
      <Button theme="solid" size="large" htmlType="submit" block>确认退出</Button>
      <a className="text-link" href="/account">保留当前会话</a>
    </form>
  </div>;
}

/** Show current sessions and password, MFA and consent controls. */
function AccountPage(props: Actions) {
  const { session } = props;
  const [sessions, setSessions] = useState<SessionRecord[] | null>(null);
  const [isSessionsFailed, setIsSessionsFailed] = useState(false);
  const [isEnrolling, setIsEnrolling] = useState(false);
  const isCompact = useMediaQuery(COMPACT_QUERY);
  const passwordForm = useForm({ password: "" });
  const consentForm = useForm({ client: "" });
  useEffect(() => {
    let isActive = true;
    void getJson<SessionRecord[]>("/api/sessions").then(value => { if (isActive) { setSessions(value); setIsSessionsFailed(false); } }).catch(() => { if (isActive) setIsSessionsFailed(true); });
    return () => { isActive = false; };
  }, [session.csrf]);
  /** Change the password; the server ends every QoAuth session afterwards. */
  function changePassword(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!passwordForm.check({ password: passwordRule(session.password_policy) })) return;
    const password = passwordForm.values.password;
    void props.run(async () => { try { await postJson("/api/password/change", { password }, session.csrf); window.location.assign("/login"); } finally { passwordForm.reset(); } });
  }
  /** Revoke consent by the application's registered client identifier. */
  function withdraw(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!consentForm.check({ client: required("请输入应用的客户端 ID。") })) return;
    const client = consentForm.values.client;
    void props.run(async () => { await postJson("/api/consent/withdraw", { client }, session.csrf); consentForm.reset(); Toast.success("已撤回该应用的授权，并撤销其现有授予。"); });
  }
  /** Revoke one session and reload the list; revoking this session signs out here. */
  function revoke(digest: string) {
    return props.run(async () => { await postJson("/api/sessions/revoke", { session: digest }, session.csrf); await props.refresh(); setSessions(await getJson<SessionRecord[]>("/api/sessions")); Toast.success("会话已撤销。"); });
  }
  if (isEnrolling && !session.has_factor) return <div className="console-narrow"><FactorForm {...props} onCancel={() => setIsEnrolling(false)} /></div>;
  const role = session.account.owner ? "实例所有者" : session.account.admin ? "管理员" : "成员";
  const revokeButton = (record: SessionRecord) => <Popconfirm title="撤销此会话？" content="该会话的 OAuth 访问令牌与刷新令牌将失效。撤销当前正在使用的会话会让你在此处退出登录。" okText="撤销" okType="danger" cancelText="取消" onConfirm={() => revoke(record.Digest)}>
    <Button type="danger" theme="borderless" disabled={props.busy}>撤销</Button>
  </Popconfirm>;
  const sessionStatus = (record: SessionRecord) => record.Assurance ? <Tag color="green" size="small">已验证</Tag> : <Tag color="orange" size="small">待完成 MFA</Tag>;
  return <div className="console-page">
    <PageHeader
      eyebrow="我的账户"
      title={session.account.username}
      description="你的登录方式、活动会话以及已授权的应用。"
      tags={<><Tag color="blue" size="large">{role}</Tag>{session.has_factor ? <Tag color="green" size="large" prefixIcon={<IconShield />}>已启用 MFA</Tag> : <Tag color="orange" size="large">仅密码</Tag>}</>}
    />
    <div className="fact-strip enter">
      <div><span>登录方式</span><strong>{session.has_factor ? "密码 + 验证器" : "密码"}</strong></div>
      <div><span>活动会话</span><strong>{sessions ? sessions.length : "—"}</strong></div>
      <div><span>角色</span><strong>{role}</strong></div>
    </div>
    {!session.has_factor ? <Banner className="enter" type="warning" title="添加验证器" description="即使密码泄露，验证器应用也能保护此账户。设置前需要在最近 5 分钟内登录过。" closeIcon={null}>
      <div className="banner-actions"><Button theme="solid" disabled={props.busy} onClick={() => setIsEnrolling(true)}>设置验证器 MFA</Button><a className="text-link" href="/login">先重新登录</a></div>
    </Banner> : null}
    <Panel id="sessions" title="活动会话" description="撤销会话后，其 OAuth 访问令牌和刷新令牌将停止工作。">
      {isSessionsFailed ? <StatusView illustration={<IllustrationFailure />} title="无法加载会话" description="会话列表加载失败。如果会话已过期，请重新登录。" />
        : !sessions ? <Skeleton active loading placeholder={<Skeleton.Paragraph rows={3} />} />
        : isCompact ? <List className="stack-list" dataSource={sessions} emptyContent={<Empty image={<IllustrationNoContent />} description="暂无活动会话。" />} renderItem={(record: SessionRecord) => <List.Item
            main={<div className="stack-item"><strong>登录于 {formatTime(record.AuthTime)}</strong><span>过期于 {formatTime(record.Expires)}</span>{sessionStatus(record)}</div>}
            extra={revokeButton(record)} />} />
        : <Table className="data-table" rowKey="Digest" pagination={false} dataSource={sessions} empty={<Empty image={<IllustrationNoContent />} description="暂无活动会话。" />} columns={[
            { title: "登录时间", dataIndex: "AuthTime", render: (value: number) => formatTime(value) },
            { title: "过期时间", dataIndex: "Expires", render: (value: number) => formatTime(value) },
            { title: "状态", dataIndex: "Assurance", render: (_value: boolean, record: SessionRecord) => sessionStatus(record) },
            { title: "", dataIndex: "Digest", align: "right", render: (_value: string, record: SessionRecord) => revokeButton(record) },
          ]} />}
    </Panel>
    <div className="panel-grid">
      <Panel id="password" title="修改密码" description="需要在最近 5 分钟内登录过。修改后所有现有 QoAuth 会话与授权都会结束，需要重新登录。">
        <form onSubmit={changePassword} noValidate>
          <PasswordField form={passwordForm} field="password" label="新密码" policy={session.password_policy} disabled={props.busy} />
          <div className="form-actions"><Button htmlType="submit" theme="solid" loading={props.busy}>修改密码</Button><a className="text-link" href="/login">重新登录以进行敏感操作</a></div>
        </form>
      </Panel>
      <Panel id="consent" title="应用授权" description="撤回某个应用的授权同意，并撤销它已持有的授予。">
        <form onSubmit={withdraw} noValidate>
          <TextField form={consentForm} field="client" label="应用客户端 ID" name="client" maxLength={64} placeholder="notes-web" autoComplete="off" disabled={props.busy} hint="客户端 ID 可向管理员获取，或在应用设置中查看。" />
          <div className="form-actions"><Button htmlType="submit" type="danger" loading={props.busy}>撤回授权并撤销授予</Button></div>
        </form>
      </Panel>
    </div>
  </div>;
}
