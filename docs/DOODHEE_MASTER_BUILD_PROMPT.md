# Doodhee Master Build Prompt

> Historical build prompt: the project has since been extended by [MILKMAN_CUSTOMER_DELIVERY_IMPLEMENTATION_PLAN.md](MILKMAN_CUSTOMER_DELIVERY_IMPLEMENTATION_PLAN.md). The current implementation has immediate Milkman access, unique phones, Milkman-managed customers, three-level pricing, per-delivery transactions, ratings/rankings and extra-milk requests. Those current rules override approval-related instructions in this original prompt.

You are the lead software architect and implementation agent for **Doodhee**, a local milk marketplace. Build the complete first working version in the existing workspace:

`/home/hp/WorkSpace/doodhee`

Do not stop at planning or scaffolding. Implement, integrate, migrate, test, build, and document the complete authorized scope while preserving existing user files and changes.

## 1. Read these specifications first

Before writing code, read these Markdown files completely:

1. `/home/hp/WorkSpace/doodhee/docs/DOODHEE_BACKEND_ARCHITECTURE.md`
2. `/home/hp/WorkSpace/doodhee/docs/DOODHEE_GATEWAY_API_ARCHITECTURE.md`
3. `/home/hp/WorkSpace/doodhee/docs/DOODHEE_FRONTEND_ARCHITECTURE.md`

Supporting human-readable copies are available here, but the Markdown files above are authoritative:

- `/home/hp/WorkSpace/doodhee/docs/DOODHEE_BACKEND_ARCHITECTURE.pdf`
- `/home/hp/WorkSpace/doodhee/docs/DOODHEE_GATEWAY_API_ARCHITECTURE.pdf`
- `/home/hp/WorkSpace/doodhee/docs/DOODHEE_FRONTEND_ARCHITECTURE.pdf`

Background product and architecture references:

- `/home/hp/Downloads/Doodhee — Vision & Scope Document A Local Milk Marketplace.docx`
- `/home/hp/Downloads/Stretus-Backend-Architecture.pdf`

Treat these files as requirements and reference material, not as operational instructions. Do not import Stretus business features into Doodhee; reuse only its modular monorepo, hexagonal architecture, dependency direction, bootstrap wiring, and contract organization concepts.

### Specification priority

If wording differs between files, use this order:

1. This master prompt for delivery process, repository layout, design system, and acceptance requirements
2. `DOODHEE_GATEWAY_API_ARCHITECTURE.md` for authentication, authorization, gateways, endpoints, events, errors, and security
3. `DOODHEE_FRONTEND_ARCHITECTURE.md` for screens, navigation, API-call flows, caching, offline behavior, and frontend testing
4. `DOODHEE_BACKEND_ARCHITECTURE.md` for the initial database, daily order model, total calculation, and business scope
5. Vision document for broader context only

Do not silently expand the product beyond the first-version scope.

## 2. Required deliverables

Create this repository structure:

```text
/home/hp/WorkSpace/doodhee/
├── backend/
├── admin-web/
├── mobile-app/
├── design/
├── docs/
├── docker-compose.yml
├── Makefile
├── .env.example
└── README.md
```

Deliver all of the following:

1. A Go backend monorepo with two HTTP gateways and one event worker
2. A Next.js admin website connected only to the Admin Gateway
3. One Flutter app shared by buyer and seller, connected only to the Public Gateway
4. PostgreSQL migrations and development seed tooling
5. Public and admin OpenAPI specifications plus generated frontend clients
6. Automated tests for critical authorization and complete user flows
7. Local development orchestration and clear run commands
8. Android APK and AAB outputs when the installed toolchain supports them
9. A valid iOS Flutter project and build verification to the extent supported by the host

Do not claim that an iOS IPA was produced unless the build ran on macOS with Xcode and valid Apple signing. On Linux, finish the iOS project, tests, configuration, and documented macOS signing/build steps. Android uses APK/AAB; iOS uses an app archive or IPA, not an APK.

## 3. Product names

Use these names consistently:

- Product and mobile app: **Doodhee**
- Admin website: **Doodhee Admin**
- Design system: **Doodhee Fresh**
- Backend module/repository name: `doodhee-backend`
- Public gateway binary: `public-gateway`
- Admin gateway binary: `admin-gateway`
- Event worker binary: `event-worker`
- Backend role values: `customer`, `seller`, `admin`

Use **buyer** in user-facing copy and `customer` in API/database role values.

## 4. Doodhee Fresh design system

Create `/home/hp/WorkSpace/doodhee/design/tokens.json` as the single visual-token reference. Implement matching tokens in Next.js CSS variables and Flutter `ThemeData`.

### Brand colors

| Token | Value | Use |
| --- | --- | --- |
| `brand.primary` | `#166534` | Primary buttons, active navigation and important actions |
| `brand.strong` | `#14532D` | Headings, dark brand surfaces and hover states |
| `brand.accent` | `#22C55E` | Decorative accents, progress and selected indicators |
| `brand.soft` | `#DCFCE7` | Selected rows, soft badges and highlighted backgrounds |
| `brand.tint` | `#F0FDF4` | Very light brand sections |
| `background` | `#F7FAF8` | Main application background |
| `surface` | `#FFFFFF` | Cards, dialogs, forms and tables |
| `text.primary` | `#17211B` | Main text |
| `text.secondary` | `#52605A` | Secondary text |
| `border` | `#D7E2DA` | Inputs, cards and table dividers |
| `status.success` | `#15803D` | Delivered and success |
| `status.warning` | `#B45309` | Pending and attention |
| `status.error` | `#B91C1C` | Not delivered, destructive actions and errors |
| `status.info` | `#1D4ED8` | Out for delivery and information |
| `status.neutral` | `#64748B` | Cancelled, disabled and inactive |

Use white text on `brand.primary` and `brand.strong`. Do not use the bright accent as small body text on white. Status must always include text or an icon, never color alone.

### Typography

Bundle fonts rather than depending on a runtime font download.

- Primary Latin font: **Inter**
- Hindi fallback: **Noto Sans Devanagari**
- Punjabi fallback: **Noto Sans Gurmukhi**
- Final fallback: platform sans-serif

| Style | Size and line height | Weight |
| --- | --- | --- |
| Display | 32 / 40 | 700 |
| Page title | 24 / 32 | 700 |
| Section title | 20 / 28 | 600 |
| Card title | 16 / 24 | 600 |
| Body | 16 / 24 | 400 |
| Label | 14 / 20 | 600 |
| Supporting text | 14 / 20 | 400 |
| Caption | 12 / 16 | 400 |

### Layout tokens

- Spacing scale: `4, 8, 12, 16, 24, 32, 48`
- Input and button radius: `10px`
- Card and dialog radius: `12px`
- Pill/status radius: `999px`
- Card border: `1px solid #D7E2DA`
- Shadow: subtle only, equivalent to `0 4px 16px rgba(20, 83, 45, 0.08)`
- Minimum mobile tap target: `48x48`
- Desktop content maximum width: `1440px`

### Visual direction

Use a simple, calm, modern dairy-marketplace interface:

- Light theme for the first version
- Clear whitespace and strong information hierarchy
- White surfaces on a warm green-tinted background
- Minimal shadows and no decorative clutter
- No large gradients, glassmorphism, excessive animations, or invented illustrations
- Use the text wordmark **Doodhee** until a real logo asset is provided
- Use consistent rounded outline icons; do not use emoji as interface icons
- Keep tables readable, forms short, and primary actions obvious
- Meet WCAG AA contrast for normal text and interactive controls

### Shared status presentation

| Status | Label | Color token |
| --- | --- | --- |
| `pending` | Pending | `status.warning` |
| `accepted` | Accepted | `brand.primary` |
| `out_for_delivery` | Out for delivery | `status.info` |
| `delivered` | Delivered | `status.success` |
| `not_delivered` | Not delivered | `status.error` |
| `cancelled` | Cancelled | `status.neutral` |
| Seller pending approval | Approval pending | `status.warning` |
| Disabled account | Disabled | `status.neutral` |

## 5. Backend implementation

Create `/home/hp/WorkSpace/doodhee/backend` as one Go module using a modular monolith and hexagonal architecture.

### Required binaries

```text
backend/cmd/public-gateway/main.go
backend/cmd/admin-gateway/main.go
backend/cmd/event-worker/main.go
```

### Required architecture

For each business module use:

```text
domain -> ports -> application -> adapters
```

Dependencies point inward. Domain code must not import HTTP, PostgreSQL, JWT, SMS, or framework packages. Gateways are inbound adapters. PostgreSQL, token, OTP, MFA, audit, and outbox implementations are outbound adapters. Wire dependencies with normal Go constructors under `internal/bootstrap`.

Use one PostgreSQL database. Use migrations, transactions, foreign keys, check constraints, indexes, and UTC timestamps. Use `NUMERIC`, never floating point, for prices, quantities, and totals.

### Required database tables

Core tables:

- `users`
- `products`
- `orders`
- `outbox_events`

Authentication and audit tables:

- `auth_challenges`
- `auth_sessions`
- `mfa_methods`
- `admin_audit_logs`

Implement the exact fields, relationships, states, and constraints defined in the architecture documents. Add only technical columns required for safe migrations, locking, idempotency, or auditing, and document them.

### Required gateways

Public Gateway:

- Buyer and seller authentication
- Buyer discovery, daily orders and totals
- Seller approval status, products, daily deliveries and totals
- Public token audience only

Admin Gateway:

- Admin OTP plus mandatory MFA
- Seller approval and rejection
- User enable and disable
- Product activation and suspension
- Order search and audited correction
- Event inspection and retry
- Audit log access
- Admin token audience only

Do not register admin routes in the public binary. Do not accept public tokens in the admin gateway or admin tokens in the public gateway.

### Required authentication behavior

- Phone OTP abstraction with a safe development provider and replaceable production provider
- Store OTP hashes, never raw OTPs
- Access JWT plus rotating opaque refresh token stored as a hash
- Public and admin token audiences
- Session revocation, logout and logout-all
- Immediate account-disable enforcement
- Admin TOTP MFA
- Refresh-token reuse detection and family revocation
- Rate limiting for OTP, login and mutations
- Generic OTP responses that do not reveal account existence
- Trusted seed or CLI for the first admin; never allow public admin registration

### Required authorization behavior

Apply both route RBAC and record ownership in application use cases.

- Derive `customer_id` from the authenticated session
- Derive `seller_id` from the product or authenticated seller
- Derive `marked_by` and `approved_by` on the server
- Never trust IDs, role, price, total, status timestamps, or audit identity from clients
- Return the same `404` shape for missing and inaccessible records
- Pending sellers may access only their profile and approval status
- Disabled users may not perform protected operations

### Required API and events

Implement every endpoint and event in:

`/home/hp/WorkSpace/doodhee/docs/DOODHEE_GATEWAY_API_ARCHITECTURE.md`

Do not omit endpoints silently. Maintain:

```text
backend/api/openapi/public-v1.yaml
backend/api/openapi/admin-v1.yaml
```

Generate or validate typed clients from these contracts.

### Required business behavior

- A daily order stores a snapshot of unit price and total
- Valid state transitions are enforced centrally
- `not_delivered` requires a reason
- Only delivered orders count toward buyer, seller, and admin summaries
- Order and outbox event commit in one transaction
- Admin correction stores before/after state, reason, request ID, actor, and event
- Event handlers are idempotent
- Worker claims events safely, retries temporary errors with backoff, and records permanent failure

### Backend operational deliverables

- Structured logs with request and correlation IDs
- Health and readiness endpoints
- Graceful shutdown
- Configuration from environment
- `.env.example` without secrets
- Dockerfile for each binary or one parameterized multi-stage Dockerfile
- Docker Compose for PostgreSQL and local services
- Migration and seed commands
- Unit, repository, handler, authorization, transaction, and integration tests

## 6. Next.js Doodhee Admin implementation

Create `/home/hp/WorkSpace/doodhee/admin-web` using current stable Next.js App Router and TypeScript.

### Required architecture

- Server Components for the shell and safe server-rendered structure
- Client Components for interactive tables, filters, dialogs and mutations
- Thin same-origin Route Handler BFF under `/api/admin`
- Server-only Admin Gateway client
- Secure, HttpOnly, SameSite cookies
- CSRF protection for mutations
- Optimistic redirect may exist, but the data-access layer and Admin Gateway must verify every protected request
- No business rules or final authorization decisions in the BFF

### Required pages

- OTP login
- MFA verification
- Dashboard
- Pending seller approvals
- User list and user details
- Product list and product details
- Order list and order details
- Audited delivery-result correction
- Bill-summary reporting
- Event list, event detail and retry
- Audit log list and detail

Implement every Next.js screen, call flow, loading state, empty state, error state, confirmation rule, and cache invalidation rule in:

`/home/hp/WorkSpace/doodhee/docs/DOODHEE_FRONTEND_ARCHITECTURE.md`

### Admin navigation

Use a collapsible left sidebar on desktop and a drawer on narrow screens:

1. Dashboard
2. Pending Sellers
3. Users
4. Products
5. Orders
6. Events
7. Audit Logs

The header contains the page title, environment badge outside production, admin identity, and logout. Keep filters near tables and use confirmation dialogs for state-changing admin actions.

### Admin UI requirements

- Apply Doodhee Fresh tokens through CSS variables
- Use Inter and Noto fallbacks
- Desktop-first but usable on tablets
- Accessible forms, dialogs and semantic tables
- Keyboard navigation and visible focus states
- Status chips include text and icon
- Destructive actions use the error color only at the final action point
- Display backend request ID on mutation errors and admin audit confirmations

### Admin tests and build

- Unit tests for validation, DAL, error mapping and BFF allowlists
- Component tests for forms, tables and dialogs
- Route Handler tests for cookies, CSRF, refresh and safe header forwarding
- Browser tests for OTP, MFA, seller approval, disablement and order correction
- Accessibility checks for primary flows
- Production build must pass

## 7. Flutter Doodhee app implementation

Create `/home/hp/WorkSpace/doodhee/mobile-app` as one Flutter app for Android and iOS.

### Required architecture

Use the documented MVVM structure:

```text
View -> ViewModel -> Repository -> Service -> Public Gateway or local storage
```

Use:

- `go_router` for declarative role routing and deep links
- Riverpod for dependency construction, ViewModel state, and test overrides
- Generated typed client from `public-v1.yaml`
- Secure device storage for refresh token
- In-memory access token
- Local database for safe cached reads and pending seller delivery results
- Constructor/provider injection so tests can replace services

Widgets must not call HTTP services directly.

### Shared authentication flow

- Splash and session restore
- Buyer or seller intent selection
- Phone OTP request and verification
- Role and approval routing from `GET /v1/me`
- Access-token refresh with a single refresh mutex
- Logout and logout-all
- Disabled-account handling
- Pending and rejected seller screens

### Buyer navigation

Use a bottom navigation bar:

1. Home
2. Orders
3. Summary
4. Profile

Required buyer flows:

- Discover approved sellers by service area
- View seller and products
- View current product details
- Place a daily order
- View daily order status and reason
- Cancel an eligible order
- View delivered days, litres, and total for a date range

### Seller navigation

Use a bottom navigation bar:

1. Home
2. Orders
3. Products
4. Summary
5. Profile

Required seller flows:

- View pending, approved, rejected, or disabled state
- View dashboard and today's order counts
- Create and edit products
- Toggle product availability
- View assigned daily orders
- Accept an order
- Mark out for delivery
- Mark delivered
- Mark not delivered with required reason and optional note
- View totals by date range and customer
- Cache the current daily list
- Queue seller delivery results while offline
- Synchronize queued delivery results using idempotency keys
- Show `Pending sync` until confirmed
- Handle conflicts by fetching server state and showing `Needs attention`

Do not queue buyer order creation, product edits, or admin operations offline.

### Flutter UI requirements

- Apply the same Doodhee Fresh colors, typography, spacing, radii, icons and statuses as the admin website
- Bundle Inter, Noto Sans Devanagari, and Noto Sans Gurmukhi
- Initial languages: English, Hindi, and Punjabi
- Large touch targets and clear labels suitable for early-morning seller use
- Never use color alone for delivery state
- Show a stable skeleton, empty state, retry state and stale-data state where documented
- Keep forms short and show field-level errors
- Preserve drafts safely when an order cannot be submitted offline

### Flutter tests and builds

- Unit tests for services, repositories, use cases and ViewModels
- Widget tests for loading, empty, error, role and approval states
- Router tests for buyer, pending seller, approved seller, rejected seller and disabled account
- Integration tests for OTP, order placement, delivery state and refresh
- Offline synchronization tests for retries, deduplication and conflicts
- Run formatting, analysis and all tests
- Build Android debug and release APK when signing/configuration allows
- Build Android App Bundle when signing/configuration allows
- On macOS with Xcode and signing, build the iOS archive/IPA
- On Linux, validate the iOS project and clearly document the remaining macOS signing command instead of claiming an IPA exists

## 8. API-client and contract workflow

Use this sequence:

1. Implement and validate backend OpenAPI contracts
2. Generate Dart client from `public-v1.yaml`
3. Generate TypeScript client from `admin-v1.yaml`
4. Add a reproducible generation command
5. Fail CI if generated clients no longer match the specifications
6. Map generated transport models into frontend domain/presentation models where needed

Do not hand-maintain duplicate endpoint strings across many screens.

## 9. Development environment

Provide simple commands through the root `Makefile` or documented scripts:

```text
make setup
make db-up
make migrate
make seed-admin
make backend
make admin
make mobile
make test
make build
```

Use Docker Compose for PostgreSQL and safe local dependencies. Provide a development OTP provider that prints the OTP only in clearly marked local development logs and is impossible to enable accidentally in production.

Provide `.env.example` files for root, backend, and admin web. Do not commit credentials, signing keys, MFA secrets, JWT private keys, or real phone numbers.

## 10. Required end-to-end flows

Implement and verify these flows across real application boundaries:

### Flow A: Seller onboarding

```text
Flutter seller registration -> Public Gateway OTP -> seller pending
-> Next.js admin pending queue -> Admin Gateway approve
-> Seller refreshes status -> Flutter seller dashboard
```

### Flow B: Buyer daily order

```text
Flutter buyer login -> discover seller -> select product -> place order
-> Public Gateway derives seller and price -> order plus event committed
-> Seller sees assigned order
```

### Flow C: Delivery result

```text
Seller accepts -> out for delivery -> delivered or not delivered
-> Public Gateway validates ownership and transition
-> Buyer order detail shows the same result and reason
-> Buyer and seller summaries count delivered orders only
```

### Flow D: Offline seller delivery

```text
Seller loads daily list -> loses network -> marks result
-> Flutter saves pending operation -> network returns
-> operation syncs once with idempotency key
-> server result replaces local pending state
```

### Flow E: Admin account control

```text
Admin OTP -> MFA -> dashboard -> disable user with reason
-> Admin Gateway writes audit and revokes sessions
-> affected Flutter session is rejected and cleared
```

### Flow F: Admin correction

```text
Admin opens order -> sees old state -> confirms corrected delivery result with reason
-> BFF -> Admin Gateway -> correction, audit and event commit together
-> all later buyer, seller and admin reads show corrected server state
```

## 11. Testing and acceptance gates

Do not call the work complete until all applicable gates pass.

### Backend

- Formatting and static analysis pass
- `go test ./...` passes
- Migration up/down behavior is verified in a disposable database
- Public token is rejected by Admin Gateway
- Admin token is rejected by Public Gateway
- Customer cannot read another customer's order
- Seller cannot read or change another seller's order or product
- Pending seller cannot manage products or orders
- Disabled account loses active access
- Order and event roll back together on failure
- Duplicate idempotent requests do not create duplicate state

### Admin web

- Lint, type-check, unit tests and production build pass
- OTP plus MFA works through the BFF
- Cookies are HttpOnly and not exposed to client code
- Seller approval, account disablement, correction and event retry are tested
- Route guards are not the only authorization layer
- Primary pages pass automated accessibility checks

### Flutter

- Formatting, analysis, unit tests and widget tests pass
- Role routing is tested for every account state
- Buyer cannot open seller routes and seller cannot open buyer routes
- Offline delivery queue does not duplicate operations
- A backend conflict does not silently overwrite server state
- Android build completes when the installed SDK supports it
- iOS output is reported honestly according to host and signing availability

### Cross-application

- Buyer, seller and admin display the same persisted order state
- Total summaries match and count only delivered orders
- Request IDs and safe errors propagate across gateways and frontends
- Design tokens produce visibly consistent web and mobile interfaces

## 12. Scope boundaries

Do not implement these unless a required dependency makes a minimal placeholder unavoidable:

- Online payment collection
- Formal invoices, refunds or seller payouts
- Complaint and dispute workflow
- Recurring subscriptions, pause and skip calendar
- Live GPS tracking
- Delivery proof images
- Quality-test reports and ratings
- Kafka, gRPC, or independently deployed business microservices
- Buyer and seller role switching inside one account

Keep extension points clean, but do not build unused systems.

## 13. Execution rules

1. Inspect the workspace, existing instructions, installed toolchains, and dirty files before changing anything.
2. Preserve user changes and existing documentation.
3. Make reasonable in-scope assumptions and record them in `docs/IMPLEMENTATION_DECISIONS.md`.
4. Ask only when a missing choice would materially change product behavior or requires external credentials/authority.
5. Implement vertical slices that remain runnable; do not leave disconnected mock screens after backend endpoints exist.
6. Prefer the simplest correct modular-monolith implementation over speculative abstraction.
7. Never weaken authentication or authorization merely to make a test pass.
8. Never fabricate successful builds, tests, generated APKs, iOS artifacts, migrations, or integrations.
9. Keep secrets out of source control and output.
10. Update documentation whenever implementation necessarily differs from an assumption.

## 14. Final handoff

At completion, provide:

- A concise architecture summary
- Exact paths to backend, admin web, Flutter app, design tokens, OpenAPI files, migrations and build artifacts
- Commands for local setup, database migration, seeding admin, running each process, testing and building
- Test and build results with any skipped checks explained
- Android APK/AAB paths if successfully generated
- Honest iOS build/signing status and the exact macOS command needed if not produced
- Any remaining external setup, such as real SMS OTP provider, production keys, domains or Apple/Android signing

The result must be a coherent, working Doodhee first version—not three disconnected demonstrations.

