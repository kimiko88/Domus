# ADR-011: Gestione dello Storage Persistente su Partizione Dedicata `/data`

## Status
Accepted

## Date
2026-10-04

## Context
Per garantire la sopravvivenza dei dati utente (database SQLite, configurazioni, credenziali, chiavi crittografiche, backup e volumi delle app) attraverso aggiornamenti A/B e riavvii, è necessaria una partizione persistente nettamente separata dal sistema operativo.

## Decision
Creiamo una partizione dedicata con etichetta **`domus-data`**, montata su **`/data`**:
- Filesystem: **ext4** con journaling (`data=ordered`) e parametro di mount `commit=30` per ridurre la frequenza di sync delle scritture sulla memoria flash.
- Struttura gerarchica rigida:
  - `/data/system/`: machine-id, configurazione di rete e stato RAUC.
  - `/data/core/`: database SQLite WAL (`domus.db`), configurazione runtime (`config.toml`), master key (`secrets.key`) e archivio backup locali.
  - `/data/supervisor/`: database metadati delle estensioni e manifest attivi.
  - `/data/apps/<app-id>/`: cartelle persistenti isolate per ciascuna estensione installata.
  - `/data/containerd/`: cache e layer delle immagini OCI gestiti da containerd.

## Consequences
### Positive
- Disaccoppiamento totale tra ciclo di vita del codice (OS immutabile) e stato dei dati.
- Procedure di backup e ripristino semplificate: l'intero stato del sistema risiede in `/data`.
- Possibilità futura di crittografare l'intera partizione `/data` con dm-crypt / LUKS2.
