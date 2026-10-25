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
  "invalid credentials": "用户名或密码不正确。",
  "try again later": "尝试次数过多，请稍后再试。",
  "invalid proof": "验证码未通过，请检查后重试。",
  "invalid origin or CSRF token": "会话已在其他标签页中变化，请刷新本页后重试。",
  "fresh administrator MFA required": "敏感管理操作需要重新登录。",
  "fresh login and unenrolled factor required": "设置验证器前请重新登录。",
  "fresh proof required": "使用恢复码前请重新登录。",
  "complete MFA first": "请先完成多因素验证。",
  "authentication required": "请登录后继续。",
  "session required": "会话已结束，请重新登录。",
  "session unavailable": "会话不可用，请重新登录。",
  "authentication unavailable": "暂时无法登录，请稍后再试。",
  "operation rejected": "服务器拒绝了这项更改，请检查输入后重试。",
  "invalid request": "请求格式无效，请刷新页面后重试。",
  "invalid application configuration": "应用设置未通过校验，请检查客户端 ID、回调地址、权限范围和资源 API。",
  "invalid username": "用户名须以字母或数字开头，只能包含字母、数字、点、短横线或下划线。",
  "invalid authorization transaction": "授权请求已失效，请返回应用重新登录。",
  "application unavailable": "该应用当前不可用。",
  "authorization rejected": "授权被拒绝。",
  "forbidden": "没有执行此操作的权限。",
  "unavailable": "服务暂时不可用，请稍后再试。",
  "method not allowed": "不支持此操作。",
};

/** Chinese explanations for the server's `password policy:` error details. */
const PASSWORD_POLICY_ERRORS: Array<[string, string]> = [
  ["choose a less common password", "密码过于常见，请换一个。"],
  ["at least", "密码长度不符合要求：Unicode 规范化后的字符数或 UTF-8 字节数超出限制。"],
  ["use valid UTF-8", "密码包含无效字符或超过 UTF-8 字节上限。"],
];

/**
 * Turn a short plain-text server error into a Chinese sentence for display.
 *
 * @param raw - Response text or error message returned by a failed request.
 * @returns A readable message; unknown errors keep the server's original text.
 */
export function describeError(raw: string): string {
  const message = raw.trim();
  if (!message) return "请求未成功，请重试。";
  const known = FRIENDLY_ERRORS[message];
  if (known) return known;
  if (message.startsWith("password policy")) {
    const match = PASSWORD_POLICY_ERRORS.find(([fragment]) => message.includes(fragment));
    if (match) return match[1];
  }
  return "请求未成功：" + message;
}
