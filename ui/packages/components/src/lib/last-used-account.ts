/**
 * The account that last signed in successfully on this browser.
 *
 * This is a convenience and never an authorisation. Nothing stored here is
 * trusted, none of it is sent anywhere, and the person it names still has to
 * authenticate. It exists so a returning visitor does not have to remember
 * which of four providers they used last time.
 *
 * It lives in localStorage rather than on the server because it is a property
 * of the browser, not of the account. Signing in on a laptop should not change
 * what a phone offers.
 */

const KEY = "authsome:last-account";

export interface LastUsedAccount {
  /** The user's id, so a host can match it against its own records. */
  userId: string;
  /** Display name, when the server gave one. */
  displayName?: string;
  /** Email or phone, shown as-is under the name. */
  identifier: string;
  avatarUrl?: string;
  /** The strategy that worked: "password", "google", "oidc:kineta", ... */
  strategy: string;
  /** Human-readable name for the strategy, for "signs in with Kineta". */
  strategyLabel?: string;
  /** When it was written, ISO 8601. */
  at: string;
}

/**
 * How long a remembered account keeps being offered.
 *
 * Ninety days. Long enough that someone who signs in every few months on
 * their own laptop still gets the shortcut, short enough that a library
 * computer or a sold phone stops offering a stranger's name and email address
 * to whoever sits down next. That record grants no access, but it is still
 * someone's identity on a screen they do not control.
 *
 * Returns milliseconds, or Infinity to keep a record forever.
 */
export function maxAgeMs(): number {
  return 90 * 24 * 60 * 60 * 1000;
}

/**
 * Reads the remembered account, or null.
 *
 * Every access is guarded. localStorage throws outright in some privacy modes,
 * returns null when site data was cleared, and holds whatever a previous
 * version of this code wrote, so a bad parse is treated the same as nothing.
 */
export function readLastAccount(): LastUsedAccount | null {
  try {
    const raw = globalThis.localStorage?.getItem(KEY);
    if (!raw) return null;
    const parsed = JSON.parse(raw) as Partial<LastUsedAccount>;
    if (!parsed || typeof parsed.userId !== "string" || typeof parsed.identifier !== "string") {
      return null;
    }
    const at = Date.parse(parsed.at ?? "");
    if (Number.isNaN(at)) return null;
    const max = maxAgeMs();
    if (Number.isFinite(max) && Date.now() - at > max) {
      forgetLastAccount();
      return null;
    }
    return parsed as LastUsedAccount;
  } catch {
    return null;
  }
}

/** Remembers an account. Called after a sign-in actually succeeds. */
export function rememberLastAccount(
  account: Omit<LastUsedAccount, "at"> & { at?: string },
): void {
  try {
    const record: LastUsedAccount = { ...account, at: account.at ?? new Date().toISOString() };
    globalThis.localStorage?.setItem(KEY, JSON.stringify(record));
    emit();
  } catch {
    // A browser that will not store it is a browser that gets no chip. That
    // is the whole consequence, so there is nothing to report.
  }
}

/**
 * Forgets the remembered account.
 *
 * Signing out does NOT call this. Being remembered is the point: the next
 * visit should still know who you are. The dismiss control on the chip calls
 * it, and so should a "this is not my device" affordance if a host adds one.
 */
export function forgetLastAccount(): void {
  try {
    globalThis.localStorage?.removeItem(KEY);
    emit();
  } catch {
    // Nothing to do. If it cannot be removed it could not have been read.
  }
}

// ── Store plumbing ─────────────────────────────────────
//
// The chip reads this through useSyncExternalStore, which needs a snapshot
// that is referentially stable between renders. readLastAccount() parses
// fresh JSON every call and would hand back a new object each time, so React
// would see a changed store on every render and loop. The cache below returns
// the same object until the stored string actually changes.

let cachedRaw: string | null = null;
let cachedValue: LastUsedAccount | null = null;

const listeners = new Set<() => void>();

function emit(): void {
  for (const l of listeners) l();
}

/** Subscribes to changes, including ones made in another tab. */
export function subscribeToLastAccount(listener: () => void): () => void {
  listeners.add(listener);
  const onStorage = (e: StorageEvent) => {
    if (e.key === KEY || e.key === null) emit();
  };
  globalThis.addEventListener?.("storage", onStorage);
  return () => {
    listeners.delete(listener);
    globalThis.removeEventListener?.("storage", onStorage);
  };
}

/** The current record, stable until the stored value changes. */
export function getLastAccountSnapshot(): LastUsedAccount | null {
  let raw: string | null;
  try {
    raw = globalThis.localStorage?.getItem(KEY) ?? null;
  } catch {
    raw = null;
  }
  if (raw !== cachedRaw) {
    cachedRaw = raw;
    cachedValue = readLastAccount();
  }
  return cachedValue;
}

/**
 * The snapshot React uses while rendering on the server, and for the first
 * client render that has to match it. There is no localStorage there, so the
 * answer is always "nothing remembered" and the chip appears after hydration.
 */
export function getLastAccountServerSnapshot(): LastUsedAccount | null {
  return null;
}
