# Milkman customer, delivery, pricing and rating implementation plan

**Status: implemented and verified on 14 November 2026.** Migration `003` is applied, generated contracts are current, and the 74-endpoint race-enabled suite passes 500 HTTP checks plus 52 business/data assertions.

## Existing system findings

- The existing `seller` role is the Milkman role. One public Go gateway serves customers and Milkmen; the admin gateway remains separate.
- `users.phone` already has a database-level global unique constraint, so one mobile number cannot represent multiple roles or users.
- `orders` already represents one dated milk delivery and stores a `unit_price` and `total_amount` snapshot. Delivered-order summaries already power customer, Milkman and admin bills.
- Before this change, seller actions required `approval_status=approved`, new sellers started as `pending`, and Flutter routed them to approval screens.
- Mutating order operations already run inside PostgreSQL transactions, use idempotency keys where retries are dangerous, and write outbox events in the same transaction.

## Design decisions

1. Keep `seller` as the internal role/API prefix. User-facing text may say Milkman.
2. Keep `orders` as the single per-delivery ledger instead of introducing a parallel delivery table. A Milkman-created completed delivery is a delivered order created in one idempotent database transaction.
3. A Milkman–customer connection is many-to-many. It has its own delivery address and active state, so an existing verified customer's global profile is never overwritten by another Milkman.
4. A customer added by a Milkman is created as an enabled, approved, unverified `customer` user when the phone is new. A successful first OTP sign-in or new-account attempt claims it and sets `phone_verified_at`; the pre-existing connection remains attached to that same user ID.
5. Price resolution is `delivery override` → `customer-product price` → `product default`. The resolved price and source are stored on the order. Once delivered, later product/customer price edits never alter it.
6. `orders.quantity` remains the billable actual quantity for delivered rows. `base_quantity` and `approved_extra_quantity` preserve how the planned quantity was formed. Completing a delivery may replace `quantity` with the actual delivered quantity and recalculates only that order total.
7. An extra-milk request belongs to an existing upcoming order. Only pending or accepted future/today orders can receive a request. Approval increases that order's planned quantity after a capacity check; rejection does not change it. Final billing still uses the actual quantity recorded at delivery.
8. One rating row is allowed per customer/Milkman pair. The same endpoint creates the first rating and updates that row afterward. Rankings sort by average rating, rating count, then Milkman name for deterministic ties.
9. Admin approval endpoints and UI actions are removed because they conflict with immediate Milkman access. Admin enable/disable, product suspension, order correction, audit and event tools remain unchanged.

## Database migration

Create migration `003_milkman_customer_delivery`:

- Update existing seller users to `approval_status='approved'` so no existing Milkman remains blocked.
- Extend `orders` with:
  - `base_quantity numeric(10,3)` seeded from current `quantity`.
  - `approved_extra_quantity numeric(10,3) default 0`.
  - `price_source` constrained to `default`, `customer`, or `delivery`.
- Add `milkman_customers` with Milkman/customer foreign keys, relationship delivery address, active flag, timestamps and one row per pair.
- Add `customer_product_prices` with Milkman/customer/product foreign keys, positive price, timestamps and one row per customer/product pair.
- Add `extra_milk_requests` with order/customer/Milkman ownership, positive extra quantity, pending/approved/rejected status, optional decision note and timestamps. Prevent more than one open/approved request for the same order.
- Add `milkman_ratings` with customer/Milkman foreign keys, integer rating 1–5, timestamps and a unique customer/Milkman pair.
- Add ownership/search/reporting indexes and a reversible down migration.

## Backend APIs

### Milkman customers and pricing

- `GET /v1/seller/customers?search=` — only connected customers; search name, phone or relationship address.
- `GET /v1/seller/customers/lookup?phone=` — exact-phone check that returns a limited existing customer match or confirms that a new customer can be created.
- `POST /v1/seller/customers` — name, phone and address; creates or safely links the unique customer record.
- `GET /v1/seller/customers/{id}` — customer plus the Milkman-owned relationship address; the client combines this with the product and price-list endpoints.
- `GET /v1/seller/customers/{id}/prices` — customer-specific prices.
- `PUT /v1/seller/customers/{id}/price` — set/update one product price.
- `DELETE /v1/seller/customers/{id}/price` — remove override and fall back to product default.

### Daily deliveries

- `POST /v1/seller/deliveries` — customer, product, date, actual quantity, optional delivery price/address/note. Creates one delivered order transaction, checks ownership/capacity, snapshots the resolved price and writes an outbox event.
- Existing `POST /v1/seller/orders/{id}/delivery-result` accepts optional actual `quantity` and optional delivery-specific `unit_price`. Existing clients may omit them, in which case the planned quantity and snapshotted price are retained.

### Ratings and rankings

- `PUT /v1/sellers/{id}/rating` — create or edit the authenticated customer's one rating.
- `GET /v1/sellers/{id}/rating` — authenticated customer's rating for that Milkman.
- `GET /v1/sellers/rankings` — ranked enabled Milkmen with average and count.
- Seller discovery/detail records include aggregate rating fields.

### Extra milk

- `POST /v1/orders/{id}/extra-milk-request` — customer requests an additional quantity for their upcoming order.
- `GET /v1/orders/{id}/extra-milk-request` — customer or owning Milkman sees request state.
- `GET /v1/seller/extra-milk-requests` — Milkman's requests with status/date filters.
- `POST /v1/seller/extra-milk-requests/{id}/approve` and `/reject` — atomic decision; approval performs capacity validation and updates planned quantity.

## Frontend changes

### Flutter customer/Milkman app

- Route every enabled seller directly to the seller workspace; remove approval waiting/rejected navigation.
- Add a Milkman Customers section with search, add-customer form and customer detail.
- Customer detail supports product price overrides and a compact “record delivery” form.
- Delivery completion asks for actual quantity and optionally a one-delivery price.
- Buyer order detail supports requesting extra milk and displays its decision status.
- Milkman workspace shows pending extra requests with approve/reject actions.
- Buyer seller cards/details show rating average/count/rank and a 1–5 rating editor.
- Order/bill views display actual delivered quantity, snapshotted unit price, total and price source.

### Next.js admin

- Remove Pending Sellers navigation and approve/reject actions.
- Keep users, enable/disable, products, orders, bill totals, events and audit tools.
- Display delivery pricing source/base/extra details where order detail data is shown.

## Verification

- Migration up/down/up on the isolated migration database.
- OTP claim of a Milkman-created customer and global phone uniqueness across roles.
- Immediate Milkman registration and access without admin actions.
- Customer ownership/search isolation between Milkmen.
- Default, customer and delivery price precedence plus immutable historical totals.
- One transaction/order per daily delivery, idempotent retries and capacity races.
- Actual-quantity billing parity for customer, Milkman and admin.
- Rating create/edit uniqueness and deterministic ranking.
- Extra request ownership, one-open-request rule, approve/reject, capacity and final actual delivery billing.
- Full Go unit/integration/API/race tests, OpenAPI/client drift check, Next.js checks and Flutter analysis/widget/storage tests.
