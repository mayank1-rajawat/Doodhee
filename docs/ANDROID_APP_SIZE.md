# Doodhee Android APK size

Measured on 14 November 2026. The ARM64 APK is now **9.30 MB**, about **96% smaller** than the original universal debug APK. MB means 1,000,000 bytes; MiB means 1,048,576 bytes.

| Build | Bytes | MB | MiB |
| --- | ---: | ---: | ---: |
| Original universal debug APK | 211,567,158 | 211.57 | 201.77 |
| Optimized ARM64 testing APK (v1.1.1) | 9,300,526 | 9.30 | 8.87 |

## What changed

- Build in release mode: remove the development Dart kernel and debug runtime overhead.
- Package one CPU architecture per APK. The retained testing build targets ARM64.
- Enable Android code/resource shrinking and Flutter's release icon tree shaking.
- Bundle the Doodhee brand mark at an app-appropriate resolution and generate density-specific launcher icons.
- Keep Dart debug symbols and the R8 mapping outside the APKs.
- Compress native libraries for the explicitly opted-in testing builds. Android extracts them during installation; **this reduces file/download size, not installed storage by the same amount**. Normal production packaging keeps its defaults. See [Android's native packaging options](https://developer.android.com/reference/tools/gradle-api/8.8/com/android/build/api/dsl/JniLibsPackaging) and [library extraction behavior](https://developer.android.com/guide/topics/manifest/application-element#extractNativeLibs).
- Move desktop SQLite test dependencies into a host-only test package. The original debug APK included extra SQLite libraries from that test hook. The release baseline already excluded them; they are not the source of the large release-mode saving.

No application screens, buyer/seller flows, offline delivery support, or bundled English/Hindi/Punjabi fonts were removed. Cupertino fallback icons are retained and tree-shaken to 848 bytes; Material icons are tree-shaken to 6,160 bytes.

## Build and share

From `/home/hp/WorkSpace/doodhee`:

```bash
PUBLIC_API_URL=http://192.168.29.127:18080 make android-share
```

| Use | File |
| --- | --- |
| ARM64 phone | `artifacts/android/doodhee-wifi-v1.1.1-arm64.apk` |
| Measured sizes and SHA-256 checksums | `artifacts/android/size-report.json` |
| Crash symbols and R8 mapping | `artifacts/android/symbols/` |

Send one matching APK. Each file contains the complete app. Do not send the old universal `app-debug.apk` when testing download size. These small APKs are release-mode **testing builds signed with the development key**; store signing requires the owner's keystore.

The current APK connects to `http://192.168.29.127:18080`, verified reachable in the user's phone browser on the same Wi-Fi. The Public Gateway must remain running on that interface. Sharing builds now require an explicit `PUBLIC_API_URL`; they no longer silently default to an emulator address. See [phone testing](PHONE_TESTING.md). For a hosted backend, rebuild after setting the actual address:

```bash
PUBLIC_API_URL=https://your-public-api make android-share
```

For a phone attached over USB, `adb reverse tcp:18080 tcp:18080` plus a build using `PUBLIC_API_URL=http://127.0.0.1:18080` is another local testing option. The backend must be running. Changing the endpoint does not require changing authentication or roles.

## Verification

- Flutter analysis passes for the app and host-only test package.
- All fifteen app tests and two storage tests pass, covering routing/profile/localization/UI, exact-phone customer lookup, the new workflow screens, connection handling, testing OTP UI, token refresh and offline delivery. Offline delivery still queues after a transport failure.
- The final APK passes Android signature verification, ZIP integrity and ZIP alignment checks.
- Its signing certificate matches the previous development APK.
- Verified package `in.doodhee.doodhee`, display name `Doodhee`, minimum API 24, target API 36, and release-mode packaging without `debuggable` or `testOnly` enabled.
- Each APK contains exactly one intended architecture, all three language fonts, and no debug kernel or host-test `libsqlite3.so`.
- The user confirmed that the phone can load the Wi-Fi health endpoint. No device is attached over ADB; APK installation and the complete login flow are awaiting confirmation on the updated branded build.

Measurements can be regenerated with:

```bash
node scripts/apk-size-report.mjs artifacts/android/doodhee-wifi-v1.1.1-arm64.apk
```

Archive the matching `symbols/` files with each distributed build for later crash diagnosis. Reported sizes are for these exact artifacts and may change after future code or dependency updates.
