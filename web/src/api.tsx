/** Typed browser access to same-origin, session-bound QoAuth endpoints. */
export interface Account { id: string; username: string; admin: boolean; enabled: boolean; owner: boolean; require_mfa: boolean }
export interface SessionState { csrf: string; account: Account; authenticated: boolean; assurance: boolean; has_factor: boolean; fresh: boolean }
export interface Application { id: string; name: string; scopes: string[]; audiences: string[]; redirects: string[]; resource: boolean; public: boolean }
export interface SessionRecord { Digest: string; AuthTime: number; Expires: number; Assurance: boolean }
export interface Activity { action: string; subject: string; time: number }

/** Read JSON without persisting tokens or sensitive response data in browser storage. */
export async function getJson<T>(path: string): Promise<T> {
  const response = await fetch(path, { credentials: "same-origin", cache: "no-store" });
  if (!response.ok) throw new Error(await response.text());
  return response.json() as Promise<T>;
}

/** Submit one CSRF-protected mutation; the browser supplies the canonical Origin. */
export async function postJson<T>(path: string, value: unknown, csrf: string): Promise<T> {
  const response = await fetch(path, { method: "POST", credentials: "same-origin", cache: "no-store", headers: { "Content-Type": "application/json", "X-CSRF-Token": csrf }, body: JSON.stringify(value) });
  if (!response.ok) throw new Error(await response.text());
  return response.json() as Promise<T>;
}
