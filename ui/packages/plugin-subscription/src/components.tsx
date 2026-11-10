/**
 * Drop-in billing components, styled like @authsome/ui-components (whose
 * stylesheet you already load for the auth forms).
 *
 * ```tsx
 * <PricingTable highlight="team" />
 * <CurrentPlanCard />
 * <UsageMeter feature="api_calls" label="API calls this month" />
 * <ManageBillingButton returnUrl={location.href} />
 * ```
 */

import {
  Badge,
  Button,
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
  Skeleton,
  cn,
} from "@authsome/ui-components";
import { useState, type ReactNode } from "react";

import { formatMoney, type Plan, type Subscription } from "./models";
import type { ChangeResult } from "./plugin";
import { useEntitlement, usePlans, useSubscription } from "./react";

const redirect = (url: string) => window.location.assign(url);

function formatDate(iso: string, locale?: string, timeZone?: string): string {
  return new Intl.DateTimeFormat(locale, { dateStyle: "medium", timeZone }).format(new Date(iso));
}

function ErrorLine({ children }: { children: ReactNode }) {
  return (
    <p role="alert" className="text-sm text-destructive">
      {children}
    </p>
  );
}

// ── PricingTable ──────────────────────────────────

export interface PricingTableProps {
  /** Take over selection yourself. Without it the table subscribes or changes plan. */
  onSelect?: (plan: Plan) => void;
  /** Called with the payment page URL when a plan needs paying for. Defaults to redirecting. */
  onCheckout?: (url: string) => void;
  /** Plan slug to draw attention to. */
  highlight?: string;
  /** Where the payment page sends the customer back to. */
  returnUrl?: string;
  /** Where it sends them if they back out. */
  cancelUrl?: string;
  /** Locale for prices. Defaults to the browser's. */
  locale?: string;
  /** Called with the server's error when a change fails; the table shows plain words. */
  onError?: (error: unknown) => void;
  className?: string;
}

/** The plans you can buy, cheapest first, with a button on each. */
export function PricingTable({
  onSelect,
  onCheckout = redirect,
  highlight,
  returnUrl,
  cancelUrl,
  locale,
  onError,
  className,
}: PricingTableProps) {
  const { plans, isLoading, error } = usePlans();
  const { subscription, canManage, subscribe, changePlan, isPending } = useSubscription();
  const [failed, setFailed] = useState(false);

  const choose = async (plan: Plan) => {
    if (onSelect) {
      onSelect(plan);
      return;
    }
    setFailed(false);
    try {
      const options = { returnUrl, cancelUrl };
      const result: ChangeResult = subscription
        ? await changePlan(plan.slug, options)
        : await subscribe(plan.slug, options);
      if (result.type === "checkout") onCheckout(result.url);
    } catch (err) {
      setFailed(true);
      onError?.(err);
    }
  };

  if (isLoading) {
    return (
      <div className={cn("grid gap-4 sm:grid-cols-3", className)}>
        {[0, 1, 2].map((i) => (
          <Skeleton key={i} className="h-48 w-full" />
        ))}
      </div>
    );
  }
  if (error) return <ErrorLine>Couldn't load plans.</ErrorLine>;

  const showButtons = canManage || Boolean(onSelect);
  return (
    <div className={cn("space-y-3", className)}>
      <div className="grid gap-4 sm:grid-cols-[repeat(auto-fit,minmax(14rem,1fr))]">
        {plans.map((plan) => {
          const current = subscription?.plan.id === plan.id;
          return (
            <Card key={plan.id} className={cn(plan.slug === highlight && "ring-2 ring-primary")}>
              <CardHeader>
                <div className="flex items-center justify-between gap-2">
                  <CardTitle>{plan.name}</CardTitle>
                  {plan.slug === highlight && <Badge>Recommended</Badge>}
                </div>
                {plan.description && <CardDescription>{plan.description}</CardDescription>}
              </CardHeader>
              <CardContent className="space-y-3">
                <p className="flex items-baseline gap-1">
                  <span className="text-2xl font-semibold">
                    {plan.price ? formatMoney(plan.price, locale) : "No charge"}
                  </span>
                  {plan.price?.interval && (
                    <span className="text-sm text-muted-foreground">/ {plan.price.interval}</span>
                  )}
                </p>
                {plan.trialDays > 0 && !subscription && (
                  <Badge variant="secondary">{plan.trialDays}-day trial</Badge>
                )}
                {plan.features.length > 0 && (
                  <ul className="space-y-1 text-sm">
                    {plan.features.map((f) => (
                      <li key={f.key}>
                        {f.name}
                        {f.type !== "boolean" && (
                          <span className="text-muted-foreground">
                            {" "}
                            {f.limit === null ? "unlimited" : `up to ${f.limit}`}
                          </span>
                        )}
                      </li>
                    ))}
                  </ul>
                )}
              </CardContent>
              {showButtons && (
                <CardFooter>
                  <Button
                    className="w-full"
                    variant={current ? "outline" : "default"}
                    disabled={current || isPending}
                    onClick={() => void choose(plan)}
                  >
                    {current ? "Current plan" : `Choose ${plan.name}`}
                  </Button>
                </CardFooter>
              )}
            </Card>
          );
        })}
      </div>
      {failed && <ErrorLine>Couldn't change your plan. Try again.</ErrorLine>}
    </div>
  );
}

// ── CurrentPlanCard ───────────────────────────────

export interface CurrentPlanCardProps {
  /** Locale for prices and dates. Defaults to the browser's. */
  locale?: string;
  /** Time zone for dates. Defaults to the browser's. */
  timeZone?: string;
  /** Called with the server's error when cancel or keep fails; the card shows plain words. */
  onError?: (error: unknown) => void;
  className?: string;
}

function statusLine(sub: Subscription, locale?: string, timeZone?: string): string {
  const date = (iso: string) => formatDate(iso, locale, timeZone);
  if (sub.isPaused) return "Paused";
  if (sub.willCancel && sub.cancelAt) return `Ends on ${date(sub.cancelAt)}`;
  if (sub.isTrialing && sub.trialEnd) return `Trial ends on ${date(sub.trialEnd)}`;
  if (sub.status === "past_due") return "Payment is overdue";
  return `Renews on ${date(sub.currentPeriodEnd)}`;
}

/** The tenant's plan: name, price, when it renews or ends, seats, and cancel/keep. */
export function CurrentPlanCard({ locale, timeZone, onError, className }: CurrentPlanCardProps) {
  const { subscription: sub, canManage, isLoading, error, cancel, resume, isPending } =
    useSubscription();
  const [failed, setFailed] = useState<string | null>(null);

  const run = async (write: () => Promise<unknown>) => {
    setFailed(null);
    try {
      await write();
    } catch (err) {
      setFailed("Couldn't update your plan. Try again.");
      onError?.(err);
    }
  };

  if (isLoading) return <Skeleton className={cn("h-40 w-full", className)} />;
  if (error) return <ErrorLine>Couldn't load your plan.</ErrorLine>;

  if (!sub) {
    return (
      <Card className={className}>
        <CardHeader>
          <CardTitle>No plan yet</CardTitle>
          <CardDescription>Pick a plan to get started.</CardDescription>
        </CardHeader>
      </Card>
    );
  }

  return (
    <Card className={className}>
      <CardHeader>
        <div className="flex items-center justify-between gap-2">
          <CardTitle>{sub.plan.name}</CardTitle>
          {sub.isTrialing && <Badge variant="secondary">Trial</Badge>}
          {sub.status === "past_due" && <Badge variant="destructive">Past due</Badge>}
        </div>
        {sub.plan.price && (
          <CardDescription>
            {`${formatMoney(sub.plan.price, locale)} / ${sub.plan.price.interval}`}
          </CardDescription>
        )}
      </CardHeader>
      <CardContent className="space-y-1 text-sm">
        <p>{statusLine(sub, locale, timeZone)}</p>
        {sub.seats && (
          <p className="text-muted-foreground">
            {sub.seats.limit === null
              ? `${sub.seats.used} seats`
              : `${sub.seats.used} of ${sub.seats.limit} seats`}
          </p>
        )}
        {failed && <ErrorLine>{failed}</ErrorLine>}
      </CardContent>
      {canManage && !sub.isPaused && (
        <CardFooter>
          {sub.willCancel ? (
            <Button disabled={isPending} onClick={() => void run(resume)}>
              Keep plan
            </Button>
          ) : (
            <Button variant="outline" disabled={isPending} onClick={() => void run(() => cancel())}>
              Cancel plan
            </Button>
          )}
        </CardFooter>
      )}
    </Card>
  );
}

// ── UsageMeter ────────────────────────────────────

export interface UsageMeterProps {
  /** The metered plan feature, e.g. "api_calls". */
  feature: string;
  /** What to call it. Defaults to the feature key. */
  label?: string;
  className?: string;
}

/** How much of a metered feature is used, against the plan's limit. */
export function UsageMeter({ feature, label, className }: UsageMeterProps) {
  const { entitlement, isLoading } = useEntitlement(feature);
  if (isLoading) return <Skeleton className={cn("h-10 w-full", className)} />;
  if (!entitlement) return null;

  const unlimited = entitlement.limit < 0;
  const percent = unlimited || entitlement.limit === 0
    ? 0
    : Math.min(100, Math.round((entitlement.used / entitlement.limit) * 100));
  return (
    <div className={cn("space-y-1", className)}>
      <div className="flex justify-between text-sm">
        <span>{label ?? feature}</span>
        <span className="text-muted-foreground">
          {unlimited ? `${entitlement.used} used` : `${entitlement.used} of ${entitlement.limit}`}
        </span>
      </div>
      <div
        role="progressbar"
        aria-label={label ?? feature}
        aria-valuemin={0}
        aria-valuemax={unlimited ? undefined : entitlement.limit}
        aria-valuenow={unlimited ? undefined : entitlement.used}
        aria-valuetext={unlimited ? `${entitlement.used} used, no limit` : `${entitlement.used} of ${entitlement.limit}`}
        className="h-2 w-full overflow-hidden rounded-full bg-secondary"
      >
        <div
          className={cn("h-full bg-primary", percent >= 90 && "bg-destructive")}
          style={{ width: `${percent}%` }}
        />
      </div>
    </div>
  );
}

// ── ManageBillingButton ───────────────────────────

export interface ManageBillingButtonProps {
  /** Where the portal sends the customer back to. */
  returnUrl?: string;
  /** Called with the portal URL. Defaults to redirecting. */
  onOpen?: (url: string) => void;
  children?: ReactNode;
  className?: string;
}

/**
 * Opens the payment provider's billing portal. Shown only to someone who can
 * manage billing; a server without a portal answers with an error message.
 */
export function ManageBillingButton({
  returnUrl,
  onOpen = redirect,
  children = "Manage billing",
  className,
}: ManageBillingButtonProps) {
  const { canManage, openPortal, isPending } = useSubscription();
  const [failed, setFailed] = useState(false);
  if (!canManage) return null;

  const open = async () => {
    setFailed(false);
    try {
      onOpen(await openPortal(returnUrl));
    } catch {
      setFailed(true);
    }
  };

  return (
    <div className="space-y-1">
      <Button variant="outline" className={className} disabled={isPending} onClick={() => void open()}>
        {children}
      </Button>
      {failed && <ErrorLine>Couldn't open the billing portal.</ErrorLine>}
    </div>
  );
}
