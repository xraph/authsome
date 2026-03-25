/**
 * Vue 3 composables for AuthSome authentication.
 *
 * Usage:
 * ```ts
 * // main.ts
 * import { createAuthPlugin } from "@authsome/ui-vue";
 * app.use(createAuthPlugin({ baseURL: "https://api.example.com" }));
 *
 * // Component.vue
 * import { useAuth, useUser } from "@authsome/ui-vue";
 * const { signIn, isAuthenticated, user } = useAuth();
 * ```
 */

import {
  computed,
  inject,
  onMounted,
  onUnmounted,
  readonly,
  ref,
  type App,
  type InjectionKey,
  type Ref,
} from "vue";
import {
  AuthManager,
  type AuthConfig,
  type AuthState,
  type AuthClient,
  type User,
  type Session,
  type Organization,
  type ListResponse,
} from "@authsome/ui-core";

/** Injection key for the AuthManager. */
const AUTH_KEY: InjectionKey<AuthManager> = Symbol("authsome");

/** Injection key for the reactive auth state. */
const AUTH_STATE_KEY: InjectionKey<Ref<AuthState>> = Symbol("authsome:state");

/**
 * Creates a Vue plugin that provides AuthSome authentication.
 *
 * ```ts
 * const app = createApp(App);
 * app.use(createAuthPlugin({ baseURL: "https://api.example.com" }));
 * ```
 */
export function createAuthPlugin(config: AuthConfig) {
  return {
    install(app: App) {
      const manager = new AuthManager(config);
      const state = ref<AuthState>(manager.getState());

      manager.subscribe((s) => {
        state.value = s;
      });

      app.provide(AUTH_KEY, manager);
      app.provide(AUTH_STATE_KEY, state);

      // Initialize on mount (hydrate from storage).
      void manager.initialize();
    },
  };
}

function useManager(): AuthManager {
  const manager = inject(AUTH_KEY);
  if (!manager) {
    throw new Error("useAuth requires the authsome plugin. Call app.use(createAuthPlugin(...)).");
  }
  return manager;
}

function useAuthState(): Ref<AuthState> {
  const state = inject(AUTH_STATE_KEY);
  if (!state) {
    throw new Error("useAuth requires the authsome plugin. Call app.use(createAuthPlugin(...)).");
  }
  return state;
}

/**
 * Main auth composable providing state and actions.
 */
export function useAuth() {
  const manager = useManager();
  const state = useAuthState();

  const isAuthenticated = computed(() => state.value.status === "authenticated");
  const isLoading = computed(() => state.value.status === "loading");

  const user = computed<User | null>(() =>
    state.value.status === "authenticated" ? state.value.user : null,
  );

  const session = computed<Session | null>(() =>
    state.value.status === "authenticated" ? state.value.session : null,
  );

  const error = computed<string | null>(() =>
    state.value.status === "error" ? state.value.error : null,
  );

  async function signIn(email: string, password: string) {
    await manager.signIn({ email, password });
  }

  async function signUp(email: string, password: string, fields?: Record<string, string>) {
    const { first_name, last_name, username, ...rest } = fields ?? {};
    const metadata = Object.keys(rest).length > 0 ? rest : undefined;
    await manager.signUp({ email, password, first_name, last_name, username, metadata });
  }

  async function signOut() {
    await manager.signOut();
  }

  async function submitMFACode(enrollmentId: string, code: string) {
    await manager.submitMFACode(enrollmentId, code);
  }

  async function submitRecoveryCode(code: string) {
    await manager.submitRecoveryCode(code);
  }

  return {
    state: readonly(state),
    isAuthenticated,
    isLoading,
    user,
    session,
    error,
    client: manager.getClient(),
    signIn,
    signUp,
    signOut,
    submitMFACode,
    submitRecoveryCode,
  };
}

/**
 * Composable to fetch and reload the user profile.
 */
export function useUser() {
  const { user: authUser, isLoading, session } = useAuth();
  const manager = useManager();
  const localUser = ref<User | null>(authUser.value);

  // Keep in sync with auth state.
  const stopWatch = computed(() => authUser.value);

  async function reload() {
    if (!session.value) return;
    const u = await manager.getClient().getMe(session.value.session_token);
    localUser.value = u;
  }

  return {
    user: computed(() => localUser.value ?? authUser.value),
    isLoading,
    reload,
  };
}

/**
 * Composable to list organizations.
 */
export function useOrganizations() {
  const { session, isAuthenticated } = useAuth();
  const manager = useManager();

  const organizations = ref<Organization[]>([]);
  const total = ref(0);
  const isLoading = ref(false);

  async function load() {
    if (!session.value) return;
    isLoading.value = true;
    try {
      const res = await manager
        .getClient()
        .listOrganizations(session.value.session_token) as unknown as ListResponse<Organization>;
      organizations.value = res.items;
      total.value = res.total;
    } finally {
      isLoading.value = false;
    }
  }

  onMounted(() => {
    if (isAuthenticated.value) {
      void load();
    }
  });

  return {
    organizations: readonly(organizations),
    total: readonly(total),
    isLoading: readonly(isLoading),
    reload: load,
  };
}

/**
 * Composable that returns the current session token.
 */
export function useSessionToken() {
  const { session } = useAuth();
  return computed(() => session.value?.session_token ?? null);
}
