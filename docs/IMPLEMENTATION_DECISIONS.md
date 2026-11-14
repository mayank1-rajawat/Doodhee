# Doodhee implementation decisions

The workspace contains three application projects: `backend`, `admin-web`, and `mobile-app`. Existing planning documents are preserved.

## Architecture

- The Go application has separate public gateway, admin gateway, and worker binaries. Shared domain rules, application use cases, ports, PostgreSQL, security, and HTTP adapters implement the modular monolith.
- Small first-version use cases share `internal/domain`, `internal/ports`, and `internal/application`. They are grouped into identity, marketplace, reporting, and administration methods. The physical per-feature package split shown in the planning tree is deferred; business rules still have no HTTP or database dependencies.
- Core entities are represented as transport-neutral records. Decimal arithmetic uses `math/big.Rat`; PostgreSQL uses fixed-precision `NUMERIC`. OpenAPI generates typed transport model accessors and operation catalogs for both frontends.
- Fifteen tables are implemented across three migrations. The Milkman delivery additions are `milkman_customers`, `customer_product_prices`, `extra_milk_requests` and `milkman_ratings`; `orders` remains the immutable per-delivery price/quantity ledger.
- Product `admin_suspended` is separate from seller-controlled `is_available`, preventing sellers from reversing an admin suspension.
- Outbox processing creates internal notifications. SMS/push delivery of business notifications requires a provider adapter later; the current screens refresh order records directly.

## Local environment

- Go and Flutter were absent. Local SDKs are installed under the ignored `.tools` directory.
- Java and Android SDK tools are also installed locally for the Android build attempt.
- Public Gateway uses `127.0.0.1:18080`; Admin Gateway uses `127.0.0.1:28081` because several common ports belong to existing projects.
- The PostgreSQL development container is `doodhee-dev-db`, listening on local port `55432`. Existing containers and databases were not modified.
- `make setup` generates a random local authentication key in ignored `.env`, preserving existing values.
- Development OTP codes appear only in the local gateway logs. Production startup refuses the development OTP adapter.
- Admin MFA uses TOTP. The trusted seed command requires a phone and an authenticator secret supplied through environment variables.

## Frontend

- Doodhee Fresh colors and bundled Inter/Noto fonts are shared across Next.js and Flutter.
- Next.js uses an allowlisted BFF with HttpOnly cookies, origin-based CSRF checks, token rotation, and gateway authorization. Browser code cannot read backend tokens.
- The Flutter app uses MVVM, Riverpod dependency providers, GoRouter role redirects, secure token storage, and SQLite caching/offline delivery queue.
- Initial language resources include English, Hindi, and Punjabi for navigation and core business labels. Some explanatory text and server validation messages remain English.
- Buyer order creation and Milkman customer/direct-delivery management are online. Milkman delivery results can queue after the daily list is cached. A secure, previously verified profile may restore a clearly marked offline view for up to 24 hours. It grants no server permissions; every synced operation must pass current backend session, role, enabled-state and ownership checks.
- Milkmen do not require approval. The retained `approval_status` field and `/v1/seller/approval-status` response exist only for compatibility; all seller registrations are created approved, and authorization uses role, enabled-state and ownership.
- Product default, customer-product and one-delivery prices resolve in that order of precedence. Every order snapshots `unit_price`, `price_source`, quantity and total so later price edits cannot alter delivered history.

## Build and external setup

- One backend `TESTING_MODE` setting controls phone OTP display in both frontends. It defaults off, is enabled locally on request, and is rejected in production. The optional `testing_otp` response field is never persisted as plaintext in the database and does not bypass verification or admin MFA. See `TESTING_MODE.md`.
- Android development builds use the generated debug keystore. Store-distribution signing requires the owner's release keystore.
- `make android-share` builds release-mode APKs for ARM64 and ARM32 separately, enables R8/resource shrinking, stores symbols externally and opts into compressed native libraries. The `localTestSigning` flag explicitly enables development signing; normal production builds keep their signing and native packaging defaults. Local HTTP also requires an explicit flag and local testing signing.
- Desktop SQLite regression tests live in `mobile-app/tooling/storage_tests`; the real mobile app retains platform SQLite and its offline queue. This isolates host-test native build hooks from mobile dependencies. No buyer/seller feature or English/Hindi/Punjabi font was removed for size reduction.
- Sharing APKs require an explicit `PUBLIC_API_URL`; emulator defaults are retained only for development commands. `make public-lan` runs the existing Public Gateway on an explicitly supplied Wi-Fi address, sharing the same database and authorization rules while preserving the user's local gateways. The current APKs use `http://192.168.29.127:18080`.
- Mobile transport failures have readable messages and remain distinct from backend HTTP rejections. The timeout covers both connection/headers and response body, and failed OTP requests are not automatically retried. Offline queues and cached session restoration retain their existing behavior.
- iOS sources are generated. Building an IPA requires macOS, Xcode, an Apple team, and signing profiles.
- Production SMS, external notification providers, production domains/TLS, signing assets, and secrets are not supplied by this implementation.
