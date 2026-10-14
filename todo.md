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
