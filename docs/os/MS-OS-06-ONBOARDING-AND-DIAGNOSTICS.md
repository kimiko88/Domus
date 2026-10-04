# Domus OS - Milestone MS-OS-06: Onboarding Appliance, mDNS & Diagnostica UI

Questo documento descrive il sistema di **onboarding zero-configuration via mDNS**, la **telemetria hardware del SoC in tempo reale**, l'**esportazione diagnostica sanitizzata** e la **modalità manutenzione SSH a tempo** implementati in Domus OS per Raspberry Pi 4 ARM64.

---

## 1. Onboarding Zero-Configuration & Risoluzione mDNS (`domus.local`)

Per eliminare la necessità di conoscere preventivamente l'indirizzo IP locale assegnato dal router DHCP, Domus OS integra il demone **Avahi mDNS/DNS-SD** nel sistema operativo base:

- **Configurazione demone**: [`os/configs/avahi-daemon.conf`](file:///c:/Users/chimi/Desktop/Programmazione/Domus/os/configs/avahi-daemon.conf)
  - Annuncio dell'hostname: `domus` nel dominio `.local`
  - Supporto dual-stack: IPv4 e IPv6 attivi
  - Risoluzione locale trasparente tramite modulo NSS `mdns4_minimal` configurato in `/etc/nsswitch.conf`
- **Definizione servizi pubblicati**: [`os/configs/domus-http.service`](file:///c:/Users/chimi/Desktop/Programmazione/Domus/os/configs/domus-http.service)
  - Annuncio HTTP standard: `_http._tcp` su porta 80
  - Annuncio appliance Domus: `_domus._tcp` su porta 80 con metadati TXT (`version=1.0.0`, `model=Raspberry Pi 4 Model B`, `arch=aarch64`)

### Flusso di Onboarding Utente
1. L'utente collega il Raspberry Pi 4 con Domus OS al router tramite cavo Ethernet o Wi-Fi e alimenta il dispositivo.
2. Dal proprio browser su PC, Mac o smartphone connesso alla stessa LAN, l'utente naviga su:
   ```
   http://domus.local
   ```
3. Il demone Avahi risponde alle query multicast DNS (porta 5353 UDP) risolvendo il nome sull'indirizzo IP del Raspberry Pi.
4. Si apre immediatamente la procedura guidata di onboarding di Domus Core per la creazione del primo account amministratore (`/setup`).

---

## 2. Telemetria Hardware & Diagnostica di Sistema

Il micro-demone **Domus Supervisor** acquisisce le metriche fisiche del Broadcom BCM2711 e dello stato del sistema operativo tramite il modulo [`internal/supervisor/diagnostics.go`](file:///c:/Users/chimi/Desktop/Programmazione/Domus/internal/supervisor/diagnostics.go):

| Metrica | Sorgente Kernel / Filesystem | Unità & Rappresentazione |
|---|---|---|
| **Temperatura SoC** | `/sys/class/thermal/thermal_zone0/temp` | Gradi Celsius (°C, es. `44.8 °C`) |
| **Carico CPU** | `/proc/loadavg` | Media su 1m, 5m, 15m con conteggio core (`runtime.NumCPU()`) |
| **Memoria RAM** | `/proc/meminfo` (`MemTotal`, `MemAvailable`, `MemFree`) | Utilizzo in MB e percentuale (es. `1536 MB / 4096 MB (37.5%)`) |
| **Spazio Disco `/data`** | Filesystem ext4 persistente via `statvfs` | Utilizzo in GB e percentuale su storage fisico |
| **Slot A/B Update** | D-Bus RAUC client (`GetStatus`) | Slot attivo (`domus-root-a`/`domus-root-b`) e stato di conferma |
| **Container & Apps** | Demone containerd (`ListContainers`) | Numero container attivi e numero Domus Apps installate |

### Endpoint REST
- **`GET /api/v1/system/diagnostics`**: Restituisce il payload JSON con tutte le metriche hardware e di runtime aggiornate in tempo reale.
- **`GET /api/v1/system/diagnostics/export`**: Genera un pacchetto completo per il supporto tecnico e il debug della community, con **sanitizzazione automatica**:
  - Rimozione e mascheramento di qualsiasi chiave contenente parole sensibili (`PASS`, `SECRET`, `TOKEN`, `KEY`, `AUTH`, `CREDENTIAL`, `PRIVATE`, `COOKIE`) con il marcatore `[REDACTED]`.

---

## 3. Modalità Manutenzione Controllata (SSH On-Demand)

In conformità all'**ADR-012 (Modello di Sicurezza Embedded e Hardening)**, il demone SSH è **rigorosamente disattivato di default** in produzione:
- Nessun demone SSH costantemente in ascolto su porta 22.
- Nessuna password di root configurata o consentita via rete.

### Attivazione a Tempo Bounded
Qualora sia necessario accedere alla shell di basso livello per interventi di manutenzione avanzata:
1. L'amministratore autenticato nella Web UI accede alla scheda **Impostazioni**.
2. Nella sezione **Modalità Manutenzione (SSH)**, seleziona la durata desiderata (15m, 30m default, 60m o 120m max) e inserisce opzionalmente la propria chiave pubblica SSH (`ssh-ed25519 ...`).
3. Il Supervisor esegue:
   - Creazione della directory `/data/system/ssh` con permessi ristretti `0700`.
   - Scrittura della chiave autorizzata in `/data/system/ssh/authorized_keys` con permessi `0600`.
   - Avvio del servizio `sshd` via systemd.
   - Avvio di un timer asincrono (`time.AfterFunc`).
4. **Scadenza automatica**: allo scadere del timer impostato, il Supervisor ferma immediatamente il servizio `sshd` e rimuove/svuota la chiave autorizzata da `/data/system/ssh/authorized_keys`.
5. **Disattivazione manuale**: l'amministratore può terminare la sessione in qualunque momento cliccando su **"Disattiva Immediatamente"** (`POST /api/v1/system/maintenance/disable`).

---

## 4. Interfaccia Utente Web & Localizzazione Multi-lingua

Le metriche di Domus OS sono integrate nella visualizzazione **Impostazioni** del frontend Vue 3 PWA:
- **Card Telemetria Hardware**: indicatore visivo con soglie di allarme per la temperatura SoC:
  - Verde (`< 60 °C`): Normale
  - Ambra (`60 °C - 75 °C`): Attenzione / Carico elevato
  - Rosso (`> 75 °C`): Critico / Rischio thermal throttling
- **Barre di utilizzo dinamiche**: RAM e partizione `/data` con gradienti moderni glassmorphism.
- **Pulsante Esporta Report Diagnostico**: download one-click del file `domus-diagnostics-YYYY-MM-DD.json`.
- **Card Modalità Manutenzione**: selettore di durata, inserimento chiave pubblica e timer di countdown in tempo reale.
- **Supporto multi-lingua completo**: tutte le stringhe, etichette e descrizioni sono localizzate in 5 lingue:
  - 🇮🇹 Italiano (`it.json`)
  - 🇬🇧 Inglese (`en.json`)
  - 🇫🇷 Francese (`fr.json`)
  - 🇪🇸 Spagnolo (`es.json`)
  - 🇩🇪 Tedesco (`de.json`)

---

## 5. Verifica e Matrice di Test (TDD)

Tutti i componenti di Milestone 6 sono coperti da test automatizzati:
1. **`test/os/diagnostics_test.go`**:
   - `TestAvahiDaemonConfig`: verifica parametri mDNS, hostname e domain.
   - `TestAvahiServiceDefinition`: validazione sintassi XML e servizi `_http._tcp` e `_domus._tcp`.
   - `TestBuildrootDefconfig_IncludesAvahiAndMDNS`: verifica presenza pacchetti Buildroot.
   - `TestPostBuildHook_ProvisionsAvahiAndNSS`: verifica installazione file e configurazione NSS.
   - `TestEndToEnd_SupervisorDiagnosticsAndMaintenanceAPI`: test end-to-end API Unix Socket.
2. **`internal/supervisor/diagnostics_test.go`**:
   - Lettura e parsing temperatura SoC, `/proc/meminfo`, `/proc/loadavg` e sanitizzazione export.
3. **`internal/supervisor/maintenance_test.go`**:
   - Ciclo di vita manutenzione SSH, permessi chiavi, clamping durata e auto-scadenza timer.
4. **`web/src/stores/system.test.ts` & `web/src/views/SettingsView.test.ts`**:
   - Test unitari e di componente Pinia/Vue con rendering delle card e switch dinamico delle lingue.
