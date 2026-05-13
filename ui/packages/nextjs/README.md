# @authsome/ui-nextjs

Next.js integration for AuthSome. Combines:

- React provider + headless components from [`@authsome/ui-react`](../react)
- Styled forms (`SignInForm`, `SignUpForm`, …) from
  [`@authsome/ui-components`](../components)
- Next-specific glue (`getServerSession`, cookie storage, proxy handler,
  middleware) and ready-made app-router pages under
  `@authsome/ui-nextjs/pages`.

## Phase 3B: verification UX and captcha

The pre-built pages exported from `@authsome/ui-nextjs/pages`
(`SignInPage`, `SignUpPage`, …) render the styled forms from
`@authsome/ui-components` ≥ 1.4.0. Those components already implement:

- A "check your inbox" panel after sign-up succeeds
  (`AuthState.status === "verification_pending"`).
- A "verify your email" panel with a Resend button after a sign-in attempt
  rejects with `error.type === "email_not_verified"`.
- Cloudflare Turnstile captcha gating, driven by
  `ClientConfig.captcha = { required, provider, site_key }`.

Next.js consumers therefore inherit Phase 3B automatically — there is no
Next-specific code path. To opt in, mount the pages and ensure
`AuthProvider` is configured with a `publishableKey` so client config
(including `captcha` and `email_verification`) is fetched:

```tsx
// app/sign-in/page.tsx
"use client";
import { SignInPage } from "@authsome/ui-nextjs/pages";

export default function Page() {
  return <SignInPage afterSignInUrl="/dashboard" />;
}
```

```tsx
// app/layout.tsx
"use client";
import { AuthProvider } from "@authsome/ui-nextjs";

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html>
      <body>
        <AuthProvider
          baseURL={process.env.NEXT_PUBLIC_AUTHSOME_URL!}
          publishableKey={process.env.NEXT_PUBLIC_AUTHSOME_KEY!}
        >
          {children}
        </AuthProvider>
      </body>
    </html>
  );
}
```

If you bring your own forms, use the headless `SignInForm` /
`SignUpForm` re-exported from this package (which come from
`@authsome/ui-react`) and follow the wiring documented in
`@authsome/ui-react`'s `useAuth` (verification states + `captchaToken`
options).
