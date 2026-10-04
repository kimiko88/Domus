# ADR-009: Aggiornamenti Atomici A/B con RAUC e U-Boot

## Status
Accepted

## Date
2026-10-04

## Context
In un appliance domotico embedded, un aggiornamento fallito o interrotto da una mancanza di corrente non deve mai rendere il sistema inavviabile (bricking). L'aggiornamento deve essere atomico, verificabile crittograficamente e capace di eseguire un rollback automatico in caso di errore di boot o mancato superamento dell'healthcheck.

## Decision
Adottiamo **RAUC (Robust Auto-Update Controller)** integrato con il bootloader **U-Boot**:
- Il disco viene partizionato con due slot di sistema operativi simmetrici: `domus-root-a` e `domus-root-b`.
- Gli aggiornamenti vengono distribuiti sotto forma di bundle compresso firmato digitalmente con certificati x.509 (`.raucb`).
- Durante l'aggiornamento, RAUC verifica la firma crittografica del bundle, scrive l'immagine sullo slot inattivo e configura le variabili di U-Boot per il riavvio sul nuovo slot.
- U-Boot implementa un contatore di tentativi di avvio (`bootcount` vs `bootlimit=3`).
- Al termine del boot, `domus-supervisor` esegue un healthcheck su Domus Core: se il servizio risponde positivamente (HTTP 200 OK), il Supervisor invia il comando D-Bus a RAUC per confermare lo slot come integro (`mark-good`).
- Se il bootloader fallisce l'avvio del kernel per 3 volte o l'healthcheck non viene superato entro il timeout, U-Boot ripristina automaticamente lo slot precedente.

## Consequences
### Positive
- Tolleranza a blackout e interruzioni di rete durante l'aggiornamento.
- Rollback automatico trasparente per l'utente finale.
- Verifica crittografica rigorosa prima dell'applicazione dell'aggiornamento.

### Negative / Trade-offs
- Richiede lo spazio sul disco per ospitare due partizioni di rootfs (2 x 1.2 GB).
