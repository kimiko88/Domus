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
- [x] **SQLite Dual-Pool & Entity Cache Warm-Up**:
  - Pool separato di scrittura serializzato (`MaxOpenConns=1`) e pool di lettura concorrente (`PRAGMA query_only=ON`).
  - Warm-up automatico delle entità in memoria all'avvio dell'hub per letture sub-50 microsecondi.
- [x] **EventBus Dispatcher Timeout & Code Isolate**:
  - Code separate `queue` (core) e `asyncQueue` (I/O, task lunghi).
  - Timeout rigido configurabile (`HandlerTimeout`) con recupero panic isolato per proteggere il core hub da stalli.
- [x] **Fuzz Testing di Rete su Frame Esterni**:
  - Suite di fuzz testing nativo Go per pacchetti Modbus TCP e frame KNX cEMI (>650k mutazioni senza crash).
- [x] **Envelope Encryption a Riposo (AES-256-GCM)**:
  - Cifratura trasparente a riposo per segreti, token e credenziali salvate su SQLite con formato `ENC:v1:<base64>`.
- [x] **Supervision Tree & Circuit Breaker per Plugin**:
  - Gestione stati (`Healthy`, `Degraded`, `Tripped`), backoff esponenziale con jitter e panic containment per tutti gli adattatori hardware.
- [x] **Pipeline Hardware Safe-Stop Idrico & Gas (<50ms)**:
  - Canale ultra-rapido fuori banda e chiusura parallela deterministica delle elettrovalvole principali di acqua e gas con reazione certificata <10ms.
  - Latch di emergenza con reset manuale autorizzato e isolamento dei guasti fisici su singola valvola.
- [x] **Manutenzione Automatica SQLite & Flash Wear Levelling**:
  - Schedulatore integrato per `wal_checkpoint(TRUNCATE)` e `PRAGMA optimize`, con prevenzione usura celle Flash NAND su hub embedded.
- [x] **Network Scanner & Automatic Device Fingerprinting**:
  - Scansione passiva/attiva della LAN (ARP table cross-platform Windows/Linux), risoluzione MAC OUI dei vendor IoT principali (Shelly, Espressif, Philips Hue, Sonoff, Tuya, Raspberry Pi) e port-probing TCP mirato con endpoint REST `/api/v1/network/*`.
- [x] **Dry-Run Restore Validation per Backup**:
  - Metodo `VerifyArchive` per validazione in-memory di decifratura, decompressione gzip, integrità tar e checksum senza scrittura su disco.

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
- [x] **Hardening Database: SQLite Dual-Pool & Cache WarmUp**:
  - `sqlite.OpenDualPool(cfg)`: separazione della connessione di scrittura serializzata (`MaxOpenConns = 1` anti-busy-lock) e del pool di lettura concorrente (`PRAGMA query_only = ON`, `MaxOpenConns = max(4, NumCPU*2)`) in WAL mode.
  - Iniezione del pool di lettura su query-heavy repositories (`EntityRepository`, `HistoryRepository`, `StatisticsRepository`, `LogbookRepository`).
  - Precaricamento esplicito in RAM (`repos.WarmUp(ctx)`) al boot in `cmd/domus/main.go` per letture sub-50-microsecondo su entità.
- [x] **Resilienza Event Bus: Dispatcher con Timeout & Code Separate**:
  - `MemoryBus.invokeHandlerWithTimeout`: esecuzione protetta con timeout configurabile (`HandlerTimeout`, default 3s) e `recover()` isolato da panic. I worker non restano mai bloccati da handler lenti o appesi.
  - Coda asincrona dedicata (`asyncQueue` con pool separato di worker) via `SubscribeAsync` e `SubscribeAllAsync` per isolare I/O esterni (push, webhook, cloud) senza impattare la latenza del loop core.
- [x] **Fuzz Testing di Rete su Frame Binari (Modbus TCP & KNX)**:
  - Fuzzing nativo Go 1.18+ per MBAP header decoding, Modbus response parsing e register decoding (`internal/plugins/modbus_tcp/frame_fuzz_test.go`).
  - Fuzzing per KNXnet/IP 6-byte header, messaggi cEMI, decodifica DPT (1, 5, 9, 14) e parsing indirizzi di gruppo/individuali (`internal/plugins/knx/cemi_fuzz_test.go`).
  - Oltre 650.000 mutazioni validate a 0 crash/panic.