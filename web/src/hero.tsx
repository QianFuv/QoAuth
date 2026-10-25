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
  { id: "capabilities", label: "Capabilities" },
  { id: "protocol", label: "Protocol" },
  { id: "operations", label: "Operations" },
  { id: "scope", label: "Scope" },
];

/** Implemented capability groups; each line describes shipped behavior only. */
const CAPABILITIES = [
  { icon: <IconUserGroup />, title: "Local accounts", lines: ["Owner-created users — no public sign-up", "Enable, disable and administrator password reset", "NFC-normalized password rules shared by CLI, API and UI"] },
  { icon: <IconShield />, title: "MFA that cannot be skipped", lines: ["TOTP authenticator enrollment and verification", "Single-use recovery codes that re-enroll, not bypass", "Administrators must enroll before managing anything"] },
  { icon: <IconApps />, title: "Applications and consent", lines: ["Server-side, public PKCE and resource API clients", "Exact callback and signed-out URLs, explicit scopes", "Per-user assignment and an explicit consent screen"] },
  { icon: <IconKey />, title: "Sessions you can end", lines: ["One SSO session across assigned applications", "List and revoke sessions, withdraw consent", "Disabling a user or app revokes its grants"] },
  { icon: <IconHistory />, title: "Redacted activity", lines: ["Security events with action, subject and time", "Never credentials, codes or secrets", "Structured operational logs without tokens"] },
  { icon: <IconServer />, title: "One small runtime", lines: ["Single Go executable with the console embedded", "One SQLite database and one issuer", "Windows and Linux amd64 binaries, Linux image"] },
];

/** OIDC and OAuth behavior implemented by the provider. */
const PROTOCOL = [
  "Authorization code flow with S256 PKCE",
  "Discovery document and JWKS",
  "Signed ID tokens and UserInfo",
  "Scoped opaque access tokens with introspection",
  "Token revocation and refresh-token rotation",
  "RP-initiated logout to registered URLs",
];

/** Maintenance commands documented in the README. */
const COMMANDS = [
  { command: "qoauth init --data-dir ./state --username owner", note: "explicit one-time owner setup" },
  { command: "qoauth serve --data-dir ./state --issuer https://auth.example.test", note: "behind your HTTPS proxy" },
  { command: "qoauth backup --data-dir ./state --snapshot ./snapshot", note: "consistent stopped backup" },
  { command: "qoauth rotate-signing --data-dir ./state --confirm", note: "signing key rotation" },
];

/** What the product deliberately does not include. */
const NOT_INCLUDED = ["Public self-registration or email delivery", "Organizations, tenants or approval workflows", "LDAP, SAML, SCIM or social login", "Clustering, Redis or an external database", "An OpenID certification claim"];

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
    <a className="skip-link" href="#top">Skip to content</a>
    <header className={"site-header" + (isScrolled ? " is-scrolled" : "")}>
      <div className="site-header-inner">
        <Brand />
        {isCompact ? <Button theme="borderless" type="tertiary" icon={<IconMenu />} aria-label="Open menu" aria-expanded={isMenuOpen} onClick={() => setIsMenuOpen(true)} />
          : <>
            <nav className="site-nav" aria-label="Homepage sections">{links}</nav>
            <div className="site-actions">
              <Button theme="borderless" type="tertiary" onClick={() => window.location.assign("/login")}>Sign in</Button>
              <Button theme="solid" onClick={() => window.location.assign("/account")}>Open console</Button>
            </div>
          </>}
      </div>
    </header>
    {isCompact ? <SideSheet visible={isMenuOpen} placement="top" height="auto" title={<Brand />} onCancel={() => setIsMenuOpen(false)} className="site-menu">
      <nav className="site-menu-links" aria-label="Homepage sections">{links}</nav>
      <div className="site-menu-actions">
        <Button theme="solid" size="large" block onClick={() => window.location.assign("/account")}>Open console</Button>
        <Button size="large" block onClick={() => window.location.assign("/login")}>Sign in</Button>
      </div>
    </SideSheet> : null}

    <main id="top">
      <section className="hero">
        <div className="hero-inner">
          <div className="hero-copy">
            <p className="hero-eyebrow reveal">Self-hosted identity · OpenID Connect provider</p>
            <h1 className="hero-title reveal reveal-1">One sign-in for the apps <span>you run yourself.</span></h1>
            <p className="hero-lead reveal reveal-2">QoAuth is a lightweight identity service for a person or small team. One Go binary, one SQLite file and an embedded console give your applications local accounts, enforced MFA, explicit consent and standard OIDC.</p>
            <div className="hero-cta reveal reveal-3">
              <Button theme="solid" size="large" icon={<IconArrowRight />} iconPosition="right" onClick={() => window.location.assign("/account")}>Open console</Button>
              <Button size="large" type="tertiary" onClick={() => window.location.assign("/login")}>Sign in</Button>
            </div>
            <ul className="hero-facts reveal reveal-4" aria-label="Runtime">
              <li>Single executable</li><li>SQLite</li><li>No Node at runtime</li><li>Windows · Linux · Docker</li>
            </ul>
          </div>
          <FlowIllustration />
        </div>
      </section>

      <section className="section" id="capabilities" aria-labelledby="capabilities-title">
        <div className="section-inner">
          <div className="section-intro">
            <p className="section-eyebrow">Capabilities</p>
            <h2 className="section-title" id="capabilities-title">A complete small authentication system — not an IAM platform.</h2>
            <p className="section-lead">Everything below ships in the current build and is enforced on the server. The console only presents what the API already authorizes.</p>
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
            <p className="section-eyebrow">Protocol</p>
            <h2 className="section-title" id="protocol-title">Standard OIDC, deliberately narrow.</h2>
            <p className="section-lead">Any conforming relying party can integrate. Ambiguous parameters, unregistered scopes and wildcard callbacks are rejected instead of guessed.</p>
            <div className="endpoint">
              <span>Discovery</span>
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
          <div className="terminal" role="img" aria-label="Example QoAuth maintenance commands">
            <div className="terminal-bar" aria-hidden="true"><span /><span /><span /><IconTerminal /></div>
            <pre>{COMMANDS.map(entry => <span className="terminal-line" key={entry.command}><span className="terminal-prompt">$</span> {entry.command}{"\n"}<span className="terminal-note"># {entry.note}</span>{"\n"}</span>)}</pre>
          </div>
          <div>
            <p className="section-eyebrow">Operations</p>
            <h2 className="section-title" id="operations-title">Maintenance you can reason about.</h2>
            <ul className="check-list">
              <li><IconLock aria-hidden="true" /><span>An OS installation lock shared by every command.</span></li>
              <li><IconLock aria-hidden="true" /><span>Consistent backups and restore into a new directory only.</span></li>
              <li><IconLock aria-hidden="true" /><span>Signing-key rotation and bounded cleanup batches.</span></li>
              <li><IconLock aria-hidden="true" /><span><code>/readyz</code> reports only ready or not ready.</span></li>
              <li><IconLock aria-hidden="true" /><span>Local owner recovery that always requires a new factor.</span></li>
            </ul>
          </div>
        </div>
      </section>

      <section className="section section-muted" id="scope" aria-labelledby="scope-title">
        <div className="section-inner scope-layout">
          <div>
            <p className="section-eyebrow">Scope</p>
            <h2 className="section-title" id="scope-title">What QoAuth intentionally leaves out.</h2>
            <p className="section-lead">Staying small is the feature. These are outside the current commitment, not hidden roadmap items.</p>
          </div>
          <ul className="scope-tags">{NOT_INCLUDED.map(item => <li key={item}><Tag size="large" type="ghost">{item}</Tag></li>)}</ul>
        </div>
      </section>

      <section className="cta-band" aria-labelledby="cta-title">
        <div className="section-inner cta-inner">
          <h2 id="cta-title">Ready to sign in?</h2>
          <p>Administrators manage users and applications from the console. Everyone else manages their own sessions and sign-in methods.</p>
          <div className="hero-cta">
            <Button theme="solid" size="large" icon={<IconArrowRight />} iconPosition="right" onClick={() => window.location.assign("/account")}>Open console</Button>
            <Button size="large" className="button-on-dark" onClick={() => window.location.assign("/login")}>Sign in</Button>
          </div>
        </div>
      </section>
    </main>
    <footer className="site-footer">
      <div className="section-inner site-footer-inner">
        <Brand />
        <p>Lightweight self-hosted identity. Runs only where you install it.</p>
      </div>
    </footer>
  </div>;
}

/** Illustrate the real sign-in sequence with example names, not live data. */
function FlowIllustration() {
  return <figure className="flow reveal reveal-2" aria-label="Illustration of an application sign-in through QoAuth">
    <div className="flow-card flow-step-1">
      <span className="flow-label">1 · Application</span>
      <div className="flow-row"><span className="flow-app">N</span><div><strong>notes.example.test</strong><small>Redirects to QoAuth with PKCE</small></div></div>
    </div>
    <div className="flow-card flow-card-main flow-step-2">
      <span className="flow-label">2 · QoAuth</span>
      <strong className="flow-title">Continue to Notes</strong>
      <div className="flow-field"><span>Password</span><i /></div>
      <div className="flow-field"><span>Authenticator code</span><i className="flow-code">••• •••</i></div>
      <ul className="flow-scopes"><li><code>openid</code></li><li><code>profile</code></li><li><code>offline_access</code></li></ul>
      <span className="flow-button">Allow and continue</span>
    </div>
    <div className="flow-card flow-step-3">
      <span className="flow-label">3 · Callback</span>
      <div className="flow-row"><span className="flow-ok" aria-hidden="true">✓</span><div><strong>Code exchanged</strong><small>Exact registered callback only</small></div></div>
    </div>
    <figcaption>Illustration · example application</figcaption>
  </figure>;
}
