# Domus OS - Milestone MS-OS-03: Dual-Slot A/B Update Engine con RAUC & U-Boot

Questo documento descrive il sistema di **aggiornamenti atomici ridondanti A/B** basato su **RAUC (Robust Auto-Update Controller)**, **U-Boot bootcounter** e **crittografia asimmetrica x.509** per Domus OS su Raspberry Pi 4 Model B (ARM64).

---

## 1. Architettura Dual-Slot A/B

Per eliminare il rischio di bricking o corruzione durante gli aggiornamenti di sistema (OTA o offline), Domus OS implementa una strategia a doppio slot simmetrico:

```
               +-------------------------------------------+
               |             U-Boot Bootloader             |
               |  (Legge: boot_slot, bootcount, bootlimit) |
               +---------------------+---------------------+
                                     |
                    +----------------+----------------+
                    |                                 |
                    v                                 v
          +-------------------+             +-------------------+
          |   domus-root-a    |             |   domus-root-b    |
          |  (SquashFS RO)    |             |  (SquashFS RO)    |
          |  Slot A - 1.2 GB  |             |  Slot B - 1.2 GB  |
          +-------------------+             +-------------------+
                    \                                 /
                     \                               /
                      v                             v
               +-------------------------------------------+
               |        Partizione Dati Condivisa          |
               |             /data (ext4 RW)               |
               +-------------------------------------------+
```

---

## 2. Flusso di Aggiornamento Atomico

1. **Download o Inserimento Bundle**: L'utente o il Supervisore riceve il bundle firmato `domus-os-<version>.raucb`.
2. **Verifica Crittografica**: RAUC verifica l'autenticità e l'integrità del bundle contro il certificato della CA presente in `/etc/rauc/ca.cert.pem`.
3. **Scrittura sullo Slot Passivo**: Se il sistema è in esecuzione su **Slot A**, RAUC scrive l'immagine SquashFS direttamente sul device a blocchi di **Slot B** (`/dev/disk/by-partlabel/domus-root-b`).
4. **Attivazione dello Slot**: RAUC aggiorna le variabili di U-Boot:
   - `boot_slot=b`
   - `bootcount=0`
   - `upgrade_available=1`
5. **Riavvio dell'Appliance**: Il sistema si riavvia sul nuovo slot.
6. **Conferma dello Stato (Healthcheck)**:
   - U-Boot avvia il kernel dello Slot B e incrementa `bootcount` a 1.
   - Domus Supervisor avvia Domus Core ed esegue l'healthcheck interno (HTTP 200 OK).
   - Se l'healthcheck ha successo, il supervisore invoca `rauc status mark-good` (oppure `domus-update mark-good`).
   - U-Boot resetta `bootcount=0` e imposta `upgrade_available=0`: **lo Slot B è confermato permanentemente**.

---

## 3. Meccanismo di Rollback Automatico

Se il nuovo slot subisce un kernel panic, un blocco hardware o Domus Core fallisce l'healthcheck:
- Il sistema si riavvia (per watchdog o timeout).
- U-Boot rileva `bootcount >= bootlimit` (soglia predefinita: **3 tentativi falliti**).
- U-Boot inverte automaticamente il puntatore allo slot sano precedente:
  ```
  boot_slot: b -> a
  bootcount: 0
  ```
- L'appliance torna operativa sullo slot funzionante **senza alcuna perdita di dati**, poiché il database e i parametri utente risiedono nella partizione `/data`.

---

## 4. PKI Crittografica per la Firma delle Release

Ogni bundle RAUC è firmato digitalmente tramite crittografia a chiave asimmetrica x.509 CMS:
- **CA Cert**: [`os/rauc/certs/ca.cert.pem`](file:///c:/Users/chimi/Desktop/Programmazione/Domus/os/rauc/certs/ca.cert.pem) - incorporato nell'immagine rootfs.
- **Signing Cert**: [`os/rauc/certs/development.cert.pem`](file:///c:/Users/chimi/Desktop/Programmazione/Domus/os/rauc/certs/development.cert.pem)
- **Signing Key**: [`os/rauc/certs/development.key.pem`](file:///c:/Users/chimi/Desktop/Programmazione/Domus/os/rauc/certs/development.key.pem)

Per rigenerare nuove chiavi:
```bash
./os/scripts/generate-rauc-keys.sh
```

---

## 5. Generazione e Applicazione di un Bundle

### 5.1 Creazione del Bundle (.raucb)
```bash
# Creazione del bundle firmato con checksum SHA256
./os/scripts/build-rauc-bundle.sh \
    --version 1.0.1 \
    --rootfs /percorso/a/rootfs.squashfs
```

### 5.2 Applicazione tramite CLI Appliance
```bash
# Consultare lo stato attuale degli slot A/B
domus-update status

# Installare un aggiornamento locale o via URL
domus-update install /data/updates/domus-os-1.0.1.raucb
# oppure
domus-update install https://releases.domus-hub.org/v1.0.1/domus-os-1.0.1.raucb

# Confermare la salute dello slot dopo il riavvio
domus-update mark-good

# Trigger manuale di rollback allo slot precedente
domus-update rollback
```
