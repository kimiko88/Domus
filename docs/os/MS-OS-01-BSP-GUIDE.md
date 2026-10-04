# Domus OS - Milestone MS-OS-01: Board Support Package (Raspberry Pi 4 ARM64)

Questo documento descrive l'implementazione del Board Support Package (BSP) per **Domus OS** su piattaforma target **Raspberry Pi 4 Model B (ARM64)**, basato su **Buildroot 2024.02 LTS**, **Linux Kernel 6.6 LTS** e **U-Boot**.

---

## 1. Caratteristiche Hardware del Target

- **SoC**: Broadcom BCM2711, Quad-core Cortex-A72 (ARMv8) 64-bit @ 1.5 GHz / 1.8 GHz.
- **Architettura**: ARM64 / AArch64 (`BR2_aarch64=y`, `BR2_cortex_a72=y`).
- **RAM Minima supportata**: 2 GB LPDDR4-3200.
- **RAM Raccomandata**: 4 GB LPDDR4-3200.
- **Storage Primario Raccomandato**: SSD USB 3.0 (tramite adattatore SATA/NVMe USB con supporto UASP).
- **Storage Secondario**: Scheda microSD ad alte prestazioni (UHS-I A2).
- **Rete**: Gigabit Ethernet nativo (controller BCM54213PE via RGMII).
- **Connettività Periferiche Domotiche**: Controller USB con dongle Zigbee su `/dev/ttyUSB0` o `/dev/ttyACM0` (chipset TI CC2652P, ConBee II, Silicon Labs CP2102N).

---

## 2. Layout del Disco e Partizionamento GPT

L'immagine disco raw generata da `genimage` (`domus-os-rpi4.img`) implementa una tabella **GPT (GUID Partition Table)** rigida con partizionamento ridondante A/B:

```
+---------------------------------------------------------------------------------------------------------+
| GUID Partition Table (GPT)                                                                              |
+-------+--------------------+----------------+----------------+--------------------+---------------------+
| Part  | Label              | Filesystem     | Dimensione     | Modalità Mount     | Descrizione         |
+-------+--------------------+----------------+----------------+--------------------+---------------------+
| 1     | domus-boot         | FAT32 (vfat)   | 128 MB         | RO (RW in update)  | Firmware RPi4,      |
|       |                    |                |                |                    | config.txt,         |
|       |                    |                |                |                    | u-boot.bin,         |
|       |                    |                |                |                    | boot.scr, Image,    |
|       |                    |                |                |                    | device tree blobs   |
+-------+--------------------+----------------+----------------+--------------------+---------------------+
| 2     | domus-root-a       | SquashFS       | 1.2 GB         | RO (Slot A)        | Immagine rootfs     |
|       |                    |                |                |                    | immutabile attiva   |
+-------+--------------------+----------------+----------------+--------------------+---------------------+
| 3     | domus-root-b       | SquashFS       | 1.2 GB         | RO (Slot B)        | Slot passivo per    |
|       |                    |                |                | (Inattivo/Update)  | aggiornamenti A/B   |
+-------+--------------------+----------------+----------------+--------------------+---------------------+
| 4     | domus-data         | ext4           | Resto disco    | RW (Persistent)    | Database SQLite,    |
|       |                    | (journaling)   | (min 512 MB)   |                    | configurazioni e    |
|       |                    |                |                |                    | volumi container    |
+-------+--------------------+----------------+----------------+--------------------+---------------------+
```

---

## 3. Struttura dei File BSP nel Repository

```
os/
├── configs/
│   └── domus_rpi4_defconfig         # Configurazione master Buildroot ARM64 per RPi4
├── board/
│   └── raspberrypi4-64/
│       ├── genimage.cfg             # Definizione GPT e assemblaggio partizioni
│       ├── config.txt               # Parametri firmware RPi (64-bit, U-Boot, UART)
│       ├── cmdline.txt              # Parametri kernel (SquashFS RO, cgroups v2)
│       ├── boot.cmd                 # Script U-Boot con logica A/B e bootcounter
│       ├── post-build.sh            # Hook filesystem: fstab, os-release, permessi
│       └── post-image.sh            # Hook generazione boot.scr, genimage, sha256
├── build-container/
│   └── Dockerfile                   # Ambiente ermetico e riproducibile per la build
└── scripts/
    └── build.sh                     # Runner per validazione e build automatica
```

---

## 4. Modalità di Compilazione Riproducibile

### Modalità 1: Validazione Sintattica Istantanea (Locale / CI)
Verifica l'integrità dei file di configurazione, la sintassi di bash e i parametri del kernel senza dover avviare la cross-compilazione completa:

```bash
# Esecuzione del runner di validazione
./os/scripts/build.sh --validate-only

# Esecuzione dei test TDD automatizzati Go
go test -v ./test/os/...
```

### Modalità 2: Compilazione Ermetica tramite Docker (Raccomandata)
Garantisce un ambiente di cross-compilazione identico su qualsiasi macchina host (Linux, macOS, Windows con Docker Desktop o WSL2):

```bash
./os/scripts/build.sh --docker
```

### Modalità 3: Compilazione Nativa su Linux / WSL2
Richiede i pacchetti host standard (`build-essential`, `bison`, `flex`, `mtools`, `genimage`):

```bash
./os/scripts/build.sh
```

---

## 5. Flashing e Scrittura dell'Immagine

L'output generato si troverà in `build/buildroot/buildroot-2024.02.5/output/images/`:
- `domus-os-rpi4.img`: Immagine disco raw completa pronta per il flashing.
- `domus-os-rpi4.img.sha256`: Checksum di integrità.

### Flashing con Raspberry Pi Imager
1. Scaricare e aprire [Raspberry Pi Imager](https://www.raspberrypi.com/software/).
2. Selezionare **Scegli Sistema Operativo** -> **Usa immagine personalizzata**.
3. Selezionare `domus-os-rpi4.img`.
4. Selezionare il drive di destinazione (SSD USB o microSD).
5. Fare clic su **Scrivi**.

### Flashing da terminale Linux / macOS
```bash
# Verificare il device corretto (es. /dev/sdb o /dev/rdisk2)
lsblk

# Scrivere l'immagine con blocco a 4M e sincronizzazione disco
sudo dd if=domus-os-rpi4.img of=/dev/sdX bs=4M status=progress conv=fsync
```

---

## 6. Procedura di Verifica e Collaudo al Boot

1. **Collegamento Seriale Debug (Opzionale)**:
   - Collegare un convertitore USB-Seriale UART (3.3V):
     - Pin 6 (GND) -> GND
     - Pin 8 (GPIO 14 / TXD) -> RXD adattatore
     - Pin 10 (GPIO 15 / RXD) -> TXD adattatore
   - Connettersi via terminale a 115200 baud (es. `screen /dev/ttyUSB0 115200` o PuTTY).

2. **Accensione**:
   - Collegare alimentatore USB-C (5V / 3A).
   - Verificare output bootloader EEPROM di Raspberry Pi.
   - U-Boot viene caricato: visualizza il banner `Domus OS Bootloader` e seleziona lo slot attivo (`domus-root-a`).
   - Il kernel Linux 6.6 si avvia e monta la rootfs in modalità SquashFS read-only (`ro`).
   - Viene montata la partizione `/data` e le cartelle effimere in `tmpfs`.
   - Viene rilasciato l'indirizzo IP via DHCP sull'interfaccia Ethernet.
