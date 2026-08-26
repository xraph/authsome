# consent

Records who agreed to what, when, and against which version of your policy.

Under GDPR the burden is on you to show consent existed, which means keeping the record. This plugin stores a
versioned record per purpose, marketing, analytics, whatever you define, keeps the full
history including revocations, and contributes the lot to the user's data export.

Versioning against a policy version is the part people forget. When your privacy policy
changes, consent gathered under the old one does not automatically carry over.

## Use it when

- You operate under GDPR, CCPA or anything with a similar burden of proof
- Marketing wants to know who opted in and you need one answer everyone trusts
- You need an audit trail showing consent existed at the moment you acted on it

## Skip it when

- You handle no personal data worth consenting to, which is rarer than it sounds
- Your consent already lives in a dedicated CMP and duplicating it here creates two sources of truth
- You have not defined your purposes yet. This plugin stores decisions, it does not design your policy

## Wiring

```go
authsome.WithPlugin(consent.New()),
```

No configuration. The purposes and policy versions are values you pass on the grant call.

## Endpoints

Mounted under `{basePath}/consent`.

| Method | Path | Purpose |
|---|---|---|
| `POST` | `/grant` | Record consent for a purpose at a policy version |
| `POST` | `/revoke` | Record a withdrawal |
| `GET` | `` | List the caller's consent records |

## Lifecycle hooks

None on the sign-in path. It registers a `DataExportContributor`, so consent history is
included in the GDPR export, and a `MigrationProvider` for its tables.

## Related

`organization` contributes memberships to the same export. `oauth2provider` is where consent
matters for third-party access grants.
