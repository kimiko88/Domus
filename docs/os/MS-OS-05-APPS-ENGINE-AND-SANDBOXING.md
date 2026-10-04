# Domus OS - Milestone MS-OS-05: Domus Apps Engine & Sandboxing

Questo documento descrive il **sistema di estensioni applicative (Domus Apps)**, il **manifest dichiarativo formale (`domus-app.yaml`)** e le **regole di isolamento e sicurezza (sandboxing)** implementate in Domus OS per Raspberry Pi 4 ARM64.

---

## 1. Modello di Sicurezza e Principio del Minimo Privilegio

In Domus OS, qualsiasi componente non-core (es. Zigbee2MQTT, broker Mosquitto, server InfluxDB, Node-RED) deve essere eseguito come **Domus App isolata**:
- **Nessun accesso al Docker o containerd socket**: le estensioni non possono gestire altri container né eseguire privilege escalation.
- **Root filesystem del container in sola lettura**: impedisce modifiche permanenti ai binari del container.
- **Storage persistente rigorosamente confinato**: ogni app riceve esclusivamente una cartella dedicata in `/data/apps/<app-id>/`.
- **Whitelisting esplicito dei dispositivi hardware**: l'accesso a periferiche fisiche USB o GPIO (es. coordinatori Zigbee su `/dev/ttyUSB0` o `/dev/ttyACM0`) deve essere dichiarato espressamente nel manifest. L'accesso a dispositivi di blocco disco (`/dev/sd*`, `/dev/mmcblk*`) o alla memoria di sistema (`/dev/mem`, `/dev/kmem`) è rigidamente rigettato dal validatore.

---

## 2. Struttura del Manifest Applicativo (`domus-app.yaml`)

Il manifest viene validato contro il JSON Schema formale [`apps/schema/app-manifest.json`](file:///c:/Users/chimi/Desktop/Programmazione/Domus/apps/schema/app-manifest.json).

```yaml
id: zigbee2mqtt
name: Zigbee2MQTT
version: "1.36.0-domus1"
description: "Universal local Zigbee coordinator supporting TI CC2652, ConBee II/III."
image: "ghcr.io/domus-hub/apps-zigbee2mqtt:1.36.0"
image_digest: "sha256:d8b74c2f1e687895bc4935e5d3fa9bc7468128913ac839352e0081d6049281a8"

permissions:
  network:
    mode: "bridge"
    inbound_ports:
      - container_port: 8080
        host_port: 8099
        protocol: "tcp"
        ui_ingress: true
  devices:
    - path: "/dev/ttyUSB0"
      description: "Coordinatore Zigbee USB"
  mqtt:
    client_id: "domus_app_zigbee2mqtt"
    publish: ["zigbee2mqtt/#"]
    subscribe: ["zigbee2mqtt/#"]
  storage:
    - path: "/data"
      volume_name: "zigbee2mqtt-data"
      quota_mb: 512

resources:
  memory_limit_mb: 256
  cpu_limit: 1.5
  restart_policy: "unless-stopped"
```

---

## 3. Ciclo di Vita delle Applicazioni

Il **Domus Supervisor** gestisce l'intero lifecycle attraverso il modulo `internal/supervisor/app`:
1. **Install (`POST /api/v1/apps/install`)**:
   - Valida la sintassi YAML e le regole di sicurezza del manifest.
   - Crea la directory persistente isolata `/data/apps/<app-id>`.
   - Salva il manifest approvato in `/data/supervisor/manifests/<app-id>.yaml`.
   - Istruisce `containerd` creando il container con i soli mount autorizzati.
2. **Start (`POST /api/v1/apps/{id}/start`)**:
   - Avvia il task associato su `containerd`.
3. **Stop (`POST /api/v1/apps/{id}/stop`)**:
   - Invia `SIGTERM` con grazia di 10 secondi e successivo `SIGKILL`.
4. **Uninstall (`DELETE /api/v1/apps/{id}`)**:
   - Ferma ed elimina il container.
   - Rimuove il manifest registrato.
   - Opzionalmente rimuove i dati persistenti in `/data/apps/<app-id>`.

---

## 4. Template di Riferimento Inclusi

- [`apps/templates/zigbee2mqtt/domus-app.yaml`](file:///c:/Users/chimi/Desktop/Programmazione/Domus/apps/templates/zigbee2mqtt/domus-app.yaml): Integrazione completa Zigbee con mapping USB e topic MQTT.
- [`apps/templates/mosquitto/domus-app.yaml`](file:///c:/Users/chimi/Desktop/Programmazione/Domus/apps/templates/mosquitto/domus-app.yaml): Broker MQTT locale con porte 1883 e 9001.
