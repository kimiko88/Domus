# Domus OS - Milestone MS-OS-04: Container Runtime containerd & Domus Supervisor (Go)

Questo documento descrive l'architettura e l'integrazione del runtime per container **`containerd`** con **cgroups v2** e del micro-demone **Domus Supervisor** scritto in Go per la gestione del ciclo di vita di Domus Core e delle estensioni su Raspberry Pi 4 ARM64.

---

## 1. Architettura del Container Engine: `containerd`

Domus OS adotta `containerd` come runtime unificato, sostituendo l'intero stack monolitico di Docker Engine per massimizzare le prestazioni ed economizzare la memoria:
- **Consumo RAM**: ~25-35 MB a riposo contro i 150+ MB di `dockerd`.
- **Integrazione Cgroups v2**: [`os/configs/containerd-config.toml`](file:///c:/Users/chimi/Desktop/Programmazione/Domus/os/configs/containerd-config.toml) configura `SystemdCgroup = true`, delegando a systemd l'accounting preciso di CPU e memoria.
- **Storage OCI su `/data`**: i layer delle immagini e gli snapshot overlayfs risiedono in `/data/containerd/root`, sopravvivendo a tutti gli aggiornamenti di sistema A/B.
- **Stato Volatile su RAM**: i PID e i runtime state risiedono in `/run/containerd` (`tmpfs`).

---

## 2. Architettura del Domus Supervisor (Go)

Il **Domus Supervisor** ([`cmd/domus-supervisor/main.go`](file:///c:/Users/chimi/Desktop/Programmazione/Domus/cmd/domus-supervisor/main.go)) è un micro-demone di orchestrazione a basso impatto, eseguito sull'host come servizio privilegiato controllato:

```
               +---------------------------------------------------+
               |             systemd (PID 1)                       |
               +-------------------------+-------------------------+
                                         |
                +------------------------+------------------------+
                |                                                 |
                v                                                 v
   +-------------------------+                       +-------------------------+
   |   containerd.service    |                       | domus-supervisor.service|
   | (gRPC: containerd.sock) |                       | (Micro-demone Go)       |
   +------------+------------+                       +------------+------------+
                |                                                 |
                |  Avvia e isola container                        | Dialoga via gRPC
                +<================================================+
                |
                v
   +---------------------------------------------------------------+
   |                      Domus Core Container                     |
   |     - Backend Go + Web UI PWA statica                         |
   |     - Volume montato: /data/core -> /data                     |
   |     - Rete: host (o internal bridge)                          |
   |     - Healthcheck: http://127.0.0.1:8080/health               |
   +---------------------------------------------------------------+
```

---

## 3. Flusso di Avvio e Conferma A/B (Healthcheck Loop)

All'avvio dell'appliance:
1. `systemd` monta la partizione dati (`/data`) e avvia `containerd.service`.
2. Viene avviato `domus-supervisor.service`.
3. Il Supervisor assicura e avvia il container `domus-core` con il volume persistente `/data/core`.
4. Viene avviato il ciclo di polling su `http://127.0.0.1:8080/health`.
5. Quando Domus Core risponde `200 OK`:
   - Il Supervisor marca `CoreHealthy = true`.
   - Se il sistema è appena stato aggiornato, invoca automaticamente `rauc status mark-good` via [`CLIRaucController`](file:///c:/Users/chimi/Desktop/Programmazione/Domus/internal/supervisor/rauc.go).
   - U-Boot resetta il bootcounter: **lo slot di boot è confermato definitivamente**.
6. Se Domus Core non diventa healthy entro i tentativi massimi (default: 15 retry):
   - Lo slot non viene confermato;
   - Al successivo riavvio, il bootcounter di U-Boot scalerà fino a innescare il rollback automatico.

---

## 4. API Unix Domain Socket del Supervisore

Il Supervisor espone un'API REST su Unix Socket protetto (`/run/domus/supervisor.sock` con permessi `0660`):

| Metodo | Endpoint | Descrizione |
|---|---|---|
| `GET` | `/api/v1/supervisor/ping` | Verifica connettività (risponde `{"status": "pong"}`) |
| `GET` | `/api/v1/supervisor/status` | Restituisce telemetria completa, stato core, slot A/B attivo e uptime |
| `GET` | `/api/v1/supervisor/health` | Ritorna 200 OK se il core è sano, 503 se in avvio o non raggiungibile |
| `POST` | `/api/v1/supervisor/core/restart` | Riavvia in modo controllato il container di Domus Core |
| `POST` | `/api/v1/supervisor/update/confirm` | Forza la conferma dello slot corrente su RAUC |
| `POST` | `/api/v1/supervisor/update/rollback` | Segnala lo slot come fallito e innesca il reboot di rollback |
