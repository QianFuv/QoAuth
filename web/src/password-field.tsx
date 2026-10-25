/** Shared new-password field; the server remains the policy authority. */
import React from "react";
import type { PasswordPolicy } from "./api";
import { TextField, type FormState } from "./form";
import { isPasswordWithinPolicy } from "./validation";

/**
 * Build the client-side rule for a new password.
 *
 * @param policy - The current server password policy.
 * @returns A rule that counts NFC code points and UTF-8 bytes, not UTF-16 length.
 */
export function passwordRule(policy: PasswordPolicy): (value: string) => string {
  return value => {
    if (value === "") return "请输入密码。";
    return isPasswordWithinPolicy(value, policy) ? "" : `Unicode 规范化后至少 ${policy.min_characters} 个字符，且不超过 ${policy.max_bytes} 个 UTF-8 字节。`;
  };
}

interface PasswordFieldProps<Keys extends string> { form: FormState<Keys>; field: Keys; label: string; policy: PasswordPolicy; disabled?: boolean }

/**
 * Render a Semi password input with the published policy as guidance.
 *
 * @param props - Form state, field key, label and current server policy.
 * @returns A labeled new-password field.
 */
export function PasswordField<Keys extends string>({ form, field, label, policy, disabled }: PasswordFieldProps<Keys>) {
  return <TextField form={form} field={field} label={label} mode="password" name="new-password" autoComplete="new-password" disabled={disabled}
    hint={`至少 ${policy.min_characters} 个字符，最多 ${policy.max_bytes} 个 UTF-8 字节。常见密码会被拒绝；支持空格和密码管理器粘贴。`} />;
}
