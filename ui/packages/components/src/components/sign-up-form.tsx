"use client";

import * as React from "react";
import { useState } from "react";
import {
  useAuth,
  useClientConfig,
  type SignupFieldConfig,
} from "@authsome/ui-react";
import type { WaitlistStatus } from "@authsome/ui-core";
import { cn } from "../lib/utils";
import { Button } from "../primitives/button";
import { Input } from "../primitives/input";
import { Label } from "../primitives/label";
import { Checkbox } from "../primitives/checkbox";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "../primitives/select";
import {
  AuthCard,
  type AuthCardAlign,
  type AuthCardVariant,
} from "./auth-card";
import { ErrorDisplay } from "./error-display";
import { LoadingSpinner } from "./loading-spinner";
import { PasswordInput } from "./password-input";
import {
  SocialButtons,
  OrDivider,
  type SocialProvider,
  type SocialButtonLayout,
} from "./social-buttons";
import { handleSocialLogin } from "../lib/social-login";
import { ArrowLeft } from "lucide-react";
import { validEmail, validateSignupField } from "../lib/form-validation";
import { EmailVerificationForm } from "./email-verification-form";
import { TurnstileWidget } from "./turnstile-widget";

export interface SignUpFormComponentProps {
  /** Callback invoked after a successful sign-up. */
  onSuccess?: () => void;
  /** Called when signup requires email verification. */
  onVerificationRequired?: (email: string) => void;
  /** URL to the sign-in page. Renders an "Already have an account?" footer link. */
  signInUrl?: string;
  /** URL to the forgot-password page. Renders a "Forgot password?" link. */
  forgotPasswordUrl?: string;
  /** Social/OAuth providers to display below the form. */
  socialProviders?: SocialProvider[];
  /** Callback when a social provider button is clicked. */
  onSocialLogin?: (providerId: string) => void;
  /** Layout mode for social login buttons. */
  socialLayout?: SocialButtonLayout;
  /** Optional logo element rendered above the title. */
  logo?: React.ReactNode;
  /** Title and description alignment. */
  align?: AuthCardAlign;
  /** Card visual style. */
  variant?: AuthCardVariant;
  /** Additional CSS class names. */
  className?: string;
  /**
   * Heading shown on the initial sign-up view. Defaults to
   * "Create your {app_name} account" when `config.branding.app_name` is set,
   * otherwise "Create an account".
   */
  title?: string;
}

/**
 * Renders a single dynamic signup field based on its type.
 */
/** The configured default value for each field that declares one. */
function defaultsFor(
  fields: SignupFieldConfig[] | null,
): Record<string, string> {
  const defaults: Record<string, string> = {};
  for (const f of fields ?? []) {
    if (f.default) defaults[f.key] = f.default;
  }
  return defaults;
}

function DynamicField({
  field,
  value,
  onChange,
  disabled,
}: {
  field: SignupFieldConfig;
  value: string;
  onChange: (value: string) => void;
  disabled: boolean;
}) {
  const fieldId = `signup-field-${field.key}`;
  const isRequired = field.validation?.required ?? false;

  const label = (
    <Label htmlFor={fieldId} className="text-[13px]">
      {field.label}
      {!isRequired && (
        <span className="ml-1 text-muted-foreground">(optional)</span>
      )}
    </Label>
  );

  switch (field.type) {
    case "radio":
    case "select":
      return (
        <div className="grid gap-1.5">
          {label}
          <Select value={value} onValueChange={onChange} disabled={disabled}>
            <SelectTrigger id={fieldId}>
              <SelectValue
                placeholder={
                  field.placeholder || `Select ${field.label.toLowerCase()}`
                }
              />
            </SelectTrigger>
            <SelectContent>
              {field.options?.map((opt) => (
                <SelectItem key={opt.value} value={opt.value}>
                  {opt.label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
          {field.description && (
            <p className="text-xs text-muted-foreground">{field.description}</p>
          )}
        </div>
      );

    case "checkbox":
    case "switch":
      return (
        <div className="flex items-center gap-2">
          <Checkbox
            id={fieldId}
            checked={value === "true"}
            onCheckedChange={(checked) => onChange(checked ? "true" : "false")}
            disabled={disabled}
          />
          <Label htmlFor={fieldId} className="text-[13px] font-normal">
            {field.label}
            {field.description && (
              <span className="ml-1 text-muted-foreground">
                : {field.description}
              </span>
            )}
          </Label>
        </div>
      );

    case "textarea":
      return (
        <div className="grid gap-1.5">
          {label}
          <textarea
            id={fieldId}
            className="flex min-h-[80px] w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50"
            placeholder={field.placeholder}
            required={isRequired}
            disabled={disabled}
            value={value}
            onChange={(e) => onChange(e.target.value)}
          />
          {field.description && (
            <p className="text-xs text-muted-foreground">{field.description}</p>
          )}
        </div>
      );

    default: {
      // text, email, number, tel, url, date, radio — all use Input
      const inputType = field.type === "radio" ? "text" : field.type || "text";
      return (
        <div className="grid gap-1.5">
          {label}
          <Input
            id={fieldId}
            type={inputType}
            placeholder={field.placeholder}
            required={isRequired}
            disabled={disabled}
            value={value}
            onChange={(e) => onChange(e.target.value)}
            minLength={field.validation?.min_len}
            maxLength={field.validation?.max_len}
            min={field.validation?.min}
            max={field.validation?.max}
            pattern={field.validation?.pattern}
          />
          {field.description && (
            <p className="text-xs text-muted-foreground">{field.description}</p>
          )}
        </div>
      );
    }
  }
}

/**
 * A fully styled sign-up form with Clerk-style UX:
 *
 * - **Social-first**: When social providers are available, they appear at the top.
 * - **Multi-step**: User enters email first, clicks "Continue", then enters additional fields & password.
 * - **Dynamic fields**: When the backend has custom signup fields configured, they are rendered automatically.
 * - **Auto-configuration**: When `publishableKey` is set on `AuthProvider`, the form
 *   auto-derives social providers and signup fields from the backend client config.
 * - When the waitlist is enabled, Continue checks approval before showing signup details.
 * - Explicit props always take precedence over auto-discovered values.
 */
export function SignUpForm({
  onSuccess,
  onVerificationRequired,
  signInUrl,
  forgotPasswordUrl,
  socialProviders: socialProvidersProp,
  onSocialLogin: onSocialLoginProp,
  socialLayout,
  logo,
  align,
  variant,
  className,
  title: titleProp,
}: SignUpFormComponentProps) {
  const { signUp, client, manager } = useAuth();
  const { config } = useClientConfig();

  const appName = config?.branding?.app_name;
  const title =
    titleProp ??
    (appName ? `Create your ${appName} account` : "Create an account");

  // Auto-derive social providers from client config when not explicitly provided.
  const socialProviders =
    socialProvidersProp ??
    (config?.social?.enabled && config.social.providers.length > 0
      ? config.social.providers.map((p) => ({ id: p.id, name: p.name }))
      : undefined);

  // Default social login: popup-based OAuth flow via startOAuth API.
  const onSocialLogin =
    onSocialLoginProp ??
    (socialProviders && socialProviders.length > 0
      ? (providerId: string) =>
          handleSocialLogin(client, providerId, () => {
            onSuccess?.();
            window.location.reload();
          })
      : undefined);

  const hasSocial =
    socialProviders && socialProviders.length > 0 && onSocialLogin;

  // Auto-derive password support from client config (default: true).
  const showPassword = config?.password?.enabled ?? true;

  // Get dynamic signup fields from config, sorted by order.
  const signupFields = React.useMemo(() => {
    const fields = config?.signup_fields;
    if (!fields || fields.length === 0) return null;
    return [...fields].sort((a, b) => a.order - b.order);
  }, [config?.signup_fields]);

  const [step, setStep] = useState<"email" | "details" | "waitlist" | "verify">(
    "email",
  );
  const [waitlistStatus, setWaitlistStatus] = useState<WaitlistStatus | null>(
    null,
  );
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [captchaAttempt, setCaptchaAttempt] = useState(0);
  const [captchaToken, setCaptchaToken] = useState<string | null>(null);

  // Captcha config (Turnstile only for now).
  const captchaCfg = config?.captcha;
  const captchaRequired = !!captchaCfg?.required;
  const captchaEnabled =
    !!captchaCfg?.required &&
    captchaCfg.provider === "turnstile" &&
    !!captchaCfg.site_key;

  // Dynamic field values — keyed by field key. Seeded from the configured
  // defaults on the first render rather than from an effect afterwards, so
  // the first paint already shows them.
  const [fieldValues, setFieldValues] = useState<Record<string, string>>(() =>
    defaultsFor(signupFields),
  );

  // Re-seed when the configured fields change. This is React's documented
  // "adjusting state when a prop changes" pattern rather than an effect, so
  // nothing is set synchronously inside one.
  //
  // Seeding has to happen on the first render as well as on a change, which is
  // what the initializer above covers. Keying only on the identity comparison
  // here would compare equal on mount and never apply the defaults at all —
  // the failure sign-up-form.test.tsx pins.
  //
  // `prev` deliberately wins the spread: a default may only fill a field the
  // user has not set.
  const [appliedFields, setAppliedFields] = useState(signupFields);
  if (signupFields !== appliedFields) {
    setAppliedFields(signupFields);
    const defaults = defaultsFor(signupFields);
    if (Object.keys(defaults).length > 0) {
      setFieldValues((prev) => ({ ...defaults, ...prev }));
    }
  }

  const setFieldValue = (key: string, value: string) => {
    setFieldValues((prev) => ({ ...prev, [key]: value }));
  };

  // Fallback: when no signup fields are configured, show first/last name fields.
  const [fallbackFirstName, setFallbackFirstName] = useState("");
  const [fallbackLastName, setFallbackLastName] = useState("");

  const handleEmailContinue = async (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    if (isSubmitting) return;
    setError(null);

    const signupEmail = email.trim().toLowerCase();
    if (!validEmail(signupEmail)) {
      setError("Please enter a valid email address.");
      return;
    }

    setEmail(signupEmail);
    setIsSubmitting(true);
    try {
      const signupConfig =
        config ??
        (client.getPublishableKey?.()
          ? await manager.fetchClientConfig()
          : null);
      if (signupConfig?.signup_enabled === false) {
        setError("Signup isn't available for this app.");
        return;
      }
      if (signupConfig?.password?.enabled === false) return;
      if (signupConfig?.waitlist?.enabled) {
        const entry = await client.joinWaitlist({ email: signupEmail });
        setWaitlistStatus(entry.status);
        if (entry.status !== "approved") {
          setStep("waitlist");
          return;
        }
      }
      setStep("details");
    } catch (err) {
      setError(
        err instanceof Error
          ? err.message
          : "Unable to check waitlist status. Please try again.",
      );
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleSignUp = async (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    if (isSubmitting) return;
    if (config?.signup_enabled === false || !showPassword) return;
    if (captchaRequired && !captchaToken) {
      setError("Please complete the captcha.");
      return;
    }
    const fields: Record<string, string> = {};
    for (const field of signupFields ?? []) {
      const value = fieldValues[field.key] ?? "";
      const problem = validateSignupField(field, value);
      if (problem) {
        setError(problem);
        return;
      }
      if (value) fields[field.key] = value;
    }
    setError(null);
    setIsSubmitting(true);

    try {
      // If using fallback (no dynamic fields), map the first/last name inputs.
      if (!signupFields) {
        if (fallbackFirstName) fields.first_name = fallbackFirstName;
        if (fallbackLastName) fields.last_name = fallbackLastName;
      }

      await signUp(
        email,
        password,
        Object.keys(fields).length > 0 ? fields : undefined,
        captchaToken ? { captchaToken } : undefined,
      );
      if (manager.getState().status === "authenticated") {
        onSuccess?.();
      } else {
        setStep("verify");
        onVerificationRequired?.(email);
      }
    } catch (err) {
      setError(
        err instanceof Error
          ? err.message
          : "Sign up failed. Please try again.",
      );
    } finally {
      setCaptchaToken(null);
      setCaptchaAttempt((value) => value + 1);
      setIsSubmitting(false);
    }
  };

  const goBack = () => {
    if (isSubmitting) return;
    setCaptchaToken(null);
    setCaptchaAttempt((value) => value + 1);
    setStep("email");
    setPassword("");
    setError(null);
    setWaitlistStatus(null);
  };

  const footer = signInUrl ? (
    <p className="text-[13px] text-muted-foreground">
      Already have an account?{" "}
      <a
        href={signInUrl}
        className="font-medium text-foreground underline-offset-4 hover:underline"
      >
        Sign in
      </a>
    </p>
  ) : undefined;

  if (step === "verify") {
    return (
      <EmailVerificationForm
        email={email}
        logo={logo}
        className={className}
        onSuccess={() => {
          if (signInUrl) window.location.assign(signInUrl);
        }}
      />
    );
  }
  if (config?.signup_enabled === false) {
    return (
      <AuthCard
        title="Signup isn't available"
        description="New accounts aren't available for this app."
        logo={logo}
        footer={footer}
        align={align}
        variant={variant}
        className={className}
      >
        {null}
      </AuthCard>
    );
  }

  if (step === "waitlist") {
    const rejected = waitlistStatus === "rejected";
    return (
      <AuthCard
        title={rejected ? "Signup isn't available" : "You're on the waitlist"}
        description={
          rejected
            ? "Access hasn't been approved for this email address."
            : `We'll email ${email} when access is approved.`
        }
        logo={logo}
        footer={footer}
        align={align}
        variant={variant}
        className={cn(className)}
      >
        <form onSubmit={handleEmailContinue} className="grid gap-3">
          <ErrorDisplay error={error} />
          <Button type="submit" disabled={isSubmitting}>
            {isSubmitting && <LoadingSpinner size="sm" className="mr-2" />}
            Check status
          </Button>
          <Button
            type="button"
            variant="ghost"
            disabled={isSubmitting}
            onClick={goBack}
          >
            Use another email
          </Button>
        </form>
      </AuthCard>
    );
  }

  /* -- Password disabled: show only social sign-up ----------- */

  if (!showPassword) {
    return (
      <AuthCard
        title={title}
        description="Get started with your account."
        logo={logo}
        footer={footer}
        align={align}
        variant={variant}
        className={cn(className)}
      >
        <div className="grid gap-4">
          {hasSocial && (
            <SocialButtons
              providers={socialProviders!}
              onProviderClick={onSocialLogin!}
              isLoading={isSubmitting}
              layout={socialLayout}
              showDivider={false}
            />
          )}

          {!hasSocial && (
            <p className="text-sm text-center text-muted-foreground py-4">
              No sign-up methods are currently available. Please contact your
              administrator.
            </p>
          )}
        </div>
      </AuthCard>
    );
  }

  /* -- Step 1: Email ---------------------------------------- */

  if (step === "email") {
    return (
      <AuthCard
        title={title}
        description="Enter your email to get started."
        logo={logo}
        footer={footer}
        align={align}
        variant={variant}
        className={cn(className)}
      >
        <div className="grid gap-4">
          {/* Social buttons first (Clerk-style) */}
          {hasSocial && (
            <SocialButtons
              providers={socialProviders!}
              onProviderClick={onSocialLogin!}
              isLoading={isSubmitting}
              layout={socialLayout}
              showDivider={false}
            />
          )}

          {hasSocial && <OrDivider />}

          <form onSubmit={handleEmailContinue} className="grid gap-3">
            <ErrorDisplay error={error} />

            <div className="grid gap-1.5">
              <Label htmlFor="signup-email" className="text-[13px]">
                Email address
              </Label>
              <Input
                id="signup-email"
                type="email"
                placeholder="name@example.com"
                autoComplete="email"
                required
                disabled={isSubmitting}
                value={email}
                onChange={(e) => setEmail(e.target.value)}
              />
            </div>

            <Button type="submit" className="w-full" disabled={isSubmitting}>
              {isSubmitting && <LoadingSpinner size="sm" className="mr-2" />}
              Continue
            </Button>
          </form>
        </div>
      </AuthCard>
    );
  }

  /* -- Step 2: Fields & Password ------------------------------ */

  return (
    <AuthCard
      title="Complete your account"
      description={email}
      logo={logo}
      footer={footer}
      align={align}
      variant={variant}
      className={cn(className)}
    >
      <div className="grid gap-4">
        <button
          type="button"
          onClick={goBack}
          className="inline-flex items-center gap-1.5 text-[13px] text-muted-foreground transition-colors hover:text-foreground"
        >
          <ArrowLeft className="h-3.5 w-3.5" />
          Use a different email
        </button>

        <form onSubmit={handleSignUp} className="grid gap-3">
          <ErrorDisplay error={error} />

          {/* Dynamic fields from config */}
          {signupFields ? (
            signupFields.map((field) => (
              <DynamicField
                key={field.key}
                field={field}
                value={fieldValues[field.key] ?? ""}
                onChange={(v) => setFieldValue(field.key, v)}
                disabled={isSubmitting}
              />
            ))
          ) : (
            /* Fallback: first & last name fields */
            <div className="grid grid-cols-2 gap-3">
              <div className="grid gap-1.5">
                <Label htmlFor="signup-first-name" className="text-[13px]">
                  First name
                  <span className="ml-1 text-muted-foreground">(optional)</span>
                </Label>
                <Input
                  id="signup-first-name"
                  type="text"
                  placeholder="John"
                  autoComplete="given-name"
                  disabled={isSubmitting}
                  value={fallbackFirstName}
                  onChange={(e) => setFallbackFirstName(e.target.value)}
                />
              </div>
              <div className="grid gap-1.5">
                <Label htmlFor="signup-last-name" className="text-[13px]">
                  Last name
                  <span className="ml-1 text-muted-foreground">(optional)</span>
                </Label>
                <Input
                  id="signup-last-name"
                  type="text"
                  placeholder="Doe"
                  autoComplete="family-name"
                  disabled={isSubmitting}
                  value={fallbackLastName}
                  onChange={(e) => setFallbackLastName(e.target.value)}
                />
              </div>
            </div>
          )}

          <div className="grid gap-1.5">
            <div className="flex items-center justify-between">
              <Label htmlFor="signup-password" className="text-[13px]">
                Password
              </Label>
              {forgotPasswordUrl && (
                <a
                  href={forgotPasswordUrl}
                  className="text-[13px] text-muted-foreground transition-colors hover:text-foreground"
                >
                  Forgot password?
                </a>
              )}
            </div>
            <PasswordInput
              id="signup-password"
              placeholder="Create a password"
              autoComplete="new-password"
              required
              autoFocus
              disabled={isSubmitting}
              value={password}
              onChange={(e) => setPassword(e.target.value)}
            />
          </div>

          {captchaRequired && !captchaEnabled && (
            <ErrorDisplay error="Verification is unavailable. Please try again later." />
          )}
          {captchaEnabled && captchaCfg?.site_key && (
            <TurnstileWidget
              key={`${captchaCfg?.provider}:${captchaCfg?.site_key}:${captchaAttempt}`}
              siteKey={captchaCfg.site_key}
              onToken={setCaptchaToken}
              onExpire={() => setCaptchaToken(null)}
              onError={() => setCaptchaToken(null)}
            />
          )}

          <Button
            type="submit"
            className="w-full"
            disabled={isSubmitting || (captchaRequired && !captchaToken)}
          >
            {isSubmitting && <LoadingSpinner size="sm" className="mr-2" />}
            Create account
          </Button>
        </form>
      </div>
    </AuthCard>
  );
}
