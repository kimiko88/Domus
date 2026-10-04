# ADR-008: Selezione di containerd come Container Runtime Unificato

## Status
Accepted

## Date
2026-10-04

## Context
Domus OS deve eseguire in container isolati sia Domus Core sia le applicazioni/estensioni (es. Zigbee2MQTT, broker MQTT, InfluxDB).
Il Raspberry Pi 4 Model B ha modelli da 2 GB e 4 GB di RAM. È necessario un container runtime leggero, stabile e integrabile programmaticamente con il supervisore di sistema scritto in Go.
Le opzioni valutate:
1. **Docker Engine (`dockerd`)**: consuma ~120-180 MB di RAM a riposo e aggiunge demoni superflui per un appliance embedded.
2. **Podman (`conmon` + `crun`)**: architettura daemonless, ma complessa configurazione di rete e dipendenze rootless in Buildroot.
3. **`containerd`**: runtime standard industriale OCI, consumo di memoria ridotto a ~25-35 MB a riposo.

## Decision
Adottiamo **`containerd`** con runtime `runc`/`crun`:
- Il demone `domus-supervisor` (scritto in Go) si integra direttamente con `containerd` importando la libreria Go ufficiale (`github.com/containerd/containerd`) tramite socket Unix gRPC (`/run/containerd/containerd.sock`).
- Non viene installato `dockerd` né la CLI client Docker.
- Ogni container viene configurato programmaticamente con isolamento cgroups v2, filesystem root in read-only e seccomp profile.

## Consequences
### Positive
- Risparmio di oltre 100 MB di memoria RAM a riposo rispetto a Docker Engine.
- Controllo programmatico deterministico dei container da parte del Supervisor senza wrapper CLI o shell exec.
- Isolamento granulare di risorse (quote CPU e memoria).

### Negative / Trade-offs
- Gli utenti non possono utilizzare il comando `docker run` da riga di comando (comportamento desiderato per un appliance gestito).
