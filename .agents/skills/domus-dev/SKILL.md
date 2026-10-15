---
name: domus-dev
description: Guidelines and automated instructions for developing, testing, and generating code for the Domus Smart Home platform (Go backend, Vue 3 web frontend, iOS Swift, and Android Jetpack Compose).
---

# Domus Development Skill

Questo skill fornisce linee guida, pattern architetturali e comandi di verifica per lo sviluppo su **Domus**.

## Quando attivare questo skill
Utilizza questo skill ogni volta che:
- Aggiungi o modifichi endpoint REST o canali WebSocket nel backend Go.
- Crei nuovi componenti o viste nel frontend Vue 3.
- Allinei o crei componenti per le companion app iOS (SwiftUI) o Android (Jetpack Compose).
- Esegui o verifichi la suite di test del progetto.

## Pattern di sviluppo

### 1. Backend Go (internal/)
- Tutti i nuovi domini e entità devono essere registrati nel registry entità.
- Per chiamate di servizio universali usare `POST /api/v1/services/{domain}/{service}` con payload JSON `{ "entity_id": "...", ... }`.
- Per eventi in tempo reale, pubblicare sul bus eventi WebSocket con topic appropriato (`alarm.state_changed`, `tag.scanned`, `entity.state_changed`).
- Gestione della concorrenza: usare sempre `sync.RWMutex` su mappe condivise e rispettare il ciclo di vita del `context.Context`.

### 2. Frontend Web (web/)
- Tutti i componenti devono supportare il tema scuro/chiaro tramite variabili CSS in `tokens.css`.
- Rispetta WCAG 2.2 AA:
  - Pulsanti e interattivi con `min-height: 44px` e `min-width: 44px`.
  - Contrasto colore minimo 4.5:1.
  - Skip link presente per navigazione da tastiera.
  - `prefers-reduced-motion` per utenti con sensibilità al movimento.
- Esegui sempre `npm run type-check` e `npm run test:run` dopo modifiche al frontend.

### 3. Mobile Companion Apps (mobile/)
- **iOS (`mobile/ios/Sources/DomusKit/`)**:
  - SwiftUI con Dynamic Type e `.accessibilityElement(children: .combine)`.
  - Design con `.ultraThinMaterial`, micro-animazioni e haptic feedback con `UIImpactFeedbackGenerator`.
- **Android (`mobile/android/app/src/main/java/com/domus/app/`)**:
  - Material 3 e Material You con colori dinamici (`dynamicDarkColorScheme`).
  - Semantica unificata con `Modifier.semantics(mergeDescendants = true)`.
  - Notifiche con priorità massima e canale allarme dedicato (`AudioAttributes.USAGE_ALARM`).
