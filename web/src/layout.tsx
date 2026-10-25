/** Shared brand, routing, responsive and authentication-frame primitives without heavy UI. */
import React, { useEffect, useState } from "react";
import { IconApps, IconKey, IconShield } from "@douyinfe/semi-icons";

/**
 * Render the QoAuth wordmark.
 *
 * @param props - Optional link target and tone for dark surfaces.
 * @returns The brand mark as a home link.
 */
export function Brand({ href = "/", isInverted = false }: { href?: string; isInverted?: boolean }) {
  return <a className={"brand" + (isInverted ? " brand-inverted" : "")} href={href} aria-label="QoAuth home">
    <span className="brand-mark" aria-hidden="true"><span /></span>
    <span className="brand-name">QoAuth</span>
  </a>;
}

/** Location fields that drive client-side rendering. */
export interface Location { path: string; hash: string; search: string }

/** Read the current browser location. */
function readLocation(): Location {
  return { path: window.location.pathname, hash: window.location.hash.replace(/^#/, ""), search: window.location.search };
}

/**
 * Track same-document navigation between server-routed console pages.
 *
 * @returns The current location, updated on history changes.
 */
export function useLocation(): Location {
  const [location, setLocation] = useState(readLocation);
  useEffect(() => {
    /** Re-read the location after back/forward or programmatic navigation. */
    function update() { setLocation(readLocation()); }
    window.addEventListener("popstate", update);
    window.addEventListener("hashchange", update);
    return () => { window.removeEventListener("popstate", update); window.removeEventListener("hashchange", update); };
  }, []);
  return location;
}

/**
 * Navigate without reloading between `/account` and `/admin`, which the server serves identically.
 *
 * @param target - A same-origin console path, optionally with a hash section.
 */
export function navigate(target: string) {
  window.history.pushState(null, "", target);
  window.dispatchEvent(new PopStateEvent("popstate"));
  window.scrollTo({ top: 0 });
}

/**
 * Subscribe to a CSS media query.
 *
 * @param query - The media query to evaluate.
 * @returns Whether the query currently matches.
 */
export function useMediaQuery(query: string): boolean {
  const [isMatch, setIsMatch] = useState(() => window.matchMedia(query).matches);
  useEffect(() => {
    const media = window.matchMedia(query);
    /** Mirror the query state into React. */
    function update() { setIsMatch(media.matches); }
    update();
    media.addEventListener("change", update);
    return () => media.removeEventListener("change", update);
  }, [query]);
  return isMatch;
}

/** Breakpoint below which tables become stacked lists and the sidebar becomes a drawer. */
export const COMPACT_QUERY = "(max-width: 767px)";

/** Breakpoint below which the console sidebar becomes a drawer. */
export const DRAWER_QUERY = "(max-width: 1023px)";

/**
 * Frame sign-in, verification, consent and logout pages.
 *
 * @param props - Page content and an optional aside heading for wide screens.
 * @returns A split layout on wide screens and a single focused column on small screens.
 */
export function AuthLayout({ children, aside }: { children: React.ReactNode; aside?: React.ReactNode }) {
  return <div className="auth-layout">
    <aside className="auth-aside" aria-hidden={aside ? undefined : true}>
      <Brand isInverted />
      <div className="auth-aside-body">{aside ?? <AuthAsideDefault />}</div>
      <p className="auth-aside-foot">Self-hosted identity · one issuer · local accounts</p>
    </aside>
    <main className="auth-main">
      <div className="auth-mobile-brand"><Brand /></div>
      <div className="auth-panel enter">{children}</div>
      <p className="auth-footnote"><a href="/">About QoAuth</a></p>
    </main>
  </div>;
}

/** Default wide-screen statement made only of implemented behavior. */
function AuthAsideDefault() {
  return <>
    <h2 className="auth-aside-title">One sign-in for the applications you run yourself.</h2>
    <ul className="auth-aside-list">
      <li><IconKey aria-hidden="true" /><span>Passwords are verified only by this QoAuth server.</span></li>
      <li><IconShield aria-hidden="true" /><span>Administrators must use an authenticator app.</span></li>
      <li><IconApps aria-hidden="true" /><span>You approve which permissions each application receives.</span></li>
    </ul>
  </>;
}

/**
 * Format a Unix time for display in the browser's locale.
 *
 * @param seconds - Unix seconds.
 * @returns A localized date and time.
 */
export function formatTime(seconds: number): string {
  return new Date(seconds * 1000).toLocaleString(undefined, { dateStyle: "medium", timeStyle: "short" });
}
