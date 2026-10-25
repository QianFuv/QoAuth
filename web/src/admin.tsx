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
import Typography from "@douyinfe/semi-ui/lib/es/typography";
import { IllustrationFailure, IllustrationNoContent } from "@douyinfe/semi-illustrations";
import { IconApps, IconChevronDown, IconChevronUp, IconHistory, IconKey, IconLock, IconPlus, IconRefresh, IconShield, IconUserGroup } from "@douyinfe/semi-icons";
import { getJson, postJson, type Account, type Activity, type Application, type SessionState } from "./api";
import { PasswordField, passwordRule } from "./password-field";
import { Field, TextField, required, useForm, type FormState } from "./form";
import { COMPACT_QUERY, DRAWER_QUERY, formatTime, navigate, useMediaQuery } from "./layout";
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
  return application.resource ? "资源 API" : application.public ? "公开客户端（PKCE）" : "服务端客户端";
}

/** Chinese labels for the activity event names the server records; unknown names show as-is. */
const ACTION_LABELS: Record<string, string> = {
  "initialize": "初始化实例",
  "password-login": "密码登录",
  "mfa-enroll": "绑定验证器",
  "mfa-verify": "多因素验证",
  "mfa-recovery": "使用恢复码恢复",
  "recovery-code": "恢复码校验",
  "authorize": "授权",
  "code-exchange": "兑换授权码",
  "code-reuse": "授权码重复使用",
  "refresh": "刷新令牌",
  "refresh-reuse": "刷新令牌重复使用",
  "revoke": "撤销令牌",
  "invalidate": "使授权失效",
  "session-revoke": "撤销会话",
  "consent-withdraw": "撤回授权同意",
  "user-create": "创建用户",
  "user-disable": "停用用户",
  "credential-reset": "重置密码",
  "application-create": "注册应用",
  "application-disable": "停用应用",
  "application-replace-disabled": "替换已停用应用",
  "assignment-change": "变更访问分配",
  "signing-rotate": "轮换签名密钥",
  "maintenance-cleanup": "维护清理",
  "restore-quarantine": "恢复隔离期",
  "restore-owner-reviewed": "恢复后所有者复核",
  "host-user-recovery": "本地恢复用户",
};

/**
 * Show an activity event by its Chinese label with the recorded event name kept beside it.
 *
 * @param props - The recorded event name.
 * @returns A label plus the original event name.
 */
function ActionLabel({ action }: { action: string }) {
  const label = ACTION_LABELS[action];
  return <span className="action-label"><span className="activity-action">{label ?? action}</span>{label ? <code className="action-code">{action}</code> : null}</span>;
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
    return run(async () => { await postJson("/api/users/disable", { user: user.id }, session.csrf); Toast.success("用户已停用，现有登录已撤销。"); await reload(); });
  }
  /** Disable an application; the server revokes its grants. */
  function disableApplication(application: Application) {
    return run(async () => { await postJson("/api/applications/disable", { client: application.id }, session.csrf); Toast.success("应用已停用，授权已撤销。"); await reload(); });
  }
  const sharedSheets = <>
    <UserSheet session={session} run={run} busy={busy} isOpen={isUserSheetOpen} onClose={() => setIsUserSheetOpen(false)} onDone={reload} />
    <ResetSheet session={session} run={run} busy={busy} users={users ?? []} target={resetTarget} onClose={() => setResetTarget(null)} onDone={reload} />
    <ApplicationSheet session={session} run={run} busy={busy} state={applicationSheet} onClose={() => setApplicationSheet(null)} onRegistered={async clientSecret => { setSecret(clientSecret); await reload(); }} />
    <Modal
      title="复制新的客户端密钥"
      visible={secret !== ""}
      closable={false}
      maskClosable={false}
      closeOnEsc={false}
      footer={<Button theme="solid" onClick={() => setSecret("")}>我已保存密钥</Button>}
      width="min(520px, calc(100vw - 32px))"
    >
      <p className="modal-text">请将其保存到应用的服务端配置中。QoAuth 只保存哈希值，密钥仅显示这一次。</p>
      <CopyBlock value={secret} label="客户端密钥" />
    </Modal>
  </>;
  const freshNotice = !session.fresh ? <Banner className="enter" type="warning" title="变更前需要重新登录" description="管理变更要求在最近 5 分钟内完成过带 MFA 的登录。" closeIcon={null}>
    <div className="banner-actions"><Button theme="solid" onClick={() => window.location.assign("/login")}>重新登录</Button></div>
  </Banner> : null;
  if (isLoadFailed) return <div className="console-page"><StatusView illustration={<IllustrationFailure />} title="管理后台暂不可用" description="管理数据加载失败。如果会话已过期，请重新登录。" action={<Button icon={<IconRefresh />} onClick={() => void reload().catch(() => setIsLoadFailed(true))}>重试</Button>} /></div>;
  if (!users || !apps || !activity) return <div className="console-page" role="status" aria-label="正在加载管理后台"><Skeleton active loading placeholder={<><Skeleton.Title style={{ width: 240, height: 32, marginBottom: 24 }} /><Skeleton.Paragraph rows={8} /></>} /></div>;
  const openRegistration = () => setApplicationSheet({ review: null });
  if (section === "users") return <div className="console-page">
    <PageHeader eyebrow="管理" title="用户" description="本实例中的本地账户。新用户需要先分配应用，才能登录该应用。" actions={<Button theme="solid" icon={<IconPlus />} onClick={() => setIsUserSheetOpen(true)}>创建用户</Button>} />
    {freshNotice}
    <UsersTable users={users} busy={busy} isFresh={session.fresh} onDisable={disableUser} onReset={user => setResetTarget(user.id)} />
    {sharedSheets}
  </div>;
  if (section === "applications") return <div className="console-page">
    <PageHeader eyebrow="管理" title="应用" description="OIDC 客户端与资源 API，使用精确回调地址和显式权限范围。" actions={<Button theme="solid" icon={<IconPlus />} onClick={openRegistration}>注册应用</Button>} />
    {freshNotice}
    <ApplicationsTable apps={apps} busy={busy} onDisable={disableApplication} onReview={application => { setSecret(""); setApplicationSheet({ review: application }); }} />
    {sharedSheets}
  </div>;
  if (section === "access") return <div className="console-page">
    <PageHeader eyebrow="管理" title="访问分配" description="为某个用户分配或移除某个应用的访问权限。" />
    {freshNotice}
    <AssignmentPanel session={session} run={run} busy={busy} users={users} apps={apps} onDone={reload} onRegister={() => navigate("/admin#applications")} />
    {sharedSheets}
  </div>;
  if (section === "activity") return <div className="console-page">
    <PageHeader eyebrow="管理" title="活动记录" description="本实例记录的最新安全事件。" />
    <ActivityTable activity={activity} users={users} apps={apps} />
    {sharedSheets}
  </div>;
  const activeApps = apps.filter(app => app.enabled);
  return <div className="console-page">
    <PageHeader
      eyebrow="管理"
      title="实例概览"
      description="本地用户以及他们可以访问的应用。"
      tags={<Tag color="green" size="large" prefixIcon={<IconShield />}>管理员 MFA 已验证</Tag>}
      actions={<><Button icon={<IconPlus />} onClick={() => setIsUserSheetOpen(true)}>创建用户</Button><Button theme="solid" icon={<IconPlus />} onClick={openRegistration}>注册应用</Button></>}
    />
    {freshNotice}
    <div className="stat-row enter">
      <button type="button" className="stat" onClick={() => navigate("/admin#users")}><IconUserGroup aria-hidden="true" /><strong>{users.length}</strong><span>本地用户</span><small>已停用 {users.filter(user => !user.enabled).length} 个</small></button>
      <button type="button" className="stat" onClick={() => navigate("/admin#applications")}><IconApps aria-hidden="true" /><strong>{activeApps.length}</strong><span>启用中的应用</span><small>已停用 {apps.length - activeApps.length} 个</small></button>
      <button type="button" className="stat" onClick={() => navigate("/admin#activity")}><IconHistory aria-hidden="true" /><strong>{activity.length}</strong><span>已记录事件</span><small>显示最近 {activity.length >= 100 ? "100" : activity.length} 条</small></button>
    </div>
    <div className="panel-grid">
      <Panel title="最近活动" description="只记录事件名称与账户标识，从不记录凭据。" toolbar={<Button theme="borderless" onClick={() => navigate("/admin#activity")}>查看全部</Button>}>
        {activity.length ? <Timeline className="activity-timeline">{activity.slice(0, 6).map((event, index) => <Timeline.Item key={`${event.time}-${index}`} time={formatTime(event.time)}><ActionLabel action={event.action} /> <SubjectLabel subject={event.subject} users={users} apps={apps} /></Timeline.Item>)}</Timeline> : <Empty image={<IllustrationNoContent />} description="暂无活动记录。" />}
      </Panel>
      <Panel title="应用" description="已注册的客户端与资源 API。" toolbar={<Button theme="borderless" onClick={() => navigate("/admin#applications")}>管理</Button>}>
        {apps.length ? <ul className="mini-list">{apps.slice(0, 6).map(app => <li key={app.id}><div><strong>{app.name}</strong><code>{app.id}</code></div>{app.enabled ? <Tag color="green" size="small">启用</Tag> : <Tag color="grey" size="small">已停用</Tag>}</li>)}</ul>
          : <Empty image={<IllustrationNoContent />} description="还没有应用。如果客户端要调用资源 API，请先注册资源 API。"><Button theme="solid" icon={<IconPlus />} onClick={openRegistration}>注册应用</Button></Empty>}
      </Panel>
    </div>
    {sharedSheets}
  </div>;
}

/** Users table on wide screens and a stacked list on small screens. */
function UsersTable({ users, busy, isFresh, onDisable, onReset }: { users: Account[]; busy: boolean; isFresh: boolean; onDisable: (user: Account) => Promise<void>; onReset: (user: Account) => void }) {
  const isCompact = useMediaQuery(COMPACT_QUERY);
  const status = (user: Account) => user.owner ? <Tag color="blue" size="small">所有者</Tag> : user.enabled ? <Tag color="green" size="small">启用</Tag> : <Tag color="grey" size="small">已停用</Tag>;
  const role = (user: Account) => <span className="role-cell">{user.admin ? "管理员" : "成员"}{user.require_mfa || user.admin ? <Tag size="small" prefixIcon={<IconLock />}>需要 MFA</Tag> : null}</span>;
  const actions = (user: Account) => user.owner ? <span className="muted-cell">通过本地命令恢复</span> : <div className="row-actions">
    <Button theme="borderless" disabled={busy || !isFresh} onClick={() => onReset(user)}>重置密码</Button>
    {user.enabled ? <Popconfirm title={`停用 ${user.username}？`} content="现有会话、验证码与授权会被撤销。之后只能通过本地 recover-user 命令恢复此账户。" okText="停用" okType="danger" cancelText="取消" onConfirm={() => onDisable(user)}>
      <Button type="danger" theme="borderless" disabled={busy}>停用</Button>
    </Popconfirm> : null}
  </div>;
  const empty = <Empty image={<IllustrationNoContent />} description="还没有用户。" />;
  return <Panel title="全部用户" description={`共 ${users.length} 个本地账户。`}>
    {isCompact ? <List className="stack-list" dataSource={users} emptyContent={empty} renderItem={(user: Account) => <List.Item main={<div className="stack-item"><strong className="wrap-anywhere">{user.username}</strong>{role(user)}<div className="stack-tags">{status(user)}</div>{actions(user)}</div>} />} />
      : <Table className="data-table" rowKey="id" pagination={users.length > 20 ? { pageSize: 20 } : false} dataSource={users} empty={empty} columns={[
          { title: "用户", dataIndex: "username", render: (_value: string, user: Account) => <div className="identity-cell"><strong>{user.username}</strong><code>{user.id}</code></div> },
          { title: "角色", dataIndex: "admin", render: (_value: boolean, user: Account) => role(user) },
          { title: "状态", dataIndex: "enabled", width: 120, render: (_value: boolean, user: Account) => status(user) },
          { title: "", dataIndex: "id", align: "right", render: (_value: string, user: Account) => actions(user) },
        ]} />}
  </Panel>;
}

/** Wrap a list of exact values, or show a dash when none are registered. */
function ValueList({ values }: { values: string[] }) {
  if (!values.length) return <span className="muted-cell">无</span>;
  return <ul className="value-list">{values.map(value => <li key={value}><code>{value}</code></li>)}</ul>;
}

/** Full registered configuration for one application. */
function ApplicationDetails({ application }: { application: Application }) {
  return <dl className="detail-grid">
    {!application.resource ? <><dt>回调地址</dt><dd><ValueList values={application.redirects ?? []} /></dd><dt>登出地址</dt><dd><ValueList values={application.logout_redirects ?? []} /></dd></> : null}
    <dt>权限范围</dt><dd><ValueList values={application.scopes ?? []} /></dd>
    <dt>{application.resource ? "受众（Audience）" : "资源 API"}</dt><dd><ValueList values={application.audiences ?? []} /></dd>
  </dl>;
}

/** Applications table with expandable configuration, or stacked cards on small screens. */
function ApplicationsTable({ apps, busy, onDisable, onReview }: { apps: Application[]; busy: boolean; onDisable: (application: Application) => Promise<void>; onReview: (application: Application) => void }) {
  const isCompact = useMediaQuery(DRAWER_QUERY);
  const [openId, setOpenId] = useState("");
  const status = (application: Application) => application.enabled ? <Tag color="green" size="small">启用</Tag> : <Tag color="grey" size="small">已停用</Tag>;
  const action = (application: Application) => application.enabled
    ? <Popconfirm title={`停用 ${application.name}？`} content="现有授权会被撤销。重新启用需要复核并替换其设置。" okText="停用" okType="danger" cancelText="取消" onConfirm={() => onDisable(application)}><Button type="danger" theme="borderless" disabled={busy}>停用</Button></Popconfirm>
    : <Button theme="borderless" disabled={busy} onClick={() => onReview(application)}>复核设置</Button>;
  const empty = <Empty image={<IllustrationNoContent />} description="还没有注册应用。" />;
  return <Panel title="已注册应用" description="展开应用可查看其精确回调地址、权限范围与资源 API。">
    {isCompact ? <List className="stack-list" dataSource={apps} emptyContent={empty} renderItem={(application: Application) => {
      const isOpen = openId === application.id;
      return <List.Item main={<div className="stack-item">
        <strong className="wrap-anywhere">{application.name}</strong>
        <code className="wrap-anywhere">{application.id}</code>
        <div className="stack-tags">{status(application)}<Tag size="small">{applicationType(application)}</Tag></div>
        <div className="row-actions">
          <Button theme="borderless" type="tertiary" icon={isOpen ? <IconChevronUp /> : <IconChevronDown />} aria-expanded={isOpen} onClick={() => setOpenId(isOpen ? "" : application.id)}>{isOpen ? "收起详情" : "详情"}</Button>
          {action(application)}
        </div>
        <Collapsible isOpen={isOpen} keepDOM={false}><ApplicationDetails application={application} /></Collapsible>
      </div>} />;
    }} />
      : <Table className="data-table" rowKey="id" pagination={apps.length > 20 ? { pageSize: 20 } : false} dataSource={apps} empty={empty} hideExpandedColumn={false} expandedRowRender={(application?: Application) => application ? <ApplicationDetails application={application} /> : null} columns={[
          { title: "应用", dataIndex: "name", render: (_value: string, application: Application) => <div className="identity-cell"><strong>{application.name}</strong><code>{application.id}</code></div> },
          { title: "类型", dataIndex: "resource", render: (_value: boolean, application: Application) => applicationType(application) },
          { title: "权限范围", dataIndex: "scopes", render: (scopes: string[]) => <div className="tag-wrap">{(scopes ?? []).map(scope => <Tag key={scope} size="small">{scope}</Tag>)}</div> },
          { title: "状态", dataIndex: "enabled", width: 110, render: (_value: boolean, application: Application) => status(application) },
          { title: "", dataIndex: "id", align: "right", render: (_value: string, application: Application) => action(application) },
        ]} />}
  </Panel>;
}

/**
 * Show an activity subject by a name the administration lists already provide.
 *
 * Users resolve to usernames and applications to names; the opaque value is
 * shortened to eight characters with the full value available to copy.
 *
 * @param props - The recorded subject and the loaded users and applications.
 * @returns A compact subject label.
 */
function SubjectLabel({ subject, users, apps }: { subject: string; users: Account[]; apps: Application[] }) {
  if (subject === "host") return <span className="subject"><span className="subject-name">主机维护</span></span>;
  const application = apps.find(app => app.id === subject);
  if (application) return <span className="subject"><span className="subject-name">{application.name}</span><code className="subject-id">{application.id}</code></span>;
  const user = users.find(account => account.id === subject);
  const shortId = subject.length > 10 ? subject.slice(0, 8) + "…" : subject;
  return <span className="subject">
    {user ? <span className="subject-name">{user.username}</span> : null}
    <Typography.Text className="subject-id" copyable={{ content: subject, copyTip: "复制完整 ID", successTip: "已复制" }}><code title={subject}>{user ? shortId : "ID " + shortId}</code></Typography.Text>
  </span>;
}

/** Latest activity events with pagination. */
function ActivityTable({ activity, users, apps }: { activity: Activity[]; users: Account[]; apps: Application[] }) {
  const isCompact = useMediaQuery(COMPACT_QUERY);
  const empty = <Empty image={<IllustrationNoContent />} description="暂无活动记录。" />;
  return <Panel title="安全事件" description="活动记录只包含事件名称与账户标识，从不包含凭据。">
    {isCompact ? <List className="stack-list" dataSource={activity.slice(0, 50)} emptyContent={empty} renderItem={(event: Activity) => <List.Item main={<div className="stack-item"><ActionLabel action={event.action} /><SubjectLabel subject={event.subject} users={users} apps={apps} /><span>{formatTime(event.time)}</span></div>} />} />
      : <Table className="data-table" rowKey={(event?: Activity) => event ? `${event.time}-${event.action}-${event.subject}` : ""} pagination={activity.length > 15 ? { pageSize: 15 } : false} dataSource={activity} empty={empty} columns={[
          { title: "事件", dataIndex: "action", render: (value: string) => <ActionLabel action={value} /> },
          { title: "主体", dataIndex: "subject", render: (value: string) => <SubjectLabel subject={value} users={users} apps={apps} /> },
          { title: "时间", dataIndex: "time", width: 200, render: (value: number) => formatTime(value) },
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
    const isValid = form.check({ username: value => value === "" ? "请输入用户名。" : /^[A-Za-z0-9][A-Za-z0-9_.-]*$/.test(value) ? "" : "须以字母或数字开头，只能包含字母、数字、点、短横线或下划线。", password: passwordRule(session.password_policy) });
    if (!isValid) return;
    const { username, password } = form.values;
    void run(async () => {
      await postJson("/api/users/create", { username, password }, session.csrf);
      close(); Toast.success("用户已创建。分配应用后即可访问。"); await onDone();
    });
  }
  return <SideSheet title="创建本地用户" visible={isOpen} onCancel={close} width={width} className="form-sheet" footer={<div className="sheet-footer"><Button onClick={close}>取消</Button><Button theme="solid" loading={busy} onClick={() => submit()}>创建用户</Button></div>}>
    <p className="sheet-lead">用户将使用这个密码登录。请通过你现有的私密渠道告知对方，QoAuth 不发送邮件。</p>
    <form onSubmit={submit} noValidate>
      <TextField form={form} field="username" label="用户名" name="username" maxLength={64} autoComplete="off" disabled={busy} />
      <PasswordField form={form} field="password" label="初始密码" policy={session.password_policy} disabled={busy} />
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
    if (!form.check({ user: required("请选择用户。"), password: passwordRule(session.password_policy) })) return;
    const { user, password } = form.values;
    void run(async () => {
      try {
        await postJson("/api/users/password", { user, password }, session.csrf);
        onClose(); Toast.success({ content: "密码已重置。现有会话、待用授权码与授权已结束；已绑定的 MFA 仍然需要。已停用的账户保持停用。", duration: 6 }); await onDone();
      } finally { form.reset(); }
    });
  }
  return <SideSheet title="重置用户密码" visible={target !== null} onCancel={close} width={width} className="form-sheet" footer={<div className="sheet-footer"><Button onClick={close}>取消</Button><Button type="danger" theme="solid" loading={busy} disabled={!session.fresh} onClick={() => submit()}>重置密码并撤销会话</Button></div>}>
    <Banner type="info" closeIcon={null} description="需要管理员最近完成 MFA 登录。此操作会结束该用户的 QoAuth 会话、待用授权码与授权，并保留其已绑定的 MFA。请通过现有私密渠道告知新密码，QoAuth 不发送邮件。" />
    <form onSubmit={submit} noValidate>
      <SelectField form={form} field="user" label="要重置的账户" placeholder="选择用户" disabled={busy} options={users.filter(user => !user.owner).map(user => ({ value: user.id, label: user.username + (user.enabled ? "" : "（已停用）") }))} />
      <PasswordField form={form} field="password" label="新密码" policy={session.password_policy} disabled={busy} />
      <button type="submit" hidden tabIndex={-1} />
    </form>
    <p className="sheet-lead">修改自己的密码请前往“我的账户”。所有者丢失凭据时仍需使用本地维护命令恢复。</p>
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
    if (!form.check({ id: required("请输入客户端 ID。"), name: required("请输入显示名称。"), scopes: value => splitCommaList(value).length ? "" : "请至少输入一个权限范围。" })) return;
    const values = form.values;
    const id = values.id;
    const application = { id, name: values.name, resource: isResource, public: kind === "public", redirects: isResource ? [] : splitLines(values.redirects), logout_redirects: isResource ? [] : splitLines(values.logout), scopes: splitCommaList(values.scopes), audiences: isResource ? [id] : splitCommaList(values.audiences) };
    const path = values.mode === "replace" ? "/api/applications/replace-disabled" : "/api/applications/create";
    void run(async () => {
      const result = await postJson<{ client_secret: string }>(path, { application }, session.csrf);
      form.reset(); onClose();
      Toast.success({ content: "应用已按复核后的设置注册。请显式分配访问权限；旧凭据与授权不会恢复。", duration: 6 });
      await onRegistered(result.client_secret);
    });
  }
  const modeId = form.idFor("mode");
  const kindId = form.idFor("kind");
  return <SideSheet title={review ? "复核已停用的应用" : "注册应用"} visible={isOpen} onCancel={onClose} width={width} className="form-sheet" footer={<div className="sheet-footer"><Button onClick={onClose}>{review ? "取消复核" : "取消"}</Button><Button theme="solid" loading={busy} onClick={() => submit()}>注册应用</Button></div>}>
    {review ? <Banner type="warning" closeIcon={null} title={review.name} description="这些是已保存的设置，不代表恢复访问。注册前请复核回调地址与权限。旧密钥永远不会返回。" /> : null}
    <form onSubmit={submit} noValidate>
      <Field id={modeId} isGroup label="注册方式" hint="替换会为机密客户端生成新密钥并清除旧的访问分配。请保持应用类型不变。替换资源 API 前，请先停用依赖它的应用，之后再按复核后的权限范围重新注册。">
        <RadioGroup type="button" name="registration-mode" value={form.values.mode} onChange={event => form.setValue("mode")(String(event.target.value))} aria-labelledby={modeId + "-label"} disabled={busy}>
          <Radio value="create">新建</Radio>
          <Radio value="replace">替换已停用应用</Radio>
        </RadioGroup>
      </Field>
      <TextField form={form} field="id" label="客户端 ID" name="client-id" maxLength={64} readonly={!!review} autoComplete="off" autoFocus={!!review} disabled={busy} />
      <TextField form={form} field="name" label="显示名称" name="display-name" maxLength={80} autoComplete="off" disabled={busy} />
      <Field id={kindId} isGroup label="应用类型">
        <RadioGroup type="pureCard" direction="vertical" name="application-kind" className="kind-picker" value={kind} onChange={event => form.setValue("kind")(String(event.target.value))} aria-labelledby={kindId + "-label"} disabled={busy}>
          <Radio value="confidential" extra="客户端密钥保存在其自己的服务器上。">服务端 Web 客户端</Radio>
          <Radio value="public" extra="浏览器或原生应用，不持有密钥，必须使用 PKCE。">公开客户端（PKCE）</Radio>
          <Radio value="resource" extra="内省令牌，并以自身 ID 作为受众。">资源 API 校验方</Radio>
        </RadioGroup>
      </Field>
      {!isResource ? <>
        <TextField form={form} field="redirects" label="精确回调地址（每行一个）" isMultiline placeholder="https://app.example.test/callback" disabled={busy} />
        <TextField form={form} field="logout" label="精确登出地址（每行一个）" isMultiline placeholder="https://app.example.test/signed-out" disabled={busy} />
      </> : null}
      <TextField form={form} field="scopes" label="允许的权限范围（逗号分隔）" placeholder="openid, profile, offline_access, notes.read" autoComplete="off" disabled={busy} />
      {!isResource ? <TextField form={form} field="audiences" label="资源 API 客户端 ID（逗号分隔）" placeholder="notes-api" autoComplete="off" disabled={busy} hint="请先注册资源 API。API 校验方以自身 ID 作为受众，且没有浏览器回调地址。" /> : null}
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
    if (!form.check({ user: required("请选择用户。"), client: required("请选择应用。") })) return;
    const { user, client, allow } = form.values;
    void run(async () => { await postJson("/api/assignments", { user, client, allow: allow === "yes" }, session.csrf); Toast.success("访问分配已更新。移除访问会撤销现有授权与授权码。"); await onDone(); });
  }
  if (!assignableApps.length) return <Panel title="分配应用访问"><StatusView illustration={<IllustrationNoContent />} title="没有可分配的应用" description="请先注册一个启用中的客户端应用再分配访问。资源 API 不分配给用户。" action={<Button theme="solid" icon={<IconPlus />} onClick={onRegister}>前往应用</Button>} /></Panel>;
  const allowId = form.idFor("allow");
  return <div className="panel-grid panel-grid-wide-left">
    <Panel title="分配应用访问" description="用户只能登录已分配给他的应用。">
      <form onSubmit={submit} noValidate>
        <SelectField form={form} field="user" label="用户" placeholder="选择用户" disabled={busy} options={users.filter(user => user.enabled).map(user => ({ value: user.id, label: user.username }))} />
        <SelectField form={form} field="client" label="应用" placeholder="选择应用" disabled={busy} options={assignableApps.map(app => ({ value: app.id, label: `${app.name} (${app.id})` }))} />
        <Field id={allowId} isGroup label="访问权限">
          <RadioGroup type="button" name="assignment-access" value={form.values.allow} onChange={event => form.setValue("allow")(String(event.target.value))} aria-labelledby={allowId + "-label"} disabled={busy}>
            <Radio value="yes">允许</Radio>
            <Radio value="no">移除并撤销</Radio>
          </RadioGroup>
        </Field>
        <div className="form-actions"><Button htmlType="submit" theme="solid" loading={busy}>保存分配</Button></div>
      </form>
    </Panel>
    <Panel title="访问规则">
      <ul className="explain-list">
        <li><IconKey aria-hidden="true" /><span>一条分配允许一个用户登录一个客户端应用，没有用户组或策略层。</span></li>
        <li><IconLock aria-hidden="true" /><span>移除访问会撤销该用户在此应用上的现有授权与待用授权码。</span></li>
        <li><IconShield aria-hidden="true" /><span>用户仍需在授权同意页上确认请求的权限范围。</span></li>
      </ul>
    </Panel>
  </div>;
}
