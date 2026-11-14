plugins {
    id("com.android.application")
    // The Flutter Gradle Plugin must be applied after the Android and Kotlin Gradle plugins.
    id("dev.flutter.flutter-gradle-plugin")
}

android {
    namespace = "in.doodhee.doodhee"
    compileSdk = flutter.compileSdkVersion
    ndkVersion = flutter.ndkVersion

    compileOptions {
        sourceCompatibility = JavaVersion.VERSION_17
        targetCompatibility = JavaVersion.VERSION_17
    }

    defaultConfig {
        // HTTP is opt-in for a locally signed sharing build. Store builds stay HTTPS-only.
        val localTestSigning = providers.gradleProperty("localTestSigning").orNull == "true"
        manifestPlaceholders["allowLocalHttp"] =
            (localTestSigning && providers.gradleProperty("allowLocalHttp").orNull == "true").toString()
        // TODO: Specify your own unique Application ID (https://developer.android.com/studio/build/application-id.html).
        applicationId = "in.doodhee.doodhee"
        // You can update the following values to match your application needs.
        // For more information, see: https://flutter.dev/to/review-gradle-config.
        minSdk = flutter.minSdkVersion
        targetSdk = flutter.targetSdkVersion
        // Uses the version code from pubspec.yaml. When using split APKs, 1000 * ABI_VERSION
        // is added automatically by Flutter. (https://developer.android.com/studio/build/configure-apk-splits#configure-APK-versions)
        // You can force using the value of versionCode by specifying the `-P force-version-code-ignoring-abi=true`
        // flag during build.
        versionCode = flutter.versionCode
        versionName = flutter.versionName
    }

    buildTypes {
        release {
            isMinifyEnabled = true
            isShrinkResources = true
            // Explicitly opt in to the existing development key for local APK sharing.
            // Never use this flag for a store upload.
            if (providers.gradleProperty("localTestSigning").orNull == "true") {
                signingConfig = signingConfigs.getByName("debug")
            }
        }
    }

    packaging {
        jniLibs {
            // Smaller downloads for sideload testing; Android extracts these on installation.
            // Leave normal production packaging unchanged.
            if (providers.gradleProperty("localTestSigning").orNull == "true" &&
                providers.gradleProperty("compactNativeLibraries").orNull == "true") {
                useLegacyPackaging = true
            }
        }
    }
}

kotlin {
    compilerOptions {
        jvmTarget = org.jetbrains.kotlin.gradle.dsl.JvmTarget.JVM_17
    }
}

flutter {
    source = "../.."
}
