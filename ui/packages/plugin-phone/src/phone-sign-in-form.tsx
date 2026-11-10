"use client";

import * as React from "react";
import {
  AuthCard,
  Button,
  ErrorDisplay,
  Label,
  LoadingSpinner,
  PhoneInput,
  type PhoneCountryCode,
  VerificationCodeInput,
  type AuthCardAlign,
  type AuthCardVariant,
} from "@authsome/ui-components";
import { usePhoneAuth } from "./react";

export interface PhoneSignInFormProps {
  /** Fired once the code is right and the session has been adopted. */
  onSuccess?: (result: { newUser: boolean }) => void;
  /** Shown on the number step. */
  title?: string;
  description?: string;
  /** Only needed on a multi-app deployment. */
  appId?: string;
  /** How many digits the code has. Match your server's setting. */
  codeLength?: number;
  /** Country the number field starts on. */
  defaultCountry?: PhoneCountryCode;
  /** Narrow the country list, e.g. ["PT", "ES", "GB"]. */
  countries?: PhoneCountryCode[];
  align?: AuthCardAlign;
  variant?: AuthCardVariant;
  className?: string;
  /** Rendered under the form, for a "use email instead" link. */
  footer?: React.ReactNode;
}

/**
 * Phone sign-in in two steps: a number, then the code that was texted to it.
 *
 * The code step auto-submits once the last digit lands, because a six-digit
 * code read off a phone has exactly one correct ending and making someone
 * press a button after it is busywork. The button stays for anyone pasting
 * or correcting.
 */
export function PhoneSignInForm({
  onSuccess,
  title = "Sign in with your phone",
  description = "We'll text you a code.",
  appId,
  codeLength = 6,
  defaultCountry = "US",
  countries,
  align,
  variant,
  className,
  footer,
}: PhoneSignInFormProps) {
  const { start, verify, isSending, isVerifying, error } = usePhoneAuth();
  const [step, setStep] = React.useState<"phone" | "code">("phone");
  // E.164, which is what the API takes. isValid gates the button, so nobody
  // can send a code to half a number.
  const [phoneNumber, setPhoneNumber] = React.useState("");
  const [isValidNumber, setIsValidNumber] = React.useState(false);
  const [code, setCode] = React.useState("");

  const sendCode = React.useCallback(async () => {
    if (!isValidNumber) return;
    try {
      await start({ phone: phoneNumber, app_id: appId });
      setStep("code");
    } catch {
      // usePhoneAuth already put it in `error`, which ErrorDisplay renders.
    }
  }, [appId, isValidNumber, phoneNumber, start]);

  const submitCode = React.useCallback(
    async (value: string) => {
      if (value.length !== codeLength || isVerifying) return;
      try {
        const res = await verify({
          phone: phoneNumber,
          code: value,
          app_id: appId,
        });
        onSuccess?.({ newUser: res.new_user });
      } catch {
        // Leave the code in place. Clearing it on a wrong answer costs
        // someone who mistyped one digit the other five.
      }
    },
    [appId, codeLength, isVerifying, onSuccess, phoneNumber, verify],
  );

  if (step === "code") {
    return (
      <AuthCard
        title="Enter your code"
        description={`We texted a ${codeLength}-digit code to ${phoneNumber}.`}
        align={align}
        variant={variant}
        className={className}
        footer={footer}
      >
        <form
          onSubmit={(e) => {
            e.preventDefault();
            void submitCode(code);
          }}
          className="grid gap-3"
        >
          <div className="flex justify-center">
            <VerificationCodeInput
              length={codeLength}
              mode="numeric"
              value={code}
              onChange={setCode}
              onComplete={(v) => void submitCode(v)}
              disabled={isVerifying}
              invalid={Boolean(error)}
              aria-label="Phone verification code"
              autoFocus
            />
          </div>

          <ErrorDisplay error={error?.message ?? null} />

          <Button
            type="submit"
            className="w-full"
            disabled={code.length !== codeLength || isVerifying}
          >
            {isVerifying ? <LoadingSpinner /> : "Verify and continue"}
          </Button>

          <Button
            type="button"
            variant="ghost"
            onClick={() => {
              setStep("phone");
              setCode("");
            }}
          >
            Use a different number
          </Button>
        </form>
      </AuthCard>
    );
  }

  return (
    <AuthCard
      title={title}
      description={description}
      align={align}
      variant={variant}
      className={className}
      footer={footer}
    >
      <form
        onSubmit={(e) => {
          e.preventDefault();
          void sendCode();
        }}
        className="grid gap-3"
      >
        <div className="grid gap-1.5">
          <Label htmlFor="authsome-phone">Phone number</Label>
          <PhoneInput
            id="authsome-phone"
            defaultCountry={defaultCountry}
            countries={countries}
            value={phoneNumber}
            onChange={(next, valid) => {
              setPhoneNumber(next);
              setIsValidNumber(valid);
            }}
            disabled={isSending}
            autoFocus
          />
        </div>

        <ErrorDisplay error={error?.message ?? null} />

        <Button
          type="submit"
          className="w-full"
          disabled={!isValidNumber || isSending}
        >
          {isSending ? <LoadingSpinner /> : "Send code"}
        </Button>
      </form>
    </AuthCard>
  );
}
