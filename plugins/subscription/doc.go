// Package subscription provides a billing/subscription management plugin for authsome.
//
// It delegates all billing logic to the ledger engine (github.com/xraph/ledger) and
// hooks into authsome lifecycle events (org creation, user signup, member changes)
// to automate subscription workflows. The plugin provides:
//
//   - Dynamic settings for configurable tenant mode, trial periods, and auto-subscription
//   - REST API routes for plan, subscription, invoice, and entitlement management
//   - A rich dashboard UI for administrators to manage billing
//   - User and organization detail section contributions
package subscription
