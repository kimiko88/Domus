# Domus — Agent & Coding Guidelines for Antigravity

Welcome to the **Domus** smart home appliance repository. This file serves as the primary instruction and guideline document for all Antigravity and AI coding agents operating within this codebase.

---

## 1. Project Overview & Architectural Vision

Domus is a **Local-First, Privacy-Focused Smart Home Operating System & Hub**, engineered for high reliability, sub-millisecond local network response times, zero cloud dependencies, and full integration across Web and Native Mobile Companion Apps (iOS & Android).

### Repository Structure
- `cmd/`: Binary entry points:
  - `cmd/domus`: Main Domus Core daemon (REST API, WebSocket Hub, Event Bus, Engine).
  - `cmd/domus-cli`: Command-line management utility.
  - `cmd/domus-supervisor`: Appliance supervisor controlling A/B updates, containers, and diagnostics.
- `internal/`: Clean Architecture modular backend:
  - `domain/`: Pure business entities and contracts (no external dependencies).
  - `application/`: Application services, use cases, and orchestrators.
  - `infrastructure/`: SQLite (with WAL & encryption), EventBus, MQTT, WebSocket Hub, Supervisor client.
  - `interfaces/`: HTTP Chi routing, REST handlers, WebSocket transport, Middleware.
  - `plugins/`: Integration bridges (Zigbee2MQTT, Z-Wave JS, ESPHome, Matter, HomeKit, Tuya Local, BLE Proxy, Wyoming, WASM).
- `web/`: Modern PWA Web Application (Vue 3, Pinia, TypeScript, Vite, Vanilla CSS Design System).
- `mobile/`:
  - `mobile/ios/`: 100% Native iOS companion app (`DomusKit` in Swift 6 + SwiftUI, WidgetKit, CarPlay, ActivityKit).
  - `mobile/android/`: 100% Native Android companion app (`com.domus.app` in Kotlin 2.x + Jetpack Compose, Glance, TileService, Biometrics).
- `os/`: Domus OS Buildroot appliance configurations, RAUC A/B atomic update engine, and storage provisioners.
- `test/`: End-to-end (`test/e2e`) and appliance OS validation suites (`test/os`).

---

## 2. Core Development Principles & Constraints

### 2.1 Backend (Go 1.23+)
1. **Clean Architecture Strictness**:
   - `domain` must never import `application`, `infrastructure`, or `interfaces`.
   - Controllers live in `internal/interfaces/http/handlers/` and must delegate logic to `internal/application/`.
   - Never write raw SQL in HTTP handlers; repositories handle persistence.
2. **Concurrency & Thread Safety**:
   - Use `sync.RWMutex` for stateful in-memory structures.
   - SQLite uses `_journal_mode=WAL` and `_busy_timeout=5000` to prevent database locks.
3. **Logging & Diagnostics**:
   - Always use structured logging with standard library `log/slog`.
   - Never use `println` or unformatted `log.Printf` in production code.
4. **Testing**:
   - Every new handler or service must include unit/integration tests with `testing.T`.
   - Verify changes with `go test ./...`.

### 2.2 Web Application (Vue 3 + TypeScript)
1. **Composition API & Script Setup**:
   - Always use `<script setup lang="ts">`.
   - Manage shared state exclusively via **Pinia stores** in `web/src/stores/`.
2. **Styling & Design Tokens**:
   - Use custom Vanilla CSS tokens defined in `web/src/assets/styles/tokens.css`.
   - Do NOT introduce Tailwind or external CSS frameworks unless explicitly instructed.
   - Maintain sleek dark-mode glassmorphism (`backdrop-filter: blur()`, subtle border glows, vibrant semantic colors).
3. **Type Safety**:
   - Strict TypeScript: no loose `any` when explicit interfaces exist in `web/src/api/types.ts`.
   - Verify types using `npm run type-check` (`vue-tsc -b`).

### 2.3 Mobile Companion Apps (iOS & Android)
1. **100% Native First**:
   - No WebViews or hybrid bridges. All UI must be written in SwiftUI (iOS) and Jetpack Compose (Android).
2. **Dual-Transport Networking**:
   - Prioritize ultra-low-latency local LAN via mDNS discovery (`_domus._tcp.local`) over direct WebSocket.
   - Seamlessly fallback to remote TLS/WireGuard endpoints with zero UI glitching.
3. **Offline-First Resilience**:
   - Cache entity state locally (SwiftData on iOS, Room/EncryptedSharedPreferences on Android).

---

## 3. Accessibility (a11y) & Visual Standards

Both the web interface and mobile applications must meet **WCAG 2.2 Level AA** accessibility standards:

### 3.1 Web Accessibility Guidelines
1. **Screen Reader Semantics**:
   - Every interactive control must have an accessible name: use `aria-label`, `aria-labelledby`, or visible text.
   - Toggles and switches must use `role="switch"` and `aria-checked="true|false"`.
   - Modals must manage focus trapping, `role="dialog"`, `aria-modal="true"`, and escape key dismissal.
2. **Keyboard Navigation**:
   - All interactive elements must be focusable (`tabindex="0"` where appropriate).
   - High-contrast, non-intrusive focus rings on `:focus-visible` (`outline: 2px solid var(--primary)` with `outline-offset: 2px`).
   - Implement a skip-link (`<a href="#main-content" class="skip-link">`) allowing keyboard users to bypass navigation.
3. **Motion & High Contrast Preferences**:
   - Respect `@media (prefers-reduced-motion: reduce)`: disable spring animations and non-essential transitions.
   - Respect `@media (forced-colors: active)`: ensure borders and text remain readable in Windows High Contrast Mode.
4. **Touch Target Sizing**:
   - Minimum touch target: **44x44px** (web & iOS) and **48x48dp** (Android).
   - Allow text selection for diagnostics, IDs, and logs (do NOT apply `user-select: none` globally to `body`).

### 3.2 Mobile Accessibility Guidelines
1. **iOS (SwiftUI)**:
   - Use `.accessibilityElement(children: .combine)`.
   - Provide informative `.accessibilityLabel`, `.accessibilityValue`, and `.accessibilityHint`.
   - Support Dynamic Type with scalable text styles (`.font(.headline)`, `.font(.subheadline)`).
2. **Android (Jetpack Compose)**:
   - Use `Modifier.semantics(mergeDescendants = true)` on composite cards.
   - Assign appropriate `contentDescription` and `role = Role.Switch` or `Role.Button`.
   - Support scale-independent pixels (`sp`) and TalkBack screen reader gestures.

---

## 4. Testing, Quality Assurance & Antigravity Workflow

When generating code or making modifications in this repository, always run and verify:

```bash
# 1. Backend Go Tests (must be 100% PASS)
go test ./...

# 2. Frontend Vitest Tests
cd web && npm run test:run

# 3. Frontend TypeScript Compilation
cd web && npm run type-check

# 4. Android Kotlin Compilation
cd mobile/android && ./gradlew.bat :app:compileDebugKotlin
```

### Critical Rules for Antigravity Agents
- **Preserve Documentation & Comments**: Never strip existing comments or docstrings.
- **No Line-Ending Corruptions**: Shell scripts in `os/scripts/` must retain Unix `\n` (LF) line endings.
- **Clickable Links**: Always format file references with markdown file links: `[Filename](file:///path/to/file)`.
- **Aesthetics Matter**: Avoid plain, generic MVP appearances. Use polished glassmorphic surfaces, subtle elevation, and responsive states.
