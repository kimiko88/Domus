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
| Xiaomi Open Platform | `xiaomi_open_platform_transport` | No | OAuth e MIoT cloud |
| Midea M-Smart | `midea_a1_protocol` | No | Driver LAN V2/V3 per A1 |

Gli ultimi cinque provider sono stati collegati al manager come adapter di trasporto/protocollo. I loro handler HTTP mantengono temporaneamente la responsabilità dei flussi di configurazione e dei record legacy cifrati, ma ricevono il client/engine dal runtime supervisionato.

Gli ID con suffisso `_transport` o `_protocol` sono intenzionali: i record legacy hanno già tipi come `imou_life`, `vesync_cloud` e `xiaomi_home_local`, con JSON cifrato non compatibile con lo schema di configurazione runtime. Non si deve far decifrare quel JSON dal factory di un provider.

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

## Roadmap per ridurre davvero il core

1. **Completare questa tranche**: tutti i provider legacy passano dal runtime, con test per factory/lifecycle e nessuna regressione delle API.
2. **Unificare la configurazione**: migrare i record legacy nel modello di istanza del plugin con migrazioni versionate e rollback; segreti sempre nel vault cifrato.
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
