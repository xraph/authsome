import React, { useState } from "react";
import type { Meta, StoryObj } from "@storybook/react";
import {
  SignInForm,
  MFAChallengeFormStyled,
  UserAvatar,
  UserProfileCard,
  SessionList,
  DeviceList,
  ChangePasswordForm,
  PasskeyList,
  Button,
  Tabs,
  TabsList,
  TabsTrigger,
  TabsContent,
  Separator,
  Badge,
} from "@authsome/ui-components";
import { MockAuthProvider, MOCK_USER } from "../../mocks/auth-provider";
import { CONFIG_ALL_ENABLED } from "../../mocks/client-config-presets";
import {
  Check,
  LogOut,
  User,
  Shield,
  Monitor,
  Fingerprint,
  KeyRound,
} from "lucide-react";

/**
 * Complete Auth Journey — a full-page, multi-step flow demonstrating
 * the entire authentication lifecycle:
 *
 * 1. Sign In (with social + email)
 * 2. MFA Challenge (6-digit TOTP code)
 * 3. Authenticated Account Dashboard (tabbed: profile, security, sessions, devices)
 */
const meta: Meta = {
  title: "Flows/Complete Auth Journey",
  tags: ["autodocs"],
  parameters: { layout: "fullscreen" },
};
export default meta;
type Story = StoryObj;

type FlowStep = "sign-in" | "mfa" | "dashboard";

/* ─────────────────────────────────────────────────── */
/*  Reusable Account Dashboard                         */
/* ─────────────────────────────────────────────────── */

function AccountDashboard({ onSignOut }: { onSignOut?: () => void }) {
  return (
    <div className="mx-auto w-full max-w-[720px]">
      {/* ── Header ──────────────────────────────── */}
      <div className="mb-6 flex items-center justify-between">
        <div className="flex items-center gap-3">
          <UserAvatar user={MOCK_USER} size="lg" />
          <div>
            <h1 className="text-base font-semibold leading-tight">
              {MOCK_USER.name}
            </h1>
            <p className="text-[13px] text-muted-foreground">
              {MOCK_USER.email}
            </p>
          </div>
        </div>
        {onSignOut && (
          <Button variant="ghost" size="sm" onClick={onSignOut}>
            <LogOut className="mr-1.5 h-3.5 w-3.5" />
            Sign out
          </Button>
        )}
      </div>

      {/* ── Tabbed content ──────────────────────── */}
      <Tabs defaultValue="profile" className="w-full">
        <TabsList className="mb-4 w-full justify-start gap-1 bg-transparent p-0">
          <TabsTrigger
            value="profile"
            className="gap-1.5 rounded-md data-[state=active]:bg-muted"
          >
            <User className="h-3.5 w-3.5" />
            Profile
          </TabsTrigger>
          <TabsTrigger
            value="security"
            className="gap-1.5 rounded-md data-[state=active]:bg-muted"
          >
            <Shield className="h-3.5 w-3.5" />
            Security
          </TabsTrigger>
          <TabsTrigger
            value="sessions"
            className="gap-1.5 rounded-md data-[state=active]:bg-muted"
          >
            <Monitor className="h-3.5 w-3.5" />
            Sessions
            <Badge
              variant="secondary"
              className="ml-0.5 h-4 min-w-[16px] px-1 text-[10px]"
            >
              3
            </Badge>
          </TabsTrigger>
          <TabsTrigger
            value="devices"
            className="gap-1.5 rounded-md data-[state=active]:bg-muted"
          >
            <Fingerprint className="h-3.5 w-3.5" />
            Devices
          </TabsTrigger>
        </TabsList>

        <Separator className="mb-4 -mt-2" />

        <TabsContent value="profile">
          <UserProfileCard className="max-w-full shadow-none" />
        </TabsContent>

        <TabsContent value="security">
          <div className="grid gap-6">
            {/* Password section */}
            <section>
              <div className="mb-2 flex items-center gap-2">
                <KeyRound className="h-3.5 w-3.5 text-muted-foreground" />
                <h3 className="text-[13px] font-medium text-muted-foreground">
                  Password
                </h3>
              </div>
              <ChangePasswordForm />
            </section>

            <Separator />

            {/* Passkeys section */}
            <section>
              <div className="mb-2 flex items-center gap-2">
                <Fingerprint className="h-3.5 w-3.5 text-muted-foreground" />
                <h3 className="text-[13px] font-medium text-muted-foreground">
                  Passkeys
                </h3>
              </div>
              <PasskeyList />
            </section>
          </div>
        </TabsContent>

        <TabsContent value="sessions">
          <SessionList />
        </TabsContent>

        <TabsContent value="devices">
          <DeviceList />
        </TabsContent>
      </Tabs>
    </div>
  );
}

/* ─────────────────────────────────────────────────── */
/*  Full-page shell with subtle background             */
/* ─────────────────────────────────────────────────── */

function PageShell({ children }: { children: React.ReactNode }) {
  return (
    <div className="min-h-screen bg-gradient-to-b from-muted/40 to-background">
      <div className="mx-auto max-w-3xl px-4 py-10 sm:px-6">{children}</div>
    </div>
  );
}

/* ─────────────────────────────────────────────────── */
/*  Full Journey renderer                              */
/* ─────────────────────────────────────────────────── */

function FullAuthFlowRenderer() {
  const [step, setStep] = useState<FlowStep>("sign-in");

  return (
    <MockAuthProvider
      clientConfig={CONFIG_ALL_ENABLED}
      signInBehavior="mfa_required"
      initialState={step === "dashboard" ? "authenticated" : "unauthenticated"}
      delay={800}
    >
      <PageShell>
        {step === "sign-in" && (
          <div className="flex min-h-[70vh] items-center justify-center">
            <div className="w-full max-w-[380px]">
              <SignInForm
                signUpUrl="/sign-up"
                forgotPasswordUrl="/forgot-password"
                onSuccess={() => setStep("mfa")}
              />
            </div>
          </div>
        )}

        {step === "mfa" && (
          <div className="flex min-h-[70vh] items-center justify-center">
            <div className="w-full max-w-[380px]">
              <MFAChallengeFormStyled
                enrollmentId="enroll_mock_totp"
                onSuccess={() => setStep("dashboard")}
              />
            </div>
          </div>
        )}

        {step === "dashboard" && (
          <>
            {/* Success toast */}
            <div className="mb-6 flex items-center gap-2.5 rounded-lg border border-green-200 bg-green-50 px-4 py-2.5 dark:border-green-900/40 dark:bg-green-950/30">
              <div className="flex h-5 w-5 shrink-0 items-center justify-center rounded-full bg-green-600">
                <Check className="h-3 w-3 text-white" />
              </div>
              <p className="text-[13px] text-green-800 dark:text-green-300">
                Signed in successfully with MFA verification
              </p>
            </div>

            <AccountDashboard onSignOut={() => setStep("sign-in")} />
          </>
        )}
      </PageShell>
    </MockAuthProvider>
  );
}

/** Complete flow: Sign In → MFA → Account Dashboard. */
export const FullJourney: Story = {
  render: () => <FullAuthFlowRenderer />,
};

/** Account dashboard only — starts already authenticated. */
export const DashboardOnly: Story = {
  render: () => (
    <MockAuthProvider
      clientConfig={CONFIG_ALL_ENABLED}
      initialState="authenticated"
      delay={800}
    >
      <PageShell>
        <AccountDashboard />
      </PageShell>
    </MockAuthProvider>
  ),
};
