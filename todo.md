1. Notifiche Push native (solo WebSocket attuale)
   Attualmente le notifiche arrivano solo via WebSocket, quindi se l'app è chiusa non arrivano nulla. Manca:

Integrazione con FCM (Android) e APNs (iOS)
Backend con token registrations per device
Canal prioritario per allarmi/intrusioni 2. Supporto Z-Wave (manca del tutto)
Zigbee e MQTT sono coperti, ma Z-Wave è lo standard diffusissimo per serrature, sensori allagamento, dispositivi medici. Serve:

Plugin zwavejs2mqtt via WebSocket
Mappatura entità equivalente a zigbee2mqtt 3. Thread/Matter nativo (solo stub)
Il codice ha CommissionMatter come stub nel server, ma Thread/Matter è il protocollo del futuro (Apple, Google, Amazon). Manca l'integrazione reale tramite matter.js o chip SDK.

4. Termostato intelligente (logica incompleta)
   ClimateCompensatorConfig esiste nel tipo TypeScript ma non c'è un servizio Go dedicato né un'interfaccia di configurazione. Manca:

Compensazione TRV (Better Thermostat)
Schedule settimanale temperatura per stanza
Integrazione con prezzi energia per pre-riscaldare nelle ore economiche
🟠 Media priorità — esperienza utente 5. Dashboard condivisibile / embedded (solo kiosk parziale)
Manca la possibilità di:

Generare URL pubblici (con token temporaneo) per dashboard singole
Embedding iframe sicuro per display touch dedicati
Layout portrait/landscape separati 6. Notifiche in-app con storico (effimere)
Le notifiche attuali spariscono. Manca:

Centro notifiche persistente con timestamp
Categorizzazione (sicurezza, energia, dispositivi)
Azioni rapide inline (es. "Disattiva allarme" direttamente dalla notifica) 7. Localizzazione geofence multipla (solo home)
Il sistema di presenza supporta solo GPS casa/fuori. Mancano:

Zone multiple (lavoro, scuola, etc.)
Automazioni basate su "quando arrivo in zona X"
Presenza per persona individuale vs presenza casa

🟡 Bassa priorità — differenziatori avanzati 9. Integrazione con assistenti vocali cloud (solo LLM locale)
Domus ha un assistente locale ottimo, ma molti utenti usano già Alexa/Google Home. Mancano:

Skill Alexa via Lambda + OAuth2
Smart Home Action per Google Home
Bridge HomeKit completo (non solo pairing) 10. Log energetico avanzato (metriche basilari)
Il sistema di energia gestisce tariffe e load shedding ma manca:

Storico consumi kWh per dispositivo
Report mensile PDF/CSV esportabile
Comparazione costo reale vs tariffe dinamiche
Integrazione con contatori smart (SMA, SolarEdge, Shelly EM) 11. Modalità vacanza / assenza estesa
Manca una modalità "house-sitter" con:

Simulazione presenza (luci random in orari verosimili)
Alert se sensori non rilevano movimento per N ore
Accesso temporaneo guest con scadenza automatica (parzialmente presente via invite) 12. Versioning delle automazioni
Attualmente una modifica sovrascrive tutto. Mancano:

Cronologia versioni per ogni automazione
Diff visuale tra versioni
Rollback a versione precedente 13. Marketplace estensioni / plugin store (stub presente)
StoreView e il backend /store/packages esistono come stub. Manca l'intera infrastruttura:

Registry pubblico di plugin
Firma crittografica dei pacchetti
Sandboxing WASM per plugin di terze parti (il plugin WASM esiste già!) 14. Integrazione con sistemi di domotica legacy
Mancano bridge per:

KNX (standard europeo edifici commerciali)
Modbus TCP (impianti industriali e FV)
DALI (illuminazione commerciale) 15. Analisi anomalie e manutenzione predittiva
Con la storia telemetrica disponibile, manca:

Rilevamento anomalie (dispositivo che consuma più del solito)
Alert "filtro HVAC da cambiare" basato su ore di funzionamento
Trend di degradazione batteria sensori

Rate limiting globale Esiste solo sul login; mancano limiti sugli endpoint di comando entità (possibile DoS)
OpenTelemetry / tracing Solo slog logging; manca distributed tracing per debug in produzione
Webhook in uscita Gli inbound webhook ci sono, ma mancano webhook outbound (chiamare URL esterni su eventi)
API GraphQL Per i client mobile, GraphQL permetterebbe query più efficienti rispetto a N chiamate REST
Test E2E frontend I test .test.ts esistono ma i .vue non sono stati compilati → coverage reale bassa

1. Verifica sui 5 Punti Specifici Richiesti
Funzionalità	Stato Reale nel Codice	Esito Verifica	Note Tecniche & Dettaglio
Rate Limiting Globale	Parziale (solo login)	❌ Non conforme	In 

server.go
 il middleware domusmw.RateLimit(srv.rateLimiter) è applicato esclusivamente a POST /api/v1/auth/login. Gli endpoint sensibili come POST /api/v1/entities/{id}/command e POST /api/v1/services/{domain}/{service} sono privi di throttling, esponendo attuatori e relay a possibili DoS o sovraccarichi.
OpenTelemetry / Tracing	Assente	❌ Non implementato	Nel backend non è presente alcuna libreria OpenTelemetry (go.opentelemetry.io/otel). Il sistema utilizza unicamente lo standard log/slog per logging testuale/JSON sincrono. Manca la propagazione del contesto di tracing W3C (traceparent) tra HTTP, chiamate a database SQLite e bus eventi interno.
Webhook in Uscita (Outbound)	Solo Inbound	❌ Non implementato	In 

webhook/service.go
 sono implementati solo gli webhook in ingresso (generazione URL /api/v1/webhook/{id} per ricevere chiamate esterne). Non esiste un motore di dispatching outbound che intercetti eventi (es. alarm.triggered, entity.state_changed) e li inoltri a URL esterni con firma crittografica HMAC-SHA256 e retry esponenziale.
API GraphQL	Assente	❌ Non implementato	Non c'è alcun layer GraphQL (es. gqlgen o schema .graphql). L'intera architettura dati è basata su REST (Chi v5) e WebSocket bidirezionale.
Test E2E Frontend	Solo Component Unit/Integration	⚠️ Parziale / Coverage reale limitata	La suite di test in web/ usa Vitest con @vue/test-utils ed emulatore DOM sintetico (happy-dom). I 241 test passano al 100%, ma non esiste una suite browser reale headless (come Playwright o Cypress) che testi il bundle Vite reale, cookie HttpOnly del browser e Web Worker/PWA.
2. Verifica Dettagliata delle Voci 1–15 di todo.md
Dall'audit del codice sorgente emergono importanti discrepanze positive (alcune funzionalità ritenute mancanti in todo.md sono in realtà già parzialmente o completamente implementate):

🔴 Funzionalità Realmente Mancanti o Incomplete
Notifiche Push Native (FCM / APNs):
In 

notification/actionable.go
 è già implementata la formattazione avanzata dei payload APNs (canali critici, suoni alarm.caf, azioni inline) e FCM (priorità alta).
Manca: Il client HTTP/2 verso gli endpoint di produzione Apple APNs e Google FCM v1 e la tabella di persistenza dei token dei device registrati.
Dashboard Condivisibile / Embedded (Fase 5):
In 

dashboard/service.go
 sono presenti solo operazioni CRUD di base. Mancano i token temporanei per generare URL pubblici protetti e i layout specifici per iframe.
Notifiche In-App con Storico Persistente (Fase 6):
Il frontend (

notifications.ts
) gestisce solo toast effimeri in memoria (con timeout di 5 secondi). Manca una tabella SQLite nel backend per lo storico persistente con categorizzazione (sicurezza, energia, domotica).
Geofencing Multiplo (Fase 7):


presence.go
 supporta solo HomeGeofence (raggio casa). Mancano zone multiple arbitrarie (lavoro, scuola, palestra) e automazioni basate su ingressi/uscite di zona specifica.
Assistenti Vocali Cloud (Fase 9):
Non esistono integrazioni Lambda/OAuth2 per Alexa Smart Home né Smart Home Action per Google Home (è presente solo l'assistente vocale locale LLM/Wyoming).
Log Energetico Avanzato (Fase 10):
Sono presenti load shedding, ottimizzatore dinamico e diagrammi Sankey, ma manca la cumulazione oraria/giornaliera dei kWh per dispositivo con esportazione CSV/PDF.
Modalità Vacanza / House-Sitter (Fase 11):
Non presente nel codice.
Versioning delle Automazioni (Fase 12):
Attualmente le automazioni vengono sovrascritte senza conservare una tabella di audit storico o rollback.
Marketplace Plugin / Store (Fase 13):


store/service.go
 è una lista statica in memoria. Mancano firma crittografica dei pacchetti e sandboxing WASM automatico.
Domotica Legacy KNX / Modbus / DALI (Fase 14):
Nessun bridge presente.
Analisi Anomalie Telemetriche (Fase 15):
Non implementata.
🟢 Funzionalità che risultavano mancanti in todo.md ma sono GIÀ Implementate
Supporto Z-Wave (Segnato come "manca del tutto" in todo.md):
In 

internal/plugins/zwave_js/
 è presente un client WebSocket completo con supporto a:
Binary Switch (CCBinarySwitch)
Multilevel Switch / Dimmer (CCMultilevelSwitch)
Sensori Multilevel (CCSensorMultilevel)
Misuratori di Consumo (CCMeter)
Notifiche ed Eventi (CCNotification)
Test di conformità validati in 

zwave_test.go
.
Thread / Matter Nativo (Segnato come "solo stub"):
In 

internal/plugins/matter/
 è implementato l'adapter con nodi, endpoint, cluster, commissioning e gestione comandi.
Termostato Intelligente & Compensazione TRV (Better Thermostat):
In 

internal/application/helper/service.go
 (DomainClimateCompensator) la logica di calcolo del setpoint compensato (CalculateCompensatedSetpoint) in base a temperatura stanza reale, fattore di calibrazione e sensore apertura finestra è già completamente implementata e funzionante.
3. Proposta di Piano di Implementazione per le Priorità
Quale di questi blocchi vuoi implementare per primo?

Sicurezza & Resilienza API:
Estendere il Rate Limiting Globale anche agli endpoint di comando entità e servizi (POST /api/v1/entities/{id}/command, POST /api/v1/services/{domain}/{service}) con un bucket token per client IP.
Implementare il motore di Webhook in Uscita (Outbound) con firma crittografica HMAC e worker pool asincrono.
Notifiche Push & Storico Notifiche:
Creare la tabella e il servizio per lo Storico Notifiche Persistente con categorizzazione e API REST per contrassegnarle come lette.
Completare il dispatching reale delle Notifiche Push Native APNs/FCM.
Observability & Testing:
Introdurre middleware OpenTelemetry per distributed tracing su HTTP, DB SQLite ed Event Bus.
Configurare Playwright per i test E2E reali su browser.


Controlla la 
### 7.1 Matrice Comparativa Funzionale[native_mobile_apps_implementation_plan.md](file;file:///c%3A/Users/chimi/Desktop/Programmazione/Domus/native_mobile_apps_implementation_plan.md) e implementa nei due progetti le funzionalità mancanti, sia ios sia android e svolgi ## 8. Piano Operativo di Allineamento e Completamento iOS (Feature Mancanti) e ### 9.2 Moduli di Presentazione Jetpack Compose (Parità Completa con iOS) e ## 10. Checklist Operativa di Validazione, Test di Regressione & Qualità del Codice