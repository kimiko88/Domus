# ADR-007: Adozione di Buildroot per la Costruzione di Domus OS

## Status
Accepted

## Date
2026-10-04

## Context
Per trasformare il Raspberry Pi 4 Model B in un appliance domotico sicuro, deterministico e aggiornabile (sull'impostazione concettuale di Home Assistant OS), è necessario un sistema operativo Linux embedded di base.
Le alternative valutate sono state:
1. **Raspberry Pi OS Minimal**: basato su Debian general-purpose, include package manager APT, interpreti e toolchain non necessari, occupando oltre 1.5 GB e rendendo l'immutabilità a sola lettura complessa e fragile.
2. **Yocto Project (Poky)**: estremamente potente ma eccessivamente complesso per un team agile, con centinaia di meta-layer e ricette BitBake.
3. **Buildroot**: generatore di sistemi Linux embedded focalizzato su semplicità, determinismo assoluto, footprint ridotto e cross-compilazione veloce.

## Decision
Adottiamo **Buildroot (2024.02 LTS)** come base tecnologica di Domus OS:
- Configurazione unificata e versionata tramite `domus_rpi4_defconfig`.
- Cross-compilazione della toolchain GCC per AArch64 (ARM64) Cortex-A72.
- Integrazione nativa del kernel Linux LTS per Raspberry Pi (`bcm2711_defconfig`).
- Generazione diretta di immagini SquashFS compresse inferiori a 200 MB.
- Totale assenza di compilatori, interpreti superflui o package manager a runtime nell'immagine finale.

## Consequences
### Positive
- Footprint ridotto: tempi di avvio < 8 secondi e consumo storage minimale.
- Immutabilità nativa: nessun rischio di modifiche incontrollate ai file di sistema da parte di pacchetti o script utente.
- Manutenibilità: singolo albero `os/` controllato da Makefile e script riproducibili.

### Negative / Trade-offs
- L'aggiunta di driver o pacchetti di sistema richiede la ricompilazione del bundle OS tramite pipeline invece di un semplice comando `apt install`.
