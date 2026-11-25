# Integrazione OpenDisplay in Domus

Questa guida riguarda i display e-paper OpenDisplay (BLE e, dove supportato, Wi-Fi) e il companion ufficiale-basato su `py-opendisplay`. Non è un'integrazione per un tablet Android generico o un kiosk.

## Architettura

```text
Domus REST API / automazioni
          |
   open_display.Adapter
          | HTTPS/HTTP + Bearer token
   opendisplay-bridge (Python 3.13)
          | BlueZ / BLE oppure LAN
       py-opendisplay
          |
     OpenDisplay e-paper
```

Il servizio Python separato contiene le dipendenze BLE/BlueZ; il processo Go Domus riceve soltanto discovery, stati e richieste azione tipizzate. Il companion va eseguito sulla macchina che vede i display via Bluetooth (in Docker Compose usa il profilo `opendisplay` e la rete host).

## Configurazione

Genera un token casuale almeno 32 byte, per esempio:

```bash
openssl rand -hex 32
```

Inserisci lo stesso valore in `DOMUS_OPENDISPLAY_BRIDGE_TOKEN`. Mantieni privato il file `.env`. Esempio:

```dotenv
DOMUS_OPENDISPLAY_ENABLED=true
DOMUS_OPENDISPLAY_BRIDGE_URL=http://127.0.0.1:10320
DOMUS_OPENDISPLAY_BRIDGE_TOKEN=INSERISCI_IL_TOKEN_GENERATO
DOMUS_OPENDISPLAY_DISCOVERY_INTERVAL=60s
DOMUS_OPENDISPLAY_BLE_SCAN_SECONDS=6
DOMUS_OPENDISPLAY_HTTP_TIMEOUT=45s
OPENDISPLAY_BRIDGE_BIND=127.0.0.1
OPENDISPLAY_BRIDGE_PORT=10320
OPENDISPLAY_BRIDGE_BLE_SCAN_SECONDS=6
OPENDISPLAY_BRIDGE_ENABLE_WIFI=true
```

Con Compose, il bridge è un servizio del profilo opzionale:

```bash
docker compose --env-file .env -f deployments/docker/docker-compose.yml --profile opendisplay up -d
```

Il bridge monta il socket D-Bus per BlueZ. In un'installazione nativa, avvia il companion con Python 3.13 e il file `deployments/opendisplay-bridge/requirements.txt`. Il token viene inviato nell'header Authorization e non deve mai comparire nei log.

### Chiavi BLE di display cifrati

Per un tag BLE legacy che richiede chiave, configura `OPENDISPLAY_BRIDGE_DEVICE_KEYS_JSON` come oggetto JSON MAC→chiave esadecimale:

```dotenv
OPENDISPLAY_BRIDGE_DEVICE_KEYS_JSON={"AA:BB:CC:DD:EE:FF":"0123456789abcdef0123456789abcdef"}
```

La chiave deve essere esattamente 16 byte (32 caratteri esadecimali). Mantienila privata. Le chiavi sono usate esclusivamente dal companion e non vengono inserite nei metadati dispositivo.

## Operazioni esposte

Gli endpoint sono sotto `/api/v1/integrations/opendisplay` e protetti da ruolo amministratore.

| Metodo | Percorso | Funzione |
|---|---|---|
| GET | `/devices` | Elenco dispositivi BLE/Wi-Fi scoperti e stato online |
| POST | `/devices/{address}/image` | Carica un'immagine raster |
| POST | `/devices/{address}/actions/led` | Attiva il LED |
| POST | `/devices/{address}/actions/buzzer` | Tono singolo |
| POST | `/devices/{address}/actions/melody` | Sequenza note |
| POST | `/devices/{address}/actions/nfc` | Scrive record NFC URL/testo/MIME o Home Assistant tag |
| POST | `/devices/{address}/actions/drawcustom` | Compone un layout ODL statico e invia l'immagine |

Esempio disegno:

```json
{
  "elements": [
    {"type":"text","x":"50%","y":22,"text":"Casa","font_size":24,"align":"center"},
    {"type":"line","x1":12,"y1":42,"x2":180,"y2":42},
    {"type":"text","x":12,"y":62,"text":"Temperatura: 21 °C"}
  ],
  "background":"white",
  "fit":"contain",
  "refresh_mode":"full",
  "dither_mode":"burkes",
  "rotation":0
}
```

La route `drawcustom` usa il renderer ODL ma limita tipi, numero di elementi, dimensioni e profondità JSON. Gli elementi che scaricano URL remoti (`dlimg`) sono intenzionalmente esclusi per evitare SSRF; anche i grafici con data provider non vengono ancora esposti.

Esempio LED:

```json
{
  "instance":0,
  "brightness":8,
  "color":[255,0,0],
  "flash_count":2,
  "loop_delay_ms":200,
  "inter_delay_ms":100,
  "repeats":3
}
```

Esempio buzzer:

```json
{"instance":0,"frequency_hz":1000,"duration_ms":200,"repeats":1}
```

Esempio NFC:

```json
{"record_type":"url","content":"https://example.com"}
```

Per `melody`, `notes` usa la notazione supportata da `BuzzerActivateConfig.melody` (note, indici, pause e durate); `record_type` per NFC è `url`, `text`, `mime` o `ha_tag`.

## Limiti attuali e passaggi successivi

- BLE discovery e discovery IP/mDNS; Wi-Fi senza TLS-PSK è supportato dal companion per upload, TLS-PSK restituisce 501 finché non viene integrato un keystore per-device.
- Il supporto alle chiavi è attualmente basato su una mappa environment JSON; la UI per pairing/backup cifrato va ancora realizzata.
- L'upload è immediato: non c'è ancora una coda persistente per tag in deep-sleep. La fase successiva dovrebbe introdurre una coda per device con scadenza, retry e drain al wake advertisement.
- I sensori/attributi attualmente importati dall'advertisement sono limitati; mancano mapping completo SHT40, pulsanti/touch, NFC status, firmware/OTA e telemetria completa.
- Il renderer espone un sottoinsieme ODL statico; non viene eseguito contenuto arbitrario né scaricata immagine remota.

Quindi il plugin offre già discovery, stati base, invio immagine, layout ODL statico e comandi LED/buzzer/NFC, ma non è ancora parity completa con tutte le funzioni dell'integrazione Home Assistant.
