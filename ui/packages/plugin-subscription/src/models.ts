/**
 * Typed models for the subscription plugin, mapped from the server's JSON.
 * Field names are camelCase; times stay RFC 3339 strings, so they survive
 * serialization (server components, caches) unchanged.
 */

/** A subscription's state. Unknown values from a newer server pass through. */
export type SubscriptionStatus =
  | "active"
  | "trialing"
  | "past_due"
  | "canceled"
  | "expired"
  | "paused"
  | (string & {});

/** An amount in the currency's minor unit (cents for usd). */
export interface Money {
  amountMinor: number;
  currency: string;
}

/** A plan's base price and how often it's charged. */
export interface Price extends Money {
  /** "monthly", "yearly", ... as the server names it. */
  interval: string;
}

/** One feature a plan includes. A null limit means unlimited. */
export interface PlanFeature {
  key: string;
  name: string;
  /** "boolean", "metered" or "seat". */
  type: string;
  limit: number | null;
  period?: string;
  softLimit: boolean;
}

export interface Plan {
  id: string;
  slug: string;
  name: string;
  description: string;
  status: string;
  trialDays: number;
  isAddon: boolean;
  /** Null for a plan without a price. */
  price: Price | null;
  features: PlanFeature[];
  metadata: Record<string, string>;
}

/** Seat use on a plan with a seat feature. A null limit means unlimited. */
export interface Seats {
  featureKey: string;
  used: number;
  limit: number | null;
}

export interface Subscription {
  id: string;
  tenantId: string;
  status: SubscriptionStatus;
  plan: Plan;
  currentPeriodStart: string;
  currentPeriodEnd: string;
  trialStart: string | null;
  trialEnd: string | null;
  /** When a cancellation scheduled for the period end takes effect. */
  cancelAt: string | null;
  canceledAt: string | null;
  pausedAt: string | null;
  seats: Seats | null;
  /** The tenant has the plan's features: active, trialing, or past due. */
  isActive: boolean;
  isTrialing: boolean;
  isPaused: boolean;
  /** A cancellation is scheduled; the plan runs until cancelAt. */
  willCancel: boolean;
  /** Whole days of trial left, null when not trialing. */
  trialDaysLeft: number | null;
}

/** Whether the tenant may use a feature, and how much of it is left. */
export interface Entitlement {
  allowed: boolean;
  used: number;
  /** -1 or a large number from the server means unlimited, as ledger reports it. */
  limit: number;
  remaining: number;
  reason?: string;
}

// ── Mapping from the server's JSON ────────────────

interface PlanFeatureJSON {
  key: string;
  name: string;
  type: string;
  limit: number | null;
  period?: string;
  soft_limit?: boolean;
}

export interface PlanJSON {
  id: string;
  slug: string;
  name: string;
  description?: string;
  status: string;
  trial_days?: number;
  is_addon?: boolean;
  currency: string;
  amount_minor?: number;
  billing_period?: string;
  features?: PlanFeatureJSON[];
  metadata?: Record<string, string>;
}

export interface SubscriptionJSON {
  id: string;
  tenant_id: string;
  status: string;
  plan: PlanJSON;
  current_period_start: string;
  current_period_end: string;
  trial_start?: string | null;
  trial_end?: string | null;
  cancel_at?: string | null;
  canceled_at?: string | null;
  paused_at?: string | null;
  seats?: { feature_key: string; used: number; limit: number | null } | null;
}

export interface EntitlementJSON {
  allowed: boolean;
  used: number;
  limit: number;
  remaining: number;
  reason?: string;
}

const DAY_MS = 86_400_000;

export function toPlan(p: PlanJSON): Plan {
  return {
    id: p.id,
    slug: p.slug,
    name: p.name,
    description: p.description ?? "",
    status: p.status,
    trialDays: p.trial_days ?? 0,
    isAddon: p.is_addon ?? false,
    price:
      p.amount_minor === undefined || p.amount_minor === null
        ? null
        : { amountMinor: p.amount_minor, currency: p.currency, interval: p.billing_period ?? "" },
    features: (p.features ?? []).map((f) => ({
      key: f.key,
      name: f.name,
      type: f.type,
      limit: f.limit ?? null,
      ...(f.period ? { period: f.period } : {}),
      softLimit: f.soft_limit ?? false,
    })),
    metadata: p.metadata ?? {},
  };
}

export function toSubscription(s: SubscriptionJSON, now = Date.now()): Subscription {
  const trialEnd = s.trial_end ?? null;
  const cancelAt = s.cancel_at ?? null;
  const trialing = s.status === "trialing";
  return {
    id: s.id,
    tenantId: s.tenant_id,
    status: s.status,
    plan: toPlan(s.plan),
    currentPeriodStart: s.current_period_start,
    currentPeriodEnd: s.current_period_end,
    trialStart: s.trial_start ?? null,
    trialEnd,
    cancelAt,
    canceledAt: s.canceled_at ?? null,
    pausedAt: s.paused_at ?? null,
    seats: s.seats
      ? { featureKey: s.seats.feature_key, used: s.seats.used, limit: s.seats.limit ?? null }
      : null,
    isActive: s.status === "active" || trialing || s.status === "past_due",
    isTrialing: trialing,
    isPaused: s.status === "paused",
    willCancel: cancelAt !== null,
    trialDaysLeft:
      trialing && trialEnd
        ? Math.max(0, Math.ceil((Date.parse(trialEnd) - now) / DAY_MS))
        : null,
  };
}

export function toEntitlement(e: EntitlementJSON): Entitlement {
  return {
    allowed: e.allowed,
    used: e.used,
    limit: e.limit,
    remaining: e.remaining,
    ...(e.reason ? { reason: e.reason } : {}),
  };
}

/**
 * Currencies the server's ledger stores with no minor unit. Every other
 * currency it stores in hundredths, whatever the currency's ISO digits are,
 * so this list has to match ledger's, not ISO 4217's.
 */
const ZERO_DECIMAL = new Set(["jpy", "krw", "vnd", "clp", "pyg", "idr"]);

/** Decimal places the server uses for a currency's amounts: 0 or 2. */
export function currencyDecimals(currency: string): number {
  return ZERO_DECIMAL.has(currency.toLowerCase()) ? 0 : 2;
}

/**
 * Formats an amount for display. amountMinor is split the way the server
 * stored it (see currencyDecimals) and shown with that many decimals.
 */
export function formatMoney(money: Money, locale?: string): string {
  const digits = currencyDecimals(money.currency);
  return new Intl.NumberFormat(locale, {
    style: "currency",
    currency: money.currency.toUpperCase(),
    minimumFractionDigits: digits,
    maximumFractionDigits: digits,
  }).format(money.amountMinor / 10 ** digits);
}
