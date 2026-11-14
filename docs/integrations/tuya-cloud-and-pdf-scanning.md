# Tuya / Smart Life Cloud integration

Domus supports two distinct Tuya paths:

- **Tuya Local**: LAN discovery and direct device control using the device local key.
- **Tuya Cloud OpenAPI**: server-to-server access to devices visible to a Tuya Developer Cloud project. This is the official Tuya OpenAPI path; it does not scrape Smart Life traffic or ask users for their Smart Life password.

## Configure the Tuya Cloud project

1. Create a Cloud Development project in the Tuya IoT Developer Platform.
2. Choose the data centre that matches the Smart Life account/device region.
3. Subscribe to the required API products and authorize the project for device management and device control.
4. Link the Smart Life/OEM application to the cloud project so the project can access devices associated with that app.
5. Set the application schema if the device-list API requires it.

Configure these variables on the Domus server (never in frontend build variables):

| Variable | Required | Description |
| --- | --- | --- |
| `TUYA_CLOUD_CLIENT_ID` | yes | Tuya Cloud Access ID / client ID |
| `TUYA_CLOUD_CLIENT_SECRET` | yes | Tuya Cloud Access Secret; keep it private |
| `TUYA_CLOUD_BASE_URL` | no | Regional OpenAPI URL; defaults to `https://openapi.tuyaeu.com` |
| `TUYA_CLOUD_SCHEMA` | no | Linked app schema used to scope device listing |

Supported regional hosts are restricted to Tuya's known HTTPS OpenAPI endpoints. The server obtains and caches a project access token using Tuya's signed `grant_type=1` flow. Credentials and tokens remain server-side.

## Domus API

All routes require an authenticated administrator:

- `GET /api/v1/integrations/tuya-cloud/devices` — list devices visible to the configured project.
- `GET /api/v1/integrations/tuya-cloud/devices/{deviceID}/functions` — list writable datapoints exposed by the device.
- `POST /api/v1/integrations/tuya-cloud/devices/{deviceID}/commands` — send up to 32 device datapoint commands; Domus checks each code against the device function list before sending.

Example request:

```json
{
  "commands": [
    { "code": "switch_1", "value": true }
  ]
}
```

Only use datapoint codes and value types exposed by the specific device's Tuya product specification. Domus validates the request shape but cannot infer a device's supported functions.

## Authorization boundary

This implementation uses the Tuya Developer Cloud project's server-side authorization and the linked app/schema permissions. It is **not** an interactive end-user OAuth consent screen for arbitrary Smart Life accounts. Device visibility depends on Tuya's project/app linkage, subscribed API products, region, and account permissions. Do not paste a Smart Life password into Domus.


## Collegamento Diretto Smart Life (QR Code + Token Server-to-Server)

Per gli utenti dell'app ufficiale **Smart Life** (o Tuya Smart), non è necessario (né supportato) il flusso OAuth 2.0 Web Redirect H5: Tuya riserva infatti l'OAuth 2.0 unicamente alle aziende che hanno pubblicato una propria app OEM proprietaria.

Il collegamento corretto, rapido e senza dipendenze cloud secondarie avviene tramite la funzione **Link Tuya App Account (QR Code)** combinata con il token Server-to-Server (`grant_type=1`):

### 1. Associazione su Tuya IoT Platform
1. Accedi al tuo progetto su [iot.tuya.com](https://iot.tuya.com/) (Cloud → Development → apri il progetto Domus).
2. Assicurati che in alto a destra sopra la tabella il Data Center sia impostato su **Central Europe Data Center** (`https://openapi.tuyaeu.com`).
3. Vai nella scheda **Devices** e poi nella sottoscheda **Link Tuya App Account**.
4. Clicca sul pulsante blu **Add App Account**: comparirà un QR Code.
5. Apri l'app **Smart Life** sullo smartphone: vai su **Io** (in basso a destra) → tocca l'icona dello scanner QR `[ - ]` in alto a destra → inquadra il QR Code → tocca **Conferma login** → seleziona **Automatic Link**, scegli permessi **Read and Write** e salva.
6. Nella tabella comparirà il tuo account Smart Life con il relativo **UID** (inizia solitamente con `eu...`).

### 2. Parametri di Configurazione in Domus
I 4 parametri possono essere inseriti comodamente dall'interfaccia web di Domus (scheda *Smart Life Cloud*) oppure configurati nel file `.env`:

| Variabile | Alias | Obbligatorio | Descrizione |
| --- | --- | --- | --- |
| `TUYA_CLIENT_ID` | `TUYA_CLOUD_CLIENT_ID` | Sì | Access ID della scheda Overview del progetto Tuya |
| `TUYA_CLIENT_SECRET` | `TUYA_CLOUD_CLIENT_SECRET` | Sì | Access Secret della scheda Overview (cifrato in SQLite) |
| `TUYA_UID` | `TUYA_CLOUD_UID` | Sì | L'UID generato in Link Tuya App Account dopo aver scansionato il QR |
| `TUYA_ENDPOINT` | `TUYA_CLOUD_BASE_URL` | No | Default: `https://openapi.tuyaeu.com` |

### 3. Flusso Server-to-Server & Local Key Handover
1. **Richiesta Token**: Domus chiama `GET /v1.0/token?grant_type=1` usando firma HMAC-SHA256, senza richiedere popup di redirect all'utente.
2. **Importazione Dispositivi**: Con il token e il `TUYA_UID`, Domus interroga `GET /v1.0/users/{TUYA_UID}/devices`, ottenendo l'elenco istantaneo dei dispositivi con le relative `local_key`.
3. **Passaggio alla LAN**: Cliccando su *"Attiva LAN Locale (Local Key)"*, le credenziali vengono salvate cifrate in `tuya_local_devices`. Da quel momento in poi, ogni comando e lettura di stato opera **esclusivamente su rete locale via LAN (TCP 6668 / UDP broadcast)** senza contattare i server Tuya.

### Endpoints REST Domus API

Tutte le rotte richiedono ruolo amministratore:

- `GET /api/v1/integrations/tuya-cloud/config` — stato della configurazione attuale.
- `POST /api/v1/integrations/tuya-cloud/config` — salva e valida le credenziali Server-to-Server con cifratura master key.
- `DELETE /api/v1/integrations/tuya-cloud/config` — rimuove la configurazione attiva.
- `GET /api/v1/integrations/tuya-cloud/devices` — elenca i dispositivi associati all'UID con flag di stato locale (`local_configured`, `local_active`, `local_ip`).
- `POST /api/v1/integrations/tuya-cloud/sync-local` — trasferisce automaticamente tutti i dispositivi e le `local_key` al runtime locale `tuya_local`.
- `POST /api/v1/integrations/tuya-cloud/devices/{deviceID}/commands` — invia comandi; se il dispositivo è configurato localmente, l'invio avviene in LAN a latenza zero.

## PDF receipt and invoice scanning

The scanner accepts PDFs up to 3 MiB. Domus first extracts an existing PDF text layer; if none is present, it rasterizes at bounded resolution and runs Tesseract OCR using Italian and English language data. At most the first 20 pages are processed, with a 20-second execution deadline and a 64 KiB extracted-text limit. The temporary PDF and page images are removed after processing; they are not persisted. The extracted text is parsed by the same receipt/invoice parser and must be reviewed before import.

The Docker runtime installs `poppler-utils`, `tesseract-ocr`, and Italian/English Tesseract language data. Non-Docker installations must provide `pdftotext`, `pdftoppm`, `tesseract`, and the `ita`/`eng` language packs on PATH.


### Publish only the callback without buying a domain (Tailscale + Cloudflare Quick Tunnel)

A Domus installation performed with `deployments/scripts/install.sh` runs as a native systemd service; Docker is not required. The installer defaults to port `8080`. Keep the Domus UI private on Tailscale and publish only the provider callback through a loopback-only Nginx proxy.

This no-domain setup uses a **Cloudflare Quick Tunnel**. Its `trycloudflare.com` hostname is temporary and changes when the tunnel is recreated, so use it for initial OAuth testing rather than as a durable production endpoint. A stable production callback needs a stable public hostname.

1. Install Nginx and `cloudflared` on the Domus ARM64 host. Install `cloudflared` using Cloudflare's official Debian/Ubuntu package repository: https://pkg.cloudflare.com/.
2. Copy `deployments/nginx/domus-tuya-oauth.conf` to `/etc/nginx/sites-available/domus-tuya-oauth`, enable it, and reload Nginx:

   ```bash
   sudo apt-get update
   sudo apt-get install -y nginx
   sudo install -m 0644 deployments/nginx/domus-tuya-oauth.conf /etc/nginx/sites-available/domus-tuya-oauth
   sudo ln -s /etc/nginx/sites-available/domus-tuya-oauth /etc/nginx/sites-enabled/domus-tuya-oauth
   sudo nginx -t
   sudo systemctl reload nginx
   ```

   The supplied proxy listens only on `127.0.0.1:8787`, forwards only the exact callback path to `127.0.0.1:8080`, accepts only GET, returns 404 for every other path, and disables access logging so OAuth query parameters are not written to Nginx access logs. It does not expose the Domus UI.
3. Verify the local proxy before opening the tunnel:

   ```bash
   curl -i http://127.0.0.1:8787/
   curl -i http://127.0.0.1:8787/api/v1/integrations/tuya-oauth/callback
   ```

   The first request must return 404. The callback request returns 400 only after all required `TUYA_OAUTH_*` settings are present and Domus has been restarted; without an enabled OAuth handler it returns 404. The 400 response is expected because no one-time `state` and authorization `code` were supplied.
4. Start a Quick Tunnel (no `cloudflared tunnel login` or domain is needed):

   ```bash
   cloudflared tunnel --url http://127.0.0.1:8787
   ```

   Keep this process running during the entire authorization. Copy the generated HTTPS hostname, for example `https://random-words.trycloudflare.com`.
5. Set the exact callback in `/etc/domus/domus.env` (keep all App Authorization secrets on the server):

   ```dotenv
   TUYA_OAUTH_CALLBACK_URL=https://random-words.trycloudflare.com/api/v1/integrations/tuya-oauth/callback
   ```

   Keep the other required `TUYA_OAUTH_*` variables from the preceding section. Protect the file with `root:domus` ownership and mode `0640` (the installer already applies these permissions when it creates the file), then restart Domus:

   ```bash
   sudo chown root:domus /etc/domus/domus.env
   sudo chmod 0640 /etc/domus/domus.env
   sudo systemctl restart domus
   sudo systemctl --no-pager --full status domus
   ```
6. Register the **identical full HTTPS callback URL** in the Tuya Developer Platform under Cloud project → Devices → Link App Account → Configure OAuth 2.0 Authorization. Confirm that the selected Smart Life app and data center match the App Authorization credentials.
7. Open Domus over Tailscale, go to Integrations, and start the Smart Life connection. Complete the consent screen. Tuya returns to the public callback; after successful code exchange Domus displays a static Italian confirmation page. Close that tab and return to the existing Domus tab over Tailscale.

The OAuth success page deliberately does not redirect to `/integrations` on the public tunnel hostname. It uses no external assets, scripts, forms, or redirects, and sends no-store and restrictive Content-Security-Policy headers.

**Quick Tunnel limitations:** the generated hostname is ephemeral, the foreground process must remain running, and restarting it can invalidate the callback registered in Tuya. If the hostname changes, update both `TUYA_OAUTH_CALLBACK_URL` and the Tuya callback registration, then restart Domus. Do not use the Quick Tunnel as a permanent integration endpoint. Cloudflare documents this mode at https://developers.cloudflare.com/cloudflare-one/networks/connectors/cloudflare-tunnel/do-more-with-tunnels/trycloudflare/.
