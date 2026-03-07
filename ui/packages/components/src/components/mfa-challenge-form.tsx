import * as React from "react";
import { REGEXP_ONLY_DIGITS } from "input-otp";
import { useAuth } from "@authsome/ui-react";
import { cn } from "../lib/utils";
import {
  InputOTP,
  InputOTPGroup,
  InputOTPSlot,
  InputOTPSeparator,
} from "../primitives/otp-input";
import { Button } from "../primitives/button";
import { Input } from "../primitives/input";
import { AuthCard } from "./auth-card";
import { ErrorDisplay } from "./error-display";
import { LoadingSpinner } from "./loading-spinner";
import { Smartphone, KeyRound, ShieldCheck } from "lucide-react";

// ── Types ──────────────────────────────────────────────

type MFAMethod = "totp" | "sms" | "recovery";

export interface MFAChallengeFormStyledProps {
  enrollmentId: string;
  onSuccess?: () => void;
  /** Optional logo element rendered above the title. */
  logo?: React.ReactNode;
  className?: string;
  /** Override available MFA methods. Auto-detected from clientConfig if omitted. */
  methods?: string[];
  /** Which method to show initially. Defaults to first available. */
  defaultMethod?: string;
}

// ── TOTP View ──────────────────────────────────────────

function TOTPView({
  enrollmentId,
  onSuccess,
}: {
  enrollmentId: string;
  onSuccess?: () => void;
}) {
  const { submitMFACode } = useAuth();
  const [code, setCode] = React.useState("");
  const [error, setError] = React.useState<string | null>(null);
  const [isSubmitting, setIsSubmitting] = React.useState(false);

  const handleSubmit = React.useCallback(
    async (otpCode: string) => {
      if (otpCode.length !== 6 || isSubmitting) return;
      setError(null);
      setIsSubmitting(true);
      try {
        await submitMFACode(enrollmentId, otpCode);
        onSuccess?.();
      } catch (err) {
        const message =
          err instanceof Error ? err.message : "Invalid code. Please try again.";
        setError(message);
        setCode("");
      } finally {
        setIsSubmitting(false);
      }
    },
    [enrollmentId, isSubmitting, onSuccess, submitMFACode],
  );

  return (
    <form
      onSubmit={(e) => {
        e.preventDefault();
        void handleSubmit(code);
      }}
      className="grid gap-3"
    >
      <div className="flex justify-center">
        <InputOTP
          maxLength={6}
          pattern={REGEXP_ONLY_DIGITS}
          value={code}
          onChange={(v) => {
            setCode(v);
            setError(null);
          }}
          onComplete={(v) => void handleSubmit(v)}
          disabled={isSubmitting}
        >
          <InputOTPGroup>
            <InputOTPSlot index={0} />
            <InputOTPSlot index={1} />
            <InputOTPSlot index={2} />
          </InputOTPGroup>
          <InputOTPSeparator />
          <InputOTPGroup>
            <InputOTPSlot index={3} />
            <InputOTPSlot index={4} />
            <InputOTPSlot index={5} />
          </InputOTPGroup>
        </InputOTP>
      </div>

      <ErrorDisplay error={error} />

      <Button
        type="submit"
        className="w-full"
        disabled={code.length !== 6 || isSubmitting}
      >
        {isSubmitting ? (
          <>
            <LoadingSpinner size="sm" className="mr-2" />
            Verifying...
          </>
        ) : (
          "Verify code"
        )}
      </Button>
    </form>
  );
}

// ── SMS View ───────────────────────────────────────────

function SMSView({ onSuccess }: { onSuccess?: () => void }) {
  const { sendSMSCode, submitSMSCode } = useAuth();
  const [smsState, setSmsState] = React.useState<"idle" | "code_sent">("idle");
  const [phoneMasked, setPhoneMasked] = React.useState("");
  const [code, setCode] = React.useState("");
  const [error, setError] = React.useState<string | null>(null);
  const [isSubmitting, setIsSubmitting] = React.useState(false);
  const [isSending, setIsSending] = React.useState(false);
  const [resendCooldown, setResendCooldown] = React.useState(0);

  // Countdown timer for resend cooldown.
  React.useEffect(() => {
    if (resendCooldown <= 0) return;
    const timer = setInterval(() => {
      setResendCooldown((prev) => prev - 1);
    }, 1000);
    return () => clearInterval(timer);
  }, [resendCooldown]);

  const handleSendCode = React.useCallback(async () => {
    if (isSending) return;
    setError(null);
    setIsSending(true);
    try {
      const res = await sendSMSCode();
      setPhoneMasked(res.phone_masked);
      setSmsState("code_sent");
      setResendCooldown(60);
    } catch (err) {
      const message =
        err instanceof Error ? err.message : "Failed to send SMS code.";
      setError(message);
    } finally {
      setIsSending(false);
    }
  }, [isSending, sendSMSCode]);

  const handleSubmit = React.useCallback(
    async (otpCode: string) => {
      if (otpCode.length !== 6 || isSubmitting) return;
      setError(null);
      setIsSubmitting(true);
      try {
        await submitSMSCode(otpCode);
        onSuccess?.();
      } catch (err) {
        const message =
          err instanceof Error ? err.message : "Invalid SMS code. Please try again.";
        setError(message);
        setCode("");
      } finally {
        setIsSubmitting(false);
      }
    },
    [isSubmitting, onSuccess, submitSMSCode],
  );

  const handleResend = React.useCallback(async () => {
    if (resendCooldown > 0 || isSending) return;
    setIsSending(true);
    try {
      const res = await sendSMSCode();
      setPhoneMasked(res.phone_masked);
      setResendCooldown(60);
    } catch {
      // Silently handle resend errors
    } finally {
      setIsSending(false);
    }
  }, [resendCooldown, isSending, sendSMSCode]);

  if (smsState === "idle") {
    return (
      <div className="grid gap-3">
        <p className="text-center text-[13px] text-muted-foreground">
          We&apos;ll send a verification code to your registered phone number.
        </p>
        <ErrorDisplay error={error} />
        <Button
          type="button"
          className="w-full"
          onClick={() => void handleSendCode()}
          disabled={isSending}
        >
          {isSending ? (
            <>
              <LoadingSpinner size="sm" className="mr-2" />
              Sending...
            </>
          ) : (
            <>
              <Smartphone className="mr-2 h-3.5 w-3.5" />
              Send code
            </>
          )}
        </Button>
      </div>
    );
  }

  return (
    <form
      onSubmit={(e) => {
        e.preventDefault();
        void handleSubmit(code);
      }}
      className="grid gap-3"
    >
      {phoneMasked && (
        <p className="text-center text-[13px] text-muted-foreground">
          Code sent to <span className="font-medium text-foreground">{phoneMasked}</span>
        </p>
      )}

      <div className="flex justify-center">
        <InputOTP
          maxLength={6}
          pattern={REGEXP_ONLY_DIGITS}
          value={code}
          onChange={(v) => {
            setCode(v);
            setError(null);
          }}
          onComplete={(v) => void handleSubmit(v)}
          disabled={isSubmitting}
        >
          <InputOTPGroup>
            <InputOTPSlot index={0} />
            <InputOTPSlot index={1} />
            <InputOTPSlot index={2} />
          </InputOTPGroup>
          <InputOTPSeparator />
          <InputOTPGroup>
            <InputOTPSlot index={3} />
            <InputOTPSlot index={4} />
            <InputOTPSlot index={5} />
          </InputOTPGroup>
        </InputOTP>
      </div>

      <ErrorDisplay error={error} />

      <Button
        type="submit"
        className="w-full"
        disabled={code.length !== 6 || isSubmitting}
      >
        {isSubmitting ? (
          <>
            <LoadingSpinner size="sm" className="mr-2" />
            Verifying...
          </>
        ) : (
          "Verify code"
        )}
      </Button>

      <p className="text-center text-[13px] text-muted-foreground">
        Didn&apos;t receive a code?{" "}
        {resendCooldown > 0 ? (
          <span className="text-muted-foreground/70">
            Resend in {resendCooldown}s
          </span>
        ) : (
          <button
            type="button"
            className="font-medium text-foreground underline-offset-4 hover:underline"
            onClick={() => void handleResend()}
          >
            Resend
          </button>
        )}
      </p>
    </form>
  );
}

// ── Recovery Code View ─────────────────────────────────

function RecoveryView({ onSuccess }: { onSuccess?: () => void }) {
  const { submitRecoveryCode } = useAuth();
  const [code, setCode] = React.useState("");
  const [error, setError] = React.useState<string | null>(null);
  const [isSubmitting, setIsSubmitting] = React.useState(false);

  const handleSubmit = React.useCallback(
    async (e: React.FormEvent) => {
      e.preventDefault();
      if (code.length === 0 || isSubmitting) return;
      setError(null);
      setIsSubmitting(true);
      try {
        await submitRecoveryCode(code.trim());
        onSuccess?.();
      } catch (err) {
        const message =
          err instanceof Error
            ? err.message
            : "Invalid recovery code. Please try again.";
        setError(message);
      } finally {
        setIsSubmitting(false);
      }
    },
    [code, isSubmitting, onSuccess, submitRecoveryCode],
  );

  return (
    <form onSubmit={handleSubmit} className="grid gap-3">
      <Input
        value={code}
        onChange={(e) => {
          setCode(e.target.value);
          setError(null);
        }}
        placeholder="Enter recovery code"
        maxLength={8}
        className="font-mono text-center tracking-widest"
        disabled={isSubmitting}
        autoFocus
      />

      <ErrorDisplay error={error} />

      <Button
        type="submit"
        className="w-full"
        disabled={code.trim().length === 0 || isSubmitting}
      >
        {isSubmitting ? (
          <>
            <LoadingSpinner size="sm" className="mr-2" />
            Verifying...
          </>
        ) : (
          "Verify recovery code"
        )}
      </Button>
    </form>
  );
}

// ── Method Switcher ────────────────────────────────────

function MethodSwitcher({
  currentMethod,
  hasTOTP,
  hasSMS,
  onSwitch,
}: {
  currentMethod: MFAMethod;
  hasTOTP: boolean;
  hasSMS: boolean;
  onSwitch: (method: MFAMethod) => void;
}) {
  const links: { method: MFAMethod; label: string; icon: React.ReactNode }[] = [];

  if (currentMethod !== "totp" && hasTOTP) {
    links.push({
      method: "totp",
      label: "Use authenticator app",
      icon: <ShieldCheck className="mr-1 inline h-3 w-3" />,
    });
  }
  if (currentMethod !== "sms" && hasSMS) {
    links.push({
      method: "sms",
      label: "Use SMS instead",
      icon: <Smartphone className="mr-1 inline h-3 w-3" />,
    });
  }
  if (currentMethod !== "recovery") {
    links.push({
      method: "recovery",
      label: "Use a recovery code",
      icon: <KeyRound className="mr-1 inline h-3 w-3" />,
    });
  }

  if (links.length === 0) return null;

  return (
    <div className="flex flex-col items-center gap-1.5 pt-1">
      {links.map(({ method, label, icon }) => (
        <button
          key={method}
          type="button"
          className="inline-flex items-center text-[13px] text-muted-foreground hover:text-foreground transition-colors"
          onClick={() => onSwitch(method)}
        >
          {icon}
          {label}
        </button>
      ))}
    </div>
  );
}

// ── Main Component ─────────────────────────────────────

const METHOD_META: Record<MFAMethod, { title: string; description: string }> = {
  totp: {
    title: "Two-factor authentication",
    description: "Enter the 6-digit code from your authenticator app",
  },
  sms: {
    title: "SMS verification",
    description: "Verify your identity with a code sent to your phone",
  },
  recovery: {
    title: "Recovery code",
    description: "Enter one of your 8-character recovery codes",
  },
};

export function MFAChallengeForm({
  enrollmentId,
  onSuccess,
  logo,
  className,
  methods: methodsProp,
  defaultMethod,
}: MFAChallengeFormStyledProps) {
  const { clientConfig } = useAuth();

  // Derive available methods from prop, clientConfig, or fallback to TOTP only.
  const availableMethods = React.useMemo(() => {
    const raw = methodsProp ?? clientConfig?.mfa?.methods ?? ["totp"];
    return raw.filter((m): m is MFAMethod => m === "totp" || m === "sms");
  }, [methodsProp, clientConfig]);

  const hasTOTP = availableMethods.includes("totp");
  const hasSMS = availableMethods.includes("sms");

  const initialMethod: MFAMethod =
    (defaultMethod as MFAMethod) ?? availableMethods[0] ?? "totp";

  const [currentMethod, setCurrentMethod] = React.useState<MFAMethod>(initialMethod);

  const meta = METHOD_META[currentMethod];

  // Show method switcher when there's more than one method OR recovery codes are available.
  const showSwitcher = availableMethods.length > 1 || currentMethod !== "recovery";

  return (
    <AuthCard
      title={meta.title}
      description={meta.description}
      logo={logo}
      className={cn(className)}
    >
      {/* key ensures clean state reset when switching methods */}
      <div key={currentMethod}>
        {currentMethod === "totp" && (
          <TOTPView enrollmentId={enrollmentId} onSuccess={onSuccess} />
        )}
        {currentMethod === "sms" && <SMSView onSuccess={onSuccess} />}
        {currentMethod === "recovery" && <RecoveryView onSuccess={onSuccess} />}
      </div>

      {showSwitcher && (
        <MethodSwitcher
          currentMethod={currentMethod}
          hasTOTP={hasTOTP}
          hasSMS={hasSMS}
          onSwitch={setCurrentMethod}
        />
      )}
    </AuthCard>
  );
}
