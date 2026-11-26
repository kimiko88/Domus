# Migrazione delle integrazioni Domus verso un runtime a plugin

## Principio di progettazione

Un'integrazione è un plugin quando contiene un protocollo, un provider o una dipendenza da un sistema esterno. Il dominio domotico rimane nel core.

Il core continua a gestire:
- modello canonico Device/Entity/Area e repository;
- Event Bus, automazioni, scene, script e template entity;
- storico, aggregazioni, retention, energia e statistiche;
- utenti, ruoli, sessioni, vault dei segreti e audit;
- API REST/WebSocket, UI, notifiche, scheduling e regole di sicurezza;
- servizi applicativi che coordinano più integrazioni.

I manifest possono dichiarare `internal: true` per gli adapter tecnici usati da altri plugin o da handler legacy. Questi tipi restano nel Registry e possono essere preparati dal runtime, ma `Manager.ListTypes()` li esclude dal catalogo utente e `Manager.Create()` ne rifiuta la creazione manuale.

Un plugin gestisce:
- discovery/pairing e connessione al dispositivo o al cloud;
- framing, autenticazione specifica, retry e riconnessione;
- polling, sottoscrizioni ed eventi vendor;
- conversione dei comandi Domus nei comandi del protocollo;
- mapping di capability native verso entità Domus.

Non creare un plugin per ogni funzione del core, né un plugin diverso per ogni modello dello stesso protocollo: un plugin deve rappresentare un provider/protocollo e può creare più istanze quando ha senso.

## Situazione delle integrazioni

| Famiglia | Tipo runtime | Istanza multipla | Ruolo |
|---|---|---:|---|
| MQTT generico | `mqtt_generic` | Sì | Broker e topic arbitrari |
| Zigbee2MQTT | `zigbee2mqtt` | Sì | Bridge Zigbee via MQTT |
| ESPHome MQTT | `esphome_mqtt` | Sì | Nodi ESPHome via MQTT |
| KNXnet/IP | `knx` | Sì | Router, group address e mapping |
| Modbus TCP | `modbus_tcp` | Sì | Inverter, meter e PLC |
| Z-Wave JS UI | `zwave_js` | Sì | Controller Z-Wave |
| Matter | `matter` | No | Controller/commissioning (ancora sperimentale) |
| BLE Proxy | `ble_proxy` | Sì | Advertisement inoltrati |
| Tuya Local / Cloud | `tuya_local`, `tuya_cloud` | Dipende dal provider | Protocollo locale e OpenAPI |
| BroadLink | `broadlink_local` | No | IR/RF locale |
| OpenDisplay | `open_display` | No | E-paper BLE/LAN via companion isolato |
| Domus Voice Satellite | `voice_satellite` | Sì | Microfono, wake-word e speaker |
| Imou Life transport | `imou_life_transport` | No | Trasporto HTTP per API e OAuth |
| VeSync transport | `vesync_transport` | No | Client cloud |
| Xiaomi miIO local | `xiaomi_miio_transport` | No | Trasporto UDP LAN |
| Xiaomi Open Platform | `xiaomi_home_cloud` | No | Istanza persistente; OAuth, callback e MIoT con segreti cifrati dal Plugin Manager |
| Midea M-Smart | `midea_a1_protocol` | No | Driver LAN V2/V3 per A1 |

Imou Life, VeSync, Xiaomi miIO locale e Midea restano adapter interni di trasporto/protocollo: i rispettivi handler mantengono ancora parte dei flussi di configurazione e dei record cifrati legacy. Xiaomi Open Platform è invece un plugin pubblico e persistente: il suo record storico `integration_xiaomi_home_cloud` viene ripristinato come istanza gestita, senza cambiare l'ID usato da device ed entità.

Gli ID con suffisso `_transport` o `_protocol` sono intenzionali per adapter tecnici che non possiedono ancora uno schema di configurazione completo. I record Xiaomi Cloud sono l'eccezione: il tipo `xiaomi_home_cloud` usa deliberatamente l'ID stabile `integration_xiaomi_home_cloud` e una configurazione Open Platform cifrata validata dal Registry.

## Altri candidati, e cosa non spostare

### Candidati a plugin successivi

- **HomeKit**: plugin che gestisce pairing, accessory database e sessioni HAP; core riceve solo entità e comandi normalizzati.
- **OCPP / wallbox**: plugin per central system, websocket OCPP, sessioni di ricarica, meter values e comandi; il core mantiene energia, costi, automazioni e dashboard.
- **Provider camera/NVR** (ONVIF, RTSP, vendor cloud): plugin trasporto per discovery, stream URI, eventi e snapshot; il core mantiene timeline, permessi, registrazione e UI.
- **Caldaie, HVAC e climatizzatori**: plugin per API LAN/cloud o protocolli seriali; il core continua a governare zone, schedule e logica climatica.
- **Meter/inverter aggiuntivi**: plugin per Modbus/SunSpec/API cloud; il servizio energia rimane nel core.
- **Altri servizi cloud** (calendari, meteo, media): plugin provider, con i servizi applicativi e i modelli canonici nel core.

### Non plugin

- automazioni, scene, script, aree, dashboard e pannelli;
- cronologia, retention, aggregazioni e statistiche;
- utenti, autorizzazioni, sessioni, audit e cifratura;
- motore di sicurezza, failover, notifiche e schedulazione.

Sono responsabilità centrali, indipendenti dal produttore o dal protocollo.

## Contratto e lifecycle

Ogni provider registrato dichiara:
- ID di tipo stabile e versione del manifest;
- schema JSON di configurazione con segreti marcati come tali;
- capability richieste;
- supporto multi-instance;
- factory che crea un adapter senza iniziare lavoro in background.

Il manager:
1. valida configurazione e molteplicità;
2. cifra e persiste le istanze create dalla UI;
3. registra e avvia l'adapter tramite Supervisor;
4. aggiorna stato e diagnostica;
5. ferma e rimuove l'istanza quando viene disabilitata/eliminata.

Gli handler di compatibilità possono usare `Prepare` per ottenere un provider condiviso durante la migrazione; le nuove integrazioni dovrebbero usare `Create` e il flusso di configurazione standard della UI.

## Verifica dei candidati non ancora pronti

Il repository contiene anche pacchetti che sembrano integrazioni ma non vanno ancora pubblicati come plugin di prodotto:

- **HomeKit** (`internal/plugins/homekit`): oggi mappa entità Domus in accessori e inoltra eventi, ma non espone ancora un listener HAP completo, pairing persistente e gestione sicura delle sessioni. Prima di attivarlo dal Plugin Store servono il vero server HAP, storage delle coppie/chiavi, reset e test con Apple Home.
- **Firewall** (`internal/plugins/firewall`): ha client per UniFi, OPNsense e MikroTik, ma non è collegato al flusso Safety/Quarantine né a route di configurazione. È un ottimo plugin futuro, da attivare soltanto dopo aver definito un contratto di isolamento, audit, verifica dello stato e rollback fail-safe.
- **OCPP** (`internal/plugins/ocpp`): il controller presente calcola la corrente in base a surplus FV e potenza contrattuale ed è usato dall'handler delle funzionalità avanzate. Questa policy deve restare nel core energia. La connessione a charge point/stazioni OCPP (WebSocket, autenticazione, BootNotification, Heartbeat, MeterValues, SetChargingProfile) deve invece essere un plugin di trasporto separato, con istanze per stazione.
- **Cloud generico** (`internal/plugins/cloud`): le implementazioni correnti sono provider semplificati e non sostituiscono i client vendor completi. Non vanno usati come scorciatoia per bypassare OAuth, firma, refresh token e gestione credenziali già presenti nei plugin specifici.
- **Camera/NVR**: il servizio camera, i permessi, le registrazioni, la timeline e gli eventi rimangono core. In futuro ONVIF, RTSP, vendor cloud e NVR esterni possono essere adapter separati che espongono stream e capability normalizzati. Non spostare il motore delle registrazioni nel plugin.

Questa distinzione è essenziale: un adapter che compila o ha qualche test unitario non equivale a un'integrazione end-to-end pronta per gli utenti.

## Roadmap per ridurre davvero il core

1. **Completare questa tranche**: i provider legacy residui passano dal runtime, con test per factory/lifecycle e nessuna regressione delle API. Tuya Cloud e Xiaomi Home Cloud dispongono già di istanze persistenti e option flow cifrati.
2. **Unificare la configurazione**: migrare i record rimanenti nel modello di istanza del plugin con migrazioni versionate e rollback; segreti sempre nel vault cifrato. Xiaomi Cloud conserva il proprio ID storico per non invalidare i riferimenti di device ed entità.
3. **Option flow**: ogni manifest dichiara pagine di setup, validazione remota, pairing, selezione device e reconfigure; il frontend genera wizard da uno schema versionato.
4. **Eventi e comandi uniformi**: capability esplicite, state schema, device discovery e command request tipizzati. Gli handler specifici restano sottili o diventano facade del runtime.
5. **Community runtime**: plugin WASI in processo separato con host-call allowlist effettiva, limiti CPU/memoria/fuel, rete consentita per capability e file system effimero. Le capability del manifest da sole non sono un sandbox.
6. **Store**: firma dei repository, SBOM, compatibilità, changelog, backup, aggiornamento atomico e rollback.

## Regole importanti

- Non caricare plugin Go/so o script community scaricati dinamicamente nel processo Domus.
- Non far leggere ai plugin direttamente SQLite o le chiavi master; usare host API.
- Non registrare credenziali, token, URL firmati, stream URL temporanei o payload privati nei log.
- Le capacità dichiarate vanno verificate e applicate dal runtime, non soltanto mostrate nella UI.
- Ogni adapter deve fermare goroutine/socket nel `Stop`, rispettare `context.Context` e avere test di timeout, reconnect, errori e concorrenza.
