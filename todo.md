# Domus — Registro Stato Sviluppo & Roadmap

Questo documento traccia lo stato reale di avanzamento, le funzionalità implementate e le future evoluzioni architetturali per il sistema **Domus Hub & Companion Apps**.

---

## 1. Funzionalità Completate & Validate (100% Funzionanti)

### 1.1 Backend Core (Go 1.23+)
- [x] **Webhooks Outbound Asincroni con HMAC-SHA256**:
  - Tabella SQLite `outbound_webhooks` (migrazione 12) con tracciamento stato HTTP e contatore fallimenti.
  - Dispatcher asincrono (`OutboundDispatcher`) agganciato al bus di dominio (`events.Bus`) con worker pool dedicato.
  - Calcolo firma crittografica HMAC-SHA256 (`X-Domus-Signature` e `X-Hub-Signature-256`) con secret token.
  - Filtraggio avanzato su pattern di eventi (es. `alarm.*`, `entity.state_changed`, `*`).
  - Endpoint REST completi: `GET /api/v1/webhooks/outbound`, `POST /api/v1/webhooks/outbound`, `DELETE /api/v1/webhooks/outbound/{id}`, `POST /api/v1/webhooks/outbound/{id}/test`.
- [x] **Hardening di Sicurezza & Protezione DoS**:
  - Middleware `MaxBodySize(4 * 1024 * 1024)` globale su Chi router per prevenire attacchi di payload inflation.
  - Rate limiting su token bucket applicato ad autenticazione (`/login`, `/kiosk-login`, `/register`).
  - Protezione CSRF automatica tramite cookie `domus_csrf` e verifica header `X-CSRF-Token`.
  - Intestazioni di sicurezza browser: CSP restrittivo, X-Frame-Options DENY, X-Content-Type nosniff, Referrer-Policy strict.
- [x] **Notifiche Push Native & Persistenza Token**:
  - Tabella `push_tokens` (migrazione 11) e `inapp_notifications` (migrazione 12).
  - Endpoint REST per registrazione token (`/api/v1/notifications/push/tokens`) e invio payload (`/send`).
  - Formattazione avanzata di payload Apple APNs (canali critici, suoni allarme) e Google FCM (alta priorità).
- [x] **Integrazione Z-Wave JS Completa**:
  - Driver client WebSocket per zwave-js-ui con supporto a BinarySwitch, MultilevelSwitch (dimmer), SensorMultilevel, Metering ed eventi Notification.
- [x] **Supporto Matter & Thread**:
  - Cluster commissioning, gestione nodi ed endpoint, mapping entità verso il bus interno.
- [x] **Termostato Intelligente & Compensazione TRV**:
  - Algoritmo di calcolo del setpoint compensato per valvole termostatiche in base a temperatura effettiva della stanza e sensori contatto finestra aperta.
- [x] **Shared Shopping List & Chores**:
  - Endpoint dedicati `/api/v1/shopping` e `/api/v1/chores` con schedulazione promemoria e sincronizzazione real-time.

---

### 1.2 Frontend Web (Vue 3, Pinia, TypeScript)
- [x] **Hardening Client HTTP (`apiClient`)**:
  - Iniezione automatica del token CSRF da cookie per tutte le mutazioni.
  - Evento globale `domus:unauthorized` su risposte HTTP 401 per gestione fluida del logout.
- [x] **Test Suite & Type Checking**:
  - 100% test superati: 74 file di test, 256 test unitari/integrazione passati in Vitest.
  - `vue-tsc -b` completato con zero errori di tipo.

---

### 1.3 Companion Apps Native (iOS & Android) — 100% Zero Stub
- [x] **Parità Funzionale & Zero Mock**:
  - Rimossi tutti i mock/stub hardcoded (array fake, ritardi artificiali).
  - Entrambe le app utilizzano esclusivamente chiamate REST tipizzate (`DomusApiClient.kt` e `DomusHttpClient.swift`) e WebSocket Hub bidirezionale.
- [x] **Discovery LAN Istantanea via mDNS**:
  - Risoluzione automatica del servizio `_domus._tcp.local` con fallback fluido su endpoint remoto.
- [x] **Funzionalità di Sistema Native**:
  - **iOS**: Swift 6, SwiftUI, WidgetKit, ActivityKit (Live Activities), CarPlay e Face ID.
  - **Android**: Kotlin 2.x, Jetpack Compose, Glance AppWidgets, Quick Settings Tiles e BiometricPrompt.
- [x] **Compilazione Android Gradle**:
  - `:app:compileDebugKotlin` completato con 0 errori.

---

## 2. Roadmap Evolutiva & Backlog Futuro

- [x] **Skill Assistenti Vocali Cloud (Opzionali)**:
  - Lambda OAuth2 bridge per Amazon Alexa Smart Home (v3 API: Discovery, PowerController, BrightnessController, ThermostatController, StateReport).
  - Smart Home Action per Google Home (intents `action.devices.SYNC`, `QUERY`, `EXECUTE` con tratti OnOff, Brightness, TemperatureSetting).
  - Architettura zero-cloud per default: endpoint opzionali protetti da bearer token crittografico (`/api/v1/cloud/alexa` e `/api/v1/cloud/google`).
- [x] **Dashboard Condivisibili con Token Temporanei**:
  - Generazione di URL crittografati a scadenza con token HMAC-SHA256 (`/share/:token`).
  - Endpoint pubblico non autenticato per visualizzatori e chioschi (`GET /api/v1/dashboards/shared/{token}`).
  - Componente Web Vue 3 `SharedDashboardView.vue` con interfaccia responsive glassmorphic dark, badge di scadenza dinamici e controlli interattivi sicuri.
- [x] **Bridge Protocolli Domotici Industriali**:
  - Modbus TCP (`internal/plugins/modbus_tcp`): supporto nativo per inverter fotovoltaici e pompe di calore (Holding Registers, Input Registers, Coils, Float32 big/little-endian, scaling factors, polling e comandi).
  - KNXnet/IP bridge (`internal/plugins/knx`): tunneling client UDP su porta standard 3671, indirizzamento a 3 livelli (`main/middle/sub`), incapsulamento cEMI e codifica/decodifica DPT (1.001 Switch, 5.001 Scaling, 9.001 Temperature °C, 14.056 Electric Power W).
- [x] **Distributed Tracing (OpenTelemetry)**:
  - Propagazione rigorosa del contesto W3C (`traceparent: 00-<trace_id>-<span_id>-01`) e header `X-Trace-ID`.
  - Middleware HTTP Chi per estrazione/generazione span root e child.
  - Propagazione asincrona attraverso `events.Event` nel bus eventi interno (`events.MemoryBus`).
  - Correlazione transazioni e query lente SQLite WAL con `LogWALTrace`.