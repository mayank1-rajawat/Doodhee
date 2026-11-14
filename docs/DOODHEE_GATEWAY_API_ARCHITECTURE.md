# Doodhee Gateway Authentication and API Architecture

> Implementation update (14 November 2026): Milkman registration no longer needs admin approval. Current public routes add customer management, per-customer prices, direct deliveries, actual delivery overrides, ratings/rankings and extra-milk requests; admin approve/reject routes were removed. See [the implemented plan](MILKMAN_CUSTOMER_DELIVERY_IMPLEMENTATION_PLAN.md) and the generated OpenAPI contracts. Approval descriptions below are retained only as the original planning baseline.

## 1. Purpose and decision

Doodhee uses one Go monorepo, one PostgreSQL database, shared business modules, and two separate HTTP gateways:

- **Public Gateway:** buyer and seller applications only
- **Admin Gateway:** admin web application only

The design follows the referenced Go architecture: modular monorepo, hexagonal architecture, inward dependencies, constructor-based dependency injection, shared contracts, REST at the edge, and event-driven processing through an outbox.

The gateways are separate binaries, but they are not separate business systems. Both call the same application use cases and share the same domain rules.

## 2. High-level architecture

```text
Buyer App --------\
                   > Public Gateway :8080 ----\
Seller App -------/                             \
                                                  > Shared Application Modules -> PostgreSQL
Admin Web --------> Admin Gateway :8081 --------/                 |
                                                                    v
                                                              Outbox Events
                                                                    |
                                                                    v
                                                               Event Worker
```

### Gateway boundaries

| Gateway | Host example | Allowed users | Main responsibility |
| --- | --- | --- | --- |
| Public Gateway | `api.doodhee.in` | Buyer and seller | Login, discovery, products, daily orders, delivery status, bill summary |
| Admin Gateway | `admin-api.doodhee.in` | Admin only | Admin login, seller approval, account control, order correction, operations and audit |

Admin routes are registered only in the admin binary. An admin token is not accepted by the public gateway, and public tokens are not accepted by the admin gateway.

## 3. Go monorepo structure

```text
doodhee-backend/
├── go.mod
├── cmd/
│   ├── public-gateway/main.go
│   ├── admin-gateway/main.go
│   └── event-worker/main.go
├── internal/
│   ├── bootstrap/
│   │   ├── public.go
│   │   ├── admin.go
│   │   └── worker.go
│   ├── platform/
│   │   ├── auth/
│   │   ├── database/
│   │   ├── httpserver/
│   │   ├── observability/
│   │   └── validation/
│   ├── identity/
│   │   ├── domain/
│   │   ├── ports/
│   │   ├── application/
│   │   └── adapters/
│   ├── users/
│   │   ├── domain/
│   │   ├── ports/
│   │   ├── application/
│   │   └── adapters/
│   ├── products/
│   │   ├── domain/
│   │   ├── ports/
│   │   ├── application/
│   │   └── adapters/
│   ├── orders/
│   │   ├── domain/
│   │   ├── ports/
│   │   ├── application/
│   │   └── adapters/
│   ├── reporting/
│   │   ├── application/
│   │   └── adapters/
│   ├── audit/
│   └── events/
│       ├── outbox/
│       └── handlers/
├── contracts/
│   ├── api/
│   ├── errors/
│   └── events/
├── api/openapi/
│   ├── public-v1.yaml
│   └── admin-v1.yaml
├── migrations/
└── deploy/
```

`reporting` reads delivered orders and calculates bill summaries. It does not create invoices or payment records.

## 4. Hexagonal architecture rule

Each module has four layers:

| Layer | Responsibility | May depend on |
| --- | --- | --- |
| Domain | Entities, value objects, state rules and validation | Nothing external |
| Ports | Repository, token, OTP, clock and event interfaces | Domain |
| Application | Use cases and transaction orchestration | Domain and ports |
| Adapters | HTTP handlers, PostgreSQL repositories, SMS provider and outbox worker | All inner layers |

The dependency direction is always:

```text
Adapters -> Application -> Ports -> Domain
```

Gateways are inbound adapters. They parse HTTP, authenticate the request, validate basic input, call a use case, and serialize the result. They must not contain business rules or execute SQL directly.

`bootstrap` creates concrete adapters and injects them into application services with normal Go constructors. No dependency-injection framework is required.

## 5. Authentication plane

### 5.1 User roles

The first version uses one role per user:

- `customer` for the buyer
- `seller`
- `admin`

A caller can request registration only as `customer` or `seller`. Admin accounts are created by a trusted migration or an existing authorized admin; public self-registration can never create an admin.

### 5.2 Account states

| Role | Initial approval state | Allowed after login |
| --- | --- | --- |
| Customer | `approved` | Buyer APIs |
| Seller | `pending` | Own profile and approval status only |
| Approved seller | `approved` | Seller APIs |
| Rejected seller | `rejected` | Own profile and rejection status only |
| Disabled user | Any | No protected operations |
| Admin | `approved` | Admin APIs after MFA |

Approval and enablement are different. An approved seller may later be disabled without losing the approval record.

### 5.3 Public login flow

1. Buyer or seller requests an OTP using a phone number and intended role.
2. The identity module applies rate limits and stores only an OTP hash with expiry.
3. The user submits the challenge ID and OTP.
4. A new customer is created as approved; a new seller is created as pending.
5. An existing user's stored role is used. The request cannot change it.
6. The gateway creates a session and returns public access and refresh tokens.
7. Pending sellers receive a valid session but can access only profile and approval-status endpoints.

### 5.4 Admin login flow

1. Admin requests an OTP through the admin gateway.
2. OTP verification confirms the account is enabled and has role `admin`.
3. The gateway returns a short-lived MFA challenge, not a normal access token.
4. Admin verifies TOTP or another configured second factor.
5. The identity module creates an admin-audience session.
6. Every admin mutation is written to the admin audit log.

MFA is mandatory for admin access. Admin self-registration is not available.

### 5.5 Token design

Access tokens are signed JWTs. Refresh tokens are random opaque values stored only as hashes.

| Claim | Meaning |
| --- | --- |
| `iss` | `doodhee` |
| `aud` | `doodhee-public` or `doodhee-admin` |
| `sub` | User UUID |
| `sid` | Session UUID |
| `role` | `customer`, `seller`, or `admin` |
| `amr` | Authentication methods, such as `otp` and `mfa` |
| `jti` | Unique access-token ID |
| `iat`, `exp` | Issued and expiry timestamps |

Do not put approval status or enabled status only in the token because they can change immediately. Protected requests must confirm that the session and account remain active.

Recommended lifetimes:

| Session type | Access token | Refresh session |
| --- | --- | --- |
| Public buyer or seller | 15 minutes | 30 days with rotation |
| Admin | 10 minutes | 8 hours with rotation |

Refresh-token reuse revokes the complete token family. Logout revokes the session.

## 6. Authorization pipeline

Every protected request passes through this order:

```text
Request ID -> Rate limit -> Token verification -> Audience check -> Session check
-> Enabled-account check -> Role check -> Approval check -> Use case -> Ownership check
```

### Important distinction

- **RBAC** answers: may this role call this operation?
- **Ownership authorization** answers: may this user access this specific record?

Both are required. For example, the seller role may read seller orders, but the use case must still enforce `orders.seller_id = authenticated_user_id`.

The backend derives `customer_id`, `seller_id`, `marked_by`, and `approved_by` from authenticated context or trusted records. These IDs must never be trusted from frontend input.

## 7. Role and permission matrix

| Capability | Customer | Pending seller | Approved seller | Admin |
| --- | --- | --- | --- | --- |
| View own profile | Yes | Yes | Yes | Yes |
| View seller approval status | No | Yes | Yes | Yes |
| Discover approved sellers and products | Yes | No | No | Yes |
| Place and cancel own pending order | Yes | No | No | Admin correction only |
| View own buyer orders and total | Yes | No | No | Yes |
| Manage seller products | No | No | Own only | Activate or suspend |
| View assigned seller orders | No | No | Own only | Yes |
| Mark delivery result | No | No | Assigned orders only | Correct with reason |
| Approve, reject, enable or disable users | No | No | No | Yes |
| Inspect or retry failed events | No | No | No | Yes |
| View admin audit records | No | No | No | Yes |

## 8. API conventions

- Public base path: `/v1`
- Admin base path: `/v1/admin`
- JSON request and response bodies
- UTC timestamps in RFC 3339 format
- Dates in `YYYY-MM-DD`
- Money returned as decimal strings, such as `"65.00"`
- Quantities returned as decimal strings, such as `"1.500"`
- Cursor pagination for list APIs
- `Idempotency-Key` required for order creation and admin corrections
- `X-Request-ID` accepted or generated and returned on every response
- OpenAPI specifications maintained separately for both gateways

### Success envelope

```json
{
  "data": {},
  "meta": {
    "request_id": "req_..."
  }
}
```

### Error envelope

```json
{
  "error": {
    "code": "ORDER_NOT_ACCESSIBLE",
    "message": "The order was not found or is not accessible.",
    "fields": {}
  },
  "meta": {
    "request_id": "req_..."
  }
}
```

Use the same not-found response for a missing record and a record owned by another user. This prevents ID discovery.

## 9. Public Gateway APIs

### 9.1 Authentication and profile

| Method | Path | Access | Purpose and main input |
| --- | --- | --- | --- |
| POST | `/v1/auth/otp/request` | Public | Request OTP: `phone`, `role` (`customer` or `seller`) |
| POST | `/v1/auth/otp/verify` | Public | Verify: `challenge_id`, `code`, and `full_name` for a new account |
| POST | `/v1/auth/token/refresh` | Refresh session | Rotate refresh token and issue new access token |
| POST | `/v1/auth/logout` | Authenticated | Revoke current session |
| POST | `/v1/auth/logout-all` | Authenticated | Revoke all sessions for the current user |
| GET | `/v1/me` | Customer or seller | Current profile, role, approval and enabled state |
| PATCH | `/v1/me` | Customer or seller | Update own name and basic address fields |

OTP request responses are generic whether or not the phone already exists. Do not reveal registered accounts.

### 9.2 Buyer discovery APIs

| Method | Path | Access | Purpose and main input |
| --- | --- | --- | --- |
| GET | `/v1/sellers` | Customer | Approved enabled sellers; filters: `pincode`, cursor |
| GET | `/v1/sellers/{seller_id}` | Customer | Public seller profile |
| GET | `/v1/sellers/{seller_id}/products` | Customer | Available products for one seller |
| GET | `/v1/products/{product_id}` | Customer | Product details, price and availability |

Only approved and enabled sellers with available products are returned.

### 9.3 Buyer order APIs

| Method | Path | Access | Purpose and main input |
| --- | --- | --- | --- |
| POST | `/v1/orders` | Customer | Place order: `product_id`, `service_date`, `quantity`, `delivery_address`, optional `customer_note` |
| GET | `/v1/orders` | Customer | Own orders; filters: `from`, `to`, `status`, `seller_id`, cursor |
| GET | `/v1/orders/{order_id}` | Customer owner | Own daily order and delivery status |
| POST | `/v1/orders/{order_id}/cancel` | Customer owner | Cancel an allowed pending order: `reason` |
| GET | `/v1/bill-summary` | Customer | Delivered-day summary; query: `from`, `to`, optional `seller_id` |

The create-order use case obtains seller ID and current unit price from the product record. It checks seller approval, product availability, date, quantity, and capacity inside one transaction.

Bill summary includes delivered days, delivered litres and total amount. Only `delivered` orders count.

### 9.4 Seller profile and product APIs

| Method | Path | Access | Purpose and main input |
| --- | --- | --- | --- |
| GET | `/v1/seller/approval-status` | Any seller | Pending, approved or rejected status |
| GET | `/v1/seller/profile` | Any seller | Own seller profile |
| PATCH | `/v1/seller/profile` | Any seller | Update allowed profile and address fields |
| POST | `/v1/seller/products` | Approved seller | Create: `name`, `unit`, `price`, `daily_capacity` |
| GET | `/v1/seller/products` | Approved seller | List own products including unavailable ones |
| GET | `/v1/seller/products/{product_id}` | Product owner | Get own product |
| PATCH | `/v1/seller/products/{product_id}` | Product owner | Update price, capacity or name |
| POST | `/v1/seller/products/{product_id}/availability` | Product owner | Set `is_available` |

A pending or rejected seller receives `403 SELLER_NOT_APPROVED` for product and order operations.

### 9.5 Seller order and summary APIs

| Method | Path | Access | Purpose and main input |
| --- | --- | --- | --- |
| GET | `/v1/seller/orders` | Approved seller | Assigned orders; filters: `service_date`, `from`, `to`, `status`, `customer_id`, cursor |
| GET | `/v1/seller/orders/{order_id}` | Assigned seller | Order and delivery details |
| POST | `/v1/seller/orders/{order_id}/accept` | Assigned seller | Move `pending` to `accepted` |
| POST | `/v1/seller/orders/{order_id}/out-for-delivery` | Assigned seller | Move `accepted` to `out_for_delivery` |
| POST | `/v1/seller/orders/{order_id}/delivery-result` | Assigned seller | `result`: `delivered` or `not_delivered`; reason required when not delivered |
| GET | `/v1/seller/bill-summary` | Approved seller | Delivered totals; query: `from`, `to`, optional `customer_id` |

The delivery-result request may include `reason_code` and `note`. The backend sets `marked_by` and `delivery_marked_at`.

## 10. Admin Gateway APIs

### 10.1 Admin authentication

| Method | Path | Access | Purpose and main input |
| --- | --- | --- | --- |
| POST | `/v1/admin/auth/otp/request` | Public admin route | Request OTP: admin phone |
| POST | `/v1/admin/auth/otp/verify` | Public admin route | Verify OTP and return MFA challenge |
| POST | `/v1/admin/auth/mfa/verify` | MFA challenge | Verify second factor and create admin session |
| POST | `/v1/admin/auth/token/refresh` | Admin refresh session | Rotate refresh token |
| POST | `/v1/admin/auth/logout` | Admin | Revoke current admin session |
| GET | `/v1/admin/me` | Admin | Current admin identity and session details |

Admin authentication uses the admin audience and cannot issue customer or seller tokens.

### 10.2 User and seller administration

| Method | Path | Access | Purpose and main input |
| --- | --- | --- | --- |
| GET | `/v1/admin/users` | Admin | Filter by role, approval state, enabled state, phone and cursor |
| GET | `/v1/admin/users/{user_id}` | Admin | User details and status history |
| GET | `/v1/admin/sellers/pending` | Admin | Pending seller queue |
| POST | `/v1/admin/sellers/{seller_id}/approve` | Admin | Approve seller; optional admin note |
| POST | `/v1/admin/sellers/{seller_id}/reject` | Admin | Reject seller; required reason |
| POST | `/v1/admin/users/{user_id}/enable` | Admin | Enable account; required reason when restoring |
| POST | `/v1/admin/users/{user_id}/disable` | Admin | Disable account and revoke sessions; required reason |

Approval endpoints are idempotent for the same final state. Every mutation records admin ID, target, reason, request ID, and before/after state.

### 10.3 Product administration

| Method | Path | Access | Purpose and main input |
| --- | --- | --- | --- |
| GET | `/v1/admin/products` | Admin | Search products by seller, availability and pincode |
| GET | `/v1/admin/products/{product_id}` | Admin | Product and seller details |
| POST | `/v1/admin/products/{product_id}/suspend` | Admin | Make unavailable; required reason |
| POST | `/v1/admin/products/{product_id}/activate` | Admin | Restore product; required reason |

### 10.4 Order and reporting administration

| Method | Path | Access | Purpose and main input |
| --- | --- | --- | --- |
| GET | `/v1/admin/orders` | Admin | Search by date, status, customer, seller and cursor |
| GET | `/v1/admin/orders/{order_id}` | Admin | Complete order and delivery record |
| POST | `/v1/admin/orders/{order_id}/correct-delivery-result` | Admin | Correct result with required reason and idempotency key |
| GET | `/v1/admin/bill-summary` | Admin | Delivered totals by date, seller or customer |
| GET | `/v1/admin/dashboard` | Admin | Counts for users, sellers, orders and delivery results |

An admin correction never silently overwrites history. It emits an event and stores an audit record with previous and new values.

### 10.5 Event and audit operations

| Method | Path | Access | Purpose and main input |
| --- | --- | --- | --- |
| GET | `/v1/admin/events` | Admin | Inspect outbox events; filter by type and status |
| GET | `/v1/admin/events/{event_id}` | Admin | Event attempts and latest error |
| POST | `/v1/admin/events/{event_id}/retry` | Admin | Retry a failed event with audit reason |
| GET | `/v1/admin/audit-logs` | Admin | Search admin actions by actor, target, action and date |

Raw OTPs, access tokens, refresh tokens and MFA secrets must never appear in audit logs.

## 11. Main request flows

### 11.1 Buyer places an order

```text
Buyer App
  -> Public Gateway
  -> Public authentication middleware
  -> CreateOrder application use case
  -> Product repository: load seller, price and availability
  -> Order repository: save order
  -> Outbox repository: save OrderPlaced in the same transaction
  -> HTTP 201 response
  -> Event worker notifies seller
```

### 11.2 Seller marks delivery

```text
Seller App
  -> Public Gateway
  -> Seller role and approval checks
  -> MarkDeliveryResult use case
  -> Verify order.seller_id equals authenticated seller
  -> Validate state transition and reason
  -> Update order plus outbox event in one transaction
  -> Customer sees the same status and total
```

### 11.3 Admin approves a seller

```text
Admin Web
  -> Admin Gateway
  -> Admin audience, session, MFA and role checks
  -> ApproveSeller use case
  -> Update approval fields
  -> Save SellerApproved event and admin audit record in one transaction
  -> Event worker notifies seller
```

## 12. Order state machine

```text
pending -> accepted -> out_for_delivery -> delivered
                                 \-----> not_delivered

pending -> cancelled
accepted -> cancelled
```

Rules:

- Only the customer may cancel their own eligible order.
- Only the assigned approved seller may accept or mark delivery progress.
- `not_delivered` requires a standard reason code.
- Completed states cannot be changed through the public gateway.
- Admin correction requires a reason, idempotency key, audit record and event.

## 13. Event catalog

| Event | Producer | Typical consumers |
| --- | --- | --- |
| `UserRegistered` | Identity | Welcome notification |
| `SellerRegistered` | Identity | Admin queue notification |
| `SellerApproved` | Admin use case | Seller notification |
| `SellerRejected` | Admin use case | Seller notification |
| `UserDisabled` | Admin use case | Session revocation and notification |
| `ProductCreated` | Seller use case | Search or cache refresh |
| `ProductAvailabilityChanged` | Seller or admin use case | Search refresh |
| `OrderPlaced` | Buyer use case | Seller notification |
| `OrderAccepted` | Seller use case | Buyer notification |
| `OrderOutForDelivery` | Seller use case | Buyer notification |
| `OrderDelivered` | Seller or admin use case | Buyer notification and summary refresh |
| `OrderNotDelivered` | Seller or admin use case | Buyer notification and summary refresh |
| `OrderCancelled` | Buyer use case | Seller notification |

Event handlers must be idempotent. The outbox worker claims rows with PostgreSQL row locking, retries temporary failures with backoff, and moves permanent failures to `failed` for admin review.

## 14. Database support for authentication and audit

The four core tables remain `users`, `products`, `orders`, and `outbox_events`. Correct authentication and admin auditing require these supporting tables:

| Table | Essential fields | Purpose |
| --- | --- | --- |
| `auth_challenges` | `id`, `phone`, `audience`, `purpose`, `code_hash`, `expires_at`, `attempt_count`, `consumed_at` | OTP and MFA challenge lifecycle |
| `auth_sessions` | `id`, `user_id`, `audience`, `refresh_token_hash`, `token_family_id`, `expires_at`, `revoked_at`, `last_seen_at` | Refresh rotation, logout and revocation |
| `mfa_methods` | `id`, `user_id`, `type`, `secret_encrypted`, `confirmed_at`, `disabled_at` | Admin second-factor configuration |
| `admin_audit_logs` | `id`, `admin_id`, `action`, `target_type`, `target_id`, `reason`, `request_id`, `before_data`, `after_data`, `created_at` | Immutable admin action history |

OTP codes and refresh tokens are never stored in plain text. MFA secrets are encrypted with a managed key.

One PostgreSQL database is still used. Recommended database credentials are separate for the public gateway, admin gateway, worker and migration process to reduce accidental privilege exposure.

## 15. Security requirements

- TLS is mandatory outside local development.
- Validate JWT signature, issuer, audience, expiry and session on every protected request.
- Require admin MFA and a separate admin token audience.
- Keep public and admin CORS allowlists separate.
- Use generic OTP responses to prevent account enumeration.
- Store OTP and refresh-token hashes, never raw values.
- Limit OTP attempts, resend frequency and requests per phone and IP.
- Rotate refresh tokens and detect token reuse.
- Revoke all sessions immediately when an account is disabled.
- Use parameterized SQL and strict request validation.
- Never trust user IDs, prices, totals, roles or timestamps supplied by clients.
- Redact tokens, OTPs, phone numbers and personal addresses from application logs.
- Apply body-size limits, timeouts and request cancellation.
- Record security-relevant admin actions in an append-only audit log.
- Use a separate admin hostname and firewall or VPN restriction when operationally possible.

## 16. Rate limits

| Area | Starting policy |
| --- | --- |
| OTP request | 1 per 30 seconds, 5 per phone per hour, plus IP limit |
| OTP verification | Maximum 5 attempts per challenge |
| Public authenticated APIs | Per user and IP |
| Admin login | Stricter per account and IP with alerting |
| Admin mutation APIs | Per admin with audit |
| Order creation | Idempotency key plus per-customer limit |

Exact values should be configuration, not constants in handlers.

## 17. HTTP status and error rules

| Status | Use |
| --- | --- |
| `200` | Successful read or update |
| `201` | Resource created |
| `204` | Successful logout with no body |
| `400` | Invalid JSON or malformed request |
| `401` | Missing, expired or invalid authentication |
| `403` | Valid identity but role, approval or state denies the operation |
| `404` | Missing or inaccessible resource |
| `409` | State conflict, duplicate idempotency key with different input, or invalid transition |
| `422` | Valid JSON with field validation errors |
| `429` | Rate limit exceeded |
| `500` | Unexpected server failure |
| `503` | Temporary dependency failure |

Stable error codes include `AUTH_INVALID_OTP`, `AUTH_SESSION_REVOKED`, `ROLE_FORBIDDEN`, `SELLER_NOT_APPROVED`, `ACCOUNT_DISABLED`, `ORDER_INVALID_TRANSITION`, `ORDER_REASON_REQUIRED`, and `IDEMPOTENCY_CONFLICT`.

## 18. Observability and operations

- Structured JSON logs with request ID, route, status, duration and authenticated subject ID
- Separate metrics for public gateway, admin gateway and worker
- Health endpoints: `/health/live` and `/health/ready`, not containing sensitive details
- Metrics for login success, OTP failures, authorization denials, order transitions and outbox lag
- Alerts for repeated admin login failures, token reuse, event backlog and high error rate
- Distributed tracing can be added later; preserve request ID and event correlation ID now

## 19. Acceptance checklist

- A public token cannot call the admin gateway.
- An admin token cannot call buyer or seller routes.
- A customer cannot read another customer's order.
- A seller cannot read or update another seller's product or order.
- A pending seller cannot create products or process orders.
- Disabling a user revokes active sessions.
- Admin login cannot complete without MFA.
- Admin actions create immutable audit records.
- Order creation cannot accept client-supplied price, seller ID or customer ID.
- Delivery result cannot be `not_delivered` without a reason.
- Bill summaries count only delivered orders and match on both sides.
- Order changes and their events commit or roll back together.
- Repeating an idempotent request does not create duplicate orders or corrections.

## 20. Not included yet

- Formal invoices, online payments, refunds and seller payouts
- Complaint and dispute workflow
- Subscriptions and automatic daily-order generation
- Delivery riders, live tracking and proof images
- Quality tests and ratings
- Kafka, gRPC and independently deployed business microservices

