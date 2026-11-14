# Doodhee mobile app

One Flutter app for buyers and sellers, targeting Android and iOS.

Run these commands from the workspace root:

```bash
make mobile          # Run in development
PUBLIC_API_URL=https://your-public-api make android-share
make android-debug   # ARM64 debug APK
```

The current sharing APK is `artifacts/android/doodhee-wifi-v1.0.4-arm64.apk`
(9.29 MB). Installed storage is larger than the compressed APK download.

The current version is 1.0.4. It uses the Doodhee doorstep-delivery icon on Android,
iOS and the sign-in screen. It separates sign-in from registration, adds a complete
profile and multiple buyer addresses, and retains the testing OTP display. The backend's
`TESTING_MODE` controls whether the OTP appears; no mobile environment flag is needed. See
[testing mode](../docs/TESTING_MODE.md).

Sharing builds require an explicit backend URL. The current APK connects to
`http://192.168.29.127:18080` for same-Wi-Fi testing. See
[phone testing](../docs/PHONE_TESTING.md) to restart the Wi-Fi gateway and read
development OTPs. For another backend, rebuild:

```bash
PUBLIC_API_URL=https://your-public-api make android-share
```

Mobile tests run in two packages: `mobile-app` contains eleven connection/routing/UI tests;
`mobile-app/tooling/storage_tests` contains two networking/offline storage tests.
The host-only package prevents desktop SQLite build hooks from entering mobile
builds. `make test` runs both suites.

See [the size report](../docs/ANDROID_APP_SIZE.md) and [workspace setup](../README.md)
for details. Production store builds need the owner's signing key. iOS builds
require macOS and Xcode.
