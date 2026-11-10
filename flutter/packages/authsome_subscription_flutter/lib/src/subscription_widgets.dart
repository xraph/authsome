import 'dart:async';

import 'package:authsome_flutter/authsome_flutter.dart';
import 'package:authsome_subscription/authsome_subscription.dart';
import 'package:flutter/material.dart';

/// Reaches the subscription plugin from the widget tree.
extension SubscriptionContext on BuildContext {
  /// The subscription plugin's API. The plugin has to be in the
  /// [AuthConfig.plugins] of the surrounding [AuthProvider].
  SubscriptionApi get subscription => AuthProvider.of(this).plugin<SubscriptionApi>();
}

/// Builds from the subscription plugin's snapshot and rebuilds when it
/// changes. Loads the subscription (and, with [loadPlans], the plans) the
/// first time it's built while signed in.
class SubscriptionBuilder extends StatefulWidget {
  const SubscriptionBuilder({required this.builder, this.loadPlans = false, super.key});

  final Widget Function(BuildContext context, SubscriptionSnapshot snapshot) builder;
  final bool loadPlans;

  @override
  State<SubscriptionBuilder> createState() => _SubscriptionBuilderState();
}

class _SubscriptionBuilderState extends State<SubscriptionBuilder> {
  SubscriptionApi? _api;

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    final auth = AuthProvider.of(context);
    final api = auth.plugin<SubscriptionApi>();
    _api = api;
    // AuthProvider is inherited, so this runs again when the user signs in.
    if (!auth.isAuthenticated) return;
    if (api.snapshot.subscription.status == ResourceStatus.idle) unawaited(api.load());
    if (widget.loadPlans && api.snapshot.plans.status == ResourceStatus.idle) {
      unawaited(api.loadPlans());
    }
  }

  @override
  Widget build(BuildContext context) {
    final api = _api!;
    return StreamBuilder<SubscriptionSnapshot>(
      stream: api.changes,
      initialData: api.snapshot,
      builder: (context, _) => widget.builder(context, api.snapshot),
    );
  }
}

// Idle counts as loading only while signed in: SubscriptionBuilder is
// about to start the load. Signed out, nothing will, so idle means empty.
bool _isLoading(BuildContext context, Resource<Object?> r) =>
    r.status == ResourceStatus.loading ||
    (r.status == ResourceStatus.idle && AuthProvider.of(context).isAuthenticated);

/// Shows [child] only when the plan includes [feature]. Loading, an error,
/// or billing not being installed all show [fallback] (or [loading] while
/// the check loads), so a paywall never opens by accident.
class EntitlementGate extends StatelessWidget {
  const EntitlementGate({
    required this.feature,
    required this.child,
    this.fallback = const SizedBox.shrink(),
    this.loading,
    super.key,
  });

  final String feature;
  final Widget child;
  final Widget fallback;
  final Widget? loading;

  @override
  Widget build(BuildContext context) {
    return SubscriptionBuilder(builder: (context, snap) {
      final entitlements = snap.entitlements;
      if (entitlements.status == ResourceStatus.ready &&
          entitlements.data[feature]?.allowed == true) {
        return child;
      }
      if (_isLoading(context, entitlements)) return loading ?? fallback;
      return fallback;
    });
  }
}

String _date(BuildContext context, DateTime date) =>
    MaterialLocalizations.of(context).formatShortDate(date.toLocal());

// Server messages are written for developers, so people see plain words
// and the error itself goes to the widget's onError.
void _showError(BuildContext context, String message) {
  ScaffoldMessenger.maybeOf(context)?.showSnackBar(SnackBar(content: Text(message)));
}

/// The plans the tenant can buy, cheapest first, with a button on each.
///
/// Without [onSelected] a tap subscribes or changes plan. When the plan
/// needs paying for, [onCheckout] gets the payment page's URL; open it with
/// your launcher of choice, e.g. `launchUrl(uri)` from url_launcher.
class PricingTable extends StatefulWidget {
  const PricingTable({
    this.onSelected,
    this.onCheckout,
    this.highlight,
    this.returnUrl,
    this.cancelUrl,
    this.onError,
    super.key,
  });

  final ValueChanged<Plan>? onSelected;

  /// Gets the payment page URL when a plan needs paying for. Without it the
  /// table can't take payment: it tells the user and passes a [StateError]
  /// to [onError].
  final ValueChanged<Uri>? onCheckout;

  /// Called with the error when a change fails; the table shows plain words.
  final ValueChanged<Object>? onError;

  /// Plan slug to draw attention to.
  final String? highlight;
  final String? returnUrl;
  final String? cancelUrl;

  @override
  State<PricingTable> createState() => _PricingTableState();
}

class _PricingTableState extends State<PricingTable> {
  bool _busy = false;

  Future<void> _choose(SubscriptionApi api, Plan plan, bool hasSubscription) async {
    final onSelected = widget.onSelected;
    if (onSelected != null) {
      onSelected(plan);
      return;
    }
    setState(() => _busy = true);
    try {
      final result = hasSubscription
          ? await api.changePlan(plan.slug, returnUrl: widget.returnUrl, cancelUrl: widget.cancelUrl)
          : await api.subscribe(plan.slug, returnUrl: widget.returnUrl, cancelUrl: widget.cancelUrl);
      if (result is CheckoutRequired && mounted) {
        final onCheckout = widget.onCheckout;
        if (onCheckout != null) {
          onCheckout(result.url);
        } else {
          debugPrint('authsome: PricingTable got a payment page but has no onCheckout to open it');
          _showError(context, "Couldn't open the payment page.");
          widget.onError?.call(StateError('Set PricingTable.onCheckout to open the payment page'));
        }
      }
    } catch (error) {
      if (mounted) _showError(context, "Couldn't change your plan. Try again.");
      widget.onError?.call(error);
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return SubscriptionBuilder(
      loadPlans: true,
      builder: (context, snap) {
        if (_isLoading(context, snap.plans)) {
          return const Center(child: Padding(padding: EdgeInsets.all(24), child: CircularProgressIndicator()));
        }
        if (snap.plans.status == ResourceStatus.error) {
          return const Text("Couldn't load plans.");
        }
        final current = snap.subscription.data;
        final showButtons = snap.canManage || widget.onSelected != null;
        return Wrap(
          spacing: 16,
          runSpacing: 16,
          children: [
            for (final plan in snap.plans.data)
              SizedBox(
                width: 240,
                child: Card(
                  shape: plan.slug == widget.highlight
                      ? RoundedRectangleBorder(
                          side: BorderSide(color: theme.colorScheme.primary, width: 2),
                          borderRadius: BorderRadius.circular(12),
                        )
                      : null,
                  child: Padding(
                    padding: const EdgeInsets.all(16),
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Row(children: [
                          Expanded(child: Text(plan.name, style: theme.textTheme.titleMedium)),
                          if (plan.slug == widget.highlight)
                            Chip(
                              label: const Text('Recommended'),
                              visualDensity: VisualDensity.compact,
                              backgroundColor: theme.colorScheme.primaryContainer,
                            ),
                        ]),
                        const SizedBox(height: 8),
                        Text(plan.price?.format() ?? 'No charge', style: theme.textTheme.headlineSmall),
                        if (plan.price != null && plan.price!.interval.isNotEmpty)
                          Text('/ ${plan.price!.interval}', style: theme.textTheme.bodySmall),
                        if (plan.trialDays > 0 && current == null) ...[
                          const SizedBox(height: 8),
                          Chip(label: Text('${plan.trialDays}-day trial')),
                        ],
                        const SizedBox(height: 8),
                        for (final f in plan.features)
                          Text(
                            f.type == 'boolean'
                                ? f.name
                                : '${f.name} ${f.limit == null ? 'unlimited' : 'up to ${f.limit}'}',
                            style: theme.textTheme.bodyMedium,
                          ),
                        if (showButtons) ...[
                          const SizedBox(height: 12),
                          SizedBox(
                            width: double.infinity,
                            child: current?.plan.id == plan.id
                                ? const OutlinedButton(onPressed: null, child: Text('Current plan'))
                                : FilledButton(
                                    onPressed: _busy
                                        ? null
                                        : () => _choose(context.subscription, plan, current != null),
                                    child: Text('Choose ${plan.name}'),
                                  ),
                          ),
                        ],
                      ],
                    ),
                  ),
                ),
              ),
          ],
        );
      },
    );
  }
}

/// The tenant's plan: name, price, when it renews or ends, seats, and
/// cancel or keep for someone who can manage billing.
class CurrentPlanCard extends StatefulWidget {
  const CurrentPlanCard({this.onError, super.key});

  /// Called with the error when cancel or keep fails; the card shows plain words.
  final ValueChanged<Object>? onError;

  @override
  State<CurrentPlanCard> createState() => _CurrentPlanCardState();
}

class _CurrentPlanCardState extends State<CurrentPlanCard> {
  bool _busy = false;

  Future<void> _run(Future<Object?> Function() write) async {
    setState(() => _busy = true);
    try {
      await write();
    } catch (error) {
      if (mounted) _showError(context, "Couldn't update your plan. Try again.");
      widget.onError?.call(error);
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  String _status(BuildContext context, Subscription sub) {
    if (sub.isPaused) return 'Paused';
    if (sub.willCancel) return 'Ends on ${_date(context, sub.cancelAt!)}';
    if (sub.isTrialing && sub.trialEnd != null) return 'Trial ends on ${_date(context, sub.trialEnd!)}';
    if (sub.status == SubscriptionStatus.pastDue) return 'Payment is overdue';
    return 'Renews on ${_date(context, sub.currentPeriodEnd)}';
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return SubscriptionBuilder(builder: (context, snap) {
      if (_isLoading(context, snap.subscription)) {
        return const Card(child: Padding(padding: EdgeInsets.all(24), child: LinearProgressIndicator()));
      }
      if (snap.subscription.status == ResourceStatus.error) {
        return const Text("Couldn't load your plan.");
      }
      final sub = snap.subscription.data;
      if (sub == null) {
        return const Card(
          child: ListTile(
            title: Text('No plan yet'),
            subtitle: Text('Pick a plan to get started.'),
          ),
        );
      }
      final api = context.subscription;
      final seats = sub.seats;
      return Card(
        child: Padding(
          padding: const EdgeInsets.all(16),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Row(children: [
                Expanded(child: Text(sub.plan.name, style: theme.textTheme.titleLarge)),
                if (sub.isTrialing) const Chip(label: Text('Trial')),
                if (sub.status == SubscriptionStatus.pastDue)
                  Chip(
                    label: const Text('Past due'),
                    backgroundColor: theme.colorScheme.errorContainer,
                  ),
              ]),
              if (sub.plan.price != null)
                Text('${sub.plan.price!.format()} / ${sub.plan.price!.interval}',
                    style: theme.textTheme.bodyMedium),
              const SizedBox(height: 8),
              Text(_status(context, sub)),
              if (seats != null)
                Text(
                  seats.limit == null ? '${seats.used} seats' : '${seats.used} of ${seats.limit} seats',
                  style: theme.textTheme.bodySmall,
                ),
              if (snap.canManage && !sub.isPaused) ...[
                const SizedBox(height: 12),
                sub.willCancel
                    ? FilledButton(
                        onPressed: _busy ? null : () => _run(api.resume),
                        child: const Text('Keep plan'),
                      )
                    : OutlinedButton(
                        onPressed: _busy ? null : () => _run(api.cancel),
                        child: const Text('Cancel plan'),
                      ),
              ],
            ],
          ),
        ),
      );
    });
  }
}

/// How much of a metered feature is used, against the plan's limit.
class UsageMeter extends StatelessWidget {
  const UsageMeter({required this.feature, this.label, super.key});

  final String feature;

  /// What to call it. Defaults to the feature key.
  final String? label;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return SubscriptionBuilder(builder: (context, snap) {
      if (_isLoading(context, snap.entitlements)) return const LinearProgressIndicator();
      final e = snap.entitlements.data[feature];
      if (e == null) return const SizedBox.shrink();
      final unlimited = e.limit < 0;
      final value = unlimited || e.limit == 0 ? 0.0 : (e.used / e.limit).clamp(0.0, 1.0);
      return Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(children: [
            Expanded(child: Text(label ?? feature)),
            Text(unlimited ? '${e.used} used' : '${e.used} of ${e.limit}', style: theme.textTheme.bodySmall),
          ]),
          const SizedBox(height: 4),
          LinearProgressIndicator(
            value: value,
            color: value >= 0.9 ? theme.colorScheme.error : null,
            semanticsLabel: label ?? feature,
          ),
        ],
      );
    });
  }
}

/// Opens the payment provider's billing portal. Shown only to someone who
/// can manage billing. [onOpen] gets the portal URL; open it with your
/// launcher of choice.
class ManageBillingButton extends StatefulWidget {
  const ManageBillingButton({required this.onOpen, this.returnUrl, this.child, super.key});

  final ValueChanged<Uri> onOpen;
  final String? returnUrl;
  final Widget? child;

  @override
  State<ManageBillingButton> createState() => _ManageBillingButtonState();
}

class _ManageBillingButtonState extends State<ManageBillingButton> {
  bool _busy = false;

  Future<void> _open(SubscriptionApi api) async {
    setState(() => _busy = true);
    try {
      widget.onOpen(await api.openPortal(returnUrl: widget.returnUrl));
    } catch (_) {
      if (mounted) {
        ScaffoldMessenger.maybeOf(context)?.showSnackBar(
          const SnackBar(content: Text("Couldn't open the billing portal")),
        );
      }
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    return SubscriptionBuilder(builder: (context, snap) {
      if (!snap.canManage) return const SizedBox.shrink();
      return OutlinedButton(
        onPressed: _busy ? null : () => _open(context.subscription),
        child: widget.child ?? const Text('Manage billing'),
      );
    });
  }
}
