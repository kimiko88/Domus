# Architettura plugin e migrazione delle integrazioni Domus

## Obiettivo

Il core Domus deve occuparsi delle funzioni domotiche comuni e non conoscere i protocolli di ogni produttore. Le integrazioni devono essere moduli registrati, configurabili per istanza e supervisionati in modo uniforme. Il modello prende da HACS l'esperienza di repository e installazione, ma mantiene separati catalogo, runtime e sicurezza.

Non tutte le integrazioni hanno però la stessa natura. Un driver di un dispositivo, un provider meteo e un'azione per l'automazione hanno contratti diversi: non vanno forzati tutti dentro `plugins.Adapter`.

## Cosa deve rimanere nel core

- Modello canonico di dispositivi, entità, aree, unità di misura, availability e storico.
- Event bus, API di comando tipizzata, autorizzazione/RBAC, cifratura dei segreti e audit.
- Motore di automazioni, scene, scheduler, dashboard, notifiche e regole di sicurezza.
- Plugin Registry, persistenza delle istanze, supervisione, diagnostica e lifecycle.
- Contratti host versionati e migrazioni dei dati.

Il core non deve conoscere DP Tuya, group address KNX, registri Modbus, topic Zigbee2MQTT o il protocollo BroadLink. Deve ricevere eventi normalizzati e inviare comandi Domus tipizzati.

## Architettura consigliata

```text
                  Domus Core
  devices/entities/events/automations/security
                 | Host API
  +--------------+-------------------------------+
  |              |               |               |
AdapterPlugin  ServicePlugin  DiscoveryPlugin  UI/ActionPlugin
  |              |               |               |
Tuya/MQTT      Weather/Voice    Hue/Shelly      widgets/actions
KNX/Zigbee     cloud providers  pairing         automation
Modbus/Z-Wave  media providers  scanners        integrations
  |
  +-- Plugin Supervisor (health, restart, backoff, logs)
  +-- Encrypted instance store (config, credentials, status)
  +-- Marketplace (signed index, version, hash, compatibility)
```

### Contratti distinti

| Contratto | Scopo | Esempi |
|---|---|---|
| `Adapter` | Collega un gateway/dispositivo, pubblica entità e riceve comandi | MQTT, KNX, Modbus, Tuya Local |
| `ServiceProvider` | Fornisce una funzione a un servizio applicativo | Open-Meteo, STT/TTS, geocoding, cloud APIs |
| `DiscoveryProvider` | Ricerca e descrive dispositivi; il pairing resta orchestrato dal core | Hue, Shelly, Kasa, Tapo, Yeelight |
| `AutomationActionProvider` | Espone azioni tipizzate per l'editor automazioni | BroadLink learn/send, OCPP charge profile |
| `PanelExtension` | Aggiunge pagine/card UI dichiarative, senza eseguire script nel browser principale | dashboard add-on |

Per gli adapter hardware si usa `plugins.Adapter`. Per i servizi il core espone porte applicative tipizzate (per esempio `weather.ForecastProvider`): i provider implementano il contratto e il motore clima non dipende dall’SDK o dal vendor concreto. Le porte per voice, camera/NVR, media e geocoding restano da estrarre con lo stesso modello.

## Stato effettivo della migrazione

### Registrati nel Registry e gestiti dal Plugin Manager

| Tipo | Funzione | Note |
|---|---|---|
| `mqtt_generic` | MQTT generico | Più istanze, broker e topic indipendenti |
| `zwave_js` | Z-Wave JS UI | WebSocket autenticato, reconnect cancellabile, namespace entità per istanza |
| `weather_openmeteo` | Forecast provider | Contratto `weather.ForecastProvider`, provider predefinito sostituibile dal core clima |
| `tuya_cloud` | Tuya Cloud OpenAPI | Provider tipizzato e supervisionato; le credenziali App Authorization e i token Smart Life OAuth risiedono nella configurazione cifrata dell'istanza, separati per amministratore |
| `vesync_cloud` | VeSync Cloud | Istanza singola `integration_vesync_cloud`; token account, account ID e metadati regionali sono cifrati, mentre email/password restano input temporanei del login |
| `imou_life` | Imou Open Platform | Istanza singola `integration_imou_life`; project keys, token utente/amministratore, callback e configurazione push sono cifrati, i campi di sessione restano interni |
| `xiaomi_home_cloud` | Xiaomi Open Platform / MIoT Cloud | Istanza singola con ID stabile `integration_xiaomi_home_cloud`; App Secret, callback e token OAuth sono cifrati nel Plugin Manager e i campi interni non sono esposti nel catalogo |
| `midea_smarthome_local` | Midea / Comfee M-Smart LAN | Istanza singola `integration_midea_smarthome`; configurazioni e chiavi M-Smart V3 per dispositivo cifrate, driver A1 V2/V3 |
| `zigbee2mqtt` | Gateway Zigbee2MQTT | Un'istanza per bridge/base topic |
| `esphome_mqtt` | Nodi ESPHome via MQTT | Un'istanza per broker/prefix |
| `knx` | KNXnet/IP | Un'istanza per gateway e mapping di group address |
| `modbus_tcp` | Modbus TCP | Un'istanza per gateway e tabella registri |
| `tuya_local` | Controllo LAN Tuya | Il runtime crea l'adapter, ma storage legacy dei dispositivi va ancora isolato per istanza |
| `broadlink_local` | IR/RF BroadLink | Il runtime crea l'adapter, ma la tabella dispositivi/comandi legacy è condivisa: mantenere una sola istanza fino alla migrazione storage |
| `open_display` | Display e-paper | Un companion per istanza Domus; può gestire molti display fisici |
| `ble_proxy` | Advertisement BLE/BTHome/iBeacon | L'ingresso dei pacchetti deve essere collegato al trasporto proxy scelto |
| `voice_satellite` | Satellite microfono/speaker Linux | Multi-istanza per stanza, token dedicato, ALSA e Wyoming wake-word |
| `matter` | Commissioning/controller Matter | **Sperimentale**: il controller di rete Matter non è ancora un'implementazione completa interoperabile |

Il campo `multi_instance` del manifest abilita le istanze multiple solo quando il plugin ha storage e stato isolati. Al momento la proprietà è attiva per MQTT Generic, Zigbee2MQTT, ESPHome MQTT, Z-Wave JS UI, KNX, Modbus TCP e BLE Proxy. Matter, OpenDisplay, Tuya Cloud, Xiaomi Home Cloud, Imou Life, VeSync Cloud e Midea / Comfee restano singola istanza; Tuya Local e BroadLink restano singola istanza finché le tabelle legacy dei dispositivi non vengono separate per integration ID. Il Manager salva le configurazioni cifrate nella tabella integrazioni, ripristina le istanze abilitate all'avvio e ne gestisce enable/disable/restart/delete. Le route storiche Tuya Local, BroadLink e OpenDisplay sono mantenute per compatibilità.

### Già presenti nel repository, ma non ancora migrati nel Registry

| Integrazione | Codice attuale | Lavoro per renderla plugin |
|---|---|---|

| HomeKit Bridge | `internal/plugins/homekit` | Prima di registrarlo completare HAP server reale/advertising, pairing persistente, unsubscribe e protezione setup code; il bridge attuale non è ancora un servizio HAP completo |
| Smart Life OAuth | `internal/plugins/tuya_cloud/oauth.go`, `TuyaOAuthHandler` | Account linking, callback e refresh sono mantenuti dal handler; token access/refresh migrati al campo interno cifrato `smart_life_oauth` dell'istanza Tuya, con separazione per amministratore. La tabella legacy viene usata solo per migrazione al primo accesso; lo state OAuth resta nel DB per CSRF |
| Tuya Cloud option flow | `TuyaCloudHandler` + Plugin Manager | GET/PUT/DELETE ora leggono e aggiornano la configurazione cifrata dell'istanza; POST resta alias per il frontend legacy. `tuya_cloud_config` viene migrata una tantum e rimossa solo dopo persistenza riuscita. DELETE rimuove solo le credenziali OpenAPI e preserva l'istanza OAuth e i token Smart Life |
| Xiaomi Home locale (miIO) | `internal/plugins/xiaomi_home` + handler locale | Adapter di trasporto già supervisionato; storage legacy dei device/token ancora da isolare per istanza. La parte Open Platform Cloud è invece migrata in `xiaomi_home_cloud` |
| Imou Life | `imou_life` + handler compatibile | Istanza persistente del Plugin Manager; progetto, token utente e amministratore, callback e push vengono cifrati nell'ID storico `integration_imou_life` |
| OCPP | `internal/plugins/ocpp` | Separare il protocollo WebSocket per charger dall'algoritmo di modulazione, che resta nel core energia |
| Firewall/router | `internal/plugins/firewall` | Provider plugin per UniFi/OPNsense/MikroTik; le policy quarantine restano nel core |
| Provider cloud generico | `internal/plugins/cloud` | Usarlo come utilità circuit-breaker comune, non come integrazione vendor unica |

### Integrazioni vendor da estrarre da `application/brand`

Il catalogo brand attuale include driver/probe per Philips Hue, Shelly, TP-Link Kasa/Tapo, Nanoleaf e Yeelight (oltre ai bridge già supportati). La migrazione corretta è un `DiscoveryProvider`/`Adapter` per vendor: probe, autenticazione, pairing e comandi specifici nel plugin; entità canoniche, aree, autorizzazioni e UX generica nel core.

### Altri provider di servizio candidati

- Meteo: `weather_openmeteo` implementa `weather.ForecastProvider`; il calcolo previsionale resta nel core clima e altri provider potranno implementare lo stesso contratto.
- STT/TTS: Wyoming/Whisper/Piper diventano provider vocali; la pipeline di intenti, contesto stanza e autorizzazioni rimane nel core assistant.
- Telecamere/NVR: ONVIF/RTSP/vendor API sono provider plugin; registrazione, timeline, utenti e policy restano servizi core.
- Media multi-room: driver AirPlay/Cast/DLNA sono provider plugin, mentre playlist, gruppi e scheduler restano nel core media.

## API correnti del Plugin Manager

Le route amministrative sono sotto `/api/v1/plugins`:

| Metodo | Endpoint |
|---|---|
| GET | `/types` — manifest e schema configurazione dei tipi built-in |
| GET | `/instances` — istanze senza configurazioni o segreti |
| POST | `/instances` — crea un'istanza e salva la configurazione cifrata |
| PUT | `/instances/{id}/config` — valida, cifra e sostituisce la configurazione con rollback dell'adapter |
| POST | `/instances/{id}/enable` |
| POST | `/instances/{id}/disable` |
| POST | `/instances/{id}/restart` |
| DELETE | `/instances/{id}` |

Le route richiedono ruolo amministratore. La configurazione in chiaro non viene restituita dalla API di elenco.

## Esempio di configurazione MQTT

```http
POST /api/v1/plugins/instances
Content-Type: application/json

{
  "id": "mqtt_second_floor",
  "type": "mqtt_generic",
  "name": "MQTT Secondo Piano",
  "enabled": true,
  "config": {
    "broker_url": "tcp://192.168.1.20:1883",
    "username": "domus",
    "password": "secret",
    "state_topic_pattern": "second_floor/+/state",
    "command_topic_template": "second_floor/%s/set"
  }
}
```

È possibile creare un secondo broker MQTT con un altro `id`, indirizzo e topic. Ogni istanza ha il proprio stato, client, configurazione e lifecycle.

## Sicurezza e confini di esecuzione

- Il Registry attuale carica factory Go già compilate nel binario: non effettua `dlopen`, non compila codice e non esegue script scaricati dal marketplace.
- Le capability dei plugin Go nativi sono dichiarazioni e guida per la revisione; **non sono un sandbox OS**. Un plugin Go ha i privilegi del processo Domus.
- I pacchetti community dovranno usare un runtime WASI in processo separato, con API host esplicite, limiti CPU/memoria/fuel, filesystem temporaneo e rete per allowlist.
- Le credenziali vanno marcate `Secret` nel manifest, cifrate prima della persistenza, mascherate nei log e mai incluse in `Instance` o manifest pubblici.
- Il core non deve passare `*sql.DB`, filesystem root o `exec.Command` ai plugin community. I plugin nativi ufficiali possono usare dipendenze host interne, ma anche lì è preferibile una Host API più stretta.
- La rimozione di un plugin deve fermare worker e subscription prima di cancellare la configurazione; la rimozione di una integrazione non deve cancellare dispositivi/entità senza una policy esplicita di retention.

## Roadmap consigliata

1. **Chiudere il runtime adapter**: test contratto, stato health, lifecycle concorrente, installazione/disabilitazione atomica, isolamento storage per istanza, diagnostica e API config update.
2. **Completare i contratti già avviati**: Z-Wave JS e Voice Satellite sono già adapter registrati; completare HomeKit solo dopo il server HAP reale, pairing e advertising.
3. **Completare e migrare provider vendor**: Tuya Cloud OpenAPI/Smart Life OAuth, Xiaomi Home Cloud, Imou Life, VeSync e Midea/Comfee sono gestiti da istanze Plugin Manager con configurazioni cifrate e campi interni protetti. Midea usa il tipo pubblico `midea_smarthome_local`, conserva l'ID storico `integration_midea_smarthome` e gestisce i token/key V3 per dispositivo; il controllo resta LAN e non richiede un account cloud.
4. **Estrarre i driver brand**: Hue, Shelly, Kasa/Tapo, Nanoleaf, Yeelight dalla scansione brand monolitica a provider installabili.
5. **Separare plugin di servizio**: meteo, voce, media, NVR e firewall con contratti specifici.
6. **UI Store/Integrations unificata**: browse, install, option-flow, pairing, status, log e restart; mantenere redirect/API di compatibilità durante la migrazione.
7. **Runtime community WASI**: solo dopo avere completato host API e capability enforcement; firma del pacchetto, SBOM, hash, sandbox e rollback obbligatori.

## Test che devono accompagnare ogni plugin

- Contract test: ID/Type, manifest, config schema, secret masking, lifecycle Start/Stop/Restart idempotente.
- Unit test del protocol parser/serializer e fuzz test per i frame non fidati.
- Integration test con broker/gateway simulato: reconnect, timeout, malformed payload, device offline e command acknowledgement.
- Concurrency/race test per stop durante discovery/polling e due istanze dello stesso tipo.
- Security test: SSRF, credenziali errate/non loggate, URL private/LAN allowlist, payload e response size limits.
- Benchmark per parser, mapper e discovery, soprattutto sul Raspberry Pi.
- E2E: crea istanza, collega gateway simulato, riceve stato, invia comando, disabilita, riabilita, riavvia Domus e verifica il ripristino.

## Stato test introdotti in questa tranche

- Test dei manifest built-in e presenza dei protocolli registrati, inclusa la factory Z-Wave JS UI.
- Test Z-Wave su connessione WebSocket, comandi, namespace per istanza e ripristino della connessione dopo uno stop/riavvio.
- Test di creazione typed per MQTT Generic, Zigbee2MQTT, ESPHome MQTT, KNX, Modbus TCP, Matter sperimentale, BLE Proxy, Tuya Local, BroadLink e OpenDisplay.
- Test con due istanze MQTT separate e test di rifiuto configurazioni errate.
- Test Manager per configurazione cifrata, restore dopo riavvio, restart, enable/disable/delete e secret masking.
- Test Smart Life OAuth per migrazione dei token legacy nel config cifrato dell'istanza, isolamento per amministratore, rimozione selettiva al disconnect e assenza dei token nelle risposte pubbliche.
- Test Tuya OpenAPI option flow per migrazione `tuya_cloud_config`, creazione su installazione nuova, GET/PUT/POST/DELETE, secret masking e preservazione dei token OAuth dopo la rimozione delle credenziali OpenAPI.
- Test che i campi interni OAuth sono validati dal Registry ma rimossi dallo schema restituito al catalogo pubblico.
- Test lifecycle per Matter, BLE Proxy, BroadLink, OpenDisplay e operazioni dinamiche del Supervisor.

Questi test non equivalgono ancora a test fisici con ogni produttore: i protocolli hardware vanno collaudati anche su dispositivi reali prima di dichiarare la compatibilità di un modello.