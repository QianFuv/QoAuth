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
    try { await task(); } catch (failure) { setError(failure instanceof Error ? failure.message : "Request failed."); }
    finally { setBusy(false); }
  }
  useEffect(() => {
    let isActive = true;
    void getJson<SessionState>("/api/session").then(value => { if (isActive) setSession(value); }).catch(() => { if (isActive) setIsUnavailable(true); });
    return () => { isActive = false; };
  }, []);
  useEffect(() => { setError(""); }, [location.path, location.hash]);
  if (isUnavailable) return <AuthLayout><StatusView illustration={<IllustrationFailure />} title="QoAuth is unavailable" description="The authentication service did not respond. Check that the server is running, then try again." action={<Button icon={<IconRefresh />} onClick={() => window.location.reload()}>Try again</Button>} /></AuthLayout>;
  if (!session) return <div className="boot" role="status"><Spin size="large" /><span>Connecting to QoAuth…</span></div>;
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
      ? session.account.admin ? <Suspense fallback={<div className="console-page" role="status" aria-label="Loading administration"><Spin size="large" /></div>}><AdminPanel {...props} section={location.hash} /></Suspense> : <StatusView illustration={<IllustrationNoAccess />} title="Administrator access required" description="This account can manage its own sign-in and sessions, but not the installation." action={<Button onClick={() => navigate("/account")}>Go to my account</Button>} />
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
    if (!form.check({ username: required("Enter your username."), password: required("Enter your password.") })) return;
    const { username, password } = form.values;
    void run(async () => { await postJson("/api/login", { username, password }, session.csrf); form.reset(); await refresh(); });
  }
  return <div className="auth-card">
    <p className="eyebrow">{applicationName ? "Single sign-on" : "Welcome back"}</p>
    <h1 className="auth-title">{applicationName ? <>Sign in to continue to <span className="accent-text">{applicationName}</span></> : "Sign in to QoAuth"}</h1>
    <p className="auth-lead">Use the local account your administrator created for you.</p>
    <form className="auth-form" onSubmit={submit} noValidate>
      <TextField form={form} field="username" label="Username" name="username" autoComplete="username" maxLength={64} autoFocus size="large" disabled={busy} />
      <TextField form={form} field="password" label="Password" name="password" mode="password" autoComplete="current-password" maxLength={1024} size="large" disabled={busy} />
      <Button htmlType="submit" theme="solid" size="large" block loading={busy} icon={<IconArrowRight />} iconPosition="right">Sign in</Button>
    </form>
    <p className="auth-hint">No public sign-up. A new installation's owner is created with the local <code>qoauth init</code> command.</p>
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
    const isCodeValid = form.check({ code: isRecovering ? required("Enter a recovery code.") : value => value === "" ? "Enter the six-digit code." : /^[0-9]{6}$/.test(value) ? "" : "Codes are exactly six digits." });
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
    <h1 className="auth-title">Save your recovery codes</h1>
    <p className="auth-lead">Each code works once and lets you replace a lost authenticator. Store them in your password manager before continuing — they are not shown again.</p>
    <CopyBlock value={codes.join("\n")} label="Recovery codes" isMultiline />
    <Button theme="solid" size="large" block icon={<IconTick />} disabled={busy} onClick={() => void run(async () => { setCodes([]); await refresh(); })}>I saved my recovery codes</Button>
  </div>;
  const title = isRecovering ? "Recover your authenticator" : session.has_factor ? "Verify your sign-in" : "Set up your authenticator";
  const lead = isRecovering ? "A recovery code lets you enroll a replacement authenticator. It does not bypass MFA." : session.has_factor ? "Enter the six-digit code from your authenticator app." : "Add a time-based one-time password before using administrator or recovered-account access.";
  return <div className="auth-card">
    {isEnrolling ? <EnrollmentSteps current={secret ? 1 : 0} /> : <div className="auth-icon" aria-hidden="true">{isRecovering ? <IconKey size="large" /> : <IconShield size="large" />}</div>}
    <p className="eyebrow">Account security</p>
    <h1 className="auth-title">{title}</h1>
    <p className="auth-lead">{lead}</p>
    {isEnrolling && !secret ? <Button theme="solid" size="large" block disabled={busy} loading={busy} onClick={() => void run(async () => { const result = await postJson<{ secret: string }>("/api/mfa/start", {}, session.csrf); setSecret(result.secret); })}>Create authenticator setup</Button> : null}
    {secret ? <div className="setup-key">
      <p>In your authenticator app, add a time-based account named <strong>QoAuth / {session.account.username}</strong> with this setup key:</p>
      <CopyBlock value={secret} label="Authenticator setup key" />
      <div className="setup-key-meta"><Tag size="small">6 digits</Tag><Tag size="small">30 seconds</Tag><Tag size="small">SHA-1</Tag></div>
    </div> : null}
    {session.has_factor || secret ? <form className="auth-form" onSubmit={verify} noValidate>
      {isRecovering
        ? <TextField key="recovery" form={form} field="code" label="Recovery code" name="code" autoComplete="one-time-code" maxLength={64} size="large" autoFocus disabled={busy} />
        : <TextField key="totp" form={form} field="code" label="Authenticator code" name="code" className="code-input" autoComplete="one-time-code" inputMode="numeric" maxLength={6} size="large" autoFocus placeholder="000000" disabled={busy} />}
      <Button htmlType="submit" theme="solid" size="large" block loading={busy}>{isRecovering ? "Recover and replace factor" : "Verify code"}</Button>
    </form> : null}
    <div className="auth-secondary">
      {session.has_factor ? <Button theme="borderless" disabled={busy} onClick={() => { form.reset(); setIsRecovering(value => !value); }}>{isRecovering ? "Use authenticator instead" : "Use a recovery code"}</Button> : null}
      {onCancel ? <Button theme="borderless" type="tertiary" disabled={busy} onClick={onCancel}>Back to my account</Button> : null}
    </div>
  </div>;
}

/** Show progress through the three real enrollment steps. */
function EnrollmentSteps({ current }: { current: number }) {
  return <Steps className="enroll-steps" type="basic" size="small" current={current} aria-label="Authenticator setup progress">
    <Steps.Step title="Create key" />
    <Steps.Step title="Verify code" />
    <Steps.Step title="Save codes" />
  </Steps>;
}

/** Plain-language descriptions for the standard scopes QoAuth implements. */
const SCOPE_DESCRIPTIONS: Record<string, string> = {
  openid: "Confirm your QoAuth identity to this application.",
  profile: "Read your basic profile, such as your username.",
  offline_access: "Stay signed in with refresh tokens until you revoke access.",
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
  if (isExpired) return <StatusView illustration={<IllustrationFailure />} title="This request expired" description="This authorization request is no longer valid. Return to the application and start signing in again." />;
  if (!interaction) return <div className="auth-card" role="status" aria-label="Loading application request"><Skeleton active placeholder={<><Skeleton.Title style={{ width: "60%", marginBottom: 16 }} /><Skeleton.Paragraph rows={4} /></>} loading /></div>;
  if (interaction.needs_login) return <LoginForm {...props} />;
  /** Let the server choose the validated callback after approval or denial. */
  async function decide(isApproved: boolean) { const result = await postJson<{ redirect: string }>("/api/consent", { request: props.requestId, approve: isApproved }, props.session.csrf); window.location.assign(result.redirect); }
  return <div className="auth-card consent-card">
    <div className="consent-apps" aria-hidden="true">
      <span className="consent-app-mark">{interaction.application.slice(0, 1).toUpperCase()}</span>
      <span className="consent-link" />
      <span className="consent-app-mark consent-app-mark-qoauth"><span /></span>
    </div>
    <p className="eyebrow">Application access</p>
    <h1 className="auth-title">Continue to <span className="accent-text">{interaction.application}</span></h1>
    <p className="auth-lead">Signed in as <strong>{props.session.account.username}</strong>. Review what this application is asking for.</p>
    <ul className="scope-list" aria-label="Requested permissions">
      {interaction.scopes.map(scope => <li key={scope}>
        <IconTick className="scope-check" aria-hidden="true" />
        <div><code>{scope}</code><span>{SCOPE_DESCRIPTIONS[scope] ?? "A permission defined by an application registered in this installation."}</span></div>
      </li>)}
    </ul>
    {interaction.audiences.length ? <div className="consent-resources"><span>Resource APIs</span>{interaction.audiences.map(audience => <Tag key={audience} size="small" color="blue">{audience}</Tag>)}</div> : null}
    <div className="consent-actions">
      <Button theme="solid" size="large" disabled={props.busy} loading={props.busy} onClick={() => void props.run(() => decide(true))}>Allow and continue</Button>
      <Button size="large" type="tertiary" disabled={props.busy} onClick={() => void props.run(() => decide(false))}>Deny</Button>
    </div>
    <p className="auth-hint">You can withdraw this consent later from My account.</p>
  </div>;
}

/** Confirm standard RP logout through a native form with session-bound CSRF. */
function LogoutConfirmation({ session }: { session: SessionState }) {
  const query = new URLSearchParams(window.location.search);
  return <div className="auth-card">
    <div className="auth-icon" aria-hidden="true"><IconExit size="large" /></div>
    <p className="eyebrow">Sign out</p>
    <h1 className="auth-title">Sign out of QoAuth?</h1>
    <p className="auth-lead">This ends this QoAuth session and revokes its grants. Other applications may keep their own cookies until they sign you out.</p>
    <form method="post" action="/end_session" className="logout-form">
      <input type="hidden" name="csrf" value={session.csrf} />
      {["id_token_hint", "post_logout_redirect_uri", "client_id", "state"].map(name => <input key={name} type="hidden" name={name} value={query.get(name) ?? ""} />)}
      <Button theme="solid" size="large" htmlType="submit" block>Confirm sign out</Button>
      <a className="text-link" href="/account">Keep this session</a>
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
    if (!consentForm.check({ client: required("Enter the application's client ID.") })) return;
    const client = consentForm.values.client;
    void props.run(async () => { await postJson("/api/consent/withdraw", { client }, session.csrf); consentForm.reset(); Toast.success("Application consent and existing grants revoked."); });
  }
  /** Revoke one session and reload the list; revoking this session signs out here. */
  function revoke(digest: string) {
    return props.run(async () => { await postJson("/api/sessions/revoke", { session: digest }, session.csrf); await props.refresh(); setSessions(await getJson<SessionRecord[]>("/api/sessions")); Toast.success("Session revoked."); });
  }
  if (isEnrolling && !session.has_factor) return <div className="console-narrow"><FactorForm {...props} onCancel={() => setIsEnrolling(false)} /></div>;
  const role = session.account.owner ? "Installation owner" : session.account.admin ? "Administrator" : "Member";
  const revokeButton = (record: SessionRecord) => <Popconfirm title="Revoke this session?" content="Its OAuth access and refresh tokens stop working. Revoking the session you are using signs you out here." okText="Revoke" okType="danger" cancelText="Cancel" onConfirm={() => revoke(record.Digest)}>
    <Button type="danger" theme="borderless" disabled={props.busy}>Revoke</Button>
  </Popconfirm>;
  const sessionStatus = (record: SessionRecord) => record.Assurance ? <Tag color="green" size="small">Verified</Tag> : <Tag color="orange" size="small">Awaiting MFA</Tag>;
  return <div className="console-page">
    <PageHeader
      eyebrow="My account"
      title={session.account.username}
      description="Your sign-in methods, active sessions and the applications you have approved."
      tags={<><Tag color="blue" size="large">{role}</Tag>{session.has_factor ? <Tag color="green" size="large" prefixIcon={<IconShield />}>MFA protected</Tag> : <Tag color="orange" size="large">Password only</Tag>}</>}
    />
    <div className="fact-strip enter">
      <div><span>Sign-in</span><strong>{session.has_factor ? "Password + authenticator" : "Password"}</strong></div>
      <div><span>Active sessions</span><strong>{sessions ? sessions.length : "—"}</strong></div>
      <div><span>Role</span><strong>{role}</strong></div>
    </div>
    {!session.has_factor ? <Banner className="enter" type="warning" title="Add an authenticator" description="An authenticator app protects this account if your password leaks. Setup needs a sign-in within the last five minutes." closeIcon={null}>
      <div className="banner-actions"><Button theme="solid" disabled={props.busy} onClick={() => setIsEnrolling(true)}>Set up authenticator MFA</Button><a className="text-link" href="/login">Sign in again first</a></div>
    </Banner> : null}
    <Panel id="sessions" title="Active sessions" description="Revoke a session to stop its OAuth access and refresh tokens.">
      {isSessionsFailed ? <StatusView illustration={<IllustrationFailure />} title="Sessions unavailable" description="Unable to load sessions. Sign in again if your session expired." />
        : !sessions ? <Skeleton active loading placeholder={<Skeleton.Paragraph rows={3} />} />
        : isCompact ? <List className="stack-list" dataSource={sessions} emptyContent={<Empty image={<IllustrationNoContent />} description="No active sessions." />} renderItem={(record: SessionRecord) => <List.Item
            main={<div className="stack-item"><strong>Signed in {formatTime(record.AuthTime)}</strong><span>Expires {formatTime(record.Expires)}</span>{sessionStatus(record)}</div>}
            extra={revokeButton(record)} />} />
        : <Table className="data-table" rowKey="Digest" pagination={false} dataSource={sessions} empty={<Empty image={<IllustrationNoContent />} description="No active sessions." />} columns={[
            { title: "Signed in", dataIndex: "AuthTime", render: (value: number) => formatTime(value) },
            { title: "Expires", dataIndex: "Expires", render: (value: number) => formatTime(value) },
            { title: "Status", dataIndex: "Assurance", render: (_value: boolean, record: SessionRecord) => sessionStatus(record) },
            { title: "", dataIndex: "Digest", align: "right", render: (_value: string, record: SessionRecord) => revokeButton(record) },
          ]} />}
    </Panel>
    <div className="panel-grid">
      <Panel id="password" title="Change password" description="Requires a sign-in within the last five minutes. All existing QoAuth sessions and grants end, and you sign in again.">
        <form onSubmit={changePassword} noValidate>
          <PasswordField form={passwordForm} field="password" label="New password" policy={session.password_policy} disabled={props.busy} />
          <div className="form-actions"><Button htmlType="submit" theme="solid" loading={props.busy}>Change password</Button><a className="text-link" href="/login">Sign in again for sensitive changes</a></div>
        </form>
      </Panel>
      <Panel id="consent" title="Application access" description="Withdraw an application's consent and revoke the grants it already holds.">
        <form onSubmit={withdraw} noValidate>
          <TextField form={consentForm} field="client" label="Application client ID" name="client" maxLength={64} placeholder="notes-web" autoComplete="off" disabled={props.busy} hint="The client ID is shown by your administrator or in the application's settings." />
          <div className="form-actions"><Button htmlType="submit" type="danger" loading={props.busy}>Revoke consent and grants</Button></div>
        </form>
      </Panel>
    </div>
  </div>;
}
