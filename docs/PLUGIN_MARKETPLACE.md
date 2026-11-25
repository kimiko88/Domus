# Domus Plugin Store: repository e sviluppo

Domus prende da HACS il modello di repository aggiungibili dall'utente, catalogo aggiornabile e pacchetti versionati, ma non copia il meccanismo di esecuzione di componenti Python dentro il processo principale.

## I tre piani

1. **Repository/catalogo**: pubblica metadati, requisiti Domus, capability dichiarate, release e hash degli artefatti.
2. **Plugin Manager**: aggiunge repository HTTPS, aggiorna l'indice, verifica firme e hash, controlla compatibilità, installa e rimuove artefatti.
3. **Runtime/integrazione**: crea istanze configurate, controlla permessi e lifecycle, traduce protocolli hardware nelle entità ed eventi Domus.

Il catalogo e il runtime sono deliberatamente separati: la presenza di un pacchetto nel catalogo non significa che il codice venga eseguito.

## Pubblicare un repository

Un repository è un URL HTTPS stabile che espone questi file:

```text
https://plugins.example/domus/
  index.json
  index.sig
  artifacts/
    acme-door-1.2.0.wasm
```

`index.json` è un documento JSON con schema versione 1. La firma Ed25519 detached in `index.sig` è la codifica Base64 della firma sui byte esatti di `index.json` (non sul JSON riformattato).

Esempio ridotto:

```json
{
  "schema_version": 1,
  "repository": {
    "id": "acme-home",
    "name": "Acme Home",
    "owner": "acme",
    "key_id": "acme-release-2026"
  },
  "packages": [
    {
      "id": "acme.door",
      "name": "Acme Door Bridge",
      "type": "wasm_plugin",
      "version": "1.2.0",
      "author": "Acme",
      "description": "Bridge locale per serrature Acme",
      "license": "MIT",
      "runtime": "wasi",
      "min_domus_version": "1.0.0",
      "capabilities": ["devices:read", "events:read", "network:lan"],
      "artifact": {
        "url": "https://plugins.example/domus/artifacts/acme-door-1.2.0.wasm",
        "sha256": "64-caratteri-esadecimali-SHA256",
        "size": 184320
      }
    }
  ]
}
```

Campi e vincoli attualmente controllati dal manager:
- `schema_version` deve essere 1; massimo 250 pacchetti per indice e 1 MiB per indice.
- Il pacchetto ha ID, nome e versione SemVer; i tipi ammessi sono `wasm_plugin`, `blueprint`, `widget`, `integration`.
- Il runtime dichiarato è `wasi` o `data`. `builtin` è riservato e non installabile da repository esterni.
- URL artefatto HTTPS sullo stesso host/porta del repository; dimensione e SHA-256 devono corrispondere.
- La compatibilità è determinata da `min_domus_version` e `max_domus_version`.

La chiave privata di firma resta sul sistema di release/CI del publisher. Su Domus si installa solo la chiave pubblica Ed25519, non la privata.

## Configurare Domus

Le impostazioni sono in `.env` (partendo da `.env.example`):

```dotenv
DOMUS_VERSION=1.0.0
DOMUS_STORE_TRUSTED_KEYS_JSON={"acme-release-2026":"BASE64_ED25519_PUBLIC_KEY"}
DOMUS_STORE_ALLOW_UNTRUSTED_REPOSITORIES=false
DOMUS_STORE_INSTALL_ROOT=./data/marketplace
DOMUS_STORE_MAX_ARTIFACT_BYTES=16777216
```

La chiave pubblica deve essere Base64 della chiave Ed25519 raw da 32 byte. L'account amministratore aggiunge l'URL del repository nella pagina Store e seleziona **Sincronizza**. Il server verifica firma, schema, compatibilità e provenienza prima di mostrare i pacchetti. I repository senza firma sono rifiutati, a meno che l'amministratore abiliti esplicitamente `DOMUS_STORE_ALLOW_UNTRUSTED_REPOSITORIES=true`; per dispositivi domestici è sconsigliato.

## Cosa fa oggi Install

Il flusso presente:
1. scarica l'indice e la firma con HTTPS;
2. verifica firma Ed25519 usando una chiave locale fidata;
3. valida schema, compatibilità, origine URL, SHA-256 e dimensione;
4. scarica l'artefatto in un file temporaneo, lo sincronizza e lo rinomina nell'area privata del Plugin Store;
5. registra lo stato di installazione nel repository SQLite;
6. alla rimozione controlla che il percorso sia dentro la root configurata.

**Importante:** l'installazione attuale mette gli artefatti in staging. Non carica librerie Go native, non esegue script scaricati e non attiva automaticamente un modulo WASM. L'esecuzione di pacchetti di community richiede un runtime WASI vero e isolato, collegato al manager tramite un protocollo host ristretto. Il parser/scaffolding in `internal/plugins/wasm` non è ancora un sandbox di produzione.

## Creare una nuova integrazione nativa

Le integrazioni ufficiali o sviluppate localmente implementano `plugins.Adapter`:

- `ID()`: ID univoco dell'istanza;
- `Type()`: tipo stabile corrispondente al manifest;
- `Start(ctx)`: apre connessioni e avvia worker;
- `Stop(ctx)`: cancella context, chiude socket e attende i worker;
- `HealthCheck(ctx)`: verifica la connessione reale.

Definisci un `plugins.Manifest` con versione API, ID, versione SemVer, capability e schema config. Registra la factory con `plugins.NewRegistry().Register(...)`. La factory decodifica con `plugins.DecodeConfig[T]`, valida l'istanza e restituisce un adapter non ancora avviato. Il chiamante lo passa poi al supervisore/lifecycle manager.

Per i plugin nativi le capability sono metadati dichiarativi, non una barriera OS: il codice ha i privilegi del processo Domus. Per questo non si devono installare e caricare dinamicamente binari Go di terze parti dal marketplace.

## OpenDisplay come plugin ufficiale

OpenDisplay è un esempio concreto del contratto:
- `internal/plugins/open_display` implementa il modello Adapter e si registra tramite `RegisterPlugin(registry)`;
- la sua factory riceve URL bridge, token, intervallo scansione, tempo BLE e timeout HTTP;
- il bootstrap Domus crea l'adapter tramite `Registry.Create`, anziché istanziarlo bypassando il registro;
- il companion Python `deployments/opendisplay-bridge` gestisce BLE e usa `py-opendisplay`; Domus non riceve privilegi Bluetooth/BlueZ;
- il bridge richiede Bearer token, controlla input e offre solo azioni esplicite, non una console di comandi.

Il modello corretto per nuove integrazioni è mantenere il core piccolo, delegare al plugin il protocollo e pubblicare soltanto entità, stati, capacità e comandi tipizzati.

## Come evolverei il prodotto verso una UX HACS

Le prossime tappe utili sono:
- **Repository ufficiali e community** con badge fiducia, manutentore, versione, compatibilità, changelog e ultime release.
- **Scheda plugin** con README/immagini, configurazione guidata, capability richieste e dispositivo supportato.
- **Update manager** con confronto versioni, backup della configurazione, download atomico, rollback e notifiche.
- **Option flow** in UI per API key, endpoint, pairing, rete e selezione device; segreti salvati cifrati nel vault Domus e mai ritornati alle API.
- **Runtime WASI isolato** in processo separato con limiti CPU/memoria/fuel, capability applicate dal runtime, filesystem temporaneo e rete per allowlist.
- **Health e diagnostica** per istanza: stato, ultima sincronizzazione, ultimo errore sanificato, log filtrati e pulsante restart.
- **Test e firma della release** obbligatori nel repository: unit, integration, fuzz, SBOM, hash, firma e compatibility matrix.

Questa è l'architettura che mantiene il modello di comodità di HACS senza trasformare l'installazione di un'integrazione domotica in esecuzione di codice non verificato con i privilegi dell'hub.
