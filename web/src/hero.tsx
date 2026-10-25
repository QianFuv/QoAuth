/**
 * Public product homepage.
 *
 * It is fully static: it makes no API request, reads no session and renders no
 * installation data, so nothing about users, applications or activity can leak.
 */
import React, { useEffect, useState } from "react";
import Button from "@douyinfe/semi-ui/lib/es/button";
import Tag from "@douyinfe/semi-ui/lib/es/tag";
import SideSheet from "@douyinfe/semi-ui/lib/es/sideSheet";
import { IconApps, IconArrowRight, IconHistory, IconKey, IconLock, IconMenu, IconServer, IconShield, IconTerminal, IconUserGroup } from "@douyinfe/semi-icons";
import { Brand, useMediaQuery } from "./layout";
import "./hero.css";

/** Section anchors shown in the homepage navigation. */
const SECTIONS = [
  { id: "capabilities", label: "能力" },
  { id: "protocol", label: "协议" },
  { id: "operations", label: "运维" },
  { id: "scope", label: "边界" },
];

/** Implemented capability groups; each line describes shipped behavior only. */
const CAPABILITIES = [
  { icon: <IconUserGroup />, title: "本地账户", lines: ["由所有者创建用户，不开放公开注册", "启用、停用账户，管理员可重置密码", "CLI、API 与界面共用 NFC 规范化密码规则"] },
  { icon: <IconShield />, title: "无法绕过的 MFA", lines: ["TOTP 验证器绑定与校验", "一次性恢复码用于重新绑定，而非绕过验证", "管理员必须先绑定验证器才能管理"] },
  { icon: <IconApps />, title: "应用与授权同意", lines: ["服务端客户端、公开 PKCE 客户端与资源 API", "精确回调与登出地址，显式权限范围", "按用户分配访问，并有明确的授权同意页"] },
  { icon: <IconKey />, title: "可随时终止的会话", lines: ["一次 SSO 会话覆盖已分配的应用", "查看并撤销会话，撤回授权同意", "停用用户或应用会撤销其授权"] },
  { icon: <IconHistory />, title: "脱敏活动记录", lines: ["记录事件、主体与时间的安全事件", "从不记录凭据、验证码或密钥", "结构化运行日志不含令牌"] },
  { icon: <IconServer />, title: "轻量运行时", lines: ["单个 Go 可执行文件，内嵌控制台", "一个 SQLite 数据库，一个签发者", "Windows 与 Linux amd64 二进制及 Linux 镜像"] },
];

/** OIDC and OAuth behavior implemented by the provider. */
const PROTOCOL = [
  "授权码流程，强制 S256 PKCE",
  "Discovery 文档与 JWKS",
  "签名 ID Token 与 UserInfo",
  "带权限范围的不透明访问令牌与令牌内省",
  "令牌撤销与刷新令牌轮换",
  "RP 发起登出，仅跳转到已注册地址",
];

/** Maintenance commands documented in the README. */
const COMMANDS = [
  { command: "qoauth init --data-dir ./state --username owner", note: "显式、一次性的所有者初始化" },
  { command: "qoauth serve --data-dir ./state --issuer https://auth.example.test", note: "部署在你的 HTTPS 反向代理之后" },
  { command: "qoauth backup --data-dir ./state --snapshot ./snapshot", note: "停机状态下的一致性备份" },
  { command: "qoauth rotate-signing --data-dir ./state --confirm", note: "轮换签名密钥" },
];

/** What the product deliberately does not include. */
const NOT_INCLUDED = ["公开自助注册或邮件发送", "组织、多租户或审批流程", "LDAP、SAML、SCIM 或社交登录", "集群、Redis 或外部数据库", "OpenID 认证声明"];

/**
 * Render the homepage with a clear route to sign-in and the console.
 *
 * @returns The public product page.
 */
export function Hero() {
  const isCompact = useMediaQuery("(max-width: 860px)");
  const [isMenuOpen, setIsMenuOpen] = useState(false);
  const [isScrolled, setIsScrolled] = useState(false);
  useEffect(() => {
    /** Add a hairline under the header once the page scrolls. */
    function update() { setIsScrolled(window.scrollY > 8); }
    update();
    window.addEventListener("scroll", update, { passive: true });
    return () => window.removeEventListener("scroll", update);
  }, []);
  useEffect(() => { if (!isCompact) setIsMenuOpen(false); }, [isCompact]);
  const links = SECTIONS.map(section => <a key={section.id} href={"#" + section.id} onClick={() => setIsMenuOpen(false)}>{section.label}</a>);
  return <div className="hero-page">
    <a className="skip-link" href="#top">跳到正文</a>
    <header className={"site-header" + (isScrolled ? " is-scrolled" : "")}>
      <div className="site-header-inner">
        <Brand />
        {isCompact ? <Button theme="borderless" type="tertiary" icon={<IconMenu />} aria-label="打开菜单" aria-expanded={isMenuOpen} onClick={() => setIsMenuOpen(true)} />
          : <>
            <nav className="site-nav" aria-label="主页栏目">{links}</nav>
            <div className="site-actions">
              <Button theme="borderless" type="tertiary" onClick={() => window.location.assign("/login")}>登录</Button>
              <Button theme="solid" onClick={() => window.location.assign("/account")}>进入控制台</Button>
            </div>
          </>}
      </div>
    </header>
    {isCompact ? <SideSheet visible={isMenuOpen} placement="top" height="auto" title={<Brand />} onCancel={() => setIsMenuOpen(false)} className="site-menu">
      <nav className="site-menu-links" aria-label="主页栏目">{links}</nav>
      <div className="site-menu-actions">
        <Button theme="solid" size="large" block onClick={() => window.location.assign("/account")}>进入控制台</Button>
        <Button size="large" block onClick={() => window.location.assign("/login")}>登录</Button>
      </div>
    </SideSheet> : null}

    <main id="top">
      <section className="hero">
        <div className="hero-inner">
          <div className="hero-copy">
            <p className="hero-eyebrow reveal">自托管身份认证 · OpenID Connect 提供方</p>
            <h1 className="hero-title reveal reveal-1">一次登录，<wbr />通行<span>你自建的<wbr />所有应用。</span></h1>
            <p className="hero-lead reveal reveal-2">QoAuth 是面向个人与小团队的轻量身份认证服务。一个 Go 程序、一个 SQLite 文件和内嵌控制台，为你的应用提供本地账户、强制 MFA、明确的授权同意与标准 OIDC。</p>
            <div className="hero-cta reveal reveal-3">
              <Button theme="solid" size="large" icon={<IconArrowRight />} iconPosition="right" onClick={() => window.location.assign("/account")}>进入控制台</Button>
              <Button size="large" type="tertiary" onClick={() => window.location.assign("/login")}>登录</Button>
            </div>
            <ul className="hero-facts reveal reveal-4" aria-label="运行环境">
              <li>单个可执行文件</li><li>SQLite</li><li>运行时无需 Node</li><li>Windows · Linux · Docker</li>
            </ul>
          </div>
          <FlowIllustration />
        </div>
      </section>

      <section className="section" id="capabilities" aria-labelledby="capabilities-title">
        <div className="section-inner">
          <div className="section-intro">
            <p className="section-eyebrow">能力</p>
            <h2 className="section-title" id="capabilities-title">完整而小巧的认证系统，而不是企业 IAM 平台。</h2>
            <p className="section-lead">以下能力均已在当前版本中实现，并由服务端强制执行。控制台只展示 API 已授权的内容。</p>
          </div>
          <div className="capability-grid">
            {CAPABILITIES.map((capability, index) => <article className={"capability" + (index === 0 ? " capability-feature" : "")} key={capability.title}>
              <span className="capability-icon" aria-hidden="true">{capability.icon}</span>
              <h3>{capability.title}</h3>
              <ul>{capability.lines.map(line => <li key={line}>{line}</li>)}</ul>
            </article>)}
          </div>
        </div>
      </section>

      <section className="section section-dark" id="protocol" aria-labelledby="protocol-title">
        <div className="section-inner protocol-layout">
          <div>
            <p className="section-eyebrow">协议</p>
            <h2 className="section-title" id="protocol-title">标准 OIDC，刻意收窄。</h2>
            <p className="section-lead">任何符合规范的依赖方都可以接入。含糊的参数、未注册的权限范围和通配回调地址会被直接拒绝，而不是被猜测。</p>
            <div className="endpoint">
              <span>发现端点</span>
              <code>/.well-known/openid-configuration</code>
            </div>
          </div>
          <ol className="protocol-list">
            {PROTOCOL.map((item, index) => <li key={item}><span className="protocol-index">{String(index + 1).padStart(2, "0")}</span><span>{item}</span></li>)}
          </ol>
        </div>
      </section>

      <section className="section" id="operations" aria-labelledby="operations-title">
        <div className="section-inner operations-layout">
          <div className="terminal" role="img" aria-label="QoAuth 维护命令示例">
            <div className="terminal-bar" aria-hidden="true"><span /><span /><span /><IconTerminal /></div>
            <pre>{COMMANDS.map(entry => <span className="terminal-line" key={entry.command}><span className="terminal-prompt">$</span> {entry.command}{"\n"}<span className="terminal-note"># {entry.note}</span>{"\n"}</span>)}</pre>
          </div>
          <div>
            <p className="section-eyebrow">运维</p>
            <h2 className="section-title" id="operations-title">可预期、可推理的运维。</h2>
            <ul className="check-list">
              <li><IconLock aria-hidden="true" /><span>所有命令共用操作系统级安装锁。</span></li>
              <li><IconLock aria-hidden="true" /><span>一致性备份，恢复只写入新目录。</span></li>
              <li><IconLock aria-hidden="true" /><span>签名密钥轮换与有上限的批量清理。</span></li>
              <li><IconLock aria-hidden="true" /><span><code>/readyz</code> 只报告就绪或未就绪。</span></li>
              <li><IconLock aria-hidden="true" /><span>本地所有者恢复始终要求重新绑定验证器。</span></li>
            </ul>
          </div>
        </div>
      </section>

      <section className="section section-muted" id="scope" aria-labelledby="scope-title">
        <div className="section-inner scope-layout">
          <div>
            <p className="section-eyebrow">边界</p>
            <h2 className="section-title" id="scope-title">QoAuth 刻意不做的事。</h2>
            <p className="section-lead">保持小巧本身就是特性。以下内容不在当前承诺范围内，也不是隐藏的路线图。</p>
          </div>
          <ul className="scope-tags">{NOT_INCLUDED.map(item => <li key={item}><Tag size="large" type="ghost">{item}</Tag></li>)}</ul>
        </div>
      </section>

      <section className="cta-band" aria-labelledby="cta-title">
        <div className="section-inner cta-inner">
          <h2 id="cta-title">准备好登录了吗？</h2>
          <p>管理员在控制台中管理用户与应用；其他用户管理自己的会话与登录方式。</p>
          <div className="hero-cta">
            <Button theme="solid" size="large" icon={<IconArrowRight />} iconPosition="right" onClick={() => window.location.assign("/account")}>进入控制台</Button>
            <Button size="large" className="button-on-dark" onClick={() => window.location.assign("/login")}>登录</Button>
          </div>
        </div>
      </section>
    </main>
    <footer className="site-footer">
      <div className="section-inner site-footer-inner">
        <Brand />
        <p>轻量自托管身份认证，只运行在你部署的地方。</p>
      </div>
    </footer>
  </div>;
}

/** Illustrate the real sign-in sequence with example names, not live data. */
function FlowIllustration() {
  return <figure className="flow reveal reveal-2" aria-label="应用通过 QoAuth 登录的示意图">
    <div className="flow-card flow-step-1">
      <span className="flow-label">1 · 应用</span>
      <div className="flow-row"><span className="flow-app">N</span><div><strong>notes.example.test</strong><small>携带 PKCE 跳转到 QoAuth</small></div></div>
    </div>
    <div className="flow-card flow-card-main flow-step-2">
      <span className="flow-label">2 · QoAuth</span>
      <strong className="flow-title">继续前往 Notes</strong>
      <div className="flow-field"><span>密码</span><i /></div>
      <div className="flow-field"><span>验证器动态码</span><i className="flow-code">••• •••</i></div>
      <ul className="flow-scopes"><li><code>openid</code></li><li><code>profile</code></li><li><code>offline_access</code></li></ul>
      <span className="flow-button">同意并继续</span>
    </div>
    <div className="flow-card flow-step-3">
      <span className="flow-label">3 · 回调</span>
      <div className="flow-row"><span className="flow-ok" aria-hidden="true">✓</span><div><strong>授权码已兑换</strong><small>只回到精确注册的回调地址</small></div></div>
    </div>
    <figcaption>示意图 · 示例应用</figcaption>
  </figure>;
}
