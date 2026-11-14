# Doodhee Frontend Architecture and Complete User Flows

> Implementation update (14 November 2026): every enabled Milkman now routes directly to the workspace. Flutter includes customer search/add, per-customer pricing, direct daily deliveries, actual delivery quantity/price, extra-milk decisions and ratings/rankings. Admin approval navigation/actions were removed. See [the implemented plan](MILKMAN_CUSTOMER_DELIVERY_IMPLEMENTATION_PLAN.md) and [current build status](BUILD_STATUS.md). Approval descriptions below are retained only as the original planning baseline.

## 1. Purpose and decision

Doodhee has two frontend applications:

- **Admin Web:** Next.js with TypeScript, used only by admins
- **Doodhee Mobile App:** one Flutter application shared by buyers and sellers

In this document, **buyer** and **customer** mean the same role. The backend role value remains `customer`.

The Flutter app calls only the Public Gateway. The Next.js application calls only the Admin Gateway through a thin same-origin Backend for Frontend layer. Neither frontend connects directly to PostgreSQL or the event worker.

## 2. High-level frontend architecture

```text
Flutter Buyer UI ----\
                       > Flutter App -> Public Gateway -> Shared Backend Modules
Flutter Seller UI ---/

Admin Browser -> Next.js Admin UI -> Next.js BFF -> Admin Gateway -> Shared Backend Modules
```

### Runtime boundaries

| Application | Users | Backend connection | Deployment |
| --- | --- | --- | --- |
| Flutter app | Buyer and seller | Public Gateway only | Android and iOS app |
| Next.js admin web | Admin only | Admin Gateway through Next.js BFF | Separate admin website |

The mobile application must not contain the Admin Gateway URL. The admin website must not call Public Gateway buyer or seller routes.

## 3. Shared frontend principles

- The backend is the source of truth for role, approval, account state, price, totals and order status.
- The frontend never sends trusted user IDs, seller IDs, prices, totals, roles or audit identities.
- UI role guards improve navigation but do not replace backend authorization.
- Every screen handles loading, empty, success, validation, offline and server-error states.
- OpenAPI files generate typed Dart and TypeScript API models.
- All mutations use the backend error envelope and display safe, actionable messages.
- Order creation and admin correction use idempotency keys.
- UTC timestamps are converted to the user's local timezone only for display.
- Money and quantities remain decimal strings until formatted for display.

## 4. Frontend repositories

The two apps are independently buildable and deployable. They may live in one source repository without sharing runtime code.

```text
frontend/
├── admin-web/          # Next.js and TypeScript
├── mobile-app/         # Flutter and Dart
├── contracts/
│   ├── public-v1.yaml
│   └── admin-v1.yaml
└── docs/
```

The backend OpenAPI specifications are the source of truth. Generated clients are committed or generated in CI using a pinned generator version. Manual edits to generated files are not allowed.

## 5. Flutter application architecture

The Flutter app uses MVVM with repositories and services:

```text
View -> ViewModel -> Repository -> API or Local Service -> Public Gateway or Device Storage
```

### Layer responsibilities

| Layer | Responsibility |
| --- | --- |
| View | Widgets, layout, accessibility, input events and rendering UI state |
| ViewModel | Screen state, commands, validation orchestration and presentation mapping |
| Repository | Source of truth for session, profile, products, orders and summaries |
| Service | Stateless wrapper for REST APIs, secure storage, local database and connectivity |
| Optional use case | Reusable logic that combines multiple repositories or handles a complex workflow |

Use cases are added only for complex or reused operations. A simple screen may call one repository directly through its ViewModel.

### Flutter dependency choices

- `go_router` for declarative routes, deep links and role redirects
- Riverpod providers for dependency construction, ViewModel lifetime and test overrides
- A typed REST client generated from `public-v1.yaml`
- Platform secure storage for refresh tokens
- A local database for cached reads and pending seller delivery operations

The architecture must not depend on widgets calling network services directly.

## 6. Flutter folder structure

```text
mobile-app/lib/
├── main.dart
├── app/
│   ├── bootstrap.dart
│   ├── router.dart
│   ├── theme.dart
│   └── localization.dart
├── core/
│   ├── api/
│   │   ├── public_api_client.dart
│   │   ├── auth_interceptor.dart
│   │   └── api_error_mapper.dart
│   ├── auth/
│   │   ├── session.dart
│   │   └── session_repository.dart
│   ├── storage/
│   │   ├── secure_token_store.dart
│   │   └── local_database.dart
│   ├── connectivity/
│   ├── errors/
│   └── widgets/
├── features/
│   ├── auth/
│   │   ├── data/
│   │   ├── domain/
│   │   └── presentation/
│   ├── profile/
│   ├── buyer_discovery/
│   ├── buyer_orders/
│   ├── buyer_summary/
│   ├── seller_approval/
│   ├── seller_products/
│   ├── seller_orders/
│   └── seller_summary/
└── generated/
    └── public_api/
```

Each feature contains its views, ViewModels and feature-specific models. Shared networking, authentication and storage stay under `core`.

## 7. Flutter session and role routing

### Session states

```text
unknown
  -> unauthenticated
  -> authenticatedCustomer
  -> pendingSeller
  -> approvedSeller
  -> rejectedSeller
  -> disabled
```

### Startup flow

1. Splash screen opens.
2. Session repository checks secure storage for a refresh token.
3. If missing, route to role selection and login.
4. If present, call `POST /v1/auth/token/refresh`.
5. If refresh succeeds, keep the access token in memory and call `GET /v1/me`.
6. Route using the server-returned role, approval state and enabled state.
7. If refresh is rejected, clear tokens and route to login.

### Role destinations

| Server state | Destination |
| --- | --- |
| Customer, enabled | Buyer home |
| Seller, pending | Seller approval-pending screen |
| Seller, approved and enabled | Seller dashboard |
| Seller, rejected | Seller rejection screen with status message |
| Any disabled account | Disabled-account screen, then logout |
| Missing or expired session | Login |

The router reads `SessionRepository`; it never accepts a role selected from local storage as authoritative.

## 8. Flutter authentication flow

### Screens and API calls

| Screen or action | API call | What happens next |
| --- | --- | --- |
| Choose Buyer or Seller | None | Sets registration intent only; it does not grant a role |
| Enter phone and request OTP | `POST /v1/auth/otp/request` | Save returned challenge ID temporarily and open OTP screen |
| Submit OTP | `POST /v1/auth/otp/verify` | Store refresh token securely, keep access token in memory, then call `/v1/me` |
| App resumes with expired access token | `POST /v1/auth/token/refresh` | Rotate refresh token and retry one failed request |
| Open profile | `GET /v1/me` | Render server role, approval and profile |
| Save profile | `PATCH /v1/me` | Update repository cache and visible profile |
| Logout | `POST /v1/auth/logout` | Clear secure tokens, local private cache and route to login |
| Logout all devices | `POST /v1/auth/logout-all` | Clear local session and route to login |

### Token handling

- Refresh token is stored in platform secure storage.
- Access token is kept in memory and sent as `Authorization: Bearer`.
- A refresh mutex ensures multiple simultaneous `401` responses trigger only one refresh request.
- After one successful refresh, failed requests may retry once.
- A second `401`, revoked session or disabled-account error clears the session.
- OTPs and tokens never appear in logs, analytics or crash reports.

## 9. Flutter navigation map

```text
/splash
/auth/role
/auth/phone
/auth/otp
/account/disabled

/buyer/home
/buyer/sellers/:sellerId
/buyer/products/:productId
/buyer/orders
/buyer/orders/new/:productId
/buyer/orders/:orderId
/buyer/summary
/buyer/profile

/seller/pending
/seller/rejected
/seller/home
/seller/products
/seller/products/new
/seller/products/:productId/edit
/seller/orders
/seller/orders/:orderId
/seller/summary
/seller/profile
```

Buyer routes redirect sellers to their seller destination. Seller routes redirect customers to buyer home. Pending and rejected sellers cannot open approved-seller routes.

## 10. Buyer screens and complete API flow

### 10.1 Buyer home

| User action | API call | UI result |
| --- | --- | --- |
| Open home | `GET /v1/sellers?pincode=...` | List approved enabled sellers; show loading, empty or retry state |
| Pull to refresh | Same request with first cursor | Replace cached seller list |
| Load next page | Same request with cursor | Append without duplicating sellers |
| Tap seller | `GET /v1/sellers/{seller_id}` | Open seller details |

The app may display cached seller data immediately, then refresh it. Stale data is visibly marked if the network request fails.

### 10.2 Seller and product discovery

| User action | API call | UI result |
| --- | --- | --- |
| Open seller | `GET /v1/sellers/{seller_id}` | Seller profile and service details |
| Load products | `GET /v1/sellers/{seller_id}/products` | Available milk products |
| Open product | `GET /v1/products/{product_id}` | Current price, unit and availability |
| Tap Order | None yet | Open order form using product ID; displayed price is informational |

The order request sends only product ID, date, quantity, address and note. The backend determines seller and price again.

### 10.3 Place an order

```text
Buyer enters date, quantity, address and optional note
  -> ViewModel validates required fields
  -> Generate Idempotency-Key
  -> POST /v1/orders
  -> Public Gateway validates session and customer role
  -> Backend loads product, seller, price and capacity
  -> Backend saves order plus OrderPlaced event
  -> App stores returned order and opens order details
```

| Result | Frontend behavior |
| --- | --- |
| `201` | Show success and open the created order |
| Validation error | Attach field messages to the form |
| Product unavailable or capacity conflict | Refresh product and explain that it is unavailable |
| Timeout after submission | Retry with the same idempotency key |
| Offline before submission | Keep the form draft locally; do not queue the order automatically |

### 10.4 Buyer order list and detail

| Screen or action | API call | UI result |
| --- | --- | --- |
| Open orders | `GET /v1/orders?from=...&to=...&status=...` | Own orders grouped by service date |
| Open order | `GET /v1/orders/{order_id}` | Daily status, quantity, price, amount, notes and reason |
| Cancel eligible order | `POST /v1/orders/{order_id}/cancel` | Update detail and list to `cancelled` |
| Resume app on order screen | `GET /v1/orders/{order_id}` | Refresh status from server |

The screen displays one of: pending, accepted, out for delivery, delivered, not delivered or cancelled. A not-delivered order displays its standard reason and seller note.

### 10.5 Buyer total summary

| User action | API call | UI result |
| --- | --- | --- |
| Open summary | `GET /v1/bill-summary?from=...&to=...` | Delivered days, litres and total amount |
| Filter seller | Add `seller_id` | Recalculate for the selected seller |
| Change month | Change `from` and `to` | Load the selected period |

Only delivered orders count. The UI must not independently recalculate a different total from partially loaded pages.

## 11. Seller screens and complete API flow

### 11.1 Pending or rejected seller

| User action | API call | UI result |
| --- | --- | --- |
| Open status screen | `GET /v1/seller/approval-status` | Pending, approved or rejected state |
| Refresh status | Same endpoint | Route to seller dashboard if newly approved |
| View profile | `GET /v1/seller/profile` | Own submitted information |
| Update allowed details | `PATCH /v1/seller/profile` | Save and refresh profile |

A pending or rejected seller cannot see product or order management navigation.

### 11.2 Seller dashboard

The dashboard loads independent sections so one failure does not blank the whole page:

- Today's orders: `GET /v1/seller/orders?service_date=YYYY-MM-DD`
- Date-range totals: `GET /v1/seller/bill-summary?from=...&to=...`
- Product availability: `GET /v1/seller/products`

The dashboard shows counts for pending, accepted, out for delivery, delivered and not delivered.

### 11.3 Seller product management

| User action | API call | UI result |
| --- | --- | --- |
| Open products | `GET /v1/seller/products` | Own products including unavailable products |
| Create product | `POST /v1/seller/products` | Add returned product to list |
| Open edit form | `GET /v1/seller/products/{product_id}` | Current product details |
| Save changes | `PATCH /v1/seller/products/{product_id}` | Replace cached product with server response |
| Toggle availability | `POST /v1/seller/products/{product_id}/availability` | Show confirmed server state |

The UI never assumes that a toggle succeeded before the server confirms it. It may show a temporary submitting state.

### 11.4 Seller daily orders

| User action | API call | UI result |
| --- | --- | --- |
| Open daily list | `GET /v1/seller/orders?service_date=...` | Assigned orders only |
| Open order | `GET /v1/seller/orders/{order_id}` | Customer delivery details |
| Accept order | `POST /v1/seller/orders/{order_id}/accept` | Status becomes accepted |
| Start delivery | `POST /v1/seller/orders/{order_id}/out-for-delivery` | Status becomes out for delivery |
| Mark delivered | `POST /v1/seller/orders/{order_id}/delivery-result` | Send `result=delivered` |
| Mark not delivered | Same endpoint | Send `result=not_delivered`, required reason and optional note |

After every mutation, update the returned order locally and invalidate the daily list and seller summary.

### 11.5 Seller offline delivery flow

The seller's current daily list is cached locally after a successful load.

```text
Seller is offline
  -> Opens cached daily orders
  -> Marks delivered or not delivered
  -> App saves pending operation with idempotency key
  -> UI shows Pending sync, not final success
  -> Connectivity returns
  -> Sync worker sends operations in order
  -> Server accepts: replace local state with server order
  -> Server conflict: show Needs attention and fetch latest order
```

Only delivery-result operations are queued initially. Product edits, order creation and admin actions are not queued offline.

Each pending operation stores:

- Local operation ID
- Order ID
- Requested result
- Reason and note
- Idempotency key
- Client timestamp for display only
- Retry count and last safe error

The backend remains authoritative for `delivery_marked_at` and final status.

### 11.6 Seller total summary

| User action | API call | UI result |
| --- | --- | --- |
| Open summary | `GET /v1/seller/bill-summary?from=...&to=...` | Delivered litres and total across customers |
| Filter customer | Add `customer_id` | Delivered rows and total for one customer |
| Change date range | Change `from` and `to` | Load new server summary |

Seller and buyer views display values returned by the backend summary APIs so they agree.

## 12. Flutter networking and state rules

### Query state

Every data ViewModel exposes these states:

```text
initial -> loading -> data | empty | error
data -> refreshing -> data | errorWithStaleData
```

### Mutation state

```text
idle -> submitting -> success | validationError | conflict | failure
```

Buttons are disabled while the same mutation is submitting. A retry reuses the same idempotency key only when it represents the same logical action.

### Error mapping

| Backend result | Flutter response |
| --- | --- |
| `401` | Refresh once; otherwise clear session |
| `403 SELLER_NOT_APPROVED` | Route seller to approval screen |
| `403 ACCOUNT_DISABLED` | Clear session and show disabled screen |
| `404` | Show unavailable and remove stale inaccessible item |
| `409 ORDER_INVALID_TRANSITION` | Fetch latest order and explain state changed |
| `422` | Show field-level validation |
| `429` | Show retry delay and disable repeated requests |
| `500` or `503` | Keep safe cached data and show retry option |

## 13. Next.js admin architecture

Use Next.js App Router with TypeScript. The website contains presentation, a server-only data-access layer and thin Route Handlers acting as a BFF.

```text
Admin Browser
  -> Next.js page and client feature
  -> Same-origin /api/admin Route Handler
  -> Server-only admin gateway client
  -> Admin Gateway
```

The BFF may validate and normalize browser input, manage HttpOnly cookies, attach request IDs and proxy requests. It must not implement seller approval, order transition or authorization rules; those stay in the Admin Gateway.

## 14. Next.js folder structure

```text
admin-web/
├── app/
│   ├── (auth)/
│   │   ├── login/page.tsx
│   │   └── mfa/page.tsx
│   ├── (protected)/
│   │   ├── layout.tsx
│   │   ├── dashboard/page.tsx
│   │   ├── sellers/pending/page.tsx
│   │   ├── users/page.tsx
│   │   ├── users/[id]/page.tsx
│   │   ├── products/page.tsx
│   │   ├── orders/page.tsx
│   │   ├── orders/[id]/page.tsx
│   │   ├── events/page.tsx
│   │   └── audit/page.tsx
│   └── api/admin/[...path]/route.ts
├── features/
│   ├── auth/
│   ├── dashboard/
│   ├── sellers/
│   ├── users/
│   ├── products/
│   ├── orders/
│   ├── events/
│   └── audit/
├── lib/
│   ├── server/
│   │   ├── admin-gateway-client.ts
│   │   ├── session.ts
│   │   └── dal.ts
│   ├── validation/
│   ├── errors/
│   └── request-id.ts
├── components/
├── generated/admin-api/
├── proxy.ts
└── tests/
```

Server Components provide the protected shell and static page structure. Interactive tables, filters, dialogs and mutations use Client Components that call same-origin Route Handlers.

## 15. Next.js admin authentication flow

```text
Admin enters phone
  -> Browser POST /api/admin/auth/otp/request
  -> Next.js BFF POST /v1/admin/auth/otp/request
  -> Browser opens OTP screen
  -> OTP verify through BFF
  -> Admin Gateway returns MFA challenge only
  -> Browser submits MFA code through BFF
  -> Admin Gateway creates admin session
  -> BFF stores tokens in Secure HttpOnly SameSite cookies
  -> Browser redirects to dashboard
```

### Session rules

- JavaScript cannot read access or refresh tokens.
- BFF reads cookies server-side and forwards authentication to Admin Gateway.
- BFF rotates refresh tokens and retries one failed request.
- `proxy.ts` may perform an optimistic redirect, but protected data access verifies the session again.
- The Admin Gateway remains authoritative for admin role, MFA, enabled state and permissions.
- Logout clears cookies only after requesting backend session revocation.
- State-changing BFF routes require CSRF protection and same-origin checks.

## 16. Admin screens and complete API flow

All browser paths below call same-origin Next.js BFF paths. The BFF forwards them to the matching `/v1/admin/...` endpoint.

### 16.1 Dashboard

| User action | Backend call | UI result |
| --- | --- | --- |
| Open dashboard | `GET /v1/admin/dashboard` | User, seller, order and delivery metrics |
| Refresh | Same endpoint | Replace dashboard snapshot |
| Click a metric | Navigate with filters | Open the relevant list page |

### 16.2 Pending seller approval

| User action | Backend call | UI result |
| --- | --- | --- |
| Open queue | `GET /v1/admin/sellers/pending` | Paginated pending sellers |
| Open seller | `GET /v1/admin/users/{user_id}` | Profile and approval information |
| Approve | `POST /v1/admin/sellers/{seller_id}/approve` | Remove from pending list and show success |
| Reject | `POST /v1/admin/sellers/{seller_id}/reject` | Require reason, then remove from queue |

Approval and rejection dialogs show a final confirmation. Duplicate clicks reuse the same logical request and are disabled while pending.

### 16.3 User administration

| User action | Backend call | UI result |
| --- | --- | --- |
| Search users | `GET /v1/admin/users` | Filtered paginated table |
| Open user | `GET /v1/admin/users/{user_id}` | User details and current status |
| Disable | `POST /v1/admin/users/{user_id}/disable` | Require reason; backend revokes sessions |
| Enable | `POST /v1/admin/users/{user_id}/enable` | Require reason and show confirmed state |

The UI does not immediately claim success for a destructive account action. It waits for the Admin Gateway response and shows the audit request ID.

### 16.4 Product administration

| User action | Backend call | UI result |
| --- | --- | --- |
| Search products | `GET /v1/admin/products` | Products with seller and availability filters |
| Open product | `GET /v1/admin/products/{product_id}` | Product and seller context |
| Suspend | `POST /v1/admin/products/{product_id}/suspend` | Require reason and refresh row |
| Activate | `POST /v1/admin/products/{product_id}/activate` | Require reason and refresh row |

### 16.5 Order administration

| User action | Backend call | UI result |
| --- | --- | --- |
| Search orders | `GET /v1/admin/orders` | Date, status, customer and seller filters |
| Open order | `GET /v1/admin/orders/{order_id}` | Full daily order record |
| Correct delivery result | `POST /v1/admin/orders/{order_id}/correct-delivery-result` | Require reason and idempotency key; refresh order |
| View totals | `GET /v1/admin/bill-summary` | Delivered totals using selected filters |

The correction dialog shows old and proposed values. After success, it displays the new server state and audit reference.

### 16.6 Event operations

| User action | Backend call | UI result |
| --- | --- | --- |
| Open events | `GET /v1/admin/events` | Filterable event table |
| Open event | `GET /v1/admin/events/{event_id}` | Attempts and safe error details |
| Retry failed event | `POST /v1/admin/events/{event_id}/retry` | Require reason; show returned event state |

### 16.7 Audit logs

| User action | Backend call | UI result |
| --- | --- | --- |
| Open audit page | `GET /v1/admin/audit-logs` | Filter by actor, target, action and date |
| Open record | Existing row data | Show before and after values with sensitive fields removed |

Audit pages are read-only.

## 17. Admin BFF forwarding rules

1. Accept only known methods and path prefixes.
2. Validate content type and body-size limit.
3. Verify the local admin session cookie.
4. Add backend access token and `X-Request-ID` server-side.
5. Forward only allowlisted headers; never forward arbitrary browser headers.
6. Apply a timeout and cancel the upstream request if the browser disconnects.
7. On one backend `401`, rotate refresh token and retry once.
8. Copy safe status, JSON body and request ID back to the browser.
9. Never expose access tokens, refresh tokens or internal gateway errors.

The BFF is not a second source of business truth. The Admin Gateway repeats authentication, role and authorization checks.

## 18. Cross-application data consistency

### Delivery status

```text
Seller marks delivery in Flutter
  -> Public Gateway updates order and emits event
  -> Buyer refreshes order or app resumes
  -> Buyer sees server status and reason
  -> Admin searches the same order through Admin Gateway
  -> All three screens render the same order record
```

### Total summary

Buyer, seller and admin screens display their gateway's summary response. No frontend independently creates a bill by summing only the currently loaded page of orders.

### Cache invalidation after mutations

| Mutation | Refresh or invalidate |
| --- | --- |
| Place order | Buyer order list, order detail and buyer summary |
| Cancel order | Buyer order list, detail and buyer summary |
| Accept or change delivery state | Seller daily list, detail, seller summary; buyer refreshes on next fetch |
| Change product | Seller product list and public product discovery |
| Approve seller | Pending seller queue, seller detail and admin dashboard |
| Disable user | User detail, lists and affected session state |
| Admin order correction | Order detail, order list and summaries |

## 19. Loading, empty and error UX

- Initial page load uses a skeleton matching the final layout.
- Empty states explain what is missing and provide the next valid action.
- Refresh errors preserve safe existing data and show that it may be stale.
- Mutation errors stay near the action or form field that caused them.
- Never show raw stack traces, database text or internal service names.
- Destructive admin actions require confirmation and a reason.
- Accessibility announcements report loading completion, validation errors and mutation results.

## 20. Security rules

### Flutter

- Store refresh tokens only in platform secure storage.
- Clear tokens and private cached data on logout or account disablement.
- Do not embed Admin Gateway URLs or admin secrets.
- Redact phone numbers, addresses, tokens and OTPs from logs and analytics.
- Treat deep-link IDs as untrusted and rely on backend ownership checks.
- Prevent screenshots on sensitive screens only if product policy requires it.

### Next.js admin

- Keep tokens in Secure, HttpOnly, SameSite cookies.
- Use CSRF protection for mutations.
- Use a strict Content Security Policy and frame protection.
- Keep Admin Gateway URL and credentials server-only.
- Validate every Route Handler input.
- Do not rely on hidden buttons, layouts or `proxy.ts` as authorization.
- Never render sensitive tokens or secrets into Client Components.

## 21. Localization, accessibility and device support

### Flutter

- Initial languages: English, Hindi and Punjabi
- All displayed text comes from localization resources
- Large tap targets and clear status labels
- Screen-reader labels for icons and status chips
- Text scaling without clipped totals or buttons
- Android-first performance, with iOS supported from the same codebase

### Admin web

- Responsive desktop-first layout with usable tablet widths
- Keyboard navigation and visible focus states
- Semantic tables, labels and dialogs
- Status is never communicated only by color
- Confirmation dialogs return focus to the triggering action

## 22. Testing strategy

### Flutter tests

- Unit tests for services, repositories, use cases and ViewModels
- Widget tests for loading, empty, data, error and role states
- Router tests for customer, pending seller, approved seller and disabled account
- Contract tests against the generated Public Gateway client
- Integration tests for login, order creation, seller delivery and token refresh
- Offline sync tests for duplicate retries, conflicts and reconnect behavior

### Next.js tests

- Unit tests for validation, error mapping, session helpers and BFF allowlists
- Component tests for tables, filters, forms and confirmation dialogs
- Route Handler tests for cookies, CSRF, refresh and safe header forwarding
- Contract tests against the generated Admin Gateway client
- Browser tests for admin OTP, MFA, seller approval, disablement and order correction
- Accessibility checks for primary admin flows

## 23. Deployment and environments

### Flutter flavors

| Flavor | Public Gateway |
| --- | --- |
| Development | Local or development API |
| Staging | Staging Public Gateway |
| Production | Production Public Gateway |

Build-time configuration contains only public environment values. Secrets are not shipped in the app.

### Next.js environments

| Environment | Admin Web | Admin Gateway |
| --- | --- | --- |
| Development | Local admin site | Development admin API |
| Staging | Staging admin site | Staging admin API |
| Production | Production admin site | Production admin API |

Admin Gateway URLs remain server-only environment variables. Production admin web and API use separate restricted hostnames.

## 24. Complete acceptance checklist

- Flutter customer login routes only to buyer screens.
- Pending seller login routes only to approval and profile screens.
- Approved seller login routes only to seller screens.
- Flutter contains no admin navigation or Admin Gateway configuration.
- Admin website cannot authenticate a customer or seller as admin.
- Admin session requires OTP and MFA.
- Buyer order creation does not send customer ID, seller ID, unit price or total.
- Seller sees and changes only assigned orders.
- Not-delivered result requires a reason.
- Offline seller delivery is visibly pending until the backend confirms it.
- Buyer, seller and admin display the same server order status.
- Buyer and seller totals count only delivered orders and match backend responses.
- Token refresh happens once for concurrent failures.
- Logout and account disablement clear frontend session state.
- Admin mutations require confirmation, reason where required and backend success.
- BFF does not contain business authorization rules.
- Every primary screen has loading, empty, error and retry behavior.
- Generated clients match both backend OpenAPI specifications.

## 25. Not included yet

- Online payments, formal invoices, refunds and seller payouts
- Complaint and dispute screens
- Subscription calendar, pause and skip flows
- Live location tracking and delivery proof images
- Quality-test reports and ratings
- Buyer and seller role switching inside one account

## 26. Architecture references

- [Next.js App Router](https://nextjs.org/docs/app)
- [Next.js Backend for Frontend guide](https://nextjs.org/docs/app/guides/backend-for-frontend)
- [Next.js authentication guide](https://nextjs.org/docs/app/guides/authentication)
- [Flutter app architecture guide](https://docs.flutter.dev/app-architecture/guide)
- [Flutter architecture recommendations](https://docs.flutter.dev/app-architecture/recommendations)
- [Flutter go_router package](https://pub.dev/packages/go_router)
- [Riverpod provider testing](https://riverpod.dev/docs/how_to/testing)

