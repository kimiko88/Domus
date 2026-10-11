# Domus Native Mobile Companion Apps: Architectural & Implementation Plan

> **Obiettivo**: Progettare e sviluppare le applicazioni companion **100% native** per **iOS** (Swift 6 + SwiftUI) e **Android** (Kotlin 2.x + Jetpack Compose), garantendo prestazioni a 60/120Hz ProMotion, latenza sub-millisecondo in rete locale, integrazioni di sistema profonde (Widget interattivi, Geofencing, Notifiche azionabili, Allarmi critici, Tag NFC, Wearables e Tablet Kiosk Mode) e zero dipendenze da framework ibridi/WebView.

---

## 1. Perché 100% Nativo (Native vs Hybrid Rationale)

Sebbene la Web App PWA di Domus offra un'interfaccia ricca, le funzionalità di automazione domestica di livello enterprise richiedono capacità di sistema non accessibili o instabili nelle sandbox Web/PWA:

| Capacità di Sistema | Web PWA / Capacitor | iOS Nativo (SwiftUI) | Android Nativo (Jetpack Compose) |
| :--- | :--- | :--- | :--- |
| **Frame Rate & Fluidità** | 60Hz limitato dal browser | **120Hz ProMotion** costante | **120Hz Adaptive Refresh** |
| **Avvio a Freddo (Cold Start)** | 1.2s - 2.5s (parsing JS/DOM) | **< 250ms** pre-compilato | **< 300ms** Baseline Profile |
| **Scoperta LAN (mDNS / Bonjour)** | Non supportato nel browser | **NWBrowser (`_domus._tcp`)** | **NsdManager (Zero-Config)** |
| **Allarmi Critici (Bypass Silenzioso)** | Impossibile | **APNs Critical Alert Entitlement** | **NotificationChannel Urgent + Alarm** |
| **Geofencing & Presenza in Background** | Inaffidabile (OS killa il browser) | **CoreLocation Significant Visits** | **FusedLocationProvider + Geofence** |
| **Scansione NFC in Background** | Richiede app aperta | **CoreNFC Background Tag Reading** | **NFC Intent Filter (`ACTION_NDEF`)** |
| **Widget Schermata Home / Blocco** | Non supportato | **WidgetKit (iOS 17/18 Interattivo)** | **Glance AppWidget (Jetpack)** |
| **Centro di Controllo / Quick Settings** | Non supportato | **iOS 18 ControlWidget** | **Quick Settings TileService** |
| **Wearables (Apple Watch / Wear OS)** | Non supportato | **watchOS App Standalone** | **Wear OS Compose Tiles** |
| **Tablet Kiosk (Parete)** | Richiede browser kiosk | **Guided Access + Proximity** | **Lock Task Mode + CameraX Wake** |

---

## 2. Architettura Software Condivisa (Clean Architecture + MVI/MVVM)

Entrambe le codebase adotteranno la medesima struttura modulare **Clean Architecture**, facilitando il debug e la consistenza funzionale tra i team.

```
┌─────────────────────────────────────────────────────────────┐
│                    Presentation Layer                       │
│   SwiftUI Views / Jetpack Compose Components                │
│   ViewModels (StateFlow / @Observable)                      │
│   Interactive Widgets (WidgetKit / Glance)                  │
└──────────────────────────────┬──────────────────────────────┘
                               │
┌──────────────────────────────▼──────────────────────────────┐
│                      Domain Layer                           │
│   UseCases (ToggleEntity, SyncState, HandleGeofence)        │
│   Domain Models (Entity, Device, Area, Alarm, Scene)         │
│   Repository Interfaces                                     │
└──────────────────────────────┬──────────────────────────────┘
                               │
┌──────────────────────────────▼──────────────────────────────┐
│                       Data Layer                            │
│   Local Cache: SwiftData / Room (Offline-First)             │
│   Network Client: URLSession / Ktor HTTP + WebSocket        │
│   Hardware Bridges: CoreLocation, CoreNFC, Biometrics       │
│   mDNS Engine: Local Hub Resolver (_domus._tcp.local)       │
└─────────────────────────────────────────────────────────────┘
```

---

## 3. Matrice Tecnologica per Piattaforma

### 3.1 Stack Tecnologico iOS (Apple Ecosystem)
- **Linguaggio & Runtime**: Swift 6, Swift Concurrency nativa (`async/await`, `AsyncStream`, `Actor`).
- **UI Framework**: SwiftUI (iOS 17+, iOS 18 ready) con `@Observable` macro (nuovo observation framework).
- **Persistenza Locale**: SwiftData + ModelContext (cache offline istantanea delle entità e delle stanze).
- **Rete & Streaming**:
  - `URLSessionWebSocketTask` con protocollo di riconnessione automatica ad esponenziale backoff.
  - `Network.framework` (`NWBrowser` / `NWPathMonitor`) per scoprire `_domus._tcp.local` e monitorare Wi-Fi vs Dati Mobili.
- **Integrazioni Hardware**:
  - `CoreLocation`: Geofencing circolare ad alta efficienza energetica + Significant Motion.
  - `UserNotifications`: Notifiche azionabili, payload immagini snapshot e allarmi critici.
  - `CoreNFC`: Lettura passiva e in background di tag NFC `domus://tag/{id}`.
  - `WidgetKit` & `AppIntents`: Widget interattivi sulla schermata home e blocco; controlli Centro di Controllo in iOS 18 (`ControlWidget`).
  - `WatchKit` (`WatchConnectivity`): App autonoma per Apple Watch con complicazioni quadrante.
  - `LocalAuthentication`: Face ID / Touch ID per autenticazione e disarmo allarme.

### 3.2 Stack Tecnologico Android (Google Ecosystem)
- **Linguaggio & Runtime**: Kotlin 2.1+, Kotlin Coroutines & `StateFlow`/`SharedFlow`.
- **UI Framework**: Jetpack Compose con Material Design 3 (Dynamic Theming / Material You).
- **Persistenza Locale**: Room DB con crittografia SQLCipher (cache offline sincronizzata).
- **Iniezione Dipendenze**: Google Hilt / Koin.
- **Rete & Streaming**:
  - Ktor Client / OkHttp WebSocket con heartbeat bidirezionale e serializzazione kotlinx.serialization.
  - `NsdManager` (Network Service Discovery) per rilevare il server Domus su mDNS locale.
  - `ConnectivityManager.NetworkCallback` per intercettare il collegamento al Wi-Fi di casa.
- **Integrazioni Hardware**:
  - `Google Play Services Location`: `GeofenceManager` + Fused Location Provider.
  - `Firebase Cloud Messaging (FCM)`: Canali di notifica ad alta priorità (Heads-up) e allarme sonoro dedicato.
  - `Android NFC Framework`: `NfcAdapter.enableReaderMode` + Intent Filter per broadcast passivo.
  - `Glance AppWidget`: Widget reattivi in Compose per la schermata home.
  - `TileService`: Riquadri rapidi nei Quick Settings (tendina superiore) per luci e scenari.
  - `Wear OS Compose`: App smartwatch con Tiles e Complicazioni.
  - `BiometricPrompt`: Impronta digitale e sblocco facciale biometrico.

---

## 4. Specifiche Dettagliate delle Funzionalità Native

### 4.1 Motore di Connessione a Bassa Latenza (Dual-Transport Local/Remote)
L'app companion implementa un orchestratore di connettività intelligente a transizione trasparente:
1. **Rilevamento Locale Istantaneo**:
   - All'apertura dell'app, se il dispositivo è connesso al Wi-Fi, il motore invia una query mDNS per il servizio `_domus._tcp.local`.
   - Se il server risponde, l'app si connette direttamente all'IP locale (es. `http://192.168.1.100:8080`) via WebSocket. Latenza comandi: **< 15 millisecondi**.
2. **Fallback Remoto Trasparente**:
   - Se l'utente esce di casa o mDNS non risponde entro 1.2 secondi, l'app passa all'endpoint remoto sicuro configurato (WireGuard, Cloudflare Tunnel o Reverse Proxy TLS).
   - Transizione zero-glitch: la UI continua a mostrare l'ultimo stato salvato in SwiftData/Room e ri-sincronizza i delta di stato non appena il socket remoto è stabilito.

```mermaid
sequenceDiagram
    participant App as Mobile App (Swift/Kotlin)
    participant mDNS as Local mDNS / Bonjour
    participant LocalHub as Domus Hub (LAN 192.168.x.x)
    participant RemoteHub as Domus Remote (TLS Tunnel)

    App->>mDNS: Query _domus._tcp.local
    alt LAN Disponibile (< 1.2s)
        mDNS-->>App: Trovato 192.168.1.100:8080
        App->>LocalHub: WebSocket Handshake (/api/v1/ws)
        LocalHub-->>App: Connesso (Latenza 12ms)
    else Fuori Casa o Nessun Riscontro
        App->>RemoteHub: Fallback HTTPS/WSS (Token JWT)
        RemoteHub-->>App: Connesso via Tunnel Remoto
    end
```

### 4.2 Multi-Tier Presence & Geofencing a Batteria Zero
L'algoritmo di rilevamento presenza nativo supera le limitazioni dei sistemi convenzionali:
- **Livello 1 (Rete Wi-Fi Locale)**: L'app registra un callback sul cambio SSID di rete. Nel momento in cui lo smartphone si aggancia al Wi-Fi domestico (es. `Casa_5GHz`), invia all'istante l'evento `presence: home` al server Domus.
- **Livello 2 (Geofencing Circolare)**: Viene definita una geofence con raggio configurabile (es. 150 metri) attorno alle coordinate dell'abitazione. Quando lo smartphone entra o esce dal perimetro, il sistema operativo risveglia l'app in background per 10 secondi per comunicare il cambio stato.
- **Livello 3 (Visit Monitoring & Beacon BLE)**: Su iOS, `CLVisit` rileva periodi prolungati di permanenza; su entrambi i sistemi operativi, il modulo BLE Proxy rileva gli iBeacon o i pacchetti Bluetooth emessi da Domus o dallo smartwatch per determinare la stanza esatta in cui si trova l'utente.

### 4.3 Notifiche Interattive & Allarmi Critici (Critical Alerts)
- **Notifiche Azionabili con Bottoni Rapidi**:
  - Le notifiche push includono bottoni contestuali definiti da categorie backend:
    - Esempio Notifica: *"Rilevata finestra aperta in salotto e pioggia in arrivo."*
    - Azione 1: `[Chiudi Tapparelle]` -> Invia comando `cover.close` via webhook diretto senza aprire l'app.
    - Azione 2: `[Ignora per 1h]` -> Snooze allerta.
  - Allegato Rich Media: Se l'allarme è generato da una telecamera, la notifica include la snapshot JPEG o la GIF animata dei 3 secondi dell'evento.
- **Allarmi Critici (Bypass Silenzioso)**:
  - Per allarmi antincendio, perdite di gas, sensori allagamento o intrusione allarme armato:
    - Su iOS: Richiesta dell'entitlement `com.apple.developer.usernotifications.critical-alerts`. La notifica suona a volume massimo (100%) anche se l'iPhone è in modalità *"Non disturbare"* o con lo switch fisico del silenzioso attivo.
    - Su Android: Canale di notifica con `IMPORTANCE_HIGH`, allarme audio continuo gestito con `RingtoneManager` e visualizzazione a pieno schermo (*Heads-up Notification*).

### 4.4 Tag NFC & QR Code Fisici
- Supporto a qualsiasi tag adesivo NTAG213 / NTAG215 / NTAG216 economico (pochi centesimi di euro).
- L'app permette di associare un tag NFC fisico a un'entità, scena o automazione:
  - Es. Tag sul comodino: *"Scena Buonanotte (spegni tutte le luci, abbassa tapparelle, arma allarme notte)"*.
  - Es. Tag vicino alla porta: *"Disarma allarme / Apri portoncino"*.
- **Lettura in Background**:
  - Su iPhone (dal modello XS in poi) e smartphone Android moderni con NFC, non è necessario aprire l'applicazione: basta accostare lo smartphone al tag bloccato. Il sistema operativo legge il payload NDEF `domus://tag/{id}` ed esegue l'automazione in background con conferma haptic.

### 4.5 Interactive Home Screen Widgets & Controlli Rapidi
- **iOS WidgetKit (iOS 17+)**:
  - Widget interattivi di dimensione Piccola, Media e Grande.
  - L'utente può toccare direttamente il pulsante di una lampada o della presa sul widget della schermata home senza che si apra l'app (eseguito via `AppIntent`).
  - Widget Schermata di Blocco (Lock Screen): indicatore temperatura stanza, stato allarme o pulsante rapido per aprire il cancello.
  - **iOS 18 ControlWidget**: Aggiunta di bottoni personalizzati nel Centro di Controllo di iOS trascinabili a piacere.
- **Android Glance AppWidgets**:
  - Widget scritti in Jetpack Compose con supporto al Material You Dynamic Color.
  - Quick Settings Tile: 4 riquadri nella tendina di notifica Android per le 4 azioni preferite (Cancello, Luce Giardino, Scenario Uscita, Allarme).

### 4.6 App per Wearables (Apple Watch & Wear OS)
- **watchOS**:
  - Complicazioni per il quadrante: Mostrano la temperatura interna, la potenza assorbita istantanea (kW) e l'icona dell'allarme.
  - Applicazione standalone sull'orologio: griglia delle scene preferite e controlli vocali rapidi via microfono dell'Apple Watch collegati all'assistente locale LLM di Domus.
- **Wear OS**:
  - Wear OS Tiles: Scorrimento laterale dalla schermata dell'orologio per mostrare i controlli rapidi della casa.
  - Complicazioni circolari e lineari per quadranti digitali.

### 4.7 Modalità Kiosk per Tablet a Parete
Per gli utenti che montano un tablet Android economico o un iPad all'ingresso come pannello domotico fisso:
- **Schermo Sempre Attivo con Sleep Intelligente**: Mantiene acceso lo schermo durante il giorno; attiva uno screensaver a bassa luminosità (orologio minimale e meteo) durante la notte.
- **Risveglio con Prossimità**: Tramite la fotocamera frontale o il sensore di luminosità (CameraX su Android), il tablet accende lo schermo automaticamente quando una persona si avvicina a meno di 1.5 metri.
- **Blocco Applicazione (Kiosk Lock)**: Utilizzo della modalità *Lock Task* (Android Device Policy) o *Accesso Guidato* (iOS) per impedire a ospiti o bambini di uscire dall'interfaccia Domus.
- **Protezione della Batteria**: Notifica o comando intelligente collegato a una presa smart Shelly che stacca l'alimentazione del tablet quando la batteria raggiunge l'80% e la riattiva al 20%, prevenendo il rigonfiamento delle batterie al litio.


### 4.8 Funzionalità Avanzate dell'Ecosistema & Hardware (iOS vs Android)

La tabella seguente riassume la corrispondenza tecnologica tra le feature avanzate di iOS e i rispettivi framework nativi Android:

| Feature Avanzata | Implementazione Apple (iOS / iPadOS) | Corrispettivo Nativo Google (Android) |
| :--- | :--- | :--- |
| **Siri & Spotlight** | `AppShortcutsProvider`, `AppIntent` & CoreSpotlight | Google Assistant App Actions (`shortcuts.xml`), Assistant Interaction API & Android Global Search |
| **Live Activities & Dynamic Island** | `ActivityKit` (`ActivityAttributes`, `ActivityConfiguration`) | Ongoing Foreground Notifications con `MediaStyle` / Progress bar, Live Updates & Android 14+ Bubble API |
| **Infotainment Auto** | `CarPlay.framework` (`CPGridTemplate`, `CPListTemplate`) | Android Auto via Android for Cars App Library (`androidx.car.app:car-app`) |
| **Autenticazione Biometrica** | `LocalAuthentication` (`LAContext`, Face ID / Touch ID / Optic ID) | `androidx.biometric:biometric` (`BiometricPrompt` con `BIOMETRIC_STRONG` Class 3) |
| **Picture-in-Picture Video** | `AVKit` (`AVPictureInPictureController`) | Android PiP Mode (`enterPictureInPictureMode`, `PictureInPictureParams.Builder`) |
| **Micro-localizzazione Indoor** | `CoreLocation` (iBeacon BLE RSSI) & `NearbyInteraction` (UWB U1/U2) | `BluetoothLeScanner` con ScanFilter RSSI & Android UWB API (`androidx.core.uwb:uwb`) |
| **Layout Tablet Multi-Colonna** | `NavigationSplitView` (3-column sidebar/content/detail) | Jetpack Compose Adaptive Navigation Suite & `ListDetailPaneScaffold` (`WindowSizeClass`) |
| **Pianta 2D / Floorplan** | SwiftUI Canvas / Vector Blueprint con Shape interattive | Jetpack Compose `Canvas` con Path rendering & gesture detection |
| **Drag & Drop Dashboard** | SwiftUI `onDrag` / `onDrop` con Spring Animations & Haptic Ticks | Jetpack Compose Reorderable Grid (`LazyVerticalGrid` + Drag Gestures) |
| **VPN On-Demand Automatica** | `NetworkExtension` (`NEVPNManager`, On-Demand SSID Rules, WireGuardKit) | Android `VpnService` con `ConnectivityManager.NetworkCallback` auto-trigger |
| **Background Maintenance Tasks** | `BackgroundTasks` (`BGTaskScheduler`, `BGAppRefreshTask`, `BGProcessingTask`) | Jetpack `WorkManager` (`PeriodicWorkRequest` con vincoli `setRequiresCharging(true)`) |

#### 4.8.1 Siri & Spotlight via AppShortcutsProvider (iOS) vs Google Assistant App Actions (Android)
- **iOS**: Con `AppShortcutsProvider`, le frasi vocali ("Ehi Siri, spegni salotto con Domus", "Ehi Siri, attiva Buonanotte") vengono registrate automaticamente all'installazione senza configurazione manuale nell'app Comandi Rapidi. Le stanze e i dispositivi vengono indicizzati tramite `CSSearchableItem` per apparire all'istante nella ricerca globale Spotlight di iOS.
- **Android**: Definizione di `res/xml/shortcuts.xml` con capability standard (es. comandi Bixby/Google Assistant) che mappano su Intent nativi per eseguire scene e switch tramite voce ("Hey Google, accendi salotto con Domus").

#### 4.8.2 Live Activities & Dynamic Island (ActivityKit) vs Android Live Ongoing Notifications
- **iOS**: Con `ActivityKit`, l'app proietta widget dinamici nell'isola dinamica (Dynamic Island) e nella Schermata di Blocco:
  - *Ritardo Allarme*: Conto alla rovescia di 30 secondi con barra di avanzamento e pulsante disarmo diretto.
  - *Elettrodomestici*: Stato lavatrice/asciugatrice ("Fine ciclo tra 14 min") con icona rotante.
  - *Allarmi Critici*: Banner persistente ad alto contrasto per allagamento, fuga gas o intrusione finché non tacitato.
- **Android**: Notifiche persistenti ad alta priorità di tipo *Live Update* con animazioni di progresso continuo, bottoni di azione rapida e supporto a *Heads-Up Notifications* e Android Bubble.

#### 4.8.3 Apple CarPlay (CarPlay.framework) vs Android Auto (Car App Library)
- **iOS**: Integrazione di `CPApplicationDelegate` con template `CPGridTemplate` per comandi prioritari in auto (apertura basculante garage, sblocco cancello carraio, luci esterne vialetto, disarmo allarme perimetrale) e `CPListTemplate` per consultazione rapida dello stato domestico durante la guida con zero distrazioni.
- **Android**: Implementazione del servizio `CarAppService` con `Screen` basato su `androidx.car.app`, conformità alle linee guida di sicurezza alla guida di Google (Driver Distraction Guidelines) con template a griglia rapida per cancello e allarme.

#### 4.8.4 Autenticazione Biometrica Nativa (Face ID / Touch ID vs BiometricPrompt)
- **iOS**: Utilizzo di `LocalAuthentication` (`LAContext`) per disarmare l'allarme, aprire serrature smart o accedere a telecamere private tramite Face ID o Touch ID a 0 attrito, con fallback al PIN numerico.
- **Android**: `BiometricPrompt` di Jetpack configurato per richiedere biometria forte (Classe 3: impronta digitale sotto schermo o riconoscimento facciale sicuro 3D) con crittografia integrata tramite Android Keystore (`Cipher` autorizzato biometricamente).

#### 4.8.5 Streaming Video Picture-in-Picture (AVKit PiP vs Android PiP Mode)
- **iOS**: Tramite `AVPictureInPictureController`, l'utente può staccare il flusso video in streaming RTSP/WebRTC della telecamera o del videocitofono in una finestra fluttuante ridimensionabile, continuando a navigare nella dashboard di Domus o in altre app.
- **Android**: Attivazione di `Activity.enterPictureInPictureMode(PictureInPictureParams)` con aspect ratio 16:9 e azioni remote personalizzate nel riquadro PiP (es. tasto per aprire cancello o silenziare sirena).

#### 4.8.6 Micro-localizzazione Stanza per Stanza (iBeacon & UWB vs Android BLE & Nearby)
- **iOS**: Rilevamento di prossimità indoor tramite beacon BLE (`CLBeaconRegion`) posizionati nelle stanze (Salotto, Cucina, Camera da letto) e chip UWB Ultra-Wideband (`NearbyInteraction`) per distanze sub-metriche. La dashboard riordina e mette automaticamente in primo piano i controlli della stanza in cui l'utente si trova fisicamente.
- **Android**: Scansione BLE a basso consumo in background tramite `BluetoothLeScanner` con filtri su UUID/Major/Minor e integrazione con le nuove librerie `androidx.core.uwb` per dispositivi Google Pixel e Samsung Galaxy dotati di chip UWB.

#### 4.8.7 Layout Nativo iPadOS & Tablet (NavigationSplitView vs ListDetailPaneScaffold)
- **iOS**: Struttura `NavigationSplitView` a tre colonne:
  - *Colonna 1 (Sidebar)*: Elenco stanze, categorie funzionali e indicatori di stato globali (luci accese, temperatura media, allarme).
  - *Colonna 2 (Contenuto)*: Griglia principale delle card con supporto drag & drop.
  - *Colonna 3 (Dettaglio / Inspector)*: Feed telecamere live, termostato circolare o cronologia consumi.
- **Android**: Supporto tablet e pieghevoli (Foldables) tramite `ListDetailPaneScaffold` di Jetpack Compose Adaptive, calcolando il layout responsive in base a `WindowWidthSizeClass.Expanded`.

#### 4.8.8 Pianta 2D / Floorplan Interattivo
- **iOS**: Rendering grafico interattivo della planimetria della casa realizzato con SwiftUI `Canvas` e `Path` vettoriali. Le stanze cambiano colore e luminosità in tempo reale al variare delle luci; icone interattive consentono di toccare lampade, termostati e sensori direttamente sulla mappa con gesture di pan e zoom fluido.
- **Android**: Componente Compose `Canvas` con rendering ottimizzato di grafiche vettoriali SVG/Path, rilevamento di coordinate di tocco (Hit Testing) e animazioni di illuminazione sfumata con `animateColorAsState`.

#### 4.8.9 Personalizzazione Dashboard & Drag & Drop
- **iOS**: Riordinamento dinamico delle card preferite nella dashboard mediante gesture di trascinamento e rilascio, arricchite da animazioni elastiche (Spring animations) e feedback aptico con `UIImpactFeedbackGenerator`. L'ordine personalizzato viene salvato in cache locale.
- **Android**: Implementazione di griglia riordinabile in Jetpack Compose con animazioni `Modifier.animateItemPlacement()` e vibrazione aptica tramite `HapticFeedbackType.LongPress`.

#### 4.8.10 Connettività Sicura: VPN On-Demand Automatica (NetworkExtension vs VpnService)
- **iOS**: Configurazione tramite `NetworkExtension` (`NEVPNManager`) di un tunnel cifrato WireGuard/IPSec con regole on-demand (`NEOnDemandRuleConnect`). Il tunnel si attiva automaticamente non appena l'iPhone si disconnette dal Wi-Fi di casa, garantendo zero esposizione pubblica e cifratura end-to-end senza richiedere Cloudflare Tunnel o porte aperte sul router.
- **Android**: Servizio `VpnService` nativo collegato a `ConnectivityManager.NetworkCallback` che stabilisce il tunnel WireGuard sicuro in background alla perdita della connessione alla rete LAN locale.

#### 4.8.11 Manutenzione in Background (BGTaskScheduler vs WorkManager)
- **iOS**: Registrazione di task periodici tramite `BGTaskScheduler` (`BGAppRefreshTask` per aggiornamenti rapidi e `BGProcessingTask` durante la notte mentre il dispositivo è in carica): pulizia cache immagini telecamere, ricalcolo statistiche consumi orari e verifica diagnostica di integrità dell'hub.
- **Android**: Definizione di `PeriodicWorkRequest` tramite `WorkManager` con `Constraints.Builder().setRequiresCharging(true).setRequiresDeviceIdle(true)`, garantendo esecuzione a impatto zero sulle prestazioni e sulla batteria.

---

## 5. Piano di Implementazione & Roadmap a Fasi

```mermaid
gantt
    title Domus Native Companion Apps - Roadmap
    dateFormat  YYYY-MM-DD
    section Fase 1: Core & Networking
    Setup Repository & Clean Architecture :2026-11-01, 14d
    mDNS Discovery & Dual-Transport Socket :2026-11-15, 14d
    Auth, Biometrics & Local Cache Sync   :2026-11-29, 14d
    section Fase 2: UI & Interazione
    Dashboard & Stanza/Entità UI         :2026-12-13, 21d
    Pannello Allarme, Clima & Telecamere :2027-01-03, 14d
    section Fase 3: Sensori & Background
    Geofencing & Presenza Multi-Tier     :2027-01-17, 14d
    Notifiche Azionabili & Allarmi Critici :2027-01-31, 14d
    Motore Tag NFC Background             :2027-02-14, 10d
    section Fase 4: Widget & Wearables
    WidgetKit (iOS) & Glance (Android)    :2027-02-24, 14d
    watchOS & Wear OS Micro-Apps          :2027-03-10, 21d
    section Fase 5: Kiosk & Rilascio
    Tablet Kiosk Mode & CameraX Wake      :2027-03-31, 14d
    CI/CD Fastlane, TestFlight & Play Beta:2027-04-14, 14d
    section Fase 6: Ecosistema Avanzato
    Siri/Assistant, Live Activities & PiP :2027-04-28, 14d
    CarPlay & Android Auto Integration    :2027-05-12, 14d
    Indoor Beacons, Floorplan & VPN      :2027-05-26, 14d
```

### Dettaglio delle Fasi di Sviluppo

#### **Fase 1: Fondamenta, Rete & Sincronizzazione Locale (Mese 1)**
- Creazione dei repository `domus-ios` (Xcode 16, Swift 6) e `domus-android` (Android Studio Ladybug, Kotlin 2.1).
- Implementazione modulo mDNS per auto-discovery del server locale `_domus._tcp.local`.
- Modulo di rete WebSocket a due canali (LAN con IP diretto vs Remoto protetto con TLS e token crittografato nel Portachiavi iOS / Android Keystore).
- Sincronizzazione entità e cache locale offline-first (SwiftData / Room DB).

#### **Fase 2: Interfaccia Utente Reattiva & Controlli Domestici (Mese 2)**
- Riproduzione nativa ad altissime prestazioni delle schede entità:
  - Luci con selezione colore HSL e luminosità fluida.
  - Interruttori e prese con visualizzazione istantanea potenza (W) e grafico consumi.
  - Termostato circolare interattivo con feedback aptico per impostazione temperatura.
  - Tastierino allarme con codice PIN, inserimento parziale/totale e ritardo acustico.
  - Streaming telecamere live a bassissima latenza (RTSP / WebRTC nativo).

#### **Fase 3: Background, Geofencing, Notifiche & NFC (Mese 3)**
- Geofencing a basso consumo con integrazione della presenza multi-tier di Domus.
- Ricezione notifiche push APNs / FCM con bottoni di risposta rapida (Actionable Notifications).
- Abilitazione Allarmi Critici con bypass della modalità silenziosa su entrambi i sistemi operativi.
- Scansione e scrittura tag NFC con esecuzione scenari a schermo bloccato.

#### **Fase 4: Widget, Control Center & Smartwatch (Mese 4)**
- Widget iOS (WidgetKit con AppIntents) e Android (Glance) per controllo diretto da schermata home.
- Aggiunta controlli nel Centro di Controllo (iOS 18 `ControlWidget`) e nei Quick Settings (Android `TileService`).
- Sviluppo companion app watchOS e Wear OS con complicazioni e quadranti personalizzati.

#### **Fase 5: Modalità Kiosk, TestFlight & Google Play Beta (Mese 5)**
- Modalità Kiosk per tablet a parete con risveglio tramite fotocamera e protezione batteria.
- Pipeline CI/CD automatizzata tramite **Fastlane** e **GitHub Actions**:
  - Compilazione automatica e deploy delle build su **Apple TestFlight** per beta tester iOS.
  - Distribuzione automatica su **Google Play Console (Internal App Sharing & Open Testing)**.

#### **Fase 6: Ecosistema Avanzato di Sistema & Integrazione Hardware (Mese 6)**
- **Interazione Vocale & Ricerca Globale**: `AppShortcutsProvider` per Siri e indicizzazione Spotlight (iOS) e Google Assistant App Actions con `shortcuts.xml` (Android).
- **Live Activities & Dynamic Island**: Monitoraggio ritardo allarme, cicli elettrodomestici e allerte critiche su Dynamic Island / Lock Screen (iOS `ActivityKit`) e Ongoing Foreground Live Notifications (Android).
- **Infotainment Auto**: Template su cruscotto per cancello, garage e allarme tramite Apple CarPlay (`CPGridTemplate`) e Android Auto (`CarAppService`).
- **Autenticazione Biometrica**: Sblocco istantaneo allarme con Face ID / Touch ID (`LocalAuthentication`) e `BiometricPrompt` (Class 3) su Android con fallback a PIN.
- **Picture-in-Picture Streaming**: Finestra video fluttuante PiP per telecamere e videocitofono (`AVKit` / Android PiP Mode).
- **Micro-localizzazione Stanza per Stanza**: Rilevamento beacon BLE e UWB con prioritizzazione automatica dei dispositivi della stanza attuale nella dashboard.
- **Layout Tablet & Floorplan 2D**: `NavigationSplitView` a 3 colonne per iPadOS/macOS e `ListDetailPaneScaffold` per Android; rendering interattivo della pianta della casa con stato luci in tempo reale e drag-and-drop per riordinare le card.
- **Sicurezza di Rete & Manutenzione**: Tunnel VPN On-Demand automatico (`NetworkExtension` / `VpnService`) fuori dal Wi-Fi di casa e background maintenance con `BGTaskScheduler` e Android `WorkManager`.

