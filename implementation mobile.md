📱 Piano di Implementazione: App Mobile Native (iOS & Android)
Per garantire la massima integrazione con il sistema operativo (geofencing a bassissimo consumo energetico, widget rapidi sulla home screen, supporto comandi vocali e risposta istantanea), è definita la seguente strategia architetturale a Doppio Canale:

mermaid
graph TD
subgraph Mobile Device
UI[Native Shell / Web Engine Capacitor 6]
BG[Background Geolocation / Significant Motion]
WIDGET[Native Widgets: WidgetKit & Glance]
SEC[Keychain / Biometrics FaceID & Fingerprint]
DISC[Local mDNS Bonjour & SSDP Discovery]
end
subgraph Domus Server Local First
HTTP[Go HTTP REST API v1]
WS[WebSocket Hub /ws]
MDNS[mDNS Daemon domus.local]
end
UI -->|REST / Bearer Auth| HTTP
UI <-->|Real-time Events| WS
BG -->|POST /presence/gps| HTTP
WIDGET -->|Quick Scenes / Load Shedding Status| HTTP
DISC -->|Resolve \_domus.\_tcp.local| MDNS
Fase 1: Stack Tecnologico & Struttura del Progetto
Approccio Ibrido Avanzato (Capacitor 6 + Plugin Nativi):
Core UI: Condivisione al 100% dell'interfaccia Vue 3 + Tailwind/CSS + Pinia già ottimizzata (PWA ready), compilata in locale e servita dal bundle dell'app (zero lag di rete).
Integrazioni Native per Piattaforma:
iOS: Swift 5.10 / SwiftUI per Widget (WidgetKit), Complicazioni Apple Watch e Background Geofencing continuo.
Android: Kotlin 1.9+ / Jetpack Compose per Widget (Glance), Tile nei Quick Settings (TileService) e Background Geofencing con Google Play Services Location API.
Fase 2: Riconoscimento e Connessione Senza Configurazione (Zero-Conf mDNS)
In Rete Locale:
Il server Domus effettua il broadcast del servizio mDNS \_domus.\_tcp.local:8080.
All'apertura dell'app mobile, il discovery nativo (NSNetServiceBrowser su iOS, NsdManager su Android) individua istantaneamente il server locale.
Accesso Remoto Sicuro:
Integrazione con Tailscale / WireGuard o reverse proxy TLS (es. Caddy con Let's Encrypt).
L'app effettua il fallback trasparente sull'URL remoto pubblico quando il ping locale a \_domus.\_tcp.local non risponde.
Fase 3: Rilevamento Presenza in Background (Geofencing a Risparmio Batterico)
iOS (CoreLocation):
Utilizzo di CLCircularGeographicRegion centrata sulle coordinate di casa registrate nel server.
Sottoscrizione a significantLocationChangeMonitoring e alle notifiche di ingresso/uscita (didEnterRegion / didExitRegion).
All'attraversamento del recinto virtuale, il sistema operativo risveglia l'app in background per 30 secondi, inviando una richiesta autenticata POST /api/v1/presence/gps al server Domus.
Android (Google Play Services Geofencing):
Registrazione del geofence tramite GeofencingClient.addGeofences().
Gestione delle transizioni tramite BroadcastReceiver / WorkManager che non viene terminato dal risparmio energetico ("Doze Mode").
