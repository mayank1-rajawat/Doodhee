# Doodhee API verification report

Verified on 14 November 2026. All **74 implemented application endpoints** passed:

- **500 HTTP checks passed**, including a successful case for every endpoint plus authentication, wrong-gateway, role, ownership, validation, state-transition and malformed-input rejections.
- **52 additional business/data assertions passed**.
- The full endpoint suite passed with Go's race detector; the complete Go suite and `go vet` also passed.
- Migration `003_milkman_customer_delivery` passed up/down/up against the isolated `doodhee_migration_test` database.
- Public, admin and Wi-Fi gateway binaries were rebuilt and their readiness checks return `ok`.

## New workflow coverage

- Every enabled Milkman receives immediate product/customer/order access without an admin approval action. The removed admin approval routes are confirmed absent.
- A phone number is globally unique. Exact-phone lookup shows a limited existing-customer match before connection, or reports that a new customer can be created. A Milkman-created, unverified customer can choose sign-in or new account; OTP claims the same user ID and existing Milkman connection.
- Customer search and data are isolated to the owning Milkman relationship.
- Default, customer-specific and delivery-specific pricing precedence is verified. The saved delivery price and total remain unchanged after the customer override is removed.
- Each directly recorded daily delivery creates a separate order/ledger row in its own idempotent transaction.
- Completing an order with actual quantity `1.800` and delivery price `70.00` produces `126.00`; buyer, Milkman and admin summaries agree and exclude undelivered rows.
- A customer can create one rating per Milkman and edit the same row. Ranking uses average rating, rating count and a deterministic name tie-break.
- Extra-milk requests cover ownership, duplicate-open-request prevention, rejection, approval, capacity validation, planned quantity update and final billing from actual delivered quantity.

## Existing regression coverage

- Buyer/Milkman registration and login, admin OTP plus MFA, wrong/expired/replayed OTPs, request/attempt limits, refresh rotation/reuse detection, logout and logout-all.
- Multiple customer addresses, default selection, update/delete and ownership isolation.
- Product create/read/update/availability, admin suspension/activation and ownership checks.
- Order create/cancel/accept/dispatch/deliver/not-deliver, reason requirements, exact decimal arithmetic, capacity and idempotency.
- Admin corrections with immutable before/after audit data, dashboards, event inspection and retry.
- Pagination, concurrent capacity reservation and concurrent idempotent retries.

## Exact endpoint results

The generated endpoint-by-endpoint table is [api-endpoints.md](../artifacts/api-verification/final/api-endpoints.md). It lists successful and expected rejection cases for all 51 public and 23 admin endpoints. The machine-readable result is [api-results.json](../artifacts/api-verification/final/api-results.json).

## Repeat the checks

From `/home/hp/WorkSpace/doodhee`, use an isolated database:

```bash
set -a; source .env; set +a
TEST_DATABASE_URL="${DATABASE_URL%/*}/doodhee_migration_test" make test-api
```

The suite creates and drops a temporary PostgreSQL schema and uses synthetic buyer, Milkman and admin identities. It does not depend on real OTP delivery or account credentials.

## Practical limits

These results cover the implemented endpoints and listed scenarios, not production load. Production SMS/push delivery is not configured. Flutter analysis, widget/storage tests and Android packaging pass, but the v1.1.1 APK has not yet been exercised end-to-end on a physical phone.
