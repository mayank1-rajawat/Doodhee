# Show OTPs while testing

The root `.env` now contains:

```env
APP_ENV=development
TESTING_MODE=true
```

After requesting a login or registration code, the phone OTP appears in a **Testing OTP** panel on both the buyer/seller mobile app and the admin website. Enter that code into the normal verification field. It expires after five minutes. Admin login then continues to the authenticator step.

The setting is enabled in this workspace now. New setup templates default it to `false`.

## Use the updated app

- Refresh the admin website at `http://127.0.0.1:3000/login`.
- Install `artifacts/android/doodhee-wifi-v1.1.1-arm64.apk` (9.30 MB).
- The mobile login screen identifies this build as **Wi-Fi test 1.1.1**.
- These APKs use `http://192.168.29.127:18080`. Keep the phone and computer on the same Wi-Fi.

Older APKs can still log in but do not render the new testing OTP panel.

## Change the setting

Set `TESTING_MODE=false` to hide OTPs on new requests. The current local gateways are running as user-managed transient services. Reload them after editing `.env`:

```bash
systemctl --user restart doodhee-public-gateway doodhee-admin-gateway doodhee-public-wifi
```

With the updated UI installed, future flag changes require only a backend restart; there is no separate Next.js environment variable or Flutter build flag. Start a new OTP request to apply the changed setting on screen.

These services remain independent of the chat command sessions. They are transient and do not configure startup after a reboot. After a reboot, the existing `make backend` and `LAN_IP=192.168.29.127 make public-lan` commands can be run in separate terminals. To switch the currently running services to those foreground commands, stop the services first:

```bash
systemctl --user stop doodhee-public-gateway doodhee-admin-gateway doodhee-public-wifi
```

## How it works

The Go backend reads the setting on startup. When enabled in `development` or `test`, an OTP request response includes an optional `data.testing_otp`. Both frontends display it only when a valid six-digit value is present. When disabled or unset, the field is omitted and the panel stays hidden.

The server refuses `TESTING_MODE=true` in production. OTP verification, expiry, request/attempt limits, role/enabled-state checks and admin MFA continue through their normal flow. Milkman registration requires no approval. The OTP remains hashed in PostgreSQL; the UI stores it only in memory and clears it when changing phone or leaving the OTP stage. Auth responses use `Cache-Control: no-store`.

This setting controls display only. It does not configure SMS delivery; the development provider still writes OTPs to local logs.

## Verified

- Go regression and race tests plus `go vet` pass, including the full endpoint suite, OTP exposure on/off for all three roles, actual displayed-code verification, admin MFA and the production startup guard.
- All 15 Flutter app tests and 2 storage tests pass, including buyer/Milkman display, exact-phone customer lookup, hidden-by-default behavior, profile/address UI, workflow screens, leading-zero preservation, manual entry and clearing the panel.
- Next.js type checks, lint, unit test and production build pass.
- Browser tests pass for enabled/disabled responses, clearing the panel and continuing to MFA. A real browser request through the admin BFF displayed the actual backend testing code.
- Live local buyer and Wi-Fi seller API requests returned testing OTPs. All three synthetic challenges used by live checks were removed afterward.
- The latest APK builds and passes signature, ZIP integrity and alignment checks. Its signing certificate matches the previous app, so it installs as an update.

Verification records: `artifacts/testing-otp/live-checks.json` and `artifacts/testing-otp/apk-verification.json`. Mobile tests are widget/network/storage tests; no Android device is attached for automated installation testing.
