# ADR-012: Modello di Sicurezza Embedded e Hardening di Default

## Status
Accepted

## Date
2026-10-04

## Context
Un hub domotico governa serrature, telecamere, sensori di sicurezza e luci di un'abitazione. Deve essere progettato con i principi di *Security by Default* e *Principle of Least Privilege*, minimizzando la superficie di attacco sia sulla rete locale (LAN) sia verso il mondo esterno.

## Decision
Adottiamo un profilo di hardening rigoroso:
1. **Accesso Amministrativo Ristretto**:
   - Il demone SSH è disabilitato di default nell'immagine di produzione.
   - Nessuna password di root preimpostata né accessibile dalla rete.
   - La modalità di manutenzione può essere abilitata esclusivamente da un utente amministratore autenticato tramite interfaccia web locale, con generazione di credenziali temporanee ad orologeria e registrazione dell'evento nell'audit log.
2. **Hardening di Rete**:
   - Nessun servizio di configurazione esposto automaticamente a Internet (UPnP disabilitato).
   - Servizio mDNS Avahi configurato per esporre solo `domus.local` sulla LAN.
   - Firewall di ingresso attivo: solo porte HTTP (80/8080) o HTTPS (443) accettate.
3. **Isolamento dei Privilegi**:
   - Domus Core e le applicazioni girano all'interno di container OCI come utenti non-root dedicati.
   - Rimozione delle Linux capabilities non necessarie (`CAP_SYS_ADMIN`, `CAP_NET_RAW`, `CAP_MKNOD`).
   - Cgroups v2 per confinamento rigoroso di CPU e memoria.

## Consequences
### Positive
- Riduzione drastica del rischio di movimento laterale in caso di violazione di altri dispositivi sulla rete LAN.
- Protezione contro tentativi di scansione ed exploit automatizzati via SSH.
- Tracciabilità completa delle attività di manutenzione.

### Negative / Trade-offs
- Il debugging di basso livello richiede di abilitare esplicitamente la modalità di manutenzione tramite la Web UI o di accedere via console seriale fisica (UART).
