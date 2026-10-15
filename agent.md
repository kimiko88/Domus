<!-- 
  Domus Antigravity Agent Configuration & Code-Generation Directives
  Refer to AGENTS.md for complete repository guidelines.
-->

# Domus Project AI Agent Directives (`agent.md`)

Benvenuto nel workspace di **Domus**, la piattaforma domotica e OS locale-first per Smart Home.

Per garantire la massima qualità del codice generato, la robustezza delle API e l'allineamento tra Backend, Frontend Web e App Mobile (iOS e Android), tutti gli agenti Antigravity devono attenersi alle seguenti direttive operative.

---

## 1. Architettura del Progetto

1. **Backend (Go 1.23+)**:
   - Clean Architecture: `internal/domain` (nessuna dipendenza esterna), `internal/application` (orchestratori e use cases), `internal/infrastructure` (driver hardware, db, mqtt, ble, zigbee), `internal/interfaces` (HTTP Chi v5, WebSocket, mDNS).
   - Local-first & offline by design. Nessuna dipendenza obbligatoria da cloud esterni.
   - Tutti gli endpoint REST e i comandi WebSocket devono essere allineati con i client mobile (`/api/v1/services/{domain}/{service}` e WebSocket `call_service`).

2. **Frontend Web (Vue 3 + TypeScript + Pinia)**:
   - Composables standardizzati: `useEntitiesStore`, `useWsStore`, `useEntityCommand`, `useAuthStore`.
   - Glassmorphism premium (token CSS in `tokens.css` e utility in `main.css`).
   - Accessibilità rigorosa WCAG 2.2 Livello AA (attributi ARIA, skip link, contrasti minimi 4.5:1, min target 44px, `prefers-reduced-motion`).

3. **App Mobile iOS (Swift 6 + SwiftUI / DomusKit)**:
   - Framework condiviso `DomusKit` con architettura MVI/MVVM.
   - Connessione dual-path automatica: LAN locale via Bonjour mDNS (`_domus._tcp`) e fallback WireGuard VPN / TLS remoto.
   - Accessibilità VoiceOver nativa con `.accessibilityElement(children: .combine)`.

4. **App Mobile Android (Kotlin 2.x + Jetpack Compose / `com.domus.app`)**:
   - Material 3 & Material You con colori dinamici (`dynamicDarkColorScheme`).
   - Notifiche critiche di allarme con `AudioAttributes.USAGE_ALARM` per bypassare il Non Disturbare.
   - Semantica TalkBack con `Modifier.semantics(mergeDescendants = true)` e target minimi di 48dp.

---

## 2. Regole Modulari per la Generazione

Le regole specializzate dettagliate sono caricate automaticamente da Antigravity da `.agents/rules/`:
- [`.agents/rules/backend_go_patterns.md`](file:///c:/Users/chimi/Desktop/Programmazione/Domus/.agents/rules/backend_go_patterns.md): Pattern Go, gestione errori, goroutine e lock mutex.
- [`.agents/rules/web_accessibility_and_styling.md`](file:///c:/Users/chimi/Desktop/Programmazione/Domus/.agents/rules/web_accessibility_and_styling.md): Linee guida a11y, CSS glassmorphism, semantica ARIA.
- [`.agents/rules/mobile_architecture_and_os.md`](file:///c:/Users/chimi/Desktop/Programmazione/Domus/.agents/rules/mobile_architecture_and_os.md): Specifiche iOS SwiftUI e Android Jetpack Compose.

---

## 3. Comandi di Validazione Obbligatori

Prima di concludere qualsiasi modifica o generare nuovo codice:
- **Backend Go**: `go test ./...`
- **Frontend Web**:
  ```bash
  cd web
  npm run type-check
  npm run test:run
  ```
