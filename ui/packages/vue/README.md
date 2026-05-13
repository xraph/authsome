# @authsome/ui-vue

Vue 3 composables for AuthSome. This package is **headless** — it ships
composables (`useAuth`, `useClientConfig`, `useUser`, `useOrganizations`,
`useSessionToken`) and no rendered components, so consumers own all
markup and styling.

## Setup

```ts
// main.ts
import { createApp } from "vue";
import { createAuthPlugin } from "@authsome/ui-vue";
import App from "./App.vue";

const app = createApp(App);
app.use(
  createAuthPlugin({
    baseURL: "https://api.example.com",
    publishableKey: "pk_…", // required to fetch /v1/client-config
  }),
);
app.mount("#app");
```

## Phase 3B: verification panels and captcha

`useAuth` exposes the same surface as `@authsome/ui-react`:

- `state.value.status === "verification_pending"` after `signUp()` resolves.
- `state.value.status === "email_not_verified"` after `signIn()` rejects with
  a backend error of `type: "email_not_verified"`.
- `resendVerification(email)` calls `POST /v1/verify-email/resend`.
- `signIn` and `signUp` both accept an optional third/fourth `options` arg of
  shape `{ captchaToken?: string }`.
- `useClientConfig()` exposes `config.value.captcha = { required, provider, site_key }`
  so the template can decide whether to render Turnstile.

### Sign-up template

```vue
<script setup lang="ts">
import { ref, computed } from "vue";
import { useAuth, useClientConfig } from "@authsome/ui-vue";

const { state, signUp, resendVerification } = useAuth();
const { config } = useClientConfig();

const email = ref("");
const password = ref("");
const captchaToken = ref<string | null>(null);
const error = ref<string | null>(null);
const isSubmitting = ref(false);

const captchaRequired = computed(
  () =>
    config.value?.captcha?.required === true &&
    config.value?.captcha?.provider === "turnstile" &&
    !!config.value?.captcha?.site_key,
);

async function onSubmit() {
  error.value = null;
  isSubmitting.value = true;
  try {
    await signUp(email.value, password.value, undefined, {
      captchaToken: captchaToken.value ?? undefined,
    });
  } catch (e) {
    error.value = e instanceof Error ? e.message : "Sign-up failed";
  } finally {
    isSubmitting.value = false;
  }
}
</script>

<template>
  <div v-if="state.status === 'verification_pending'">
    <h2>Check your inbox</h2>
    <p>We sent a verification link to {{ state.email }}.</p>
    <button @click="resendVerification(state.email)">Resend email</button>
  </div>

  <form v-else @submit.prevent="onSubmit">
    <input v-model="email" type="email" required />
    <input v-model="password" type="password" required />

    <!-- Render Turnstile when captcha is required. See <TurnstileWidget> below. -->
    <TurnstileWidget
      v-if="captchaRequired"
      :site-key="config!.captcha!.site_key!"
      @token="captchaToken = $event"
    />

    <button
      type="submit"
      :disabled="isSubmitting || (captchaRequired && !captchaToken)"
    >
      Create account
    </button>
    <p v-if="error">{{ error }}</p>
  </form>
</template>
```

### Sign-in template

```vue
<script setup lang="ts">
import { ref } from "vue";
import { useAuth, useClientConfig } from "@authsome/ui-vue";

const { state, signIn, resendVerification } = useAuth();
const { config } = useClientConfig();

const email = ref("");
const password = ref("");
const captchaToken = ref<string | null>(null);
const error = ref<string | null>(null);

async function onSubmit() {
  error.value = null;
  try {
    await signIn(email.value, password.value, {
      captchaToken: captchaToken.value ?? undefined,
    });
  } catch (e) {
    // The manager has already promoted email_not_verified errors into
    // state.status === "email_not_verified" — only show the message for
    // unrelated failures.
    if (state.value.status !== "email_not_verified") {
      error.value = e instanceof Error ? e.message : "Sign-in failed";
    }
  }
}
</script>

<template>
  <div v-if="state.status === 'email_not_verified'">
    <h2>Verify your email</h2>
    <p>{{ state.email }} hasn't been verified yet.</p>
    <button @click="resendVerification(state.email)">Resend link</button>
  </div>

  <form v-else @submit.prevent="onSubmit">
    <input v-model="email" type="email" required />
    <input v-model="password" type="password" required />
    <TurnstileWidget
      v-if="config?.captcha?.required && config.captcha.provider === 'turnstile' && config.captcha.site_key"
      :site-key="config.captcha.site_key"
      @token="captchaToken = $event"
    />
    <button type="submit">Sign in</button>
    <p v-if="error">{{ error }}</p>
  </form>
</template>
```

### TurnstileWidget.vue

The React package ships a `<TurnstileWidget>` in `@authsome/ui-components`.
For Vue, drop the following SFC into your project (it loads the Turnstile
script once and re-uses it across mounts):

```vue
<script setup lang="ts">
import { onMounted, onBeforeUnmount, ref } from "vue";

const props = defineProps<{ siteKey: string }>();
const emit = defineEmits<{ (e: "token", token: string): void }>();

const container = ref<HTMLDivElement | null>(null);
const widgetId = ref<string | null>(null);

const SCRIPT_SRC = "https://challenges.cloudflare.com/turnstile/v0/api.js";

declare global {
  interface Window {
    turnstile?: {
      render: (
        el: HTMLElement,
        opts: { sitekey: string; callback: (token: string) => void },
      ) => string;
      remove: (id: string) => void;
    };
  }
}

function ensureScript(): Promise<void> {
  if (typeof window === "undefined") return Promise.resolve();
  if (window.turnstile) return Promise.resolve();
  if (document.querySelector(`script[src="${SCRIPT_SRC}"]`)) {
    return new Promise((resolve) => {
      const tick = () => (window.turnstile ? resolve() : setTimeout(tick, 50));
      tick();
    });
  }
  return new Promise((resolve, reject) => {
    const s = document.createElement("script");
    s.src = SCRIPT_SRC;
    s.async = true;
    s.defer = true;
    s.onload = () => resolve();
    s.onerror = () => reject(new Error("Failed to load Turnstile"));
    document.head.appendChild(s);
  });
}

onMounted(async () => {
  await ensureScript();
  if (!container.value || !window.turnstile) return;
  widgetId.value = window.turnstile.render(container.value, {
    sitekey: props.siteKey,
    callback: (token) => emit("token", token),
  });
});

onBeforeUnmount(() => {
  if (widgetId.value && window.turnstile) {
    window.turnstile.remove(widgetId.value);
  }
});
</script>

<template>
  <div ref="container" />
</template>
```

The script-load + cleanup logic mirrors
`ui/packages/components/src/components/turnstile-widget.tsx` in the React
implementation.
