# Domus 🏠 — Manuale Utente

> **Versione**: 1.0.0 · **Aggiornato**: Ottobre 2026  
> **Piattaforme**: Raspberry Pi 4 ARM64 (Domus OS), Linux/Docker, macOS/Windows (sviluppo)

---

## Indice

1. [Introduzione](#1-introduzione)
2. [Installazione](#2-installazione)
   - 2.1 [Domus OS — Flash immagine su Raspberry Pi 4](#21-domus-os--flash-immagine-su-raspberry-pi-4)
   - 2.2 [Docker Compose](#22-docker-compose)
   - 2.3 [Installazione nativa su Linux](#23-installazione-nativa-su-linux)
   - 2.4 [Sviluppo locale (Go + Node)](#24-sviluppo-locale-go--node)
3. [Primo avvio — Onboarding](#3-primo-avvio--onboarding)
4. [Navigazione nell'interfaccia](#4-navigazione-nellinterfaccia)
5. [Dashboard e Stanze](#5-dashboard-e-stanze)
6. [Dispositivi ed Entità](#6-dispositivi-ed-entità)
7. [Automazioni](#7-automazioni)
8. [Scene](#8-scene)
9. [Cronologia e Grafici](#9-cronologia-e-grafici)
10. [Gestione Famiglia e Permessi](#10-gestione-famiglia-e-permessi)
11. [Impostazioni e Configurazione](#11-impostazioni-e-configurazione)
12. [Diagnostica Hardware (Domus OS)](#12-diagnostica-hardware-domus-os)
13. [Modalità Manutenzione SSH](#13-modalità-manutenzione-ssh)
14. [Backup e Ripristino](#14-backup-e-ripristino)
15. [Log di Audit](#15-log-di-audit)
16. [Riferimento Configurazione (`domus.toml`)](#16-riferimento-configurazione-domustoml)
17. [Tuya Local & Smart Life (LAN-First)](#17-tuya-local--smart-life-lan-first)
18. [Mappa della Casa e Piani (Floor Plan)](#18-mappa-della-casa-e-piani-floor-plan)
19. [Faccende Domestiche e Promemoria (Chores)](#19-faccende-domestiche-e-promemoria-chores)
20. [App Mobile iOS e Android (Capacitor)](#20-app-mobile-ios-e-android-capacitor)
21. [Risoluzione dei Problemi](#21-risoluzione-dei-problemi)
22. [Domande Frequenti (FAQ)](#22-domande-frequenti-faq)

---

## 1. Introduzione

**Domus** è un hub domotico locale *privacy-first* progettato per funzionare interamente nella rete di casa, senza cloud obbligatorio e senza account esterni. È scritto in Go moderno e Vue 3 + TypeScript, ottimizzato per girare su un Raspberry Pi 4 con meno di 30 MB di RAM di occupazione del backend.

### Caratteristiche principali

| Funzionalità | Descrizione |
|---|---|
| **Local-first** | Automazioni, dashboard e telemetria senza Internet |
| **Ultra leggero** | Backend ~25 MB RAM, database SQLite WAL embedded |
| **MQTT nativo** | Supporto Zigbee2MQTT, ESPHome, MQTT generico |
| **Automazioni DSL** | Motore a regole JSON: trigger, condizioni, azioni |
| **Multi-famiglia RBAC** | Admin, Operator, Viewer con invito tramite link |
| **Domus OS** | Immagine Linux embedded per RPi 4, aggiornamenti A/B atomici |
| **PWA installabile** | Funziona come app nativa su iOS, Android, desktop |
| **5 lingue** | IT EN FR ES DE |

> [!NOTE]
> Domus **non richiede** account su server remoti, abbonamenti o connessione Internet per il funzionamento quotidiano. Internet serve solo per aggiornare Domus stesso.

---

## 2. Installazione

### 2.1 Domus OS — Flash immagine su Raspberry Pi 4

Domus OS è la modalità di installazione consigliata: un'immagine Linux embedded completa e immutabile da scrivere direttamente su SSD/microSD.

#### Requisiti hardware

| Componente | Minimo | Consigliato |
|---|---|---|
| **Board** | Raspberry Pi 4 Model B | Raspberry Pi 4 Model B 4 GB |
| **Storage** | microSD 16 GB (Classe 10) | SSD USB 3.0 64 GB+ |
| **RAM** | 2 GB | 4 GB o 8 GB |
| **Rete** | Ethernet LAN | Ethernet Gigabit |
| **Alimentazione** | USB-C 5V/3A | Alimentatore ufficiale RPi 15W |

#### Procedura di flash

**Su Windows** (con [Raspberry Pi Imager](https://www.raspberrypi.com/software/)):

1. Scarica l'immagine `domus-os-rpi4-arm64.img.xz`.
2. Apri **Raspberry Pi Imager** → scegli **"Use Custom"** → seleziona il file `.img.xz`.
3. Seleziona il tuo SSD/microSD come destinazione.
4. Clicca **Write** e attendi il completamento (la verifica è automatica).

**Su macOS/Linux** (con `dd`):

```bash
# Decomprimere e scrivere in un solo passaggio
xzcat domus-os-rpi4-arm64.img.xz | sudo dd of=/dev/sdX bs=4M status=progress conv=fsync
sync
```

> [!CAUTION]
> Sostituisci `/dev/sdX` con il percorso corretto del tuo dispositivo storage. L'uso errato di `dd` può cancellare irrecuperabilmente altri dischi. Verifica con `lsblk` o `diskutil list`.

#### Primo avvio con Domus OS

1. Inserisci la microSD o collega l'SSD al Raspberry Pi.
2. Connetti il Raspberry Pi alla rete tramite cavo Ethernet.
3. Collega l'alimentazione.
4. Dopo circa 60 secondi, apri un browser e naviga su:

```
http://domus.local
```

Il dispositivo viene annunciato automaticamente nella LAN tramite **mDNS** (Avahi). Non è necessario conoscere l'indirizzo IP assegnato dal router.

> [!TIP]
> Se `domus.local` non funziona su Windows, assicurati che il servizio **Bonjour** sia installato (incluso con iTunes). Su Android, usa un'app "Network Scanner" per trovare l'IP del Raspberry Pi.

---

### 2.2 Docker Compose

Modalità consigliata per ambienti containerizzati e test rapidi.

**Prerequisiti**: Docker Engine 24+ · Docker Compose v2+

```bash
git clone https://github.com/domus-hub/domus.git
cd domus

# Avvia hub Domus + broker MQTT Mosquitto
docker compose -f deployments/docker/docker-compose.yml up -d --build
```

L'hub sarà disponibile su `http://localhost:8080`.

```bash
# Fermare il servizio
docker compose -f deployments/docker/docker-compose.yml down
```

I dati vengono persistiti nel volume Docker `domus_data`.

---

### 2.3 Installazione nativa su Linux

Script di installazione automatizzato per Raspberry Pi OS, Debian e Ubuntu.

```bash
git clone https://github.com/domus-hub/domus.git
cd domus

sudo ./deployments/scripts/install.sh
```

Lo script esegue automaticamente:
- Rilevamento architettura (`arm64` / `amd64`)
- Creazione utente di sistema non privilegiato `domus`
- Compilazione o copia dei binari
- Installazione unità systemd con hardening
- Abilitazione e avvio del servizio

```bash
# Disinstallazione
sudo ./deployments/scripts/uninstall.sh
```

---

### 2.4 Sviluppo locale (Go + Node)

**Prerequisiti**: Go 1.23+ · Node.js 20+

```bash
# Backend Go
make run

# Frontend Vue 3 (secondo terminale)
cd web && npm install && npm run dev
```

- Frontend hot-reload: `http://localhost:5173`
- Backend API: `http://localhost:8080`
- Swagger UI: `http://localhost:8080/docs`

---

## 3. Primo avvio — Onboarding

Al primo accesso, Domus mostra una **procedura guidata** per configurare l'account amministratore principale.

### Creazione account amministratore

Compila i campi nella pagina `/setup`:

| Campo | Descrizione |
|---|---|
| **Nome utente** | Identificativo univoco (es. `admin`, `mario`) |
| **Password** | Minimo 8 caratteri. Hashata con **Argon2id** — mai memorizzata in chiaro |
| **Email** | Opzionale, usata solo per identificazione |

> [!IMPORTANT]
> L'endpoint `/api/v1/auth/setup` è disponibile **una sola volta**. Dopo la creazione del primo admin, viene bloccato automaticamente (`409 Conflict`).

### Configurazione lingua

Vai in **Impostazioni** → seleziona la lingua preferita:  
Italiano · English · Français · Español · Deutsch

La scelta viene salvata nel browser e applicata immediatamente.

---

## 4. Navigazione nell'interfaccia

L'interfaccia è una **Progressive Web App** (PWA) con design glassmorphico in dark mode, installabile su qualsiasi dispositivo come app nativa.

### Barra di navigazione

| Voce | Descrizione |
|---|---|
| **Dashboard** | Panoramica entità, stanze e stato dispositivi |
| **Aree** | Mappa gerarchica stanze/zone della casa |
| **Dispositivi** | Elenco hardware integrato |
| **Automazioni** | Gestione regole e DSL automazioni |
| **Scene** | Preset multi-dispositivo attivabili manualmente |
| **Cronologia** | Grafici storici delle entità |
| **Audit** | Log di sicurezza (solo Admin) |
| **Famiglia** | Gestione membri e inviti (solo Admin) |
| **Impostazioni** | Configurazione, lingua, diagnostica OS |

### Installare la PWA

- **iOS/Safari**: condividi → "Aggiungi alla schermata Home"
- **Android/Chrome**: menu ⋮ → "Aggiungi a schermata Home"
- **Desktop/Chrome**: icona nell'omnibar → "Installa Domus"

---

## 5. Dashboard e Stanze

La dashboard mostra tutte le entità attive. Le card si aggiornano **in tempo reale** via WebSocket.

Ogni card mostra: icona, nome, stato corrente, stanza, disponibilità e controllo rapido (toggle, slider).

### Gestione Aree

1. Vai in **Aree** → **"+ Nuova Area"**
2. Inserisci nome, icona e area genitore opzionale
3. Clicca **"Salva"**

Le aree supportano gerarchie (es. `Casa` → `Primo Piano` → `Camera da Letto`).

---

## 6. Dispositivi ed Entità

### Concetti chiave

| Termine | Significato |
|---|---|
| **Dispositivo** | Hardware fisico (es. lampadina Zigbee, sensore temperatura) |
| **Entità** | Capacità logica esposta da un dispositivo (es. `light.camera_letto`) |
| **Integrazione** | Adattatore che connette un protocollo a Domus |

### Protocolli supportati

| Protocollo | Modalità | Note |
|---|---|---|
| **MQTT generico** | Bidirezionale | Configura mapping topic↔entità in `domus.toml` |
| **Zigbee2MQTT** | Auto-discovery | Rilevamento automatico dispositivi Zigbee |
| **ESPHome** | Telemetria + Comandi | Mapper MQTT per sensori/attuatori ESPHome |
| **Tuya / Xiaomi / eWeLink / VeSync** | Cloud con circuit-breaker | Richiede Internet, con graceful fallback |

### Inviare un comando

Dalla card entità, usa i controlli inline. Per comandi avanzati:

1. Apri il pannello dettaglio dell'entità
2. Clicca **"Invia Comando"**
3. Scegli il servizio (es. `turn_on`, `set_brightness`)
4. Imposta i parametri e conferma

I comandi sono asincroni (`202 Accepted`). L'entità si aggiorna quando il dispositivo risponde via MQTT.

---

## 7. Automazioni

Le automazioni eseguono azioni automaticamente in risposta a eventi, stati o orari.

### Struttura

```
[Trigger] → [Condizioni (opzionale)] → [Azioni]
```

### Tipi di Trigger

| Tipo | `type` DSL | Parametri chiave |
|---|---|---|
| Cambio stato entità | `state` | `entity_id`, `from`, `to` |
| Soglia numerica | `numeric_state` | `entity_id`, `above` / `below` |
| Evento puntuale | `time` | `at` (es. `"07:30"`) |
| Evento applicazione | `event` | `event_type` |

### Tipi di Azione

| Tipo | `type` DSL | Parametri chiave |
|---|---|---|
| Comando entità | `call_service` | `entity_id`, `service`, `params` |
| Attiva scena | `activate_scene` | `scene_id` |
| Attendi | `delay` | `seconds` |

### Esempio: Accendi le luci al tramonto

```json
{
  "name": "Luci tramonto",
  "trigger": { "type": "time", "at": "20:00" },
  "conditions": [
    { "type": "weekday", "weekdays": ["mon","tue","wed","thu","fri","sat","sun"] }
  ],
  "actions": [
    {
      "type": "call_service",
      "entity_id": "light.soggiorno",
      "service": "turn_on",
      "params": { "brightness": 180 }
    }
  ]
}
```

### Modalità di esecuzione

| Modalità | Comportamento |
|---|---|
| `single` | Ignora nuovi trigger se già in esecuzione |
| `restart` | Cancella l'esecuzione corrente e ricomincia |
| `queued` | Accoda le nuove esecuzioni |

---

## 8. Scene

Le scene sono **preset multi-dispositivo** attivabili con un click o da automazioni.

**Creare una scena**:
1. Vai in **Scene** → **"+ Nuova Scena"**
2. Assegna nome e icona
3. Aggiungi azioni (`call_service`)
4. Salva

**Attivare una scena**:
- Manualmente: pulsante **"Attiva"** nella card
- Da automazione: azione `activate_scene` con `scene_id`
- Via API: `POST /api/v1/scenes/{id}/activate`

---

## 9. Cronologia e Grafici

**Cronologia** mostra l'andamento nel tempo di qualsiasi entità numerica.

1. Vai in **Cronologia**
2. Seleziona una o più entità
3. Scegli l'intervallo: **1h / 6h / 24h / 7g / 30g**

> [!NOTE]
> Retention predefinita: 30 giorni per stati sensori, 90 giorni per audit log. Configurabile in `domus.toml` sezione `[retention]`.

---

## 10. Gestione Famiglia e Permessi

Domus supporta più utenti con **permessi differenziati per ruolo** (RBAC).

### Ruoli

| Ruolo | Permessi |
|---|---|
| **Admin** | Gestione completa: utenti, automazioni, dispositivi, impostazioni, backup, audit |
| **Operator** | Comanda entità, attiva scene, gestisce automazioni |
| **Viewer** | Solo lettura: vede stati, cronologia, dashboard |

### Invitare un membro della famiglia

1. Vai in **Famiglia** → **"Invita Membro"**
2. Seleziona il ruolo (`operator` o `viewer`)
3. Scegli la scadenza (default: 48 ore)
4. Copia il link generato (`http://domus.local/register?token=...`)
5. Invia il link al membro

Il destinatario apre il link, crea le proprie credenziali e viene registrato con il ruolo assegnato.

### Gestire i membri (solo Admin)

- **Cambia ruolo**: seleziona il membro → "Modifica Ruolo"
- **Rimuovi membro**: pulsante "Rimuovi"
- **Revoca invito**: nella lista inviti → "Revoca"

> [!WARNING]
> Non è possibile rimuovere o degradare l'**ultimo amministratore** del sistema.

---

## 11. Impostazioni e Configurazione

| Scheda | Contenuto |
|---|---|
| **Generale** | Nome hub, lingua, fuso orario |
| **Profilo** | Cambio password e dati personali |
| **Sistema** | Metriche in tempo reale, versione, uptime |
| **Diagnostica** (Domus OS) | Telemetria SoC, slot A/B, export report |
| **Manutenzione** (Domus OS) | Abilitazione SSH temporanea |

---

## 12. Diagnostica Hardware (Domus OS)

> [!NOTE]
> Disponibile solo con **Domus OS** su Raspberry Pi 4.

### Metriche disponibili

| Metrica | Fonte | Note |
|---|---|---|
| **Temperatura SoC** | `/sys/class/thermal/thermal_zone0/temp` | Verde < 60°C · Ambra 60-75°C · Rosso > 75°C |
| **Carico CPU** | `/proc/loadavg` | Media 1m, 5m, 15m |
| **RAM** | `/proc/meminfo` | MB usati / totali + percentuale |
| **Disco `/data`** | `statvfs` | GB usati / totali su partizione persistente |
| **Slot A/B** | RAUC via D-Bus | Slot attivo e stato conferma |
| **App (container)** | containerd | Numero app attive |

### Export diagnostico

Clicca **"Esporta Report Diagnostico"** per scaricare `domus-diagnostics-YYYY-MM-DD.json`. I dati sensibili vengono automaticamente mascherati con `[REDACTED]`.

---

## 13. Modalità Manutenzione SSH

> [!NOTE]
> Disponibile solo su **Domus OS**. SSH è disattivato per default.

### Attivazione

1. Vai in **Impostazioni** → **"Modalità Manutenzione (SSH)"**
2. Seleziona la durata: 15m · 30m (default) · 60m · 120m (massimo)
3. (Opzionale) Incolla la tua **chiave pubblica SSH** (`ssh-ed25519 AAAA...`)
4. Clicca **"Attiva SSH"**

### Connessione

```bash
ssh root@domus.local
```

### Disattivazione

- **Automatica**: allo scadere del timer impostato
- **Manuale**: clicca **"Disattiva Immediatamente"**

> [!CAUTION]
> La chiave pubblica viene eliminata alla disattivazione. Dovrai reinserirla alla prossima sessione.

---

## 14. Backup e Ripristino

### Creare un backup

```bash
# Backup non cifrato (.tar.gz)
domus-cli backup -db /var/lib/domus/domus.db -config /etc/domus/domus.toml -out /backups

# Backup cifrato AES-256-GCM (.tar.gz.enc) — raccomandato
domus-cli backup -db /var/lib/domus/domus.db -out /backups -passphrase "La-mia-password-sicura"
```

### Ripristinare da backup

> [!WARNING]
> Il ripristino sovrascrive il database corrente. Ferma Domus prima: `sudo systemctl stop domus`

```bash
# Ripristino da backup non cifrato
domus-cli restore -archive /backups/domus-backup-20261004-120000.tar.gz -out /var/lib/domus

# Ripristino da backup cifrato
domus-cli restore \
  -archive /backups/domus-backup-20261004-120000.tar.gz.enc \
  -out /var/lib/domus \
  -passphrase "La-mia-password-sicura"

sudo systemctl start domus
```

### Automazione backup (cron)

```bash
# Backup giornaliero alle 03:00, mantieni ultimi 7 giorni
0 3 * * * /usr/local/bin/domus-cli backup -db /var/lib/domus/domus.db \
  -out /backups -passphrase "$(cat /etc/domus/backup.key)" && \
  find /backups -name "domus-backup-*.tar.gz.enc" -mtime +7 -delete
```

---

## 15. Log di Audit

La sezione **Audit** (solo Admin) registra ogni azione significativa:

| Campo | Descrizione |
|---|---|
| **Data/Ora** | Timestamp UTC |
| **Attore** | Utente che ha eseguito l'azione |
| **Azione** | Tipo di evento (login, entity_command, user_invite, ecc.) |
| **Target** | Oggetto dell'azione (ID entità, ID utente) |
| **Payload** | Dettagli JSON dell'operazione |

I log sono **immutabili**: non cancellabili dall'interfaccia. Retention default: 90 giorni.

---

## 16. Riferimento Configurazione (`domus.toml`)

Percorso: `/etc/domus/domus.toml` (nativa) o `./domus.toml` (locale).  
Tutti i parametri possono essere sovrascritti con variabili d'ambiente `DOMUS_*`.

```toml
[server]
host = "0.0.0.0"          # 0.0.0.0 = tutte le interfacce
port = 8080               # Porta HTTP
cookie_secure = false     # true se dietro reverse proxy HTTPS

[database]
path = "./data/domus.db"  # Percorso SQLite
wal_mode = true           # Write-Ahead Logging (consigliato)
busy_timeout_ms = 5000    # Timeout lock database (ms)
max_open_conns = 1        # 1 per SQLite single-writer

[logging]
level = "info"            # debug | info | warn | error
format = "json"           # json (produzione) | text (sviluppo)

[auth]
session_ttl = "24h"               # Durata sessione utente
rate_limit_login_per_min = 10     # Tentativi login max per minuto/IP
argon2_time = 1                   # Parametro temporale Argon2id
argon2_memory = 65536             # Memoria Argon2id (64 MiB)
argon2_threads = 4                # Thread Argon2id

[retention]
state_history_days = 30     # Giorni storico stati sensori
audit_log_days = 90         # Giorni retention log audit
automation_run_days = 30    # Giorni retention esecuzioni automazioni
---

## 17. Tuya Local & Smart Life (LAN-First)

Domus supporta la comunicazione diretta via LAN con i dispositivi Tuya e Smart Life (prese smart, lampadine CCT/RGB, termostati, ciabatte smart).

- **Porta UDP 6666/6667**: ascolto broadcast locale per discovery automatica di nuovi dispositivi.
- **Porta TCP 6668**: canale dati crittografato AES-128 (v3.1, v3.3) o HMAC-SHA256 (v3.4, v3.5).
- **Zero Cloud**: dopo aver estratto la `local_key` iniziale, i dispositivi comunicano interamente sulla rete locale senza dipendere dai server Tuya.

---

## 18. Mappa della Casa e Piani (Floor Plan)

La sezione **Mappa Piani** (`/floorplan`) consente di visualizzare la casa suddivisa in livelli (es. Piano Terra, Primo Piano, Giardino).

- **Piani Multi-Livello**: navigazione a schede tra i piani configurati con livello numerico.
- **Posizionamento Visuale (Drag-and-Drop)**: in modalità *Modifica Posizioni*, seleziona un dispositivo e fai clic sulla planimetria per impostare le coordinate X/Y percentuali (responsive su ogni schermo).
- **Controllo Diretto dai Marker**: fai clic su un marker di luce o presa per accenderla/spegnerla istantaneamente con feedback visivo a luminescenza (glow).

---

## 19. Faccende Domestiche e Promemoria (Chores)

La sezione **Faccende** (`/chores`) organizza i compiti domestici e le scadenze:

- **Ricorrenze**: compiti una tantum o ricorrenti (giornalieri, settimanali, mensili).
- **Assegnazione & Priorità**: assegna i compiti ai membri della famiglia con priorità *Bassa*, *Media*, *Alta*, o *Urgente*.
- **Promemoria Notifiche Automatiche**: il background scheduler controlla periodicamente i compiti in scadenza e invia notifiche in-app e push.
- **Completamento con Auto-Riprogrammazione**: contrassegnando un compito ricorrente come completato, Domus genera automaticamente la scadenza successiva.

---

## 20. App Mobile iOS e Android (Capacitor)

Domus include una configurazione pronta per Capacitor (`web/capacitor.config.ts`), permettendo di distribuire l'applicazione come app nativa per iOS e Android:

```bash
cd web
npm install @capacitor/core @capacitor/cli @capacitor/ios @capacitor/android
npx cap add ios
npx cap add android
npx cap sync
```

- **Discovery mDNS**: l'app rileva automaticamente il server `domus.local` sulla rete Wi-Fi di casa.
- **Fallback Remoto**: connessione sicura da remoto tramite HTTPS / Tailscale / WireGuard.
- **Notifiche Push Native**: ricezione promemoria compiti e allarmi sensori anche con app chiusa.

---

## 21. Risoluzione dei Problemi

### `domus.local` non raggiungibile

1. Verifica che dispositivo e Raspberry Pi siano sulla stessa rete LAN.
2. Su Windows, installa [Bonjour Print Services](https://support.apple.com/kb/DL999).
3. Trova l'IP del Raspberry Pi nel pannello DHCP del router e accedi direttamente: `http://192.168.x.y`.

### Il servizio non si avvia

```bash
sudo journalctl -u domus -n 100 --no-pager
sudo systemctl status domus
```

### Password amministratore dimenticata

```bash
sudo systemctl stop domus
domus-cli reset-password -db /var/lib/domus/domus.db -user admin
sudo systemctl start domus
```

### Entità sempre offline

1. Verifica il broker MQTT: `systemctl status mosquitto`
2. Controlla i log MQTT: `journalctl -u domus | grep mqtt`
3. Verifica che il dispositivo pubblichi sul topic MQTT corretto.

### Database bloccato (`SQLITE_BUSY`)

```toml
[database]
busy_timeout_ms = 10000  # Aumenta a 10 secondi
```

### Temperatura SoC critica (> 75°C)

1. Verifica che il dissipatore sia installato correttamente.
2. Controlla il flusso d'aria attorno al case.
3. Considera un case con ventola attiva per ambienti caldi.

---

## 18. Domande Frequenti (FAQ)

**D: Domus funziona senza connessione Internet?**  
R: Sì. Tutte le funzioni core funzionano completamente offline. Internet serve solo per aggiornare il software e le integrazioni cloud opzionali.

**D: Posso usare Domus senza Raspberry Pi?**  
R: Sì. Domus gira su qualsiasi Linux x86-64 via Docker o installazione nativa.

**D: Quanti dispositivi supporta Domus?**  
R: Non c'è un limite fisso. Su Raspberry Pi 4 con 4 GB RAM, funziona correttamente con centinaia di entità MQTT simultanee.

**D: I dati escono dalla mia rete?**  
R: No, mai. Domus non trasmette dati a server remoti. Le integrazioni cloud opzionali comunicano direttamente con i cloud dei produttori, non tramite Domus.

**D: Come aggiorno Domus OS?**  
R: Domus OS usa aggiornamenti **A/B atomici** con RAUC. Un aggiornamento non riuscito ripristina automaticamente la slot precedente.

**D: Posso aggiungere un'integrazione personalizzata?**  
R: Sì. Implementa l'interfaccia `Integration` in Go e aggiungi il tuo adapter in `internal/plugins/`.

---

## Appendice A: API REST — Panoramica rapida

Documentazione completa e interattiva: `http://domus.local/docs`

| Gruppo | Endpoint principale |
|---|---|
| **Auth** | `POST /api/v1/auth/login` · `GET /api/v1/auth/me` |
| **Onboarding** | `POST /api/v1/auth/setup` |
| **Inviti** | `POST /api/v1/family/invites` · `POST /api/v1/auth/register` |
| **Entità** | `GET /api/v1/entities` · `POST /api/v1/entities/{id}/command` |
| **Dispositivi** | `GET /api/v1/devices` |
| **Aree** | `GET /api/v1/areas` · `POST /api/v1/areas` |
| **Mappa Piani** | `GET /api/v1/floors` · `POST /api/v1/floors` · `GET/PUT /api/v1/floors/{id}/markers` |
| **Faccende Domestiche** | `GET /api/v1/chores` · `POST /api/v1/chores` · `POST /api/v1/chores/{id}/complete` |
| **Automazioni** | `GET /api/v1/automations` · `POST /api/v1/automations` |
| **Scene** | `POST /api/v1/scenes/{id}/activate` |
| **Cronologia** | `GET /api/v1/entities/{id}/history?from=...&to=...` |
| **Famiglia** | `GET /api/v1/family/members` · `PUT /api/v1/family/members/{id}/role` |
| **Sistema** | `GET /api/v1/system/health` · `GET /api/v1/system/metrics` |
| **OS Diagnostica** | `GET /api/v1/system/diagnostics` · `GET /api/v1/system/diagnostics/export` |
| **OS Manutenzione** | `POST /api/v1/system/maintenance/enable` · `POST /api/v1/system/maintenance/disable` |
| **Audit** | `GET /api/v1/audit` |

---

## Appendice B: WebSocket

**Endpoint**: `ws://domus.local/ws`  
**Autenticazione**: cookie di sessione valido (`domus_session`)

### Formato messaggi

```json
{
  "type": "entity_state_changed",
  "entity_id": "light.soggiorno",
  "state": { "on": true, "brightness": 180 },
  "changed_at": "2026-10-04T18:30:00Z"
}
```

---

*Domus v1.0.0 — Ottobre 2026 · Licenza MIT*
