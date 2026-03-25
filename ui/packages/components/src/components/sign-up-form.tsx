"use client";

import * as React from "react";
import { useState } from "react";
import { useAuth, useClientConfig, type SignupFieldConfig } from "@authsome/ui-react";
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
import { AuthCard, type AuthCardAlign, type AuthCardVariant } from "./auth-card";
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

export interface SignUpFormComponentProps {
  /** Callback invoked after a successful sign-up. */
  onSuccess?: () => void;
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
}


/**
 * Renders a single dynamic signup field based on its type.
 */
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
    case "select":
      return (
        <div className="grid gap-1.5">
          {label}
          <Select value={value} onValueChange={onChange} disabled={disabled}>
            <SelectTrigger id={fieldId}>
              <SelectValue placeholder={field.placeholder || `Select ${field.label.toLowerCase()}`} />
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
                — {field.description}
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
      const inputType =
        field.type === "radio" ? "text" : field.type || "text";
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
 * - Explicit props always take precedence over auto-discovered values.
 */
export function SignUpForm({
  onSuccess,
  signInUrl,
  forgotPasswordUrl,
  socialProviders: socialProvidersProp,
  onSocialLogin: onSocialLoginProp,
  socialLayout,
  logo,
  align,
  variant,
  className,
}: SignUpFormComponentProps) {
  const { signUp, client } = useAuth();
  const { config } = useClientConfig();

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

  const [step, setStep] = useState<"email" | "details">("email");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);

  // Dynamic field values — keyed by field key.
  const [fieldValues, setFieldValues] = useState<Record<string, string>>({});

  // Initialize defaults when signup fields change.
  React.useEffect(() => {
    if (signupFields) {
      const defaults: Record<string, string> = {};
      for (const f of signupFields) {
        if (f.default && !fieldValues[f.key]) {
          defaults[f.key] = f.default;
        }
      }
      if (Object.keys(defaults).length > 0) {
        setFieldValues((prev) => ({ ...defaults, ...prev }));
      }
    }
  }, [signupFields]); // eslint-disable-line react-hooks/exhaustive-deps

  const setFieldValue = (key: string, value: string) => {
    setFieldValues((prev) => ({ ...prev, [key]: value }));
  };

  // Fallback: when no signup fields are configured, show the classic "Name" field.
  const [fallbackName, setFallbackName] = useState("");

  const handleEmailContinue = (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    setError(null);

    if (!email.trim()) {
      setError("Please enter your email address.");
      return;
    }

    setStep("details");
  };

  const handleSignUp = async (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    setError(null);
    setIsSubmitting(true);

    try {
      // Build the fields object.
      const fields: Record<string, string> = { ...fieldValues };

      // If using fallback (no dynamic fields), map name to first_name.
      if (!signupFields && fallbackName) {
        fields.first_name = fallbackName;
      }

      await signUp(
        email,
        password,
        Object.keys(fields).length > 0 ? fields : undefined,
      );
      onSuccess?.();
    } catch (err) {
      setError(
        err instanceof Error
          ? err.message
          : "Sign up failed. Please try again.",
      );
    } finally {
      setIsSubmitting(false);
    }
  };

  const goBack = () => {
    setStep("email");
    setPassword("");
    setError(null);
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

  /* -- Password disabled: show only social sign-up ----------- */

  if (!showPassword) {
    return (
      <AuthCard
        title="Create an account"
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
        title="Create an account"
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

            <Button
              type="submit"
              className="w-full"
              disabled={isSubmitting}
            >
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
            /* Fallback: classic Name field */
            <div className="grid gap-1.5">
              <Label htmlFor="signup-name" className="text-[13px]">
                Name
                <span className="ml-1 text-muted-foreground">(optional)</span>
              </Label>
              <Input
                id="signup-name"
                type="text"
                placeholder="John Doe"
                autoComplete="name"
                disabled={isSubmitting}
                value={fallbackName}
                onChange={(e) => setFallbackName(e.target.value)}
              />
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

          <Button
            type="submit"
            className="w-full"
            disabled={isSubmitting}
          >
            {isSubmitting && <LoadingSpinner size="sm" className="mr-2" />}
            Create account
          </Button>
        </form>
      </div>
    </AuthCard>
  );
}
