# Domus OS - Milestone MS-OS-02: Read-Only SquashFS & Storage Persistente

Questo documento descrive l'architettura di **immutabilità del filesystem**, la gestione dello **storage persistente su partizione dedicata `/data`** e le strategie di **protezione contro guasti da blackout (power-loss resilience)** e usura delle memorie flash (flash wear-out) implementate in **Domus OS**.

---

## 1. Modello di Immutabilità: Stateless Read-Only RootFS

In Domus OS, l'intero sistema operativo base risiede in una partizione montata come filesystem **SquashFS a sola lettura**:
- **Immunità da manomissione**: nessun binario di sistema (`/bin`, `/sbin`, `/usr`, `/lib`) può essere sovrascritto, corrotto o eliminato a runtime.
- **Resilienza a blackout improvvisi**: poiché la rootfs non effettua mai scritture su disco fisico, un'interruzione di corrente improvvisa non può provocare corruzione del filesystem di boot.
- **Cartelle volatili in RAM (`tmpfs`)**:
  - `/tmp`: `tmpfs` dimensionato a max 256 MB (opzioni: `nosuid,nodev,mode=1777`).
  - `/run`: `tmpfs` per socket IPC e PID file.
  - `/var/volatile`: `tmpfs` per log e lock effimeri.

---

## 2. Partizione Dati Persistente (`/data`)

Tutti i dati generati dall'utente, il database SQLite, i segreti crittografici, le configurazioni di rete e i volumi dei container risiedono esclusivamente nella partizione etichettata **`domus-data`**, montata su **`/data`**.

### 2.1 Opzioni di Mount per Protezione Flash
La mount unit systemd [`os/systemd/data.mount`](file:///c:/Users/chimi/Desktop/Programmazione/Domus/os/systemd/data.mount) applica:
```ini
[Mount]
What=/dev/disk/by-partlabel/domus-data
Where=/data
Type=ext4
Options=defaults,noatime,commit=30,errors=remount-ro
```
- **`noatime`**: disabilita l'aggiornamento dell'orario di ultimo accesso in lettura, riducendo le scritture su flash dell'80% per operazioni di consultazione.
- **`commit=30`**: ritarda il flush periodico del journal ext4 a 30 secondi (anziché i 5s standard), raggruppando le scritture e allungando la vita utile della memoria SSD o microSD.
- **`errors=remount-ro`**: in caso di guasto hardware irreparabile, rimonta il disco in sola lettura evitando di corrompere i dati storici.

---

## 3. Albero delle Directory Persistenti su `/data`

Il servizio [`domus-storage-init.service`](file:///c:/Users/chimi/Desktop/Programmazione/Domus/os/systemd/domus-storage-init.service) esegue lo script [`domus-storage-init.sh`](file:///c:/Users/chimi/Desktop/Programmazione/Domus/os/scripts/domus-storage-init.sh) all'avvio:

```
/data/
├── system/                     # [0755] Parametri persistenti di sistema
│   ├── machine-id              # [0444] Identificativo univoco hardware dell'appliance
│   ├── network/                # [0700] Configurazioni protette IP e Wi-Fi
│   ├── ssh/                    # [0700] Host keys e flag modalità manutenzione
│   └── rauc/                   # [0755] Stato e cronologia aggiornamenti A/B
├── core/                       # [0750] Dati applicativi Domus Core
│   ├── domus.db                # Database SQLite di produzione (modalità WAL)
│   ├── config.toml             # Configurazione runtime dell'hub
│   ├── secrets.key             # [0600] Master key crittografica locale
│   └── backups/                # [0700] Archivio backup cifrati locali
├── supervisor/                 # [0750] Metadati del supervisore
│   ├── supervisor.db           # Database stato estensioni e audit log
│   └── manifests/              # [0755] Manifest verificati delle app installate
├── apps/                       # [0755] Volumi persistenti isolati per container estensioni
│   └── <app-id>/
├── containerd/                 # [0711] Layer storage delle immagini OCI containerd
└── logs/                       # [2755]
    └── journal/                # Log systemd journald persistenti (limitati a 100 MB)
```

---

## 4. Limitazione e Protezione Log (`journald`)

Per impedire la saturazione dello storage fisico e ridurre l'usura flash:
- La configurazione [`os/systemd/journald.conf`](file:///c:/Users/chimi/Desktop/Programmazione/Domus/os/systemd/journald.conf) impone:
  - `SystemMaxUse=100M`: tetto massimo rigido per l'intero archivio log.
  - `SystemMaxFileSize=10M`: rotazione automatica del singolo file di log a 10 MB.
  - `SyncIntervalSec=5m`: sincronizzazione su disco ogni 5 minuti (con buffer in memoria).
  - `ForwardToSyslog=no`: elimina duplicazioni nei file `/var/log/syslog`.

---

## 5. Ripristino di Fabbrica (Factory Reset)

L'utilità [`domus-factory-reset.sh`](file:///c:/Users/chimi/Desktop/Programmazione/Domus/os/scripts/domus-factory-reset.sh) consente di ripristinare l'appliance allo stato originale:
- Elimina completamente i dati di `/data` (database, app, segreti, log).
- L'immagine SquashFS di base non viene toccata e rimane integra.
- Opzioni disponibili:
  ```bash
  # Reset con richiesta interattiva di conferma
  sudo domus-factory-reset

  # Reset immediato non interattivo (per automazioni o manutenzione)
  sudo domus-factory-reset --force

  # Reset preservando la connettività di rete (Wi-Fi e IP)
  sudo domus-factory-reset --force --keep-network
  ```
