/** Small user/application administration without organization or policy machinery. */
import React, { useCallback, useEffect, useState } from "react";
import Button from "@douyinfe/semi-ui/lib/es/button";
import Banner from "@douyinfe/semi-ui/lib/es/banner";
import Select from "@douyinfe/semi-ui/lib/es/select";
import Radio from "@douyinfe/semi-ui/lib/es/radio";
import RadioGroup from "@douyinfe/semi-ui/lib/es/radio/radioGroup";
import Tag from "@douyinfe/semi-ui/lib/es/tag";
import Table from "@douyinfe/semi-ui/lib/es/table";
import List from "@douyinfe/semi-ui/lib/es/list";
import SideSheet from "@douyinfe/semi-ui/lib/es/sideSheet";
import Modal from "@douyinfe/semi-ui/lib/es/modal";
import Popconfirm from "@douyinfe/semi-ui/lib/es/popconfirm";
import Skeleton from "@douyinfe/semi-ui/lib/es/skeleton";
import Empty from "@douyinfe/semi-ui/lib/es/empty";
import Timeline from "@douyinfe/semi-ui/lib/es/timeline";
import Collapsible from "@douyinfe/semi-ui/lib/es/collapsible";
import Toast from "@douyinfe/semi-ui/lib/es/toast";
import { IllustrationFailure, IllustrationNoContent } from "@douyinfe/semi-illustrations";
import { IconApps, IconChevronDown, IconChevronUp, IconHistory, IconKey, IconLock, IconPlus, IconRefresh, IconShield, IconUserGroup } from "@douyinfe/semi-icons";
import { getJson, postJson, type Account, type Activity, type Application, type SessionState } from "./api";
import { PasswordField, passwordRule } from "./password-field";
import { Field, TextField, required, useForm, type FormState } from "./form";
import { COMPACT_QUERY, formatTime, navigate, useMediaQuery } from "./layout";
import { CopyBlock, PageHeader, Panel, StatusView } from "./ui";
import { splitCommaList, splitLines } from "./validation";

interface AdminProps { session: SessionState; run: (task: () => Promise<void>) => Promise<void>; busy: boolean; section: string }

/**
 * Describe an application's registered type.
 *
 * @param application - A registered application.
 * @returns The human-readable application type.
 */
function applicationType(application: Application): string {
  return application.resource ? "Resource API" : application.public ? "Public client (PKCE)" : "Server-side client";
}

/**
 * Manage local users, registered applications and their explicit assignments.
 *
 * @param props - Session, mutation boundary and the selected console section.
 * @returns The administration page for the selected section.
 */
export function AdminPanel({ session, run, busy, section }: AdminProps) {
  const [users, setUsers] = useState<Account[] | null>(null);
  const [apps, setApps] = useState<Application[] | null>(null);
  const [activity, setActivity] = useState<Activity[] | null>(null);
  const [isLoadFailed, setIsLoadFailed] = useState(false);
  const [secret, setSecret] = useState("");
  const [isUserSheetOpen, setIsUserSheetOpen] = useState(false);
  const [resetTarget, setResetTarget] = useState<string | null>(null);
  const [applicationSheet, setApplicationSheet] = useState<{ review: Application | null } | null>(null);
  /** Fetch independent administration views together. */
  const reload = useCallback(async () => {
    const [nextUsers, nextApps, nextActivity] = await Promise.all([getJson<Account[]>("/api/users"), getJson<Application[]>("/api/applications"), getJson<Activity[]>("/api/activity")]);
    setUsers(nextUsers); setApps(nextApps); setActivity(nextActivity); setIsLoadFailed(false);
  }, []);
  useEffect(() => { void reload().catch(() => setIsLoadFailed(true)); }, [reload]);
  /** Disable a non-owner user; the server revokes their existing authentication. */
  function disableUser(user: Account) {
    return run(async () => { await postJson("/api/users/disable", { user: user.id }, session.csrf); Toast.success("User disabled; existing authentication was revoked."); await reload(); });
  }
  /** Disable an application; the server revokes its grants. */
  function disableApplication(application: Application) {
    return run(async () => { await postJson("/api/applications/disable", { client: application.id }, session.csrf); Toast.success("Application disabled and grants revoked."); await reload(); });
  }
  const sharedSheets = <>
    <UserSheet session={session} run={run} busy={busy} isOpen={isUserSheetOpen} onClose={() => setIsUserSheetOpen(false)} onDone={reload} />
    <ResetSheet session={session} run={run} busy={busy} users={users ?? []} target={resetTarget} onClose={() => setResetTarget(null)} onDone={reload} />
    <ApplicationSheet session={session} run={run} busy={busy} state={applicationSheet} onClose={() => setApplicationSheet(null)} onRegistered={async clientSecret => { setSecret(clientSecret); await reload(); }} />
    <Modal
      title="Copy the new client secret"
      visible={secret !== ""}
      closable={false}
      maskClosable={false}
      closeOnEsc={false}
      footer={<Button theme="solid" onClick={() => setSecret("")}>I saved the secret</Button>}
      width="min(520px, calc(100vw - 32px))"
    >
      <p className="modal-text">Store this in the application's server configuration. QoAuth keeps only a hash and shows the secret this one time.</p>
      <CopyBlock value={secret} label="Client secret" />
    </Modal>
  </>;
  const freshNotice = !session.fresh ? <Banner className="enter" type="warning" title="Fresh sign-in required for changes" description="Administrative changes need a sign-in with MFA within the last five minutes." closeIcon={null}>
    <div className="banner-actions"><Button theme="solid" onClick={() => window.location.assign("/login")}>Sign in again</Button></div>
  </Banner> : null;
  if (isLoadFailed) return <div className="console-page"><StatusView illustration={<IllustrationFailure />} title="Administration unavailable" description="Unable to load administration data. Sign in again if your session expired." action={<Button icon={<IconRefresh />} onClick={() => void reload().catch(() => setIsLoadFailed(true))}>Try again</Button>} /></div>;
  if (!users || !apps || !activity) return <div className="console-page" role="status" aria-label="Loading administration"><Skeleton active loading placeholder={<><Skeleton.Title style={{ width: 240, height: 32, marginBottom: 24 }} /><Skeleton.Paragraph rows={8} /></>} /></div>;
  const openRegistration = () => setApplicationSheet({ review: null });
  if (section === "users") return <div className="console-page">
    <PageHeader eyebrow="Administration" title="Users" description="Local accounts in this installation. New users need an application assignment before they can sign in to it." actions={<Button theme="solid" icon={<IconPlus />} onClick={() => setIsUserSheetOpen(true)}>Create user</Button>} />
    {freshNotice}
    <UsersTable users={users} busy={busy} isFresh={session.fresh} onDisable={disableUser} onReset={user => setResetTarget(user.id)} />
    {sharedSheets}
  </div>;
  if (section === "applications") return <div className="console-page">
    <PageHeader eyebrow="Administration" title="Applications" description="OIDC clients and resource APIs with exact callbacks and explicit scopes." actions={<Button theme="solid" icon={<IconPlus />} onClick={openRegistration}>Register application</Button>} />
    {freshNotice}
    <ApplicationsTable apps={apps} busy={busy} onDisable={disableApplication} onReview={application => { setSecret(""); setApplicationSheet({ review: application }); }} />
    {sharedSheets}
  </div>;
  if (section === "access") return <div className="console-page">
    <PageHeader eyebrow="Administration" title="Access" description="Assign or remove one user's access to one application." />
    {freshNotice}
    <AssignmentPanel session={session} run={run} busy={busy} users={users} apps={apps} onDone={reload} onRegister={() => navigate("/admin#applications")} />
    {sharedSheets}
  </div>;
  if (section === "activity") return <div className="console-page">
    <PageHeader eyebrow="Administration" title="Activity" description="The latest security events recorded by this installation." />
    <ActivityTable activity={activity} />
    {sharedSheets}
  </div>;
  const activeApps = apps.filter(app => app.enabled);
  return <div className="console-page">
    <PageHeader
      eyebrow="Administration"
      title="Installation overview"
      description="Local users and the applications they can access."
      tags={<Tag color="green" size="large" prefixIcon={<IconShield />}>Administrator MFA verified</Tag>}
      actions={<><Button icon={<IconPlus />} onClick={() => setIsUserSheetOpen(true)}>Create user</Button><Button theme="solid" icon={<IconPlus />} onClick={openRegistration}>Register application</Button></>}
    />
    {freshNotice}
    <div className="stat-row enter">
      <button type="button" className="stat" onClick={() => navigate("/admin#users")}><IconUserGroup aria-hidden="true" /><strong>{users.length}</strong><span>Local users</span><small>{users.filter(user => !user.enabled).length} disabled</small></button>
      <button type="button" className="stat" onClick={() => navigate("/admin#applications")}><IconApps aria-hidden="true" /><strong>{activeApps.length}</strong><span>Active applications</span><small>{apps.length - activeApps.length} disabled</small></button>
      <button type="button" className="stat" onClick={() => navigate("/admin#activity")}><IconHistory aria-hidden="true" /><strong>{activity.length}</strong><span>Recorded events</span><small>Latest {activity.length >= 100 ? "100" : activity.length} shown</small></button>
    </div>
    <div className="panel-grid">
      <Panel title="Recent activity" description="Event names and account identifiers only — never credentials." toolbar={<Button theme="borderless" onClick={() => navigate("/admin#activity")}>View all</Button>}>
        {activity.length ? <Timeline className="activity-timeline">{activity.slice(0, 6).map((event, index) => <Timeline.Item key={`${event.time}-${index}`} time={formatTime(event.time)}><span className="activity-action">{event.action}</span> <code className="activity-subject">{event.subject}</code></Timeline.Item>)}</Timeline> : <Empty image={<IllustrationNoContent />} description="No activity recorded yet." />}
      </Panel>
      <Panel title="Applications" description="Registered clients and resource APIs." toolbar={<Button theme="borderless" onClick={() => navigate("/admin#applications")}>Manage</Button>}>
        {apps.length ? <ul className="mini-list">{apps.slice(0, 6).map(app => <li key={app.id}><div><strong>{app.name}</strong><code>{app.id}</code></div>{app.enabled ? <Tag color="green" size="small">Active</Tag> : <Tag color="grey" size="small">Disabled</Tag>}</li>)}</ul>
          : <Empty image={<IllustrationNoContent />} description="No applications yet. Register a resource API first if clients will call one."><Button theme="solid" icon={<IconPlus />} onClick={openRegistration}>Register application</Button></Empty>}
      </Panel>
    </div>
    {sharedSheets}
  </div>;
}

/** Users table on wide screens and a stacked list on small screens. */
function UsersTable({ users, busy, isFresh, onDisable, onReset }: { users: Account[]; busy: boolean; isFresh: boolean; onDisable: (user: Account) => Promise<void>; onReset: (user: Account) => void }) {
  const isCompact = useMediaQuery(COMPACT_QUERY);
  const status = (user: Account) => user.owner ? <Tag color="blue" size="small">Owner</Tag> : user.enabled ? <Tag color="green" size="small">Active</Tag> : <Tag color="grey" size="small">Disabled</Tag>;
  const role = (user: Account) => <span className="role-cell">{user.admin ? "Administrator" : "Member"}{user.require_mfa || user.admin ? <Tag size="small" prefixIcon={<IconLock />}>MFA required</Tag> : null}</span>;
  const actions = (user: Account) => user.owner ? <span className="muted-cell">Recover with local command</span> : <div className="row-actions">
    <Button theme="borderless" disabled={busy || !isFresh} onClick={() => onReset(user)}>Reset password</Button>
    {user.enabled ? <Popconfirm title={`Disable ${user.username}?`} content="Existing sessions, codes and grants are revoked. The account can be restored only with the local recover-user command." okText="Disable" okType="danger" cancelText="Cancel" onConfirm={() => onDisable(user)}>
      <Button type="danger" theme="borderless" disabled={busy}>Disable</Button>
    </Popconfirm> : null}
  </div>;
  const empty = <Empty image={<IllustrationNoContent />} description="No users yet." />;
  return <Panel title="All users" description={`${users.length} local ${users.length === 1 ? "account" : "accounts"}.`}>
    {isCompact ? <List className="stack-list" dataSource={users} emptyContent={empty} renderItem={(user: Account) => <List.Item main={<div className="stack-item"><strong className="wrap-anywhere">{user.username}</strong>{role(user)}<div className="stack-tags">{status(user)}</div>{actions(user)}</div>} />} />
      : <Table className="data-table" rowKey="id" pagination={users.length > 20 ? { pageSize: 20 } : false} dataSource={users} empty={empty} columns={[
          { title: "User", dataIndex: "username", render: (_value: string, user: Account) => <div className="identity-cell"><strong>{user.username}</strong><code>{user.id}</code></div> },
          { title: "Role", dataIndex: "admin", render: (_value: boolean, user: Account) => role(user) },
          { title: "Status", dataIndex: "enabled", width: 120, render: (_value: boolean, user: Account) => status(user) },
          { title: "", dataIndex: "id", align: "right", render: (_value: string, user: Account) => actions(user) },
        ]} />}
  </Panel>;
}

/** Wrap a list of exact values, or show a dash when none are registered. */
function ValueList({ values }: { values: string[] }) {
  if (!values.length) return <span className="muted-cell">None</span>;
  return <ul className="value-list">{values.map(value => <li key={value}><code>{value}</code></li>)}</ul>;
}

/** Full registered configuration for one application. */
function ApplicationDetails({ application }: { application: Application }) {
  return <dl className="detail-grid">
    {!application.resource ? <><dt>Callback URLs</dt><dd><ValueList values={application.redirects ?? []} /></dd><dt>Signed-out URLs</dt><dd><ValueList values={application.logout_redirects ?? []} /></dd></> : null}
    <dt>Scopes</dt><dd><ValueList values={application.scopes ?? []} /></dd>
    <dt>{application.resource ? "Audience" : "Resource APIs"}</dt><dd><ValueList values={application.audiences ?? []} /></dd>
  </dl>;
}

/** Applications table with expandable configuration, or stacked cards on small screens. */
function ApplicationsTable({ apps, busy, onDisable, onReview }: { apps: Application[]; busy: boolean; onDisable: (application: Application) => Promise<void>; onReview: (application: Application) => void }) {
  const isCompact = useMediaQuery(COMPACT_QUERY);
  const [openId, setOpenId] = useState("");
  const status = (application: Application) => application.enabled ? <Tag color="green" size="small">Active</Tag> : <Tag color="grey" size="small">Disabled</Tag>;
  const action = (application: Application) => application.enabled
    ? <Popconfirm title={`Disable ${application.name}?`} content="Existing grants are revoked. Re-enabling requires reviewing and replacing its settings." okText="Disable" okType="danger" cancelText="Cancel" onConfirm={() => onDisable(application)}><Button type="danger" theme="borderless" disabled={busy}>Disable</Button></Popconfirm>
    : <Button theme="borderless" disabled={busy} onClick={() => onReview(application)}>Review settings</Button>;
  const empty = <Empty image={<IllustrationNoContent />} description="No applications registered yet." />;
  return <Panel title="Registered applications" description="Expand an application to see its exact callbacks, scopes and resource APIs.">
    {isCompact ? <List className="stack-list" dataSource={apps} emptyContent={empty} renderItem={(application: Application) => {
      const isOpen = openId === application.id;
      return <List.Item main={<div className="stack-item">
        <strong className="wrap-anywhere">{application.name}</strong>
        <code className="wrap-anywhere">{application.id}</code>
        <div className="stack-tags">{status(application)}<Tag size="small">{applicationType(application)}</Tag></div>
        <div className="row-actions">
          <Button theme="borderless" type="tertiary" icon={isOpen ? <IconChevronUp /> : <IconChevronDown />} aria-expanded={isOpen} onClick={() => setOpenId(isOpen ? "" : application.id)}>{isOpen ? "Hide details" : "Details"}</Button>
          {action(application)}
        </div>
        <Collapsible isOpen={isOpen} keepDOM={false}><ApplicationDetails application={application} /></Collapsible>
      </div>} />;
    }} />
      : <Table className="data-table" rowKey="id" pagination={apps.length > 20 ? { pageSize: 20 } : false} dataSource={apps} empty={empty} hideExpandedColumn={false} expandedRowRender={(application?: Application) => application ? <ApplicationDetails application={application} /> : null} columns={[
          { title: "Application", dataIndex: "name", render: (_value: string, application: Application) => <div className="identity-cell"><strong>{application.name}</strong><code>{application.id}</code></div> },
          { title: "Type", dataIndex: "resource", render: (_value: boolean, application: Application) => applicationType(application) },
          { title: "Scopes", dataIndex: "scopes", render: (scopes: string[]) => <div className="tag-wrap">{(scopes ?? []).map(scope => <Tag key={scope} size="small">{scope}</Tag>)}</div> },
          { title: "Status", dataIndex: "enabled", width: 110, render: (_value: boolean, application: Application) => status(application) },
          { title: "", dataIndex: "id", align: "right", render: (_value: string, application: Application) => action(application) },
        ]} />}
  </Panel>;
}

/** Latest activity events with pagination. */
function ActivityTable({ activity }: { activity: Activity[] }) {
  const isCompact = useMediaQuery(COMPACT_QUERY);
  const empty = <Empty image={<IllustrationNoContent />} description="No activity recorded yet." />;
  return <Panel title="Security events" description="Activity records contain event names and account identifiers, never credentials.">
    {isCompact ? <List className="stack-list" dataSource={activity.slice(0, 50)} emptyContent={empty} renderItem={(event: Activity) => <List.Item main={<div className="stack-item"><strong>{event.action}</strong><code className="wrap-anywhere">{event.subject}</code><span>{formatTime(event.time)}</span></div>} />} />
      : <Table className="data-table" rowKey={(event?: Activity) => event ? `${event.time}-${event.action}-${event.subject}` : ""} pagination={activity.length > 15 ? { pageSize: 15 } : false} dataSource={activity} empty={empty} columns={[
          { title: "Event", dataIndex: "action", render: (value: string) => <span className="activity-action">{value}</span> },
          { title: "Subject", dataIndex: "subject", render: (value: string) => <code className="wrap-anywhere">{value}</code> },
          { title: "Time", dataIndex: "time", width: 200, render: (value: number) => formatTime(value) },
        ]} />}
  </Panel>;
}

/** Width for form drawers: full width on phones, fixed on wider screens. */
function useSheetWidth(): string | number {
  return useMediaQuery(COMPACT_QUERY) ? "100%" : 520;
}

interface SheetProps { session: SessionState; run: (task: () => Promise<void>) => Promise<void>; busy: boolean; onDone: () => Promise<void> }

/** Option shape accepted by Semi Select. */
interface Option { value: string; label: string }

/**
 * Render a labeled Semi Select bound to a form field.
 *
 * @param props - Form state, field key, label, options and optional hint.
 * @returns The labeled select.
 */
function SelectField<Keys extends string>({ form, field, label, options, placeholder, hint, disabled }: { form: FormState<Keys>; field: Keys; label: string; options: Option[]; placeholder: string; hint?: string; disabled?: boolean }) {
  const id = form.idFor(field);
  const error = form.errors[field];
  return <Field id={id} label={label} hint={hint} error={error}>
    <Select id={id} className="select-full" value={form.values[field] || undefined} onChange={value => form.setValue(field)(String(value ?? ""))} optionList={options} placeholder={placeholder} filter={options.length > 6} validateStatus={error ? "error" : "default"} disabled={disabled} aria-labelledby={id + "-label"} />
  </Field>;
}

/** Create an ordinary local user; owner privileges are not an input field. */
function UserSheet({ session, run, busy, isOpen, onClose, onDone }: SheetProps & { isOpen: boolean; onClose: () => void }) {
  const form = useForm({ username: "", password: "" });
  const width = useSheetWidth();
  /** Close the drawer and clear the entered password. */
  function close() { form.reset(); onClose(); }
  /** Submit the new account after client-side checks. */
  function submit(event?: React.FormEvent<HTMLFormElement>) {
    event?.preventDefault();
    const isValid = form.check({ username: value => value === "" ? "Enter a username." : /^[A-Za-z0-9][A-Za-z0-9_.-]*$/.test(value) ? "" : "Start with a letter or digit; use letters, digits, dot, dash or underscore.", password: passwordRule(session.password_policy) });
    if (!isValid) return;
    const { username, password } = form.values;
    void run(async () => {
      await postJson("/api/users/create", { username, password }, session.csrf);
      close(); Toast.success("User created. Assign an application to enable access."); await onDone();
    });
  }
  return <SideSheet title="Create a local user" visible={isOpen} onCancel={close} width={width} className="form-sheet" footer={<div className="sheet-footer"><Button onClick={close}>Cancel</Button><Button theme="solid" loading={busy} onClick={() => submit()}>Create user</Button></div>}>
    <p className="sheet-lead">The user signs in with this password. Share it through your existing private channel — QoAuth sends no email.</p>
    <form onSubmit={submit} noValidate>
      <TextField form={form} field="username" label="Username" name="username" maxLength={64} autoComplete="off" disabled={busy} />
      <PasswordField form={form} field="password" label="Initial password" policy={session.password_policy} disabled={busy} />
      <button type="submit" hidden tabIndex={-1} />
    </form>
  </SideSheet>;
}

/** Reset only a selected local user's password, retaining their enrolled factor. */
function ResetSheet({ session, run, busy, users, target, onClose, onDone }: SheetProps & { users: Account[]; target: string | null; onClose: () => void }) {
  const form = useForm({ user: "", password: "" });
  const width = useSheetWidth();
  const setUser = form.setValue("user");
  useEffect(() => { if (target !== null) setUser(target); }, [target]);
  /** Close the drawer and clear the replacement password. */
  function close() { form.reset(); onClose(); }
  /** Submit the replacement password and always clear the form. */
  function submit(event?: React.FormEvent<HTMLFormElement>) {
    event?.preventDefault();
    if (!form.check({ user: required("Choose a user."), password: passwordRule(session.password_policy) })) return;
    const { user, password } = form.values;
    void run(async () => {
      try {
        await postJson("/api/users/password", { user, password }, session.csrf);
        onClose(); Toast.success({ content: "Password reset. Existing sessions, pending codes and grants ended; enrolled MFA is still required. Disabled accounts remain disabled.", duration: 6 }); await onDone();
      } finally { form.reset(); }
    });
  }
  return <SideSheet title="Reset a user's password" visible={target !== null} onCancel={close} width={width} className="form-sheet" footer={<div className="sheet-footer"><Button onClick={close}>Cancel</Button><Button type="danger" theme="solid" loading={busy} disabled={!session.fresh} onClick={() => submit()}>Reset password and revoke sessions</Button></div>}>
    <Banner type="info" closeIcon={null} description="Requires fresh administrator MFA. This ends the user's QoAuth sessions, pending codes and grants and keeps their enrolled MFA. Share the replacement through your existing private channel; QoAuth sends no email." />
    <form onSubmit={submit} noValidate>
      <SelectField form={form} field="user" label="Account to reset" placeholder="Choose a user" disabled={busy} options={users.filter(user => !user.owner).map(user => ({ value: user.id, label: user.username + (user.enabled ? "" : " (disabled)") }))} />
      <PasswordField form={form} field="password" label="Replacement password" policy={session.password_policy} disabled={busy} />
      <button type="submit" hidden tabIndex={-1} />
    </form>
    <p className="sheet-lead">Use My account to change your own password. Lost-owner recovery remains a local maintenance command.</p>
  </SideSheet>;
}

/** Empty registration values. */
const EMPTY_APPLICATION = { mode: "create", kind: "confidential", id: "", name: "", redirects: "", logout: "", scopes: "", audiences: "" };

/**
 * Prefill registration values from a disabled application under review.
 *
 * @param review - The application being reviewed, or null for a new registration.
 * @returns Form values for the registration drawer.
 */
function applicationValues(review: Application | null): typeof EMPTY_APPLICATION {
  if (!review) return EMPTY_APPLICATION;
  return { mode: "replace", kind: review.resource ? "resource" : review.public ? "public" : "confidential", id: review.id, name: review.name, redirects: review.redirects?.join("\n") ?? "", logout: review.logout_redirects?.join("\n") ?? "", scopes: review.scopes?.join(", ") ?? "", audiences: review.audiences?.join(", ") ?? "" };
}

/** Register reviewed callbacks/resources or explicitly replace a disabled client. */
function ApplicationSheet({ session, run, busy, state, onClose, onRegistered }: Omit<SheetProps, "onDone"> & { state: { review: Application | null } | null; onClose: () => void; onRegistered: (secret: string) => Promise<void> }) {
  const review = state?.review ?? null;
  return <ApplicationSheetBody key={state ? (review ? "review:" + review.id : "create") : "closed"} session={session} run={run} busy={busy} isOpen={state !== null} review={review} onClose={onClose} onRegistered={onRegistered} />;
}

/** Registration drawer body, remounted per registration so values never leak between reviews. */
function ApplicationSheetBody({ session, run, busy, isOpen, review, onClose, onRegistered }: Omit<SheetProps, "onDone"> & { isOpen: boolean; review: Application | null; onClose: () => void; onRegistered: (secret: string) => Promise<void> }) {
  const form = useForm(applicationValues(review));
  const width = useSheetWidth();
  const kind = form.values.kind;
  const isResource = kind === "resource";
  /** Build the exact application payload the server validates. */
  function submit(event?: React.FormEvent<HTMLFormElement>) {
    event?.preventDefault();
    if (!form.check({ id: required("Enter a client ID."), name: required("Enter a display name."), scopes: value => splitCommaList(value).length ? "" : "Enter at least one scope." })) return;
    const values = form.values;
    const id = values.id;
    const application = { id, name: values.name, resource: isResource, public: kind === "public", redirects: isResource ? [] : splitLines(values.redirects), logout_redirects: isResource ? [] : splitLines(values.logout), scopes: splitCommaList(values.scopes), audiences: isResource ? [id] : splitCommaList(values.audiences) };
    const path = values.mode === "replace" ? "/api/applications/replace-disabled" : "/api/applications/create";
    void run(async () => {
      const result = await postJson<{ client_secret: string }>(path, { application }, session.csrf);
      form.reset(); onClose();
      Toast.success({ content: "Application registered with reviewed settings. Assign access explicitly; prior credentials and grants are not restored.", duration: 6 });
      await onRegistered(result.client_secret);
    });
  }
  const modeId = form.idFor("mode");
  const kindId = form.idFor("kind");
  return <SideSheet title={review ? "Review disabled application" : "Register an application"} visible={isOpen} onCancel={onClose} width={width} className="form-sheet" footer={<div className="sheet-footer"><Button onClick={onClose}>{review ? "Cancel review" : "Cancel"}</Button><Button theme="solid" loading={busy} onClick={() => submit()}>Register application</Button></div>}>
    {review ? <Banner type="warning" closeIcon={null} title={review.name} description="These are saved settings, not restored access. Review callbacks and permissions before registering. The previous secret is never returned." /> : null}
    <form onSubmit={submit} noValidate>
      <Field id={modeId} isGroup label="Registration action" hint="Replacement creates a new secret for confidential clients and clears old assignments. Keep the application type unchanged. Disable dependent applications before replacing a resource API, then register them again with reviewed scopes.">
        <RadioGroup type="button" name="registration-mode" value={form.values.mode} onChange={event => form.setValue("mode")(String(event.target.value))} aria-labelledby={modeId + "-label"} disabled={busy}>
          <Radio value="create">Create new</Radio>
          <Radio value="replace">Replace disabled</Radio>
        </RadioGroup>
      </Field>
      <TextField form={form} field="id" label="Client ID" name="client-id" maxLength={64} readonly={!!review} autoComplete="off" autoFocus={!!review} disabled={busy} />
      <TextField form={form} field="name" label="Display name" name="display-name" maxLength={80} autoComplete="off" disabled={busy} />
      <Field id={kindId} isGroup label="Application type">
        <RadioGroup type="pureCard" direction="vertical" name="application-kind" className="kind-picker" value={kind} onChange={event => form.setValue("kind")(String(event.target.value))} aria-labelledby={kindId + "-label"} disabled={busy}>
          <Radio value="confidential" extra="Keeps a client secret on its own server.">Server-side web client</Radio>
          <Radio value="public" extra="Browser or native app without a secret; requires PKCE.">Public client with PKCE</Radio>
          <Radio value="resource" extra="Introspects tokens and uses its own ID as audience.">Resource API validator</Radio>
        </RadioGroup>
      </Field>
      {!isResource ? <>
        <TextField form={form} field="redirects" label="Exact callback URLs, one per line" isMultiline placeholder="https://app.example.test/callback" disabled={busy} />
        <TextField form={form} field="logout" label="Exact signed-out URLs, one per line" isMultiline placeholder="https://app.example.test/signed-out" disabled={busy} />
      </> : null}
      <TextField form={form} field="scopes" label="Allowed scopes, comma separated" placeholder="openid, profile, offline_access, notes.read" autoComplete="off" disabled={busy} />
      {!isResource ? <TextField form={form} field="audiences" label="Resource API client IDs, comma separated" placeholder="notes-api" autoComplete="off" disabled={busy} hint="Register resource APIs first. An API validator uses its own ID as its audience and has no browser callbacks." /> : null}
      <button type="submit" hidden tabIndex={-1} />
    </form>
  </SideSheet>;
}

/** Grant or remove the selected user's application assignment. */
function AssignmentPanel({ session, run, busy, users, apps, onDone, onRegister }: SheetProps & { users: Account[]; apps: Application[]; onRegister: () => void }) {
  const form = useForm({ user: "", client: "", allow: "yes" });
  const assignableApps = apps.filter(app => app.enabled && !app.resource);
  /** Save one explicit assignment change. */
  function submit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!form.check({ user: required("Choose a user."), client: required("Choose an application.") })) return;
    const { user, client, allow } = form.values;
    void run(async () => { await postJson("/api/assignments", { user, client, allow: allow === "yes" }, session.csrf); Toast.success("Assignment updated. Removed access revokes existing grants and codes."); await onDone(); });
  }
  if (!assignableApps.length) return <Panel title="Assign application access"><StatusView illustration={<IllustrationNoContent />} title="No assignable applications" description="Register an active client application before assigning access. Resource APIs are not assigned to users." action={<Button theme="solid" icon={<IconPlus />} onClick={onRegister}>Go to applications</Button>} /></Panel>;
  const allowId = form.idFor("allow");
  return <div className="panel-grid panel-grid-wide-left">
    <Panel title="Assign application access" description="Users can only sign in to applications they are assigned to.">
      <form onSubmit={submit} noValidate>
        <SelectField form={form} field="user" label="User" placeholder="Choose a user" disabled={busy} options={users.filter(user => user.enabled).map(user => ({ value: user.id, label: user.username }))} />
        <SelectField form={form} field="client" label="Application" placeholder="Choose an application" disabled={busy} options={assignableApps.map(app => ({ value: app.id, label: `${app.name} (${app.id})` }))} />
        <Field id={allowId} isGroup label="Access">
          <RadioGroup type="button" name="assignment-access" value={form.values.allow} onChange={event => form.setValue("allow")(String(event.target.value))} aria-labelledby={allowId + "-label"} disabled={busy}>
            <Radio value="yes">Allow</Radio>
            <Radio value="no">Remove and revoke</Radio>
          </RadioGroup>
        </Field>
        <div className="form-actions"><Button htmlType="submit" theme="solid" loading={busy}>Save assignment</Button></div>
      </form>
    </Panel>
    <Panel title="How access works">
      <ul className="explain-list">
        <li><IconKey aria-hidden="true" /><span>An assignment lets one user sign in to one client application. There is no group or policy layer.</span></li>
        <li><IconLock aria-hidden="true" /><span>Removing access revokes that user's existing grants and pending codes for the application.</span></li>
        <li><IconShield aria-hidden="true" /><span>Users still approve the requested scopes on the consent screen.</span></li>
      </ul>
    </Panel>
  </div>;
}
