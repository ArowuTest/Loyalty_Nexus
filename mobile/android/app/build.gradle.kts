import java.util.Properties
import java.io.FileInputStream

plugins {
    id("com.android.application")
    id("kotlin-android")
    // The Flutter Gradle Plugin must be applied after the Android and Kotlin Gradle plugins.
    id("dev.flutter.flutter-gradle-plugin")
    // Firebase / Google Services
    id("com.google.gms.google-services")
    id("com.google.firebase.crashlytics")
}

// Load signing credentials from key.properties (not committed to git)
val keyPropertiesFile = rootProject.file("key.properties")
val keyProperties = Properties()
if (keyPropertiesFile.exists()) {
    keyProperties.load(FileInputStream(keyPropertiesFile))
}

android {
    namespace = "ng.loyaltynexus.loyalty_nexus"
    compileSdk = flutter.compileSdkVersion
    // speech_to_text requires NDK 28.2.13676358; NDKs are backward compatible so
    // the highest requested version wins (Flutter prints this exact instruction).
    ndkVersion = "28.2.13676358"

    compileOptions {
        // Enable core library desugaring (required by flutter_local_notifications)
        isCoreLibraryDesugaringEnabled = true
        sourceCompatibility = JavaVersion.VERSION_11
        targetCompatibility = JavaVersion.VERSION_11
    }

    kotlinOptions {
        jvmTarget = JavaVersion.VERSION_11.toString()
    }

    signingConfigs {
        create("release") {
            // Read each value and FAIL LOUDLY if it is missing or blank.
            //
            // These were previously `... as String? ?: ""`. That silent default is
            // how an empty alias reached the signer and produced, ten minutes into
            // the build, deep inside :app:packageRelease:
            //     KeytoolException: No key with alias '' found in keystore
            // A signing credential that is silently blank is strictly worse than a
            // build that stops immediately and says which value is missing.
            fun requireProp(name: String): String {
                // This block is CONFIGURED for every build, debug included. When
                // key.properties is absent entirely that is the normal local-dev
                // case — return empty and let the buildTypes guard below decide
                // (it already fails hard only for RELEASE tasks). Throwing here
                // would break `flutter run` for anyone without a keystore.
                if (!keyPropertiesFile.exists()) return ""
                val v = (keyProperties[name] as String?)?.trim()
                if (v.isNullOrEmpty()) {
                    throw GradleException(
                        "Release signing: '$name' is missing or blank in " +
                        "android/key.properties. Every release-signing value must be " +
                        "present — a blank one silently yields an unusable artifact. " +
                        "In CI these come from the ANDROID_* repository secrets."
                    )
                }
                return v
            }

            keyAlias = requireProp("keyAlias")
            keyPassword = requireProp("keyPassword")
            storePassword = requireProp("storePassword")
            // Only resolve a file when there is genuinely one to resolve.
            val storeFileName = requireProp("storeFile")
            storeFile = if (storeFileName.isEmpty()) null else file(storeFileName)
        }
    }

    defaultConfig {
        applicationId = "ng.loyaltynexus.loyalty_nexus"
        // 24 = Android 7.0. Raised from 23: video_player_android 2.12.0 (pulled in by
        // Flutter 3.44) declares minSdk 24, which failed the manifest merger.
        // record_android needs >= 23, so 24 satisfies everything.
        minSdk = 24
        targetSdk = flutter.targetSdkVersion
        versionCode = flutter.versionCode
        versionName = flutter.versionName
    }

    buildTypes {
        getByName("debug") {
            signingConfig = signingConfigs.getByName("debug")
        }
        getByName("release") {
            // FAIL FAST — never silently degrade to debug signing.
            // The old fallback shipped a DEBUG-SIGNED release artifact with no
            // warning if key.properties was missing. Play rejects it, and any
            // tester who installed it cannot upgrade in place (signature
            // mismatch forces an uninstall). A local debug/profile run is
            // unaffected; only assembling a RELEASE without a keystore fails.
            signingConfig = if (keyPropertiesFile.exists()) {
                signingConfigs.getByName("release")
            } else if (project.gradle.startParameter.taskNames.any {
                    it.contains("Release") || it.contains("release")
                }) {
                throw GradleException(
                    "Release build requested but android/key.properties is missing. " +
                    "A debug-signed release cannot be distributed: Play rejects it and " +
                    "testers cannot upgrade in place. Provide the keystore (CI injects it) " +
                    "or build a debug variant instead."
                )
            } else {
                signingConfigs.getByName("debug")
            }
            isMinifyEnabled = true
            isShrinkResources = true
            proguardFiles(
                getDefaultProguardFile("proguard-android-optimize.txt"),
                "proguard-rules.pro"
            )
        }
    }
}

dependencies {
    coreLibraryDesugaring("com.android.tools:desugar_jdk_libs:2.1.4")
}

flutter {
    source = "../.."
}
