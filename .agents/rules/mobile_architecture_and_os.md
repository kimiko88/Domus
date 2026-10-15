# Mobile Companion Apps Architecture & OS-Specific Guidelines

These rules apply when writing, modifying, or refactoring code in `mobile/ios/` and `mobile/android/`.

## 1. Shared Architecture (Clean Architecture)
- **Domain Layer**: Platform-agnostic models, use cases, and repository interfaces.
- **Data Layer**: Network clients (HTTP + WebSocket), offline storage (SwiftData / Room), and OS hardware bridges.
- **Presentation Layer**: Declarative UI (SwiftUI on iOS, Jetpack Compose on Android) with observable ViewModels.

## 2. iOS Specific Guidelines (`mobile/ios/Sources/DomusKit/`)
1. **Swift 6 & Concurrency**:
   - Use `async/await`, `actor`, and `@Observable` macros.
   - All network calls must handle disconnection gracefully.
2. **SwiftUI Aesthetics & Accessibility**:
   - Use system materials: `.background(.ultraThinMaterial)`.
   - Apply `.accessibilityElement(children: .combine)`.
   - Provide `.accessibilityLabel`, `.accessibilityValue`, and `.accessibilityAction`.
   - Ensure touch targets are at least **44x44 pt**.
3. **Hardware Integrations**:
   - CoreLocation (circular geofence at 150m + significant visits).
   - WidgetKit with AppIntents for interactive home/lock screen controls.
   - APNs Critical Alerts for alarm intrusions (`com.apple.developer.usernotifications.critical-alerts`).

## 3. Android Specific Guidelines (`mobile/android/app/src/main/`)
1. **Kotlin 2.x & Jetpack Compose**:
   - Material Design 3 with Material You dynamic color palettes (`dynamicDarkColorScheme`/`dynamicLightColorScheme`).
   - Adaptive layouts with `ListDetailPaneScaffold` for tablets and foldables.
2. **Compose Accessibility**:
   - Use `Modifier.semantics(mergeDescendants = true)` on composite cards.
   - Provide `contentDescription` on all icon buttons and switches.
   - Minimum interactive touch target: **48.dp**.
3. **Android OS Integration**:
   - **Glance AppWidgets**: Interactive Compose widgets with `actionRunCallback`.
   - **TileService**: Quick Settings tiles for instant control from notification shade.
   - **WorkManager**: Background synchronization with charging/idle constraints.
   - **NotificationChannel**: `IMPORTANCE_HIGH` with `AudioAttributes.USAGE_ALARM` for critical alarms.
   - **BiometricPrompt**: Hardware Keystore authentication (Class 3 / `BIOMETRIC_STRONG`).
   - **CameraX Kiosk**: Front-camera proximity screen wake + `LockTaskMode` for wall-mounted tablets.
   - **Android 14/15 Standards**: Predictive Back (`PredictiveBackHandler`), edge-to-edge system bars, and 16 KB memory page compatibility.
