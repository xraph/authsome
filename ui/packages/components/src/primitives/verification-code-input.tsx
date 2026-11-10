"use client";

import * as React from "react";
import {
  OTPInput,
  REGEXP_ONLY_DIGITS,
  REGEXP_ONLY_DIGITS_AND_CHARS,
} from "input-otp";

import { cn } from "../lib/utils";

/** What a code is allowed to contain. */
export type VerificationCodeMode = "numeric" | "alphanumeric";

/** How large each box is drawn. */
export type VerificationCodeSize = "default" | "compact";

export interface VerificationCodeInputProps {
  /** How many characters the code has. */
  length?: number;
  /**
   * Which characters are accepted. "alphanumeric" also upper-cases as you
   * type, because every code authsome issues is case-insensitive and a
   * lowercase "b" sitting next to an upper-case "B" reads as a bug.
   */
  mode?: VerificationCodeMode;
  /** Controlled value. Leave it out and the component holds its own. */
  value?: string;
  onChange?: (value: string) => void;
  /** Fires once the last character is filled in. */
  onComplete?: (value: string) => void;
  /**
   * Insert a separator every n characters, for codes that are shown grouped,
   * like the device flow's WDJB-MJHT. 0 renders one flat run.
   */
  groupSize?: number;
  autoFocus?: boolean;
  disabled?: boolean;
  /** Renders the boxes in the destructive colour and sets aria-invalid. */
  invalid?: boolean;
  /**
   * Box size. "default" is the 6-character field the sign-in screens use.
   * "compact" is for longer codes. Left alone, anything over 6 characters
   * picks "compact" on its own, which is what keeps an 8-character device
   * code inside a 400px card.
   */
  size?: VerificationCodeSize;
  /** Accessible name for the group of boxes. */
  "aria-label"?: string;
  className?: string;
  containerClassName?: string;
}

const patterns: Record<VerificationCodeMode, string> = {
  numeric: REGEXP_ONLY_DIGITS,
  alphanumeric: REGEXP_ONLY_DIGITS_AND_CHARS,
};

/**
 * A one-time code field: separated boxes, one character each.
 *
 * Built on `input-otp`, which keeps a single real input behind the boxes. That
 * is what makes paste, mobile SMS autofill and password managers work, so the
 * underlying input keeps `autocomplete="one-time-code"` and the boxes stay
 * presentational.
 */
export function VerificationCodeInput({
  length = 6,
  mode = "numeric",
  value,
  onChange,
  onComplete,
  groupSize = 0,
  autoFocus,
  disabled,
  invalid,
  size,
  "aria-label": ariaLabel,
  className,
  containerClassName,
}: VerificationCodeInputProps) {
  const resolvedSize: VerificationCodeSize = size ?? (length > 6 ? "compact" : "default");

  const [internal, setInternal] = React.useState("");
  const isControlled = value !== undefined;
  const current = isControlled ? value : internal;

  // Disabling the field while a code is checked blurs it, and nothing gives
  // the focus back when it is re-enabled. The cost lands on exactly the
  // person who needs it most: someone whose code was wrong, who now has to
  // click back into the boxes before they can fix a digit. So when the field
  // comes back and the code is still incomplete, it takes focus again.
  const inputRef = React.useRef<HTMLInputElement>(null);
  const wasDisabled = React.useRef(Boolean(disabled));
  React.useEffect(() => {
    const nowDisabled = Boolean(disabled);
    if (wasDisabled.current && !nowDisabled && current.length < length) {
      inputRef.current?.focus();
    }
    wasDisabled.current = nowDisabled;
  }, [disabled, current, length]);

  const handleChange = React.useCallback(
    (next: string) => {
      const normalised = mode === "alphanumeric" ? next.toUpperCase() : next;
      if (!isControlled) setInternal(normalised);
      onChange?.(normalised);
    },
    [isControlled, mode, onChange],
  );

  // Where the separators go. An empty set means one flat run of boxes, which
  // is what every screen in the refreshed design uses.
  const breaks = React.useMemo(() => {
    if (!groupSize || groupSize >= length) return new Set<number>();
    const s = new Set<number>();
    for (let i = groupSize; i < length; i += groupSize) s.add(i);
    return s;
  }, [groupSize, length]);

  return (
    <OTPInput
      ref={inputRef}
      data-slot="verification-code"
      maxLength={length}
      pattern={patterns[mode]}
      value={current}
      onChange={handleChange}
      // input-otp hands the completed value to onComplete. Use it rather than
      // `current`, which is still the previous render's value at this point:
      // onChange and onComplete fire in the same tick, before React re-renders.
      onComplete={
        onComplete
          ? (next: string) =>
              onComplete(mode === "alphanumeric" ? next.toUpperCase() : next)
          : undefined
      }
      autoFocus={autoFocus}
      disabled={disabled}
      // input-otp widens the field to make room for a password manager badge,
      // and polls document.elementFromPoint while focused to find one. A
      // one-time code is never something a password manager should offer to
      // fill, the badge would sit on top of the boxes, and the polling throws
      // under jsdom, so the whole mechanism is off.
      pushPasswordManagerStrategy="none"
      inputMode={mode === "numeric" ? "numeric" : "text"}
      autoComplete="one-time-code"
      aria-label={ariaLabel ?? "Verification code"}
      aria-invalid={invalid || undefined}
      containerClassName={cn(
        // No w-full here. In a parent that sizes to its content the row would
        // have nothing to be 100% of, and the boxes would collapse to
        // hairlines. Left alone the row is as wide as its boxes, and when the
        // parent is narrower than that the boxes shrink instead of spilling.
        "flex items-center gap-2 has-disabled:opacity-50",
        containerClassName,
      )}
      className={cn("disabled:cursor-not-allowed", className)}
      render={({ slots }) => (
        <>
          {slots.map((slot, i) => (
            <React.Fragment key={i}>
              {breaks.has(i) && (
                <span
                  aria-hidden
                  className="mx-0.5 h-px w-2.5 shrink-0 bg-border"
                />
              )}
              <Slot {...slot} invalid={invalid} size={resolvedSize} />
            </React.Fragment>
          ))}
        </>
      )}
    />
  );
}

// A width rather than a grow factor. The box wants to be this wide, and
// flex-shrink (on by default, with min-w-0 below to let it go under
// min-content) takes it narrower when eight of them have to fit a 400px card.
const slotSizes: Record<VerificationCodeSize, string> = {
  default: "h-14 w-12 text-xl",
  compact: "h-12 w-10 text-base",
};

function Slot({
  char,
  hasFakeCaret,
  isActive,
  invalid,
  size,
}: {
  char: string | null;
  hasFakeCaret: boolean;
  isActive: boolean;
  invalid?: boolean;
  size: VerificationCodeSize;
}) {
  return (
    <div
      data-slot="verification-code-slot"
      data-active={isActive}
      className={cn(
        "relative flex min-w-0 shrink items-center justify-center rounded-lg border border-input bg-background font-medium tabular-nums transition-all outline-none",
        slotSizes[size],
        "data-[active=true]:z-10 data-[active=true]:border-ring data-[active=true]:ring-[3px] data-[active=true]:ring-ring/40",
        invalid &&
          "border-destructive data-[active=true]:border-destructive data-[active=true]:ring-destructive/30",
      )}
    >
      {char}
      {hasFakeCaret && (
        <div className="pointer-events-none absolute inset-0 flex items-center justify-center">
          <div className="h-6 w-px animate-caret-blink bg-foreground duration-1000" />
        </div>
      )}
    </div>
  );
}
