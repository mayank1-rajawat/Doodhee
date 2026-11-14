# Doodhee Simple Backend Architecture

> Implementation update (14 November 2026): Milkmen now register without admin approval. Migration `003_milkman_customer_delivery` adds Milkman/customer relationships, customer-product prices, extra-milk requests and ratings; `orders` remains the per-delivery ledger with actual quantity and immutable price snapshots. See [the implemented plan](MILKMAN_CUSTOMER_DELIVERY_IMPLEMENTATION_PLAN.md) and [current build status](BUILD_STATUS.md). Approval descriptions below are retained only as the original planning baseline.

## 1. First version decision

- One Go backend monorepo using a modular monolith
- One PostgreSQL database
- Three frontends: customer, seller, and admin
- Two REST gateways: public buyer/seller and admin-only
- Internal event-driven flow using an outbox table
- No payments, invoices, payouts, complaints, microservices, Kafka, or gRPC yet

The first complete flow is: seller approval, product creation, daily order, delivery status, non-delivery reason, and total bill view.

## 2. High-level architecture

```text
Customer App ----\
Seller App ------- > Public Gateway ----\
                                           > Shared Business Modules -> PostgreSQL
Admin Web --------> Admin Gateway -------/                  |
                                                              v
                                                        Outbox Events
                                                              |
                                                              v
                                                         Event Worker
```

The gateways and worker are separate binaries built from the same Go monorepo. Business rules remain in shared application modules rather than inside either gateway.

## 3. Go project structure

```text
cmd/public-gateway/main.go
cmd/admin-gateway/main.go
cmd/event-worker/main.go
internal/bootstrap/
internal/platform/
  auth/
  database/
internal/identity/
  domain/ ports/ application/ adapters/
internal/users/
  domain/ ports/ application/ adapters/
internal/products/
  domain/ ports/ application/ adapters/
internal/orders/
  domain/ ports/ application/ adapters/
internal/events/
  outbox/ handlers/
contracts/
api/openapi/
  public-v1.yaml
  admin-v1.yaml
migrations/
```

Each module uses this direction:

```text
HTTP adapter -> application use case -> port interface -> database adapter
```

The complete login, role, gateway, and endpoint design is documented in `DOODHEE_GATEWAY_API_ARCHITECTURE.md`.

## 4. Initial database

Only four tables are required for this version.

### 4.1 users

One table stores customers, sellers, and admins.

| Column | Type | Purpose |
| --- | --- | --- |
| `id` | UUID, primary key | User identity |
| `phone` | VARCHAR, unique | Login by phone and OTP |
| `full_name` | VARCHAR | User name |
| `role` | VARCHAR | `customer`, `seller`, or `admin` |
| `approval_status` | VARCHAR | `pending`, `approved`, or `rejected` |
| `is_enabled` | BOOLEAN | Admin enable or disable control |
| `phone_verified_at` | TIMESTAMPTZ, nullable | Phone verification time |
| `approved_by` | UUID, nullable, foreign key to `users.id` | Admin who approved the user |
| `approved_at` | TIMESTAMPTZ, nullable | Approval time |
| `address_line` | TEXT, nullable | Basic address |
| `city` | VARCHAR, nullable | City |
| `pincode` | VARCHAR, nullable | Delivery area |
| `created_at` | TIMESTAMPTZ | Creation time |
| `updated_at` | TIMESTAMPTZ | Last update time |

Customers can be approved automatically. Sellers start as `pending` and require admin approval.

### 4.2 products

Stores milk products created by sellers.

| Column | Type | Purpose |
| --- | --- | --- |
| `id` | UUID, primary key | Product identity |
| `seller_id` | UUID, foreign key to `users.id` | Product owner |
| `name` | VARCHAR | Cow Milk, Buffalo Milk, etc. |
| `unit` | VARCHAR | Initially `litre` |
| `price` | NUMERIC(12,2) | Current price per unit in INR |
| `daily_capacity` | NUMERIC(10,3) | Maximum daily quantity |
| `is_available` | BOOLEAN | Product availability |
| `created_at` | TIMESTAMPTZ | Creation time |
| `updated_at` | TIMESTAMPTZ | Last update time |

Only an approved and enabled seller can create or update their own products.

### 4.3 orders

Each row represents one customer's milk delivery for one day. It is also the shared daily record used by the customer and seller.

| Column | Type | Purpose |
| --- | --- | --- |
| `id` | UUID, primary key | Order identity |
| `customer_id` | UUID, foreign key to `users.id` | Customer who placed the order |
| `seller_id` | UUID, foreign key to `users.id` | Seller responsible for delivery |
| `product_id` | UUID, foreign key to `products.id` | Ordered milk product |
| `service_date` | DATE | Delivery day |
| `quantity` | NUMERIC(10,3) | Quantity in litres |
| `unit_price` | NUMERIC(12,2) | Price copied when ordered |
| `total_amount` | NUMERIC(12,2) | Quantity multiplied by unit price |
| `status` | VARCHAR | `pending`, `accepted`, `out_for_delivery`, `delivered`, `not_delivered`, or `cancelled` |
| `not_delivered_reason` | VARCHAR, nullable | Standard reason code |
| `delivery_note` | TEXT, nullable | Short explanation for both sides |
| `delivery_address` | TEXT | Address copied when ordered |
| `customer_note` | TEXT, nullable | Customer delivery instruction |
| `marked_by` | UUID, nullable, foreign key to `users.id` | Seller or admin who marked the result |
| `delivery_marked_at` | TIMESTAMPTZ, nullable | Time delivery result was marked |
| `created_at` | TIMESTAMPTZ | Order placement time |
| `updated_at` | TIMESTAMPTZ | Last update time |

Suggested non-delivery reasons:

- `customer_unavailable`
- `seller_unavailable`
- `insufficient_milk`
- `address_issue`
- `route_or_weather_issue`
- `cancelled_before_delivery`
- `other`

Both sides see the same status, reason, note, quantity, price, and amount.

### 4.4 outbox_events

Stores events reliably for background processing.

| Column | Type | Purpose |
| --- | --- | --- |
| `id` | UUID, primary key | Event identity |
| `event_type` | VARCHAR | Example: `OrderDelivered` |
| `aggregate_type` | VARCHAR | Example: `order` |
| `aggregate_id` | UUID | Related record ID |
| `payload` | JSONB | Event data |
| `status` | VARCHAR | `pending`, `published`, or `failed` |
| `attempt_count` | INTEGER | Processing attempts |
| `last_error` | TEXT, nullable | Latest processing error |
| `created_at` | TIMESTAMPTZ | Event creation time |
| `published_at` | TIMESTAMPTZ, nullable | Successful processing time |

The order change and its event must be saved in the same database transaction.

## 5. Daily order and delivery flow

1. Admin approves and enables the seller.
2. Seller creates a milk product with price and capacity.
3. Customer places an order for a specific day.
4. Backend obtains `seller_id` and price from the product instead of trusting the request.
5. Backend saves the order and `OrderPlaced` event together.
6. Seller sees only orders assigned to them.
7. Seller marks the order `delivered` or `not_delivered`.
8. A non-delivered order requires a reason and may include a short note.
9. The customer sees the same daily status immediately.
10. The status change creates `OrderDelivered` or `OrderNotDelivered`.

## 6. Total bill view without a billing table

For now, the bill is a calculated summary, not a stored invoice.

```text
Total bill = sum of total_amount for delivered orders in the selected date range
```

`not_delivered`, `cancelled`, and other incomplete orders do not count toward the bill.

### Customer view

- Date and daily status
- Product and delivered quantity
- Unit price and daily amount
- Non-delivery reason when applicable
- Total delivered days
- Total delivered litres
- Total bill for the selected month or date range

### Seller view

- The same daily details for each customer
- Total delivered litres per customer
- Total bill per customer
- Total delivered litres and sales across all customers

Both views must use the same orders query so the totals cannot disagree.

## 7. Role-based access

| Role | Access |
| --- | --- |
| Customer | View approved products and orders where `customer_id` is their user ID |
| Seller | Manage their products and orders where `seller_id` is their user ID |
| Admin | View all records and approve, reject, enable, or disable users |

Foreign keys connect records but do not control visibility. Every Go use case must filter using the authenticated user ID and role.

Only the assigned seller or an admin can mark an order delivered or not delivered. The backend sets `marked_by`; it must not trust a user ID supplied by the frontend.

## 8. Events for the first version

- `SellerApproved`
- `ProductCreated`
- `OrderPlaced`
- `OrderAccepted`
- `OrderOutForDelivery`
- `OrderDelivered`
- `OrderNotDelivered`
- `OrderCancelled`

These events can trigger notifications and summary refreshes. This is event-driven architecture, not event sourcing: the current state remains in the normal tables.

## 9. Important rules and indexes

- Use `NUMERIC`, never floating-point types, for quantity and money.
- Keep `unit_price` on the order so later product price changes do not change old totals.
- Require a reason when status is `not_delivered`.
- Clear the non-delivery reason if a valid correction changes the status to `delivered`.
- Keep all timestamps in UTC and display them in the user's local timezone.
- Unique index on `users(phone)`.
- Index on `products(seller_id, is_available)`.
- Index on `orders(seller_id, service_date, status)`.
- Index on `orders(customer_id, service_date)`.
- Index on `outbox_events(status, created_at)`.

## 10. Add later

- Subscriptions, pauses, skips, and automatic daily-order generation
- Multiple customer addresses
- Multiple products in one order
- Formal invoices, payments, refunds, and seller payouts
- Complaints and dispute resolution
- Delivery proof and rider tracking
- Quality testing and ratings
- Kafka or separate services only when scale requires them

