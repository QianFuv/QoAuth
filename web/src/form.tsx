/**
 * Lightweight form primitives around Semi field components.
 *
 * Semi's `Form` statically bundles every field type (date, tree, upload and more),
 * so these helpers keep labels, hints and inline errors without that weight.
 */
import React, { useCallback, useId, useState } from "react";
import Input from "@douyinfe/semi-ui/lib/es/input";
import TextArea from "@douyinfe/semi-ui/lib/es/input/textarea";

/** Validation rules keyed by field; each returns an error message or an empty string. */
export type Rules<Keys extends string> = Partial<Record<Keys, (value: string) => string>>;

/** State and helpers for one form. */
export interface FormState<Keys extends string> {
  values: Record<Keys, string>;
  errors: Partial<Record<Keys, string>>;
  setValue: (key: Keys) => (value: string) => void;
  check: (rules: Rules<Keys>) => boolean;
  reset: () => void;
  idFor: (key: Keys) => string;
}

/**
 * Track field values and inline errors for one form.
 *
 * @param initial - Initial value for every field.
 * @returns Values, errors and update, validation and reset helpers.
 */
export function useForm<Keys extends string>(initial: Record<Keys, string>): FormState<Keys> {
  const prefix = useId();
  const [values, setValues] = useState(initial);
  const [errors, setErrors] = useState<Partial<Record<Keys, string>>>({});
  const idFor = useCallback((key: Keys) => prefix + key, [prefix]);
  /** Update one field and clear its error. */
  const setValue = (key: Keys) => (value: string) => {
    setValues(current => ({ ...current, [key]: value }));
    setErrors(current => (current[key] ? { ...current, [key]: undefined } : current));
  };
  /** Validate every ruled field, show errors and focus the first invalid field. */
  function check(rules: Rules<Keys>): boolean {
    const next: Partial<Record<Keys, string>> = {};
    let firstInvalid: Keys | null = null;
    for (const key of Object.keys(rules) as Keys[]) {
      const message = rules[key]?.(values[key] ?? "") ?? "";
      if (message) { next[key] = message; firstInvalid ??= key; }
    }
    setErrors(next);
    if (firstInvalid) document.getElementById(idFor(firstInvalid))?.focus();
    return firstInvalid === null;
  }
  /** Clear values, including secrets, and errors. */
  function reset() { setValues(initial); setErrors({}); }
  return { values, errors, setValue, check, reset, idFor };
}

/** Rule helper for required fields. */
export function required(message: string): (value: string) => string {
  return value => (value === "" ? message : "");
}

/** Props shared by labeled fields. */
interface FieldProps { id: string; label: React.ReactNode; hint?: React.ReactNode; error?: string; isGroup?: boolean; children: React.ReactNode }

/**
 * Render a label, control, and either an inline error or a hint.
 *
 * @param props - Control id, label, hint, error, whether it labels a radio group, and the control.
 * @returns A labeled field block.
 */
export function Field({ id, label, hint, error, isGroup = false, children }: FieldProps) {
  return <div className={"field" + (error ? " field-invalid" : "")} role={isGroup ? "group" : undefined} aria-labelledby={isGroup ? id + "-label" : undefined}>
    {isGroup ? <span className="field-label" id={id + "-label"}>{label}</span> : <label className="field-label" htmlFor={id} id={id + "-label"}>{label}</label>}
    {children}
    {error ? <p className="field-error" id={id + "-message"} role="alert">{error}</p> : hint ? <p className="field-hint" id={id + "-message"}>{hint}</p> : null}
  </div>;
}

/** Props for a labeled Semi text input bound to a form field. */
type TextFieldProps<Keys extends string> = {
  form: FormState<Keys>;
  field: Keys;
  label: React.ReactNode;
  hint?: React.ReactNode;
  isMultiline?: boolean;
} & Omit<React.ComponentProps<typeof Input>, "value" | "onChange" | "id" | "form">;

/**
 * Render a Semi `Input` (or `TextArea`) wired to a form field with accessible messages.
 *
 * @param props - Form state, field key, label, hint and native input attributes.
 * @returns The labeled field.
 */
export function TextField<Keys extends string>({ form, field, label, hint, isMultiline = false, ...inputProps }: TextFieldProps<Keys>) {
  const id = form.idFor(field);
  const error = form.errors[field];
  const describedBy = error || hint ? id + "-message" : undefined;
  return <Field id={id} label={label} hint={hint} error={error}>
    {isMultiline
      ? <TextArea id={id} value={form.values[field]} onChange={form.setValue(field)} validateStatus={error ? "error" : "default"} aria-invalid={error ? true : undefined} aria-describedby={describedBy} autosize={{ minRows: 2, maxRows: 6 }} placeholder={typeof inputProps.placeholder === "string" ? inputProps.placeholder : undefined} disabled={inputProps.disabled} />
      : <Input id={id} {...inputProps} value={form.values[field]} onChange={form.setValue(field)} validateStatus={error ? "error" : "default"} aria-invalid={error ? true : undefined} aria-describedby={describedBy} />}
  </Field>;
}
