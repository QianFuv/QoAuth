/** Minimal login, factor, consent and self-service pages. */
import React, { useEffect, useState } from "react";
import Button from "@douyinfe/semi-ui/lib/es/button";
import { getJson, postJson, type SessionState, type SessionRecord } from "./api";
import { AdminPanel } from "./admin";

type Task = () => Promise<void>;
interface Actions { session: SessionState; refresh: () => Promise<void>; run: (task: Task) => Promise<void>; busy: boolean }

/** Render the application with a single explicit error and mutation boundary. */
export function Portal() {
  const [session, setSession] = useState<SessionState | null>(null);
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
    try { await task(); } catch (failure) { setError(failure instanceof Error ? failure.message : "Request failed"); }
    finally { setBusy(false); }
  }
  useEffect(() => { let active = true; void getJson<SessionState>("/api/session").then(value => { if (active) setSession(value); }).catch(() => { if (active) setError("Unable to load the authentication service."); }); return () => { active = false; }; }, []);
  const requestId = new URLSearchParams(window.location.search).get("request");
  return <>
    <header className="topbar"><a className="brand" href="/">Q<span>OAuth</span></a><span className="tagline">Your apps. One identity.</span>{session?.authenticated ? <span className="identity">{session.account.username}</span> : null}</header>
    <main>
      {session?.assurance ? <nav aria-label="Main navigation"><a href="/account">My account</a>{session.account.admin ? <a href="/admin">Administration</a> : null}<Button disabled={busy} onClick={() => void run(async () => { await postJson("/api/logout", {}, session.csrf); window.location.assign("/"); })}>Sign out</Button></nav> : null}
      {error ? <div className="notice error" role="alert">{error}</div> : null}
      {!session ? <p role="status">Connecting to QoAuth…</p> : <PortalContent session={session} refresh={refresh} run={run} busy={busy} requestId={requestId} />}
      <footer>Self-hosted identity · QoAuth</footer>
    </main>
  </>;
}

/** Keep login and privileged views separate while relying on server authorization. */
function PortalContent(props: Actions & { requestId: string | null }) {
  const { session, requestId } = props;
  if (window.location.pathname === "/end_session") return <LogoutConfirmation session={session} />;
  if (!session.authenticated) return <LoginForm {...props} />;
  if (!session.assurance) return <FactorForm {...props} key={session.csrf} />;
  if (requestId) return <ConsentPage {...props} requestId={requestId} key={session.csrf} />;
  if (window.location.pathname === "/login") return <LoginForm {...props} />;
  if (window.location.pathname === "/admin") return session.account.admin ? <AdminPanel {...props} /> : <section><h1>Administrator access required</h1></section>;
  return <AccountPage {...props} />;
}

/** Submit a password only to the same-origin login endpoint. */
function LoginForm({ session, refresh, run, busy }: Actions) {
  /** Capture form values before awaiting, then clear password fields. */
  function submit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault(); const form = event.currentTarget; const data = new FormData(form);
    void run(async () => { await postJson("/api/login", { username: data.get("username"), password: data.get("password") }, session.csrf); form.reset(); await refresh(); });
  }
  return <section className="auth-card"><div className="eyebrow">WELCOME BACK</div><h1>Sign in to QoAuth</h1><p className="muted">Use your local account to continue to your applications.</p><form onSubmit={submit}>
    <label>Username<input name="username" autoComplete="username" required maxLength={64} autoFocus /></label>
    <label>Password<input name="password" type="password" autoComplete="current-password" required maxLength={1024} /></label>
    <Button htmlType="submit" theme="solid" block loading={busy}>Sign in</Button>
  </form><p className="hint">New installation? The owner is created with the local initialization command.</p></section>;
}

/** Enroll or prove a factor; recovery always returns to restricted enrollment. */
function FactorForm({ session, refresh, run, busy }: Actions) {
  const [secret, setSecret] = useState("");
  const [codes, setCodes] = useState<string[]>([]);
  const [recovering, setRecovering] = useState(false);
  /** Verify the selected factor proof without retaining the input. */
  function verify(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault(); const form = event.currentTarget; const code = new FormData(form).get("code");
    const path = recovering ? "/api/mfa/recover" : session.has_factor ? "/api/mfa/verify" : "/api/mfa/confirm";
    void run(async () => { const result = await postJson<{ recovery_codes?: string[] }>(path, { code }, session.csrf); form.reset(); if (result.recovery_codes) { setSecret(""); setCodes(result.recovery_codes); } else await refresh(); });
  }
  if (codes.length) return <section className="auth-card"><h1>Save your recovery codes</h1><p>Each code can be used once. Store them in your password manager before continuing.</p><pre className="secret" aria-label="Recovery codes">{codes.join("\n")}</pre><Button theme="solid" onClick={() => void run(async () => { setCodes([]); await refresh(); })}>I saved my recovery codes</Button></section>;
  return <section className="auth-card"><div className="eyebrow">ACCOUNT SECURITY</div><h1>{recovering ? "Recover your authenticator" : session.has_factor ? "Verify your sign-in" : "Set up your authenticator"}</h1>
    <p className="muted">{recovering ? "A recovery code allows replacement enrollment. It does not bypass MFA." : session.has_factor ? "Enter the six-digit code from your authenticator." : "Complete this step before using administrator or recovered-account access."}</p>
    {!session.has_factor && !secret ? <Button theme="solid" disabled={busy} onClick={() => void run(async () => { const result = await postJson<{ secret: string }>("/api/mfa/start", {}, session.csrf); setSecret(result.secret); })}>Create authenticator setup</Button> : null}
    {secret ? <><p>Add a time-based account named <strong>QoAuth / {session.account.username}</strong> with this setup key:</p><code className="secret">{secret}</code><p className="hint">Six digits · 30 seconds · SHA-1</p></> : null}
    {session.has_factor || secret ? <form onSubmit={verify}><label>{recovering ? "Recovery code" : "Authenticator code"}<input name="code" required autoComplete="one-time-code" inputMode={recovering ? "text" : "numeric"} pattern={recovering ? undefined : "[0-9]{6}"} maxLength={recovering ? 64 : 6} /></label><Button htmlType="submit" theme="solid" loading={busy}>{recovering ? "Recover and replace factor" : "Verify code"}</Button></form> : null}
    {session.has_factor ? <Button type="tertiary" disabled={busy} onClick={() => setRecovering(value => !value)}>{recovering ? "Use authenticator instead" : "Use a recovery code"}</Button> : null}
  </section>;
}

interface Interaction { application: string; scopes: string[]; audiences: string[]; needs_login: boolean; needs_consent: boolean }
/** Present the server's immutable requested permissions and explicit consent. */
function ConsentPage(props: Actions & { requestId: string }) {
  const [interaction, setInteraction] = useState<Interaction | null>(null);
  const [error, setError] = useState("");
  useEffect(() => { let active = true; void getJson<Interaction>("/api/interaction?request=" + encodeURIComponent(props.requestId)).then(value => { if (active) setInteraction(value); }).catch(() => { if (active) setError("This authorization request expired. Return to the application and try again."); }); return () => { active = false; }; }, [props.requestId, props.session.csrf]);
  if (error) return <p role="alert">{error}</p>;
  if (!interaction) return <p role="status">Loading application request…</p>;
  if (interaction.needs_login) return <LoginForm {...props} />;
  /** Let the server choose the validated callback after approval or denial. */
  async function decide(approve: boolean) { const result = await postJson<{ redirect: string }>("/api/consent", { request: props.requestId, approve }, props.session.csrf); window.location.assign(result.redirect); }
  return <section className="auth-card"><div className="eyebrow">APPLICATION ACCESS</div><h1>Continue to {interaction.application}</h1><p>Signed in as <strong>{props.session.account.username}</strong></p><h2>Requested permissions</h2><ul>{interaction.scopes.map(scope => <li key={scope}><code>{scope}</code></li>)}</ul>{interaction.audiences.length ? <p className="muted">Resource: {interaction.audiences.join(", ")}</p> : null}<div className="actions"><Button theme="solid" disabled={props.busy} onClick={() => void props.run(() => decide(true))}>Allow and continue</Button><Button disabled={props.busy} onClick={() => void props.run(() => decide(false))}>Deny</Button></div></section>;
}

/** Confirm standard RP logout through a native form with session-bound CSRF. */
function LogoutConfirmation({ session }: { session: SessionState }) {
  const query = new URLSearchParams(window.location.search);
  return <section className="auth-card"><h1>Sign out of QoAuth?</h1><p>This ends this QoAuth session and revokes its grants. Other applications may retain their own cookies.</p><form method="post" action="/end_session"><input type="hidden" name="csrf" value={session.csrf} />{["id_token_hint", "post_logout_redirect_uri", "client_id", "state"].map(name => <input key={name} type="hidden" name={name} value={query.get(name) ?? ""} />)}<Button theme="solid" htmlType="submit">Confirm sign out</Button><a href="/account">Keep this session</a></form></section>;
}

/** Show current sessions and minimal password/consent controls. */
function AccountPage(props: Actions) {
  const [sessions, setSessions] = useState<SessionRecord[]>([]);
  const [message, setMessage] = useState("");
  const [isEnrolling, setIsEnrolling] = useState(false);
  useEffect(() => { void getJson<SessionRecord[]>("/api/sessions").then(setSessions).catch(() => setMessage("Unable to load sessions.")); }, [props.session.csrf]);
  /** Change the password only after server-enforced fresh authentication. */
  function changePassword(event: React.FormEvent<HTMLFormElement>) { event.preventDefault(); const form = event.currentTarget; const password = new FormData(form).get("password"); void props.run(async () => { await postJson("/api/password/change", { password }, props.session.csrf); form.reset(); window.location.assign("/"); }); }
  /** Revoke consent by the application's registered client identifier. */
  function withdraw(event: React.FormEvent<HTMLFormElement>) { event.preventDefault(); const form = event.currentTarget; const client = new FormData(form).get("client"); void props.run(async () => { await postJson("/api/consent/withdraw", { client }, props.session.csrf); setMessage("Application consent and existing grants revoked."); form.reset(); }); }
  if (isEnrolling && !props.session.has_factor) return <FactorForm {...props} />;
  return <><div className="page-heading"><div><div className="eyebrow">MY ACCOUNT</div><h1>{props.session.account.username}</h1></div><span className="badge">{props.session.has_factor ? "MFA protected" : "Password sign-in"}</span></div>{!props.session.has_factor ? <Button disabled={props.busy} onClick={() => setIsEnrolling(true)}>Set up authenticator MFA</Button> : null}{message ? <p role="status">{message}</p> : null}<div className="grid">
    <section><h2>Active sessions</h2><p className="muted">Revoke a session to stop its OAuth access and refresh tokens.</p><ul className="rows">{sessions.map(session => <li key={session.Digest}><span>{new Date(session.AuthTime * 1000).toLocaleString()}<small>{session.Assurance ? "Verified" : "Awaiting MFA"}</small></span><Button disabled={props.busy} onClick={() => void props.run(async () => { await postJson("/api/sessions/revoke", { session: session.Digest }, props.session.csrf); await props.refresh(); setSessions(await getJson<SessionRecord[]>("/api/sessions")); })}>Revoke</Button></li>)}</ul></section>
    <section><h2>Change password</h2><p className="muted">Requires a login within the last five minutes. All existing QoAuth sessions and grants will end.</p><form onSubmit={changePassword}><label>New password<input type="password" name="password" minLength={15} maxLength={1024} required autoComplete="new-password" /></label><Button htmlType="submit" disabled={props.busy}>Change password</Button></form><a href="/login">Sign in again for sensitive changes</a></section>
    <section><h2>Application access</h2><form onSubmit={withdraw}><label>Application client ID<input name="client" required maxLength={64} /></label><Button htmlType="submit" disabled={props.busy}>Revoke consent and grants</Button></form></section>
  </div></>;
}
