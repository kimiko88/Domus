# ADR-013: Architettura del Supervisore e Sandboxing delle Domus Apps

## Status
Accepted

## Date
2026-10-04

## Context
Per consentire all'ecosistema Domus di estendere le proprie capacità (supporto a Zigbee2MQTT, broker MQTT locali, server di telemetria, storage di serie storiche), il sistema deve permettere l'installazione di applicazioni/plugin di terze parti senza compromettere la stabilità e la sicurezza del core né i dati degli utenti.

## Decision
Introduciamo il **Domus Supervisor** e il modello dichiarativo **Domus Apps Manifest**:
1. **Domus Supervisor**:
   - Micro-demone scritto in Go in esecuzione sull'host come servizio controllato (`domus-supervisor.service`).
   - Gestisce il ciclo di vita (install, configure, start, stop, update, uninstall) di Domus Core e delle applicazioni.
   - Espone un'API locale esclusivamente tramite Unix Domain Socket (`/run/domus/supervisor.sock`).
   - Comunica direttamente con `containerd` via gRPC socket.
   - Non ospita logica applicativa o di automazione domotica.
2. **Isolamento Sandboxed delle Apps**:
   - Ogni app viene definita da un file manifest `domus-app.yaml` validato contro un JSON Schema rigido.
   - Modello a permessi espliciti: l'accesso a porte di rete, dispositivi fisici (es. chiavetta Zigbee su `/dev/ttyUSB0`), topic MQTT e quote di memoria/CPU deve essere formalmente dichiarato nel manifest e approvato dall'utente prima dell'installazione.
   - I container delle app non hanno accesso al socket di `containerd` né al database primario di Domus.

## Consequences
### Positive
- Isolamento ermetico dei componenti aggiuntivi.
- Un'applicazione difettosa o compromessa non può interrompere Domus Core né accedere al filesystem host.
- Modello di autorizzazioni chiaro e trasparente per l'utente finale.
