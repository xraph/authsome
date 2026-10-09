import type { SignupFieldConfig } from "@authsome/ui-core";

export function validEmail(value: string): boolean {
  return /^[^\s@]+@[^\s@]+$/.test(value);
}

export function validateSignupField(
  field: SignupFieldConfig,
  value: string,
): string | null {
  const text = value.trim();
  const rules = field.validation;
  const toggle = field.type === "checkbox" || field.type === "switch";
  if (rules?.required && (!text || (toggle && text !== "true")))
    return `${field.label} is required.`;
  if (!text) return null;
  if (
    (field.type === "select" || field.type === "radio") &&
    !field.options?.some((option) => option.value === text)
  )
    return `Choose a valid ${field.label.toLowerCase()}.`;
  if (field.type === "email" && !validEmail(text))
    return `${field.label} must be an email address.`;
  if (rules?.min_len != null && [...text].length < rules.min_len)
    return `${field.label} must be at least ${rules.min_len} characters.`;
  if (rules?.max_len != null && [...text].length > rules.max_len)
    return `${field.label} must be at most ${rules.max_len} characters.`;
  if (field.type === "number" || rules?.min != null || rules?.max != null) {
    const number = Number(text);
    if (
      !Number.isFinite(number) ||
      (field.type === "number" && !Number.isInteger(number))
    )
      return `${field.label} must be a number.`;
    if (rules?.min != null && number < rules.min)
      return `${field.label} must be at least ${rules.min}.`;
    if (rules?.max != null && number > rules.max)
      return `${field.label} must be at most ${rules.max}.`;
  }
  if (rules?.pattern) {
    try {
      if (!new RegExp(`^(?:${rules.pattern})$`).test(text))
        return `${field.label} is not in the expected format.`;
    } catch {
      /* The server remains authoritative for unsupported patterns. */
    }
  }
  return null;
}
