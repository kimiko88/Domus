# ADR-010: Filesystem Immutabile in SquashFS e RootFS Stateless

## Status
Accepted

## Date
2026-10-04

## Context
Le memorie flash (microSD ed SSD economici) subiscono degradazione (wear-out) a causa di continue scritture di file di log, file temporanei e lock file. Inoltre, un'interruzione di corrente durante una scrittura su partizioni ext4 montate in sola scrittura può causare la corruzione del filesystem di avvio.

## Decision
Adottiamo un'architettura **Stateless e Immutabile** per il sistema operativo:
1. La partizione di root (`/`) viene compressa e montata in sola lettura come filesystem **SquashFS**.
2. Tutte le cartelle effimere del sistema (`/tmp`, `/run`, `/var/volatile`) vengono montate in RAM tramite **`tmpfs`**.
3. Nessun file di log di sistema viene scritto continuamente sul disco primario; i log persistenti di journald sono confinati e limitati a un massimo rigido di 100 MB con rotazione automatica.
4. Il reset di fabbrica dell'appliance (Factory Reset) si ottiene formattando la sola partizione dati, lasciando intatto e incontaminato il sistema operativo base.

## Consequences
### Positive
- Immunità assoluta da corruzioni del filesystem causate da black-out improvvisi.
- Notevole allungamento della vita utile della memoria microSD o SSD.
- Garanzia di integrità: i binari di sistema non possono essere modificati, cancellati o infettati a runtime.

### Negative / Trade-offs
- Le modifiche di configurazione a livello OS richiedono meccanismi di override espliciti mappati su `/data` o ricompilazione del sistema.
