# Doodhee build and verification status

Implemented on 14 November 2026 in `/home/hp/WorkSpace/doodhee`.

## Delivered applications

| Project | Delivery |
| --- | --- |
| `backend` | Go Public Gateway, Admin Gateway, event worker, migration and admin-seed CLIs |
| `admin-web` | Next.js admin site with OTP/MFA, protected BFF, users, products, delivery pricing details, orders, totals, events and audit views |
| `mobile-app` | Flutter Android/iOS project with separate login/registration, multiple buyer addresses, Milkman customer/search/delivery tools, three-level pricing, extra-milk requests, ratings/rankings, totals and offline delivery queue |
| `design` | Doodhee Fresh tokens, canonical delivery icon, and Inter/Noto font assets bundled in both frontends |

## Verified

- Testing OTP display is implemented and enabled locally with `TESTING_MODE=true`. Both frontends show the optional backend code. Production rejects the flag; admin MFA remains required. Version 1.1.1, backend flag-on/off tests, 15 Flutter app tests, 2 storage tests and web build checks pass. See `TESTING_MODE.md`.
- All 74 implemented application API endpoints pass the race-enabled endpoint suite: 500 HTTP checks and 52 additional business/data assertions. This includes exact-phone customer lookup/connection, immediate Milkman access, customer OTP claim, global phone uniqueness, customer isolation, pricing precedence/history, separate deliveries, ratings/rankings, extra-milk requests and actual-delivery billing. See `API_VERIFICATION_REPORT.md`.
- Go unit tests and PostgreSQL integration tests pass.
- Go race detector and `go vet` pass.
- Migration up/down/up passes against the isolated `doodhee_migration_test` database.
- Public/admin token audience separation is tested.
- Milkman/customer ownership, immediate Milkman access, delivery transitions, reason requirements, monetary arithmetic and idempotency are tested.
- OTP replay rejection, admin MFA, refresh-token reuse detection and logout are tested.
- Next.js type checking, ESLint, unit test and production build pass.
- The existing browser flow covers admin OTP/MFA, immediate Milkman access, buyer/Milkman order, delivery, admin correction, matching totals and account disablement; the new business workflows are covered by the isolated real-HTTP/PostgreSQL suite.
- Automated dashboard accessibility check passes with no reported violations.
- Production web dependency audit reports zero known vulnerabilities at verification time.
- Flutter analysis passes with no issues.
- Fifteen Flutter app tests and two storage tests pass, including exact-phone customer lookup, direct Milkman routing, customer/pricing/rating/extra-milk screens, localized labels, login and profile/address UI, testing OTP display/hiding, delivery chips, transport failures, concurrent refresh and offline idempotency.
- Public and admin OpenAPI files validate. Generated clients are checked for drift.
- Android debug APK and debug AAB builds pass.
- The optimized release-mode ARM64 testing APK v1.1.1 builds successfully at 9.30 MB with the launcher and in-app brand icon. The original universal debug APK was 211.57 MB (201.77 MiB).
- The current APK passes signature, ZIP integrity and alignment checks; its signing certificate matches the previous development APK.
- Wi-Fi login diagnosis: the prior APK used the emulator address and the original gateway listened only on loopback. A Public Gateway instance now listens on `192.168.29.127:18080`, the current APK contains that URL, a real OTP request returned HTTP 200, and the user confirmed the health endpoint works in the phone browser. The existing local public/admin gateways continue to respond.
- The same Doodhee doorstep-delivery mark is now used for Android/iOS app icons, the mobile sign-in header, admin branding, browser icon and Apple touch icon.
- APK signature verification passes (v2 signing); package name is `in.doodhee.doodhee`, display name is `Doodhee`, minimum Android API is 24, and target API is 36.

## Build artifacts

| Artifact | Path |
| --- | --- |
| Public Gateway binary | `artifacts/public-gateway` |
| Admin Gateway binary | `artifacts/admin-gateway` |
| Event worker binary | `artifacts/event-worker` |
| Next.js production output | `admin-web/.next` |
| Latest ARM64 testing APK | `artifacts/android/doodhee-wifi-v1.1.1-arm64.apk` |
| Android size measurements / SHA-256 | `artifacts/android/size-report.json` |
| External Dart symbols and R8 mapping | `artifacts/android/symbols/` |
| Android development AAB | `mobile-app/build/app/outputs/bundle/debug/app-debug.aab` |
| Admin dashboard screenshot | `artifacts/admin-dashboard.png` |

The small testing APKs run in release mode with shrinking and native-library compression, explicitly signed with the existing development key. They currently use the same-Wi-Fi Public Gateway URL `http://192.168.29.127:18080`. `android-share` requires an explicit `PUBLIC_API_URL` on future builds. Android extracts the compressed libraries during installation, so installed storage is larger than download size. The original APK and AAB remain debug builds. Store distribution still requires owner signing and a reachable production backend. See `ANDROID_APP_SIZE.md` and `PHONE_TESTING.md`.

The three gateway listeners now run as user-managed transient services: `doodhee-public-gateway`, `doodhee-admin-gateway` and `doodhee-public-wifi`. They load the root `.env` and remain running independently of chat command sessions. The Wi-Fi gateway's development OTP log is `/tmp/doodhee-public-lan-dev.log`. The Next.js preview and development PostgreSQL container remain available. Restart commands are in `TESTING_MODE.md`.

## Remaining external setup and verification

- iOS source exists, but an IPA cannot be built or signed on this Linux host. Use macOS/Xcode and an Apple signing team as described in the README.
- Android store release signing requires the owner's keystore.
- Production SMS OTP and external delivery-notification providers require configuration/adapters. Production startup refuses the development OTP logger.
- Production domains, TLS and deployment secrets must be supplied by the owner.
- A complete real-device/emulator Flutter end-to-end test has not been run. Phone-browser backend reachability and the earlier login path are confirmed; this v1.1.1 feature build still needs installation on the phone. Other mobile verification consists of analysis, unit/widget/storage tests, the isolated backend workflow suite and successful Android packaging.
- Hindi/Punjabi resources cover navigation, status, forms and common actions. Some explanatory copy and backend validation messages remain English.

See `IMPLEMENTATION_DECISIONS.md` for physical package organization and implementation assumptions. Existing architecture PDFs and Word documents remain planning references; source Markdown and this status file describe the implementation.
