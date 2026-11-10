"use client";

import * as React from "react";
import {
  AsYouType,
  getCountries,
  getCountryCallingCode,
  parsePhoneNumberFromString,
  type CountryCode,
} from "libphonenumber-js/min";

import { InputAddon, InputGroup, InputGroupInput } from "./input-group";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "./select";

/** ISO 3166-1 alpha-2 country code, as libphonenumber knows them. */
export type PhoneCountryCode = CountryCode;

export interface PhoneInputProps {
  /** The number in E.164 ("+351912345678"), or "" when nothing valid yet. */
  value?: string;
  /**
   * Fires on every keystroke with the E.164 number and whether it is a
   * complete, valid number for the selected country. Partial input still
   * reports a value, so a host can keep it in state, and `isValid` is what
   * a submit button should be gated on.
   */
  onChange?: (value: string, isValid: boolean) => void;
  /** Which country the field starts on. */
  defaultCountry?: CountryCode;
  /** Narrow the country list. Left out, every country libphonenumber knows. */
  countries?: CountryCode[];
  disabled?: boolean;
  autoFocus?: boolean;
  /**
   * Placeholder for the number field. There is deliberately no default: a
   * generic example is wrong for most countries, and libphonenumber only
   * ships real per-country examples in a separate metadata file that is
   * larger than the hint is worth. The label above the field carries the
   * meaning.
   */
  placeholder?: string;
  /** Draws the field in the destructive colour. */
  invalid?: boolean;
  id?: string;
  className?: string;
  "aria-label"?: string;
}

/** "Portugal", from the country code, in the viewer's own language. */
function countryName(code: CountryCode, display: Intl.DisplayNames | null): string {
  return display?.of(code) ?? code;
}

/** Regional-indicator pair, which renders as a flag wherever the font has one. */
function flag(code: CountryCode): string {
  return String.fromCodePoint(
    ...[...code].map((c) => 0x1f1e6 - 65 + c.charCodeAt(0)),
  );
}

/**
 * A phone number field: country picker plus a national number that formats
 * itself as you type.
 *
 * The value handed back is always E.164, because that is what the API wants
 * and what a database should hold. What the person sees is their own
 * country's grouping, which is the only format they can check at a glance
 * against the handset in front of them.
 */
export function PhoneInput({
  value,
  onChange,
  defaultCountry = "US",
  countries,
  disabled,
  autoFocus,
  placeholder,
  invalid,
  id,
  className,
  "aria-label": ariaLabel,
}: PhoneInputProps) {
  const list = React.useMemo(
    () => (countries?.length ? countries : (getCountries() as CountryCode[])),
    [countries],
  );

  // Intl.DisplayNames is missing on older engines, so names fall back to the
  // two-letter code rather than the field failing to render.
  const display = React.useMemo(() => {
    try {
      return new Intl.DisplayNames(undefined, { type: "region" });
    } catch {
      return null;
    }
  }, []);

  const parsed = React.useMemo(
    () => (value ? parsePhoneNumberFromString(value) : undefined),
    [value],
  );

  const [country, setCountry] = React.useState<CountryCode>(
    parsed?.country ?? defaultCountry,
  );

  // What the person sees. Derived from the E.164 value so a controlled parent
  // stays the source of truth, but held locally so that typing a space or a
  // bracket does not get rewritten underneath the cursor.
  const [text, setText] = React.useState(() =>
    parsed ? parsed.formatNational() : "",
  );

  const emit = React.useCallback(
    (nationalText: string, forCountry: CountryCode) => {
      const digits = nationalText.replace(/\D/g, "");
      if (!digits) {
        onChange?.("", false);
        return;
      }
      const e164 = `+${getCountryCallingCode(forCountry)}${digits}`;
      const number = parsePhoneNumberFromString(e164);
      onChange?.(e164, Boolean(number?.isValid()));
    },
    [onChange],
  );

  const handleText = (next: string) => {
    // AsYouType is stateful per instance, so it gets a fresh one each time
    // rather than being reused across keystrokes. Reusing one makes deleting
    // a character format against stale input.
    const formatted = new AsYouType(country).input(next);
    // When someone is deleting, AsYouType can re-add the separator they just
    // removed and the cursor sticks. Taking the raw text while it is shorter
    // than what the formatter produced avoids that.
    setText(formatted.length < next.length ? next : formatted);
    emit(next, country);
  };

  const handleCountry = (next: string) => {
    const code = next as CountryCode;
    setCountry(code);
    setText(new AsYouType(code).input(text.replace(/\D/g, "")));
    emit(text, code);
  };

  const sorted = React.useMemo(
    () =>
      [...list].sort((a, b) =>
        countryName(a, display).localeCompare(countryName(b, display)),
      ),
    [list, display],
  );

  return (
    <InputGroup className={className} invalid={invalid}>
      <InputAddon side="start">
        <Select value={country} onValueChange={handleCountry} disabled={disabled}>
          <SelectTrigger
            aria-label="Country calling code"
            className="h-full w-[76px] rounded-none border-0 bg-transparent px-2.5 text-foreground shadow-none focus:ring-0 focus-visible:ring-0"
          >
            <SelectValue>
              <span className="flex items-center gap-1.5">
                <span aria-hidden>{flag(country)}</span>
                <span className="tabular-nums">
                  +{getCountryCallingCode(country)}
                </span>
              </span>
            </SelectValue>
          </SelectTrigger>
          <SelectContent className="max-h-72">
            {sorted.map((code) => (
              <SelectItem key={code} value={code}>
                <span className="flex items-center gap-2">
                  <span aria-hidden>{flag(code)}</span>
                  <span className="flex-1">{countryName(code, display)}</span>
                  <span className="text-muted-foreground tabular-nums">
                    +{getCountryCallingCode(code)}
                  </span>
                </span>
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </InputAddon>

      <InputGroupInput
        id={id}
        type="tel"
        inputMode="tel"
        autoComplete="tel-national"
        aria-label={ariaLabel ?? "Phone number"}
        placeholder={placeholder}
        value={text}
        onChange={(e) => handleText(e.target.value)}
        disabled={disabled}
        autoFocus={autoFocus}
      />
    </InputGroup>
  );
}
