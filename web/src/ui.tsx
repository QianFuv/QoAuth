/** Console frame, panels and shared status/copy blocks built from Semi components. */
import React, { useEffect, useState } from "react";
import Nav from "@douyinfe/semi-ui/lib/es/navigation";
import SideSheet from "@douyinfe/semi-ui/lib/es/sideSheet";
import Button from "@douyinfe/semi-ui/lib/es/button";
import Avatar from "@douyinfe/semi-ui/lib/es/avatar";
import Empty from "@douyinfe/semi-ui/lib/es/empty";
import Typography from "@douyinfe/semi-ui/lib/es/typography";
import { IconApps, IconExit, IconGridView, IconHistory, IconKey, IconMenu, IconUser, IconUserGroup } from "@douyinfe/semi-icons";
import type { SessionState } from "./api";
import { Brand, DRAWER_QUERY, navigate, useMediaQuery } from "./layout";

const { Text } = Typography;

/** Console navigation keys mapped to same-origin paths. */
const NAV_TARGETS: Record<string, string> = {
  account: "/account",
  "admin-overview": "/admin",
  "admin-users": "/admin#users",
  "admin-applications": "/admin#applications",
  "admin-access": "/admin#access",
  "admin-activity": "/admin#activity",
};

/** Props for the authenticated console frame. */
interface ConsoleShellProps {
  session: SessionState;
  selectedKey: string;
  isBusy: boolean;
  onSignOut: () => void;
  children: React.ReactNode;
}

/**
 * Frame authenticated pages with a sidebar on wide screens and a drawer on small screens.
 *
 * @param props - Session identity, active navigation key, sign-out handler and page content.
 * @returns The console layout.
 */
export function ConsoleShell({ session, selectedKey, isBusy, onSignOut, children }: ConsoleShellProps) {
  const isCompact = useMediaQuery(DRAWER_QUERY);
  const [isDrawerOpen, setIsDrawerOpen] = useState(false);
  useEffect(() => { if (!isCompact) setIsDrawerOpen(false); }, [isCompact]);
  const items = [
    { itemKey: "account", text: "我的账户", icon: <IconUser /> },
    ...(session.account.admin ? [
      { itemKey: "admin-overview", text: "概览", icon: <IconGridView /> },
      { itemKey: "admin-users", text: "用户", icon: <IconUserGroup /> },
      { itemKey: "admin-applications", text: "应用", icon: <IconApps /> },
      { itemKey: "admin-access", text: "访问分配", icon: <IconKey /> },
      { itemKey: "admin-activity", text: "活动记录", icon: <IconHistory /> },
    ] : []),
  ];
  /** Move to the chosen console section and close the drawer. */
  function select(key: string) {
    const target = NAV_TARGETS[key];
    if (target) navigate(target);
    setIsDrawerOpen(false);
  }
  const navigation = <Nav
    className="console-nav"
    mode="vertical"
    selectedKeys={[selectedKey]}
    items={items}
    onSelect={data => select(String(data.itemKey))}
    footer={{ collapseButton: false }}
  />;
  const identity = <div className="console-identity">
    <Avatar size="small" color="blue" alt={session.account.username}>{session.account.username.slice(0, 1).toUpperCase()}</Avatar>
    <div className="console-identity-text">
      <Text strong ellipsis={{ showTooltip: true }}>{session.account.username}</Text>
      <Text type="tertiary" size="small">{session.account.owner ? "所有者" : session.account.admin ? "管理员" : "成员"}</Text>
    </div>
    <Button theme="borderless" type="tertiary" icon={<IconExit />} aria-label="退出登录" disabled={isBusy} onClick={onSignOut} />
  </div>;
  return <div className="console">
    {isCompact ? <header className="console-topbar">
      <Brand href="/account" />
      <Button theme="borderless" type="tertiary" icon={<IconMenu />} aria-label="打开导航" aria-expanded={isDrawerOpen} onClick={() => setIsDrawerOpen(true)} />
    </header> : <aside className="console-sidebar">
      <div className="console-sidebar-brand"><Brand href="/account" /></div>
      {session.account.admin ? <p className="console-nav-label">工作台</p> : null}
      {navigation}
      <div className="console-sidebar-foot">{identity}</div>
    </aside>}
    {isCompact ? <SideSheet
      className="console-drawer"
      visible={isDrawerOpen}
      placement="left"
      width="min(320px, 86vw)"
      title={<Brand href="/account" />}
      onCancel={() => setIsDrawerOpen(false)}
      bodyStyle={{ padding: 0, display: "flex", flexDirection: "column" }}
    >
      {navigation}
      <div className="console-sidebar-foot">{identity}</div>
    </SideSheet> : null}
    <main className="console-main" id="main">{children}</main>
  </div>;
}

/**
 * Render a console page heading with optional status tags and actions.
 *
 * @param props - Eyebrow, title, description, tags and action controls.
 * @returns The page header.
 */
export function PageHeader({ eyebrow, title, description, tags, actions }: { eyebrow: string; title: React.ReactNode; description?: React.ReactNode; tags?: React.ReactNode; actions?: React.ReactNode }) {
  return <header className="page-header enter">
    <div className="page-header-text">
      <p className="eyebrow">{eyebrow}</p>
      <h1 className="page-title">{title}</h1>
      {description ? <p className="page-description">{description}</p> : null}
      {tags ? <div className="page-tags">{tags}</div> : null}
    </div>
    {actions ? <div className="page-actions">{actions}</div> : null}
  </header>;
}

/**
 * Group related console content with a heading and optional toolbar.
 *
 * @param props - Section title, description, toolbar and content.
 * @returns A bordered content region.
 */
export function Panel({ title, description, toolbar, children, id, className }: { title: React.ReactNode; description?: React.ReactNode; toolbar?: React.ReactNode; children: React.ReactNode; id?: string; className?: string }) {
  return <section className={"panel enter" + (className ? " " + className : "")} id={id} aria-labelledby={id ? id + "-title" : undefined}>
    <div className="panel-head">
      <div>
        <h2 className="panel-title" id={id ? id + "-title" : undefined}>{title}</h2>
        {description ? <p className="panel-description">{description}</p> : null}
      </div>
      {toolbar ? <div className="panel-toolbar">{toolbar}</div> : null}
    </div>
    <div className="panel-body">{children}</div>
  </section>;
}

/**
 * Show a one-time value in monospace with a Semi copy control.
 *
 * @param props - The value, an accessible label and whether to preserve line breaks.
 * @returns A copyable value block.
 */
export function CopyBlock({ value, label, isMultiline = false }: { value: string; label: string; isMultiline?: boolean }) {
  return <div className={"copy-block" + (isMultiline ? " copy-block-multiline" : "")}>
    <code aria-label={label}>{value}</code>
    <Text className="copy-block-action" copyable={{ content: value, copyTip: "复制", successTip: "已复制" }} />
  </div>;
}

/**
 * Render a centered status view for empty, denied, failed and expired states.
 *
 * @param props - Illustration, title, description and optional action.
 * @returns A Semi Empty state.
 */
export function StatusView({ illustration, title, description, action }: { illustration: React.ReactNode; title: string; description: React.ReactNode; action?: React.ReactNode }) {
  return <div className="status-view enter">
    <Empty image={illustration} title={title} description={description} />
    {action ? <div className="status-view-action">{action}</div> : null}
  </div>;
}
