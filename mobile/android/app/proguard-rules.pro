# R8 / ProGuard rules — Loyalty Nexus
#
# CONTEXT: `isMinifyEnabled = true` had been set for a long time, but until now
# EVERY CI build was `assembleDebug`, so R8 had literally never run on this
# project. The first release build failed at :app:minifyReleaseWithR8.
#
# Dart code is AOT-compiled into libapp.so and is NOT touched by R8. What R8 can
# break is the Java/Kotlin plugin layer — especially anything reached reflectively
# or only from native code.

# ── The failure that surfaced first ───────────────────────────────────────────
# Flutter's embedding references Play Core (deferred components / split installs):
#   Missing class com.google.android.play.core.splitinstall.SplitInstallManager
#   (referenced from PlayStoreDeferredComponentManager ... and 5 other contexts)
# This app does NOT use deferred components, so the classes are legitimately
# absent. Flutter's own guidance is to silence them rather than add the
# dependency — pulling in Play Core just to satisfy R8 would ship dead weight.
-dontwarn com.google.android.play.core.**
-dontwarn com.google.android.play.core.splitcompat.**
-dontwarn com.google.android.play.core.splitinstall.**
-dontwarn com.google.android.play.core.tasks.**

# ── Flutter engine ────────────────────────────────────────────────────────────
-keep class io.flutter.** { *; }
-keep class io.flutter.plugins.** { *; }
-keep class io.flutter.embedding.** { *; }
-dontwarn io.flutter.embedding.**

# ── Firebase (core / messaging / analytics / crashlytics) ─────────────────────
-keep class com.google.firebase.** { *; }
-keep class com.google.android.gms.** { *; }
-dontwarn com.google.firebase.**
-dontwarn com.google.android.gms.**

# Crashlytics needs line numbers and source files to produce readable stacks.
# Without these the reports that justify shipping Crashlytics at all are useless.
-keepattributes SourceFile,LineNumberTable
-keep public class * extends java.lang.Exception

# ── Media: video_player + just_audio (ExoPlayer / media3) ─────────────────────
# ExoPlayer is a well-known R8 casualty: it instantiates renderers and extractors
# by reflection, so stripping them yields runtime "Decoder init failed" errors
# that only appear in release builds.
-keep class com.google.android.exoplayer2.** { *; }
-keep class androidx.media3.** { *; }
-dontwarn com.google.android.exoplayer2.**
-dontwarn androidx.media3.**

# ── Audio capture: record + speech_to_text ────────────────────────────────────
-keep class com.llfbandit.record.** { *; }
-keep class com.csdcorp.speech_to_text.** { *; }

# ── webview_flutter ───────────────────────────────────────────────────────────
# JavascriptInterface methods are called from JS by name; R8 cannot see those
# call sites and will happily rename them.
-keep class io.flutter.plugins.webviewflutter.** { *; }
-keepclassmembers class * {
    @android.webkit.JavascriptInterface <methods>;
}

# ── Kotlin coroutines (used by several plugins) ───────────────────────────────
-keepclassmembers class kotlinx.coroutines.** { volatile <fields>; }
-dontwarn kotlinx.coroutines.**

# ── Annotations / generics R8 needs to keep semantics intact ──────────────────
-keepattributes *Annotation*,Signature,InnerClasses,EnclosingMethod

# ── Parcelables ───────────────────────────────────────────────────────────────
-keepclassmembers class * implements android.os.Parcelable {
    public static final ** CREATOR;
}

# ── Enums (valueOf/values are reflective) ─────────────────────────────────────
-keepclassmembers enum * {
    public static **[] values();
    public static ** valueOf(java.lang.String);
}
