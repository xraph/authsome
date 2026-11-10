/// Typed models for the subscription plugin, mapped from the server's JSON.
library;

/// A subscription's state. A value from a newer server reads as [unknown].
enum SubscriptionStatus {
  active,
  trialing,
  pastDue,
  canceled,
  expired,
  paused,
  unknown;

  /// Parses the server's snake_case status.
  static SubscriptionStatus parse(String value) => switch (value) {
        'active' => active,
        'trialing' => trialing,
        'past_due' => pastDue,
        'canceled' => canceled,
        'expired' => expired,
        'paused' => paused,
        _ => unknown,
      };
}

/// Currencies with no minor unit. Everything else is treated as two places.
const _zeroDecimal = {'bif', 'clp', 'djf', 'gnf', 'jpy', 'kmf', 'krw', 'mga', 'pyg', 'rwf', 'ugx', 'vnd', 'vuv', 'xaf', 'xof', 'xpf'};

const _symbols = {'usd': r'$', 'eur': '€', 'gbp': '£', 'jpy': '¥', 'inr': '₹', 'cad': r'CA$', 'aud': r'A$'};

/// An amount in the currency's minor unit (cents for usd).
class Money {
  const Money({required this.amountMinor, required this.currency});

  final int amountMinor;
  final String currency;

  /// Decimal places the currency uses.
  int get decimals => _zeroDecimal.contains(currency.toLowerCase()) ? 0 : 2;

  /// The amount in major units, e.g. 29.0 for 2900 usd cents.
  double get major => amountMinor / _pow10(decimals);

  /// A plain rendering such as `$29.00` or `NGN 10.50`. Use your own
  /// NumberFormat for locale-aware output; [major] and [currency] are there
  /// for that.
  String format() {
    final amount = major.toStringAsFixed(decimals);
    final symbol = _symbols[currency.toLowerCase()];
    return symbol != null ? '$symbol$amount' : '${currency.toUpperCase()} $amount';
  }

  static int _pow10(int n) => n == 0 ? 1 : 100;
}

/// A plan's base price and how often it's charged.
class Price extends Money {
  const Price({required super.amountMinor, required super.currency, required this.interval});

  /// "monthly", "yearly", ... as the server names it.
  final String interval;
}

/// One feature a plan includes. A null [limit] means unlimited.
class PlanFeature {
  const PlanFeature({
    required this.key,
    required this.name,
    required this.type,
    required this.limit,
    required this.softLimit,
    this.period,
  });

  final String key;
  final String name;

  /// "boolean", "metered" or "seat".
  final String type;
  final int? limit;
  final String? period;
  final bool softLimit;

  factory PlanFeature.fromJson(Map<String, dynamic> json) => PlanFeature(
        key: json['key'] as String,
        name: json['name'] as String,
        type: json['type'] as String,
        limit: (json['limit'] as num?)?.toInt(),
        period: json['period'] as String?,
        softLimit: json['soft_limit'] as bool? ?? false,
      );
}

class Plan {
  const Plan({
    required this.id,
    required this.slug,
    required this.name,
    required this.description,
    required this.status,
    required this.trialDays,
    required this.isAddon,
    required this.price,
    required this.features,
    required this.metadata,
  });

  final String id;
  final String slug;
  final String name;
  final String description;
  final String status;
  final int trialDays;
  final bool isAddon;

  /// Null for a plan without a price.
  final Price? price;
  final List<PlanFeature> features;
  final Map<String, String> metadata;

  factory Plan.fromJson(Map<String, dynamic> json) {
    final amount = json['amount_minor'] as num?;
    return Plan(
      id: json['id'] as String,
      slug: json['slug'] as String,
      name: json['name'] as String,
      description: json['description'] as String? ?? '',
      status: json['status'] as String,
      trialDays: (json['trial_days'] as num?)?.toInt() ?? 0,
      isAddon: json['is_addon'] as bool? ?? false,
      price: amount == null
          ? null
          : Price(
              amountMinor: amount.toInt(),
              currency: json['currency'] as String,
              interval: json['billing_period'] as String? ?? '',
            ),
      features: [
        for (final f in (json['features'] as List? ?? const []))
          PlanFeature.fromJson(Map<String, dynamic>.from(f as Map)),
      ],
      metadata: Map<String, String>.from(json['metadata'] as Map? ?? const {}),
    );
  }
}

/// Seat use on a plan with a seat feature. A null [limit] means unlimited.
class Seats {
  const Seats({required this.featureKey, required this.used, required this.limit});

  final String featureKey;
  final int used;
  final int? limit;
}

class Subscription {
  const Subscription({
    required this.id,
    required this.tenantId,
    required this.status,
    required this.plan,
    required this.currentPeriodStart,
    required this.currentPeriodEnd,
    this.trialStart,
    this.trialEnd,
    this.cancelAt,
    this.canceledAt,
    this.pausedAt,
    this.seats,
  });

  final String id;
  final String tenantId;
  final SubscriptionStatus status;
  final Plan plan;
  final DateTime currentPeriodStart;
  final DateTime currentPeriodEnd;
  final DateTime? trialStart;
  final DateTime? trialEnd;

  /// When a cancellation scheduled for the period end takes effect.
  final DateTime? cancelAt;
  final DateTime? canceledAt;
  final DateTime? pausedAt;
  final Seats? seats;

  /// The tenant has the plan's features: active, trialing, or past due.
  bool get isActive =>
      status == SubscriptionStatus.active ||
      status == SubscriptionStatus.trialing ||
      status == SubscriptionStatus.pastDue;
  bool get isTrialing => status == SubscriptionStatus.trialing;
  bool get isPaused => status == SubscriptionStatus.paused;

  /// A cancellation is scheduled; the plan runs until [cancelAt].
  bool get willCancel => cancelAt != null;

  /// Whole days of trial left, null when not trialing.
  int? get trialDaysLeft {
    final end = trialEnd;
    if (!isTrialing || end == null) return null;
    final left = end.difference(DateTime.now()).inMilliseconds / Duration.millisecondsPerDay;
    return left <= 0 ? 0 : left.ceil();
  }

  factory Subscription.fromJson(Map<String, dynamic> json) {
    DateTime? date(String key) {
      final v = json[key] as String?;
      return v == null ? null : DateTime.parse(v);
    }

    final seats = json['seats'] as Map?;
    return Subscription(
      id: json['id'] as String,
      tenantId: json['tenant_id'] as String,
      status: SubscriptionStatus.parse(json['status'] as String),
      plan: Plan.fromJson(Map<String, dynamic>.from(json['plan'] as Map)),
      currentPeriodStart: DateTime.parse(json['current_period_start'] as String),
      currentPeriodEnd: DateTime.parse(json['current_period_end'] as String),
      trialStart: date('trial_start'),
      trialEnd: date('trial_end'),
      cancelAt: date('cancel_at'),
      canceledAt: date('canceled_at'),
      pausedAt: date('paused_at'),
      seats: seats == null
          ? null
          : Seats(
              featureKey: seats['feature_key'] as String,
              used: (seats['used'] as num).toInt(),
              limit: (seats['limit'] as num?)?.toInt(),
            ),
    );
  }
}

/// Whether the tenant may use a feature, and how much of it is left.
class Entitlement {
  const Entitlement({
    required this.allowed,
    required this.used,
    required this.limit,
    required this.remaining,
    this.reason,
  });

  final bool allowed;
  final int used;

  /// Negative means unlimited, as ledger reports it.
  final int limit;
  final int remaining;
  final String? reason;

  factory Entitlement.fromJson(Map<String, dynamic> json) => Entitlement(
        allowed: json['allowed'] as bool,
        used: (json['used'] as num).toInt(),
        limit: (json['limit'] as num).toInt(),
        remaining: (json['remaining'] as num).toInt(),
        reason: json['reason'] as String?,
      );
}
