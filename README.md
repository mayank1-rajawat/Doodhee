# Doodhee

Local milk marketplace with a Go backend, Next.js admin website, and one Flutter buyer/seller application.

## Projects

| Folder | Application |
| --- | --- |
| `backend` | Public Gateway, Admin Gateway and event worker |
| `admin-web` | Doodhee Admin website and server-side BFF |
| `mobile-app` | Doodhee app for Android and iOS |
| `design/tokens.json` | Doodhee Fresh visual tokens |
| `docs` | Specifications, master prompt and implementation notes |

## Run locally

Requirements: Docker, Node.js, Go, Flutter, and an Android SDK for Android builds. This workspace has Go/Flutter/Java/Android SDK installations under `.tools`; the Makefile uses them when present.

```bash
make setup
make db-up
make migrate
```

Start these in separate terminals:

```bash
make backend
make admin
make mobile
```

Addresses:

- Admin website: `http://127.0.0.1:3000`
- Public Gateway: `http://127.0.0.1:18080`
- Admin Gateway: `http://127.0.0.1:28081`
- PostgreSQL: `127.0.0.1:55432`

Both gateways expose `/health/live` and `/health/ready`.

The Android emulator uses `http://10.0.2.2:18080`. For a physical Android device connected by USB, use:

```bash
.tools/android/platform-tools/adb reverse tcp:18080 tcp:18080
PUBLIC_API_URL=http://127.0.0.1:18080 make mobile
```

Debug Android builds permit local HTTP. The explicitly opted-in `android-share` testing build also permits local HTTP. Production releases stay HTTPS-only.

## First admin

Create a new admin through the trusted CLI. Public registration cannot create admins.

```bash
export ADMIN_PHONE='+countrycode-and-your-number'
export ADMIN_TOTP_SECRET='YOUR_AUTHENTICATOR_BASE32_SECRET'
make seed-admin
```

Enroll the same Base32 secret in your authenticator using issuer `Doodhee`, 6 digits, SHA1 and a 30-second period. The CLI refuses to overwrite an existing account.

Open the admin website and request the phone OTP. With `TESTING_MODE=true` in the root `.env`, the code appears directly on the admin verification screen and in the updated mobile app. Otherwise read it from the gateway's **local development** log. Admin login then asks for the authenticator code. See [testing OTP display](docs/TESTING_MODE.md) for the current setting and restart command. Production startup rejects the development OTP provider; supply a production OTP adapter before deploying.

Buyer and Milkman signup happens inside the Flutter app. Every enabled Milkman can start immediately; admin approval is not required. Mobile numbers are unique across all account types. Buyers can manage multiple delivery addresses from Profile.

## First working flow

1. Register a Milkman in Flutter and add a milk product with its default price.
2. Add customers by name, address and unique mobile number. Their first OTP sign-in or new-account attempt claims the same customer record automatically.
3. Optionally set a customer-specific product price, then search the customer and record each daily delivery separately.
4. A buyer may also place an upcoming order and request extra milk. The Milkman approves or rejects that request.
5. At completion, the Milkman records the actual quantity and may set a one-delivery price.
6. Bills include delivered rows only and use each delivery's saved quantity and price snapshot.
7. Buyers can rate each Milkman once and later edit that same rating; discovery shows rating and rank.
8. Admin can enable/disable accounts, suspend products and correct delivery results with an audited reason.

## Verify

```bash
make test
node scripts/check-contracts.mjs
PUBLIC_API_URL=https://your-public-api make build
```

`make test` runs database-backed Go integration tests against the configured development database, along with web type/lint/tests and Flutter analysis/tests. Test records use synthetic identities.

To verify every implemented API endpoint with isolated test data, run `make test-api`. It creates and removes its own temporary PostgreSQL schema and writes endpoint results under `artifacts/api-verification/final/`. See [the API verification report](docs/API_VERIFICATION_REPORT.md) for all 74 endpoints and 552 passing checks/assertions.

The browser test requires the three backend processes and Next.js development server. It seeds its own synthetic admin, reads development OTP logs at `/tmp/doodhee-admin-dev.log` and `/tmp/doodhee-public-dev.log`, and uses installed Google Chrome:

```bash
cd admin-web
node tests/e2e.mjs
```

To create those logs, run the public/admin commands with their stdout redirected to the matching `/tmp` paths. Browser checks cover login/MFA, accessibility, immediate Milkman access, buyer order, delivery, correction, totals and disablement.

## API contracts

```bash
make contracts
```

- `backend/api/openapi/public-v1.yaml`
- `backend/api/openapi/admin-v1.yaml`
- `admin-web/generated/admin-api/`
- `mobile-app/lib/generated/public_api/`

The `.yaml` files use JSON syntax, which is valid YAML. HTTP input validation and generation share the same request-field definitions.

## Mobile builds

Create small APKs for sharing with testers:

```bash
PUBLIC_API_URL=https://your-public-api make android-share
```

Outputs:

- Current phone build: `artifacts/android/doodhee-wifi-v1.1.1-arm64.apk` (9.30 MB).

The workspace currently keeps only this latest ARM64 APK. It is a complete app. Its size was measured on 14 November 2026 and can change with future code.

These builds use release optimization, resource shrinking, compressed native libraries and the existing **development signing key**. Compression reduces download size; Android extracts native libraries during installation. Normal production packaging is unchanged. Debug symbols and the Android R8 mapping are saved outside the APKs in `artifacts/android/symbols/`.

`android-share` requires an explicit backend URL to prevent accidentally distributing an emulator-only build. The current APK was rebuilt for same-Wi-Fi testing at `http://192.168.29.127:18080`. Your phone successfully opened that server's health check. Keep the computer, Public Gateway and phone connected to the same Wi-Fi. See [phone testing](docs/PHONE_TESTING.md) for restart commands and development OTP instructions.

For a hosted backend, rebuild with its actual URL:

```bash
PUBLIC_API_URL=https://your-public-api make android-share
```

For development with debugging enabled, use `make android-debug` (ARM64). The old universal `app-debug.apk` is about 212 MB; share the new testing APK instead. `make build` also generates the small testing APKs and therefore requires `PUBLIC_API_URL`.

For an Android store release, configure a release signing keystore, then run `flutter build appbundle --release --dart-define=PUBLIC_API_URL=https://your-public-api`.

On macOS with Flutter and Xcode:

```bash
cd mobile-app
flutter pub get
open ios/Runner.xcworkspace
# Select your Apple team and signing profile in Xcode.
flutter build ipa --release \
  --dart-define=PUBLIC_API_URL=https://your-public-api
```

This Linux workspace cannot generate or sign an iOS IPA.

See `docs/ANDROID_APP_SIZE.md`, `docs/IMPLEMENTATION_DECISIONS.md` and `docs/BUILD_STATUS.md` for measurements, implementation choices, verification results and remaining setup.
