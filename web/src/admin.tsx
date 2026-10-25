/** Small user/application administration without organization or policy machinery. */
import React, { useCallback, useEffect, useState } from "react";
import { PasswordField } from "./password-field";
import Button from "@douyinfe/semi-ui/lib/es/button";
import { getJson, postJson, type Account, type Activity, type Application, type SessionState } from "./api";

interface AdminProps { session: SessionState; run: (task: () => Promise<void>) => Promise<void>; busy: boolean }
/** Parse explicitly entered scope/audience lists without inventing permissions. */
function list(value: FormDataEntryValue | null): string[] { return String(value ?? "").split(",").map(item => item.trim()).filter(Boolean); }

/** Manage local users, registered applications and their explicit assignments. */
export function AdminPanel({ session, run, busy }: AdminProps) {
  const [users, setUsers] = useState<Account[]>([]);
  const [apps, setApps] = useState<Application[]>([]);
  const [activity, setActivity] = useState<Activity[]>([]);
  const [message, setMessage] = useState("");
  const [secret, setSecret] = useState("");
  /** Fetch independent administration views together. */
  const reload = useCallback(async () => { const [nextUsers, nextApps, nextActivity] = await Promise.all([getJson<Account[]>("/api/users"), getJson<Application[]>("/api/applications"), getJson<Activity[]>("/api/activity")]); setUsers(nextUsers); setApps(nextApps); setActivity(nextActivity); }, []);
  useEffect(() => { void reload().catch(() => setMessage("Unable to load administration. Sign in again if your session expired.")); }, [reload]);
  /** Create an ordinary local user; owner privileges are not an input field. */
  function createUser(event: React.FormEvent<HTMLFormElement>) { event.preventDefault(); const form = event.currentTarget; const data = new FormData(form); void run(async () => { await postJson("/api/users/create", { username: data.get("username"), password: data.get("password") }, session.csrf); form.reset(); setMessage("User created. Assign an application to enable access."); await reload(); }); }
  /** Register reviewed callbacks/resources or explicitly replace a disabled client. */
  function createApplication(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault(); const form = event.currentTarget; const data = new FormData(form); const kind = data.get("kind"); const id = String(data.get("id")); const resource = kind === "resource";
    const application = { id, name: data.get("name"), resource, public: kind === "public", redirects: resource ? [] : list(data.get("redirects")), logout_redirects: resource ? [] : list(data.get("logout")), scopes: list(data.get("scopes")), audiences: resource ? [id] : list(data.get("audiences")) };
    const path = data.get("mode") === "replace" ? "/api/applications/replace-disabled" : "/api/applications/create";
    void run(async () => { const result = await postJson<{ client_secret: string }>(path, { application }, session.csrf); setSecret(result.client_secret); form.reset(); setMessage("Application registered with reviewed settings. Assign access explicitly; prior credentials and grants are not restored."); await reload(); });
  }
  /** Grant or remove the selected user's application assignment. */
  function assign(event: React.FormEvent<HTMLFormElement>) { event.preventDefault(); const data = new FormData(event.currentTarget); void run(async () => { await postJson("/api/assignments", { user: data.get("user"), client: data.get("client"), allow: data.get("allow") === "yes" }, session.csrf); setMessage("Assignment updated. Removed access revokes existing grants and codes."); await reload(); }); }
  return <><div className="page-heading"><div><div className="eyebrow">INSTALLATION</div><h1>Administration</h1><p className="muted">Local users and the applications they can access.</p></div><span className="badge">Administrator MFA verified</span></div>
    {!session.fresh ? <div className="notice">Sensitive changes require a fresh sign-in. <a href="/login">Sign in again</a></div> : null}
    {message ? <p role="status">{message}</p> : null}
    {secret ? <section className="notice"><h2>Copy the new client secret</h2><p>Store this in the application's server configuration. It is shown only now.</p><code className="secret">{secret}</code><Button onClick={() => setSecret("")}>I saved the secret</Button></section> : null}
    <div className="summary"><span><strong>{users.length}</strong> local users</span><span><strong>{apps.length}</strong> active applications</span><span><strong>{activity.length}</strong> recent events</span></div>
    <div className="grid">
      <section><h2>Users</h2><ul className="rows">{users.map(user => <li key={user.id}><span>{user.username}<small>{user.owner ? "Installation owner" : user.enabled ? "Active" : "Disabled"}</small></span>{!user.owner && user.enabled ? <Button type="danger" disabled={busy} onClick={() => void run(async () => { await postJson("/api/users/disable", { user: user.id }, session.csrf); setMessage("User disabled; existing authentication was revoked."); await reload(); })}>Disable</Button> : null}</li>)}</ul><details><summary>Create a local user</summary><form onSubmit={createUser}><label>Username<input name="username" required maxLength={64} pattern="[A-Za-z0-9][A-Za-z0-9_.-]*" autoComplete="off" /></label><PasswordField label="Initial password" policy={session.password_policy} /><Button htmlType="submit" theme="solid" disabled={busy}>Create user</Button></form></details></section>
      <section><h2>Applications</h2><ul className="rows">{apps.map(app => <li key={app.id}><span>{app.name}<small>{app.id} · {app.resource ? "Resource API" : app.public ? "Public client" : "Server-side client"}</small></span><Button disabled={busy} onClick={() => void run(async () => { await postJson("/api/applications/disable", { client: app.id }, session.csrf); setMessage("Application disabled and grants revoked."); await reload(); })}>Disable</Button></li>)}</ul><details><summary>Register an application</summary><form onSubmit={createApplication}>
        <label>Registration action<select name="mode"><option value="create">Create a new application</option><option value="replace">Replace a disabled application with reviewed settings</option></select></label><p className="hint">Replacement creates a new secret for confidential clients and clears old assignments. Keep the application type unchanged. Disable dependent applications before replacing a resource API, then register them again with reviewed scopes.</p>
        <label>Client ID<input name="id" required maxLength={64} /></label><label>Display name<input name="name" required maxLength={80} /></label><label>Application type<select name="kind"><option value="confidential">Server-side web client</option><option value="public">Public client with PKCE</option><option value="resource">Resource API validator</option></select></label>
        <label>Exact callback URLs<input name="redirects" placeholder="https://app.example.test/callback" /></label><label>Exact signed-out URLs<input name="logout" placeholder="https://app.example.test/signed-out" /></label><label>Allowed scopes, comma separated<input name="scopes" required placeholder="openid, profile, offline_access, notes.read" /></label><label>Resource API client IDs, comma separated<input name="audiences" placeholder="notes-api" /></label><p className="hint">Register resource APIs first. An API validator uses its own ID as its audience and has no browser callbacks.</p><Button htmlType="submit" theme="solid" disabled={busy}>Register application</Button>
      </form></details></section>
      <section><h2>Assign application access</h2><form onSubmit={assign}><label>User<select name="user" required><option value="">Choose a user</option>{users.filter(user => user.enabled).map(user => <option key={user.id} value={user.id}>{user.username}</option>)}</select></label><label>Application<select name="client" required><option value="">Choose an application</option>{apps.filter(app => !app.resource).map(app => <option key={app.id} value={app.id}>{app.name}</option>)}</select></label><label>Access<select name="allow"><option value="yes">Allow</option><option value="no">Remove and revoke</option></select></label><Button htmlType="submit" disabled={busy}>Save assignment</Button></form></section>
      <section><h2>Recent activity</h2><ol className="events">{activity.slice(0,15).map((event,index) => <li key={`${event.time}-${index}`}><span>{event.action}</span><small>{new Date(event.time*1000).toLocaleString()}</small></li>)}</ol><p className="hint">Activity records contain event names and account identifiers, never credentials.</p></section>
    </div></>;
}
