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
- [x] **IoT Network Guardian & WAN Leak Detection**:
  - Boundary isolation per subnet/VLAN IoT e MAC untrusted, ispezione dei flussi di rete e alert immediato (`network.leak_alert`) per tentativi di connessione a server WAN esterni con cooldown anti-flood.
- [x] **Hotplug USB Dinamico via udev (Domus OS)**:
  - Monitoraggio real-time di `/dev/serial/by-id/` con classificazione automatica dei dongle Zigbee (CC2652, Sonoff, ConBee, SkyConnect) e Z-Wave (Zooz 700/800, Aeotec), con emissione eventi `usb.plugged` / `usb.unplugged` e riconnessione a caldo.
- [x] **Simulatore di Presenza Intelligente ("Modalità Vacanza")**:
  - Replay realistico delle routine di luci e tapparelle registrate con applicazione di jitter stocastico ($\pm 15$ min) per simulare la presenza umana a casa disabitata.
- [x] **Automazioni basate su Pendolarismo ed ETA**:
  - Tracciamento dinamico della traiettoria e calcolo ETA d'arrivo a casa (`CommuteTracker`) per pre-riscaldamento/climatizzazione anticipata e avvio intelligente ricarica auto.
- [x] **Manutenzione Predittiva & Rilevamento Anomalie**:
  - Rilevamento micro-perdite idriche continue da flussimetro (con soglie differenziate giorno/notte).
  - Rilevamento porte frigo/freezer rimaste socchiuse con protezione deterioramento alimenti.
  - Diagnostica anomalie motori/pompe (blocco rotore $> 135\%$ e cavitazione a secco $< 30\%$).
  - Calcolo del punto di rugiada (*Dew Point*) tramite formula di Magnus-Tetens e valutazione rischio muffa per pareti e ponti termici.
- [x] **Standard OCPP 1.6J / 2.0.1 & Ricarica Solare Intelligente**:
  - Server OCPP WebSocket nativo per colonnine/wallbox con modulazione continua della corrente (6A - 32A) in base alla sovrapproduzione fotovoltaica, stato di carica (SoC) e protezione anti-blackout contatore.

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
- [x] **Federazione Multi-Hub Parent/Satellite (WireGuard & mTLS)**:
  - Architettura distribuita parent-hub centrale e satelliti periferici (`internal/application/federation`).
  - Namespacing automatico entità remote (`satellite_id:domain.entity`), proxying trasparente dei comandi bidirezionali e replicazione dinamica degli stati.
  - Heartbeat continuo e failover isolato per outbuilding, dépendance o case secondarie.
- [x] **Edge AI Computer Vision & NVR Locale**:
  - Biometria facciale dei familiari residenti via cosine similarity locale ($\ge 0.65$) su embedding vettoriali (`internal/application/camera`).
  - Riconoscimento automatico targhe autorizzate (ALPR/ANPR) per azionamento intelligente varchi carrabili e garage.
  - Pubblicazione automatica di eventi arricchiti sul bus di dominio per automazioni ed alert push istantanei.
- [x] **Offline Voice Pipeline & LLM Tool Calling (Ollama)**:
  - Function calling strutturato con schema OpenAI/Ollama per `control_device`, `set_climate`, `activate_scene` e `query_entity_state` (`internal/application/assistant`).
  - Esecuzione deterministica con verifica entità e fallback fluido sull'engine euristico locale.
- [x] **Coda Comandi Offline & Idempotency Key (`X-Idempotency-Key`)**:
  - Middleware HTTP universale `Idempotency` su `/api/v1/*` (`internal/interfaces/http/middleware/idempotency.go`).
  - Cache thread-safe con TTL configurabile, detection conflitti concorrenti in-flight (`409 Conflict`) e replay trasparente delle risposte con header `X-Cache-Lookup: HIT-IDEMPOTENT` per prevenire doppi scatti su comandi fisici.
- [x] **E2E Chaos & Fault Injection Testing Suite**:
  - Suite di chaos test avanzata (`test/e2e/chaos_fault_injection_test.go`): simulazione di failure storm su bus hardware e verifica transizione/recupero automatico del `Supervisor` (Tripped -> Degraded -> Healthy), storm concorrente di 50 goroutine con Idempotency Key, e trigger d'emergenza Safe-Stop ad alta frequenza con risposta sottomillisecondo e latching deterministico.
- [x] **Isolamento Router Firewall & IoT Device Quarantine**:
  - Plugin di controllo router e apparati di rete (`internal/plugins/firewall/`) con driver per UniFi Network Controller (`block-sta`), OPNsense (`alias_util domus_quarantine`) e MikroTik RouterOS (`address-list`).
  - Integrazione con `NetworkGuardian`: azione di isolamento istantaneo su MAC/IP address e dispatch evento `network.device_quarantined`.
  - Endpoint REST `/api/v1/cameras/faces`, `/api/v1/cameras/plates`, `/api/v1/nvr/status` montati nel router HTTP e collegati al motore di vision locale.
- [x] **Streaming Replication per Alta Affidabilità (HA SQLite WAL Streamer)**:
  - Streaming in tempo reale delle pagine WAL di SQLite da Leader a Standby con digest SHA-256 e verifica di monotonicità sequenziale (`internal/application/failover/wal_streamer.go`).
  - Prevenzione split-brain e calcolo dinamico del replication transit time lag (< 5ms).
  - Suite unitaria e di integrazione passata al 100% (`internal/application/failover/wal_streamer_test.go`).
- [x] **Frontend Web: Gestione Biometria Volti Familiari & Targhe ALPR**:
  - Modalità vista dedicata in `web/src/views/CamerasView.vue` (`currentViewMode = 'ai-faces'`) con schede biometriche per residenti (avatar, ruolo, embedding 128-D attivo, simulazione test match cosine similarity &ge; 0.65), form modale di registrazione e rimozione REST.
  - Sezione veicoli autorizzati con render grafico delle targhe automobilistiche europee (`[IT] AB 123 CD`), modello auto, badge apertura automatica cancello carrabile e form di aggiunta/eliminazione.
  - Type-safe con interfacce `FamilyFace` e `AuthorizedVehicle` in `web/src/api/types.ts`.
- [x] **IoT Network Guardian REST & UI di Confinamento LAN**:
  - Endpoint REST completi in `NetworkHandler`: `GET /api/v1/network/leaks`, `POST /api/v1/network/quarantine`, `POST /api/v1/network/unblock` montati nel server HTTP Chi.
  - Tab dedicato `LAN & IoT Guardian` in `web/src/views/IntegrationsView.vue` con banner telemetria (host LAN, apparati confinati, alert WAN leaks), scansione ARP attiva/passiva, port probing TCP e azione 1-click di isolamento/sblocco apparati rogue.
- [x] **Visual Flow-Based Automation Editor (Node Canvas)**:
  - Nuova vista a pagina intera `web/src/views/FlowEditorView.vue` collegata alla rotta `/automations/flow/:id?` e pulsante rapido "Editor a Nodi" in `AutomationsView.vue`.
  - Motore canvas interattivo con pan, zoom, grid a punti, curve di Bézier SVG per i collegamenti e libreria nodi categorized (Triggers, Conditions, Logic & Flow, Actions).
  - Compilatore AST bidirezionale (Graph <-> Automation JSON) per caricamento e salvataggio trasparente sull'engine Go.
  - Inspector drawer reattivo per parametri dei nodi e simulazione animata visuale "Test Run" step-by-step.
- [x] **Frontend Wallbox OCPP & Ricarica Solare Intelligente (EV)**:
  - Sezione dedicata "Ricarica Solare Intelligente & Wallbox OCPP" in `web/src/views/EnergyDashboardView.vue`.
  - Telemetria live Wallbox: Potenza Attiva (W), Corrente Erogata (A), SoC Veicolo (%), Energia Sessione (kWh), badge stato Live (Charging/Suspended) con pulse-dot animato.
  - Selettore visuale di 4 modalità di gestione carica: Solo Solare (IEC 61851 ≥6A), Solare Ottimizzato (6A base + boost), Fast Charge, Notturna Economica.
  - Banner algoritmo di bilanciamento dinamico con surplus FV e carichi casa in tempo reale.
  - Particella SVG animata inverter→EV nel Power Wheel quando wallbox attiva.
  - API types TypeScript (`WallboxStatusResponse`, `WallboxTelemetry`, `WallboxModulationResult`, `WallboxChargingMode`).
  - 2 nuovi test Vitest per rendering telemetria e interazione mode-change/modulation.
- [x] **Frontend Simulatore di Presenza Vacanza (Vacation Mode)**:
  - Sezione "Simulatore di Presenza — Modalità Vacanza" in `web/src/views/ZonesView.vue`.
  - Toggle ARIA-compliant (`role="switch"`) per abilitare/disabilitare la simulazione con aggiornamento backend REST.
  - Banner stato dinamico (Simulazione Attiva/Disattivata) con pulse-dot animato in verde.
  - Lista azioni simulate (entity_id, command, typical_time) con modale di aggiunta nuove azioni.
  - Integrazione API: `GET/POST /api/v1/presence/vacation`, `POST /api/v1/presence/vacation/actions`.
  - API types TypeScript (`VacationModeStatus`, `VacationSimulatedAction`, `CommuteStatus`).
  - 3 nuovi test Vitest per rendering, toggle e modale azioni.
- [x] **Stato Validazione & Suite di Test (100% PASS)**:
  - Backend Go: 100% superato su tutti i package di dominio, applicazione, infrastruttura, plugin, e2e ed os (`go test ./...`).
  - Frontend Vue 3: **86 file di test, 324 test passati in Vitest** (0 fallimenti).
  - TypeScript Compilation: `vue-tsc -b` completato con 0 errori.