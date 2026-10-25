/** Shared new-password feedback; the server remains the policy authority. */
import React, { useId } from "react";
import type { PasswordPolicy } from "./api";

interface PasswordFieldProps { label: string; policy: PasswordPolicy }

/** Count NFC code points and UTF-8 bytes rather than HTML's UTF-16 length. */
export function PasswordField({ label, policy }: PasswordFieldProps) {
  const hintId = useId();
  /** Update browser validity without storing or trimming the password. */
  function validate(event: React.FormEvent<HTMLInputElement>) {
    const input = event.currentTarget;
    const normalized = input.value.normalize("NFC");
    const encoder = new TextEncoder();
    const isValid = Array.from(normalized).length >= policy.min_characters && encoder.encode(input.value).length <= policy.max_bytes && encoder.encode(normalized).length <= policy.max_bytes;
    input.setCustomValidity(isValid ? "" : `Use at least ${policy.min_characters} characters after Unicode normalization and at most ${policy.max_bytes} UTF-8 bytes.`);
  }
  return <><label>{label}<input name="password" type="password" required autoComplete="new-password" aria-describedby={hintId} onInput={validate} /></label><p id={hintId} className="hint">At least {policy.min_characters} characters after Unicode normalization; at most {policy.max_bytes} UTF-8 bytes. Common passwords are rejected. Spaces and password-manager paste are supported.</p></>;
}
