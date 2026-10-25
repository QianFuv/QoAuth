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
    if (value === "") return "Enter a password.";
    return isPasswordWithinPolicy(value, policy) ? "" : `Use at least ${policy.min_characters} characters after Unicode normalization and at most ${policy.max_bytes} UTF-8 bytes.`;
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
    hint={`At least ${policy.min_characters} characters; at most ${policy.max_bytes} UTF-8 bytes. Common passwords are rejected. Spaces and password-manager paste work.`} />;
}
