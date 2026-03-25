import React, { useCallback, useMemo, useState } from "react";
import {
  AuthContext,
  type AuthContextValue,
} from "@authsome/ui-react";
import type { AuthState, User, Session, AuthClient, ClientConfig, Organization } from "@authsome/ui-core";

/** Mock user for Storybook stories. */
export const MOCK_USER: User = {
  id: "user_mock_123",
  email: "jane@example.com",
  email_verified: true,
  name: "Jane Doe",
  username: "janedoe",
  image: "",
  phone: "+1234567890",
  banned: false,
  metadata: {},
  created_at: "2024-01-01T00:00:00Z",
  updated_at: "2024-06-01T00:00:00Z",
};

/** Mock session for Storybook stories. */
export const MOCK_SESSION: Session = {
  session_token: "mock_session_token_abc123",
  refresh_token: "mock_refresh_token_xyz789",
  expires_at: "2099-12-31T23:59:59Z",
};

/** Mock organizations for Storybook stories. */
export const MOCK_ORGANIZATIONS: Organization[] = [
  {
    id: "org_1",
    name: "Acme Corp",
    slug: "acme-corp",
    app_id: "app_1",
    created_by: "user_mock_123",
    created_at: "2024-01-01T00:00:00Z",
    updated_at: "2024-06-01T00:00:00Z",
  },
  {
    id: "org_2",
    name: "Widgets Inc",
    slug: "widgets-inc",
    app_id: "app_1",
    created_by: "user_mock_123",
    created_at: "2024-03-01T00:00:00Z",
    updated_at: "2024-06-01T00:00:00Z",
  },
  {
    id: "org_3",
    name: "Startup Labs",
    slug: "startup-labs",
    app_id: "app_1",
    created_by: "user_mock_456",
    created_at: "2024-05-01T00:00:00Z",
    updated_at: "2024-06-01T00:00:00Z",
  },
];

/** Mock passkeys for Storybook stories. */
export const MOCK_PASSKEYS = [
  {
    id: "cred_1",
    display_name: "MacBook Pro Touch ID",
    created_at: "2024-06-15T10:30:00Z",
    transport: ["internal"],
  },
  {
    id: "cred_2",
    display_name: "YubiKey 5C",
    created_at: "2024-08-20T14:00:00Z",
    transport: ["usb", "nfc"],
  },
];

/** Mock devices for Storybook stories. */
export const MOCK_DEVICES = [
  {
    id: "dev_1",
    app_id: "app_1",
    user_id: "user_mock_123",
    name: "Chrome on MacBook Pro",
    browser: "Chrome 120",
    os: "macOS 14.2",
    ip_address: "192.168.1.100",
    type: "desktop",
    trusted: true,
    fingerprint: "fp_abc123",
    last_seen_at: new Date(Date.now() - 1000 * 60 * 5).toISOString(), // 5 min ago
    created_at: "2024-01-15T08:00:00Z",
    updated_at: new Date(Date.now() - 1000 * 60 * 5).toISOString(),
  },
  {
    id: "dev_2",
    app_id: "app_1",
    user_id: "user_mock_123",
    name: "Safari on iPhone",
    browser: "Safari 17",
    os: "iOS 17.2",
    ip_address: "10.0.0.50",
    type: "mobile",
    trusted: true,
    fingerprint: "fp_def456",
    last_seen_at: new Date(Date.now() - 1000 * 60 * 60 * 2).toISOString(), // 2 hours ago
    created_at: "2024-03-10T12:00:00Z",
    updated_at: new Date(Date.now() - 1000 * 60 * 60 * 2).toISOString(),
  },
  {
    id: "dev_3",
    app_id: "app_1",
    user_id: "user_mock_123",
    name: "Firefox on iPad",
    browser: "Firefox 121",
    os: "iPadOS 17.2",
    ip_address: "172.16.0.25",
    type: "tablet",
    trusted: false,
    fingerprint: "fp_ghi789",
    last_seen_at: new Date(Date.now() - 1000 * 60 * 60 * 24 * 3).toISOString(), // 3 days ago
    created_at: "2024-05-20T16:00:00Z",
    updated_at: new Date(Date.now() - 1000 * 60 * 60 * 24 * 3).toISOString(),
  },
];

/** Mock sessions for Storybook stories. */
export const MOCK_SESSIONS = [
  {
    id: "sess_1",
    session_token: "mock_session_token_abc123",
    device: "Chrome on MacBook Pro",
    ip_address: "192.168.1.100",
    last_active: new Date(Date.now() - 1000 * 60 * 2).toISOString(),
    created_at: "2024-06-01T08:00:00Z",
  },
  {
    id: "sess_2",
    session_token: "sess_token_other_1",
    device: "Safari on iPhone",
    ip_address: "10.0.0.50",
    last_active: new Date(Date.now() - 1000 * 60 * 60 * 4).toISOString(),
    created_at: "2024-06-10T12:00:00Z",
  },
  {
    id: "sess_3",
    session_token: "sess_token_other_2",
    device: "Firefox on Windows",
    ip_address: "172.16.0.25",
    last_active: new Date(Date.now() - 1000 * 60 * 60 * 24).toISOString(),
    created_at: "2024-06-15T16:00:00Z",
  },
];

/** Initial state presets for stories. */
export type MockInitialState =
  | "authenticated"
  | "unauthenticated"
  | "loading"
  | "mfa_required"
  | "error";

function stateFromPreset(
  preset: MockInitialState,
  user: User,
  session: Session,
): AuthState {
  switch (preset) {
    case "authenticated":
      return { status: "authenticated", user, session };
    case "unauthenticated":
      return { status: "unauthenticated" };
    case "loading":
      return { status: "loading" };
    case "mfa_required":
      return { status: "mfa_required", session };
    case "error":
      return { status: "error", error: "Something went wrong" };
  }
}

export interface MockAuthProviderProps {
  children: React.ReactNode;
  initialState?: MockInitialState;
  /** Simulated delay for async operations in ms. */
  delay?: number;
  /** If true, signIn/signUp will throw an error. */
  simulateError?: boolean;
  /** Custom error message for simulated errors. */
  errorMessage?: string;
  /** Custom mock user. */
  user?: User;
  /** Optional client config for auto-configuration stories. */
  clientConfig?: ClientConfig;
}

/**
 * MockAuthProvider provides controllable auth state for Storybook stories.
 * It provides values to the real AuthContext so `useAuth()` works in components.
 */
export function MockAuthProvider({
  children,
  initialState = "unauthenticated",
  delay = 1000,
  simulateError = false,
  errorMessage = "Invalid email or password",
  user = MOCK_USER,
  clientConfig = null,
}: MockAuthProviderProps) {
  const [state, setState] = useState<AuthState>(
    stateFromPreset(initialState, user, MOCK_SESSION),
  );

  const wait = useCallback(
    () => new Promise<void>((resolve) => setTimeout(resolve, delay)),
    [delay],
  );

  const signIn = useCallback(
    async (_email: string, _password: string) => {
      setState({ status: "loading" });
      await wait();
      if (simulateError) {
        setState({ status: "error", error: errorMessage });
        throw new Error(errorMessage);
      }
      setState({ status: "authenticated", user, session: MOCK_SESSION });
    },
    [wait, simulateError, errorMessage, user],
  );

  const signUp = useCallback(
    async (_email: string, _password: string, _name?: string) => {
      setState({ status: "loading" });
      await wait();
      if (simulateError) {
        setState({ status: "error", error: errorMessage });
        throw new Error(errorMessage);
      }
      setState({ status: "authenticated", user, session: MOCK_SESSION });
    },
    [wait, simulateError, errorMessage, user],
  );

  const signOut = useCallback(async () => {
    setState({ status: "loading" });
    await wait();
    setState({ status: "unauthenticated" });
  }, [wait]);

  const submitMFACode = useCallback(
    async (_enrollmentId: string, _code: string) => {
      setState({ status: "loading" });
      await wait();
      if (simulateError) {
        setState({ status: "mfa_required", session: MOCK_SESSION });
        throw new Error("Invalid MFA code");
      }
      setState({ status: "authenticated", user, session: MOCK_SESSION });
    },
    [wait, simulateError, user],
  );

  const submitRecoveryCode = useCallback(
    async (_code: string) => {
      setState({ status: "loading" });
      await wait();
      setState({ status: "authenticated", user, session: MOCK_SESSION });
    },
    [wait, user],
  );

  const mockClient = useMemo(() => {
    const client = {
      forgotPassword: async () => {
        await wait();
        if (simulateError) throw new Error("Email not found");
      },
      resetPassword: async () => {
        await wait();
        if (simulateError) throw new Error("Invalid or expired token");
      },
      changePassword: async () => {
        await wait();
        if (simulateError) throw new Error("Current password is incorrect");
      },
      updateMe: async (body: any) => {
        await wait();
        if (simulateError) throw new Error("Failed to update profile");
        return { ...user, ...body };
      },
      listOrganizations: async () => {
        await wait();
        return { items: MOCK_ORGANIZATIONS, total: MOCK_ORGANIZATIONS.length, organizations: MOCK_ORGANIZATIONS };
      },
      createOrganization: async (body: any) => {
        await wait();
        return {
          ...body,
          id: "org_new",
          app_id: "app_1",
          created_by: user.id,
          created_at: new Date().toISOString(),
          updated_at: new Date().toISOString(),
        };
      },
      getMe: async () => {
        await wait();
        return user;
      },
      sendMagicLink: async () => {
        await wait();
        if (simulateError) throw new Error("Email not found");
      },
      // Passkey methods
      listPasskeys: async () => {
        await wait();
        return { credentials: MOCK_PASSKEYS };
      },
      passkeyLoginBegin: async () => {
        await wait();
        return { options: {} };
      },
      passkeyLoginFinish: async () => {
        await wait();
        if (simulateError) throw new Error("Passkey verification failed");
        return { status: "ok", user_id: user.id };
      },
      passkeyRegisterBegin: async () => {
        await wait();
        return { options: {} };
      },
      passkeyRegisterFinish: async () => {
        await wait();
        return { id: "cred_new", display_name: "New Passkey", status: "ok" };
      },
      deletePasskey: async () => {
        await wait();
        return { status: "ok" };
      },
      // Device methods
      listDevices: async () => {
        await wait();
        return { devices: MOCK_DEVICES };
      },
      trustDevice: async (_deviceId: string) => {
        await wait();
        const device = MOCK_DEVICES.find((d) => d.id === _deviceId);
        return { ...device, trusted: !device?.trusted };
      },
      deleteDevice: async () => {
        await wait();
        return { status: "ok" };
      },
      // Session methods
      listSessions: async () => {
        await wait();
        return { sessions: MOCK_SESSIONS };
      },
      revokeSession: async () => {
        await wait();
        return { status: "ok" };
      },
    };
    return client as unknown as AuthClient;
  }, [wait, simulateError, user]);

  const currentUser = state.status === "authenticated" ? state.user : null;
  const currentSession =
    state.status === "authenticated" || state.status === "mfa_required"
      ? state.session
      : null;

  const value = useMemo<AuthContextValue>(
    () => ({
      state,
      manager: {} as AuthContextValue["manager"],
      client: mockClient,
      user: currentUser,
      session: currentSession,
      isAuthenticated: state.status === "authenticated",
      isLoading: state.status === "loading",
      clientConfig: clientConfig ?? null,
      isConfigLoaded: clientConfig !== null,
      signIn,
      signUp,
      signOut,
      submitMFACode,
      submitRecoveryCode,
      sendSMSCode: async () => ({ sent: true, phone_masked: "+1***1234", expires_in_seconds: 300 }),
      submitSMSCode: async () => {},
    }),
    [
      state,
      mockClient,
      currentUser,
      currentSession,
      clientConfig,
      signIn,
      signUp,
      signOut,
      submitMFACode,
      submitRecoveryCode,
    ],
  );

  return (
    <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
  );
}
