/**
 * Pure form helpers shared by the UI and its Node test runner.
 *
 * This module keeps the `.ts` extension so Node's built-in type stripping can
 * execute it directly in tests without adding a test framework dependency.
 */

/** Server-published password limits; the server remains the policy authority. */
export interface PasswordLimits {
  min_characters: number;
  max_bytes: number;
}

/**
 * Check a new password with the same NFC character and UTF-8 byte rules as the server.
 *
 * @param password - The candidate password, never trimmed or stored.
 * @param limits - The current server password policy.
 * @returns True when the password satisfies the published length limits.
 */
export function isPasswordWithinPolicy(password: string, limits: PasswordLimits): boolean {
  const encoder = new TextEncoder();
  const normalized = password.normalize("NFC");
  return Array.from(normalized).length >= limits.min_characters
    && encoder.encode(password).length <= limits.max_bytes
    && encoder.encode(normalized).length <= limits.max_bytes;
}

/**
 * Split an explicitly entered comma-separated scope or audience list.
 *
 * @param value - Raw text entered by an administrator.
 * @returns Trimmed, non-empty entries in their original order.
 */
export function splitCommaList(value: unknown): string[] {
  return String(value ?? "").split(",").map(item => item.trim()).filter(Boolean);
}

/**
 * Split exact callback URLs at line breaks so literal commas inside URLs survive.
 *
 * @param value - Raw multi-line text entered by an administrator.
 * @returns Trimmed, non-empty URLs in their original order.
 */
export function splitLines(value: unknown): string[] {
  return String(value ?? "").split(/\r?\n/).map(item => item.trim()).filter(Boolean);
}

const FRIENDLY_ERRORS: Record<string, string> = {
  "invalid credentials": "Incorrect username or password.",
  "try again later": "Too many attempts. Wait a moment and try again.",
  "invalid proof": "That code was not accepted. Check the code and try again.",
  "invalid origin or CSRF token": "Your session changed in another tab. Reload this page and try again.",
  "fresh administrator MFA required": "Sign in again to make sensitive administrative changes.",
  "fresh login and unenrolled factor required": "Sign in again before setting up an authenticator.",
  "fresh proof required": "Sign in again before using a recovery code.",
  "complete MFA first": "Complete multi-factor verification first.",
  "authentication required": "Sign in to continue.",
  "session required": "Your session ended. Sign in again.",
  "operation rejected": "The server rejected this change. Review the values and try again.",
  "invalid application configuration": "The application settings were rejected. Check client ID, callbacks, scopes and resource APIs.",
  "invalid username": "Usernames start with a letter or digit and use only letters, digits, dot, dash or underscore.",
};

/**
 * Turn a short plain-text server error into a sentence for display.
 *
 * @param raw - Response text or error message returned by a failed request.
 * @returns A readable message that never adds information the server did not send.
 */
export function describeError(raw: string): string {
  const message = raw.trim();
  if (!message) return "Request failed. Try again.";
  const known = FRIENDLY_ERRORS[message];
  if (known) return known;
  const sentence = message.charAt(0).toUpperCase() + message.slice(1);
  return /[.!?]$/.test(sentence) ? sentence : sentence + ".";
}
