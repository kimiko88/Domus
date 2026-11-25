# Sviluppare plugin e nuove integrazioni per Domus

Questa guida descrive il contratto attuale per aggiungere un'integrazione protocollo/vendor senza modificare il core a ogni passaggio.

## Architettura

Ogni integrazione è un adapter che traduce il protocollo del dispositivo nel modello comune Domus:

```text
Dispositivo / Cloud API / Gateway
            |
       Plugin Adapter
            |
  Discovery + Mapping + Commands
            |
  Repository Device / Entity
            |
       Event Bus Domus
            |
   Automazioni / UI / Storico
```

L'adapter implementa `plugins.Adapter`:
- `ID()`: identificativo univoco dell'istanza (es. `mqtt_cucina`);
- `Type()`: tipo registrato (es. `mqtt_generic`);
- `Start(ctx)`: apre socket, sottoscrizioni e goroutine;
- `Stop(ctx)`: ferma tutto e rilascia le risorse;
- `HealthCheck(ctx)`: verifica con timeout la connessione reale.

Il `Supervisor` gestisce lifecycle, health check, isolamento dei panic e circuit breaker. La factory costruisce l'istanza, il Supervisor la avvia: non avviare goroutine permanenti dentro la factory.

## Manifest e registry

Per ogni tipo di plugin definisci un `Manifest` con:
- `api_version`: attualmente `domus.plugin/v1`;
- `id`: chiave stabile in minuscolo, per esempio `acme_bridge`;
- `name`, `version` SemVer e descrizione;
- `capabilities`: permessi dichiarati;
- `config`: schema dei parametri, inclusi campi obbligatori, enum e segreti.

Il registry verifica manifest, capability dichiarate, tipo/formato dei parametri, campi sconosciuti e dimensione massima (64 KiB). Le factory sono funzioni Go registrate esplicitamente; la creazione non avvia l'adapter. Dopo `Create`, passa l'istanza al `Supervisor.Register` e quindi avvia il Supervisor.

Esempio di manifest:

```json
{
  "api_version": "domus.plugin/v1",
  "id": "acme_bridge",
  "name": "Acme local bridge",
  "version": "1.0.0",
  "description": "Bridge LAN per sensori e attuatori Acme",
  "capabilities": ["devices:read", "devices:write", "events:read", "network:lan"],
  "config": [
    {"name": "host", "type": "string", "required": true},
    {"name": "port", "type": "integer", "required": true},
    {"name": "poll_interval", "type": "duration", "required": false},
    {"name": "api_token", "type": "string", "required": true, "secret": true}
  ]
}
```

## Scheletro di adapter

Crea una directory `internal/plugins/acme_bridge` e implementa il contratto:

```go
package acme_bridge

import (
    "context"
    "encoding/json"
    "fmt"
    "sync"

    "github.com/domus-hub/domus/internal/plugins"
)

type Config struct {
    Host         string `json:"host"`
    Port         int    `json:"port"`
    PollInterval string `json:"poll_interval"`
    APIToken     string `json:"api_token"`
}

type Adapter struct {
    id     string
    cfg    Config
    mu     sync.Mutex
    cancel context.CancelFunc
}

var _ plugins.Adapter = (*Adapter)(nil)

func New(id string, cfg Config) (*Adapter, error) {
    if id == "" || cfg.Host == "" || cfg.Port < 1 || cfg.Port > 65535 {
        return nil, fmt.Errorf("acme_bridge: invalid configuration")
    }
    return &Adapter{id: id, cfg: cfg}, nil
}

func (a *Adapter) ID() string   { return a.id }
func (a *Adapter) Type() string { return "acme_bridge" }

func (a *Adapter) Start(ctx context.Context) error {
    a.mu.Lock()
    defer a.mu.Unlock()
    if a.cancel != nil {
        return nil
    }
    runCtx, cancel := context.WithCancel(ctx)
    a.cancel = cancel
    go a.pollLoop(runCtx)
    return nil
}

func (a *Adapter) Stop(ctx context.Context) error {
    a.mu.Lock()
    cancel := a.cancel
    a.cancel = nil
    a.mu.Unlock()
    if cancel != nil {
        cancel()
    }
    return nil
}

func (a *Adapter) HealthCheck(ctx context.Context) error {
    // Esegui un ping leggero con ctx; non limitarti a controllare un boolean.
    return a.ping(ctx)
}

func (a *Adapter) pollLoop(ctx context.Context) {
    // Usa ticker con ctx.Done(), timeout per ogni I/O e backoff per i retry.
}

func (a *Adapter) ping(ctx context.Context) error {
    return nil
}

func Factory(ctx context.Context, id string, raw json.RawMessage, deps plugins.Dependencies) (plugins.Adapter, error) {
    cfg, err := plugins.DecodeConfig[Config](raw)
    if err != nil {
        return nil, err
    }
    adapter, err := New(id, cfg)
    if err != nil {
        return nil, err
    }
    // Usa deps.Bus, deps.Entities, deps.Devices e deps.Logger soltanto quando necessari.
    _ = deps
    return adapter, nil
}
```

## Registrazione

Nel bootstrap della composizione applicativa (dopo aver creato repository ed event bus), registra esplicitamente il tipo:

```go
registry := plugins.NewRegistry()
manifest := plugins.Manifest{
    APIVersion: plugins.ManifestAPIVersion,
    ID: "acme_bridge",
    Name: "Acme local bridge",
    Version: "1.0.0",
    Capabilities: []plugins.Capability{
        plugins.CapabilityDevicesRead,
        plugins.CapabilityDevicesWrite,
        plugins.CapabilityEventsRead,
        plugins.CapabilityNetworkLAN,
    },
    Config: []plugins.ConfigField{
        {Name: "host", Type: "string", Required: true},
        {Name: "port", Type: "integer", Required: true},
        {Name: "api_token", Type: "string", Required: true, Secret: true},
    },
}
if err := registry.Register(manifest, acme_bridge.Factory); err != nil {
    return err
}

adapter, err := registry.Create(ctx, "acme_bridge", "acme_cucina", jsonConfig, plugins.Dependencies{
    Bus: bus, Entities: repos.Entities, Devices: repos.Devices,
})
if err != nil { return err }
if err := supervisor.Register(adapter); err != nil { return err }
```

Il registry offre `List()` per presentare i tipi installati e `Create()` per costruire un'istanza dopo la validazione. La configurazione dell'istanza va mantenuta nello storage sicuro Domus; non inserire token in manifest, log o repository.

## Stato dispositivo, eventi e comandi

- Discovery: crea/aggiorna un `device.Device` e le relative `entity.Entity` con ID stabili.
- State: valida e normalizza i valori, chiama il repository per aggiornare lo stato e pubblica `events.EntityStateChanged`.
- Command: sottoscrivi `events.CommandRequested`, controlla entity key e comando consentito, poi invia il pacchetto al dispositivo.
- Availability: aggiorna `available` soltanto in base a heartbeat, polling o connessione reale.
- Idempotenza: deduplica discovery e comandi; non creare una nuova entità a ogni reconnect.
- Stop: annulla goroutine, chiude socket, unsubscribes dall'event bus e attende i worker.

Test minimi richiesti: mapping unit test, framing/parser test, command round-trip con server simulato, reconnect, timeout, device offline, config invalida, race test e fuzz del parser. Per protocolli di rete, non fidarti di lunghezze o JSON ricevuti dal dispositivo.

## Sicurezza e limiti del runtime

Le capability del manifest sono dichiarazioni validate e utili per review/UI: **non sono un sandbox per codice Go nativo**. Un adapter Go compilato nel processo Domus ha i privilegi del processo; accetta soltanto plugin nativi di cui ti fidi e revisiona ogni dipendenza. Non eseguire sorgenti o binari scaricati automaticamente da store non fidati.

Il pacchetto `internal/plugins/wasm` presente nel repository contiene parser e scaffolding, ma non deve essere considerato un runtime WASM/WASI production-ready né una barriera di sicurezza per codice ostile. Prima di distribuire plugin terzi servono un vero runtime WASI isolato, limiti CPU/memoria/fuel, host-call allowlist effettiva, firma dei pacchetti, verifica hash, aggiornamenti atomici e test di escape sandbox.

### Checklist prima della PR

1. Manifest e config validati, segreti fuori dal codice.
2. Adapter lifecycle idempotente e cancellabile.
3. Network I/O con context, deadline e size limits.
4. Mapping stabile per device/entity.
5. Test unitari, integrazione, race e fuzz.
6. Log strutturati senza token, payload sensibili o audio grezzo.
7. Documentazione con protocollo, pairing, credenziali e recovery.
