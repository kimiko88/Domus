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


## Interactive Smart Life account authorization (OAuth 2.0)

This is separate from the project-level `TUYA_CLOUD_CLIENT_ID` / `TUYA_CLOUD_CLIENT_SECRET` flow. It uses a Tuya **App Authorization** key pair and the authorization-code flow (`grant_type=2`) to grant Domus access to one Smart Life user account.

Before enabling it in Domus:

1. In the Tuya Developer Platform, open the Cloud project, then **Devices → Link App Account → Configure OAuth 2.0 Authorization**.
2. Enable authorization for the target Smart Life/OEM app and select the matching data center.
3. Register the exact public HTTPS callback URL, for example `https://domus.example.com/api/v1/integrations/tuya-oauth/callback`.
4. Create/use the **App Authorization** client ID and secret for this OAuth flow. Do not reuse the project simple-mode credentials unless Tuya explicitly issued the pair for App Authorization.
5. Configure the H5 authorization URL generated/provided for the app in `TUYA_OAUTH_AUTHORIZE_URL`. Domus adds `client_id`, `callback_url`, and a one-time `state` value to that URL.

Server environment:

| Variable | Required | Description |
| --- | --- | --- |
| `TUYA_OAUTH_APP_CLIENT_ID` | yes | App Authorization client ID |
| `TUYA_OAUTH_APP_CLIENT_SECRET` | yes | App Authorization secret; server-only |
| `TUYA_OAUTH_IDENTIFIER` | yes | App authorization identifier / configured bundle identifier |
| `TUYA_OAUTH_AUTHORIZE_URL` | yes | Tuya H5 authorization URL for the linked app |
| `TUYA_OAUTH_CALLBACK_URL` | yes | Exact HTTPS callback registered in the Tuya console |
| `TUYA_OAUTH_BASE_URL` | no | Regional Tuya OpenAPI endpoint; defaults to EU |

The callback is intentionally unauthenticated because Tuya calls it directly. It accepts only a short-lived, single-use random state created by an authenticated administrator and bound to that Domus user. The authorization code is exchanged server-side. The database stores access and refresh tokens only after AES-256-GCM envelope encryption using Domus' existing master key. Token values are never returned by status APIs. Refresh is serialized in-process and the newly rotated refresh token is persisted together with the access token before a request uses it. Disconnect deletes the encrypted token record.

The OAuth endpoints are administrator-only except for the provider callback:

- `POST /api/v1/integrations/tuya-oauth/authorize` — create a 10-minute state and return the Tuya authorization URL.
- `GET /api/v1/integrations/tuya-oauth/callback` — exchange the provider code and store the encrypted user tokens.
- `GET /api/v1/integrations/tuya-oauth/status` — report whether the current administrator has a linked account.
- `GET /api/v1/integrations/tuya-oauth/devices` — refresh tokens when needed and list devices belonging to that authorized Tuya UID.
- `POST /api/v1/integrations/tuya-oauth/disconnect` — remove the current administrator's connection.

Use HTTPS end-to-end and keep the Domus master key backed up securely. Losing the master key makes encrypted OAuth credentials unrecoverable. Tuya requires the callback URL and H5 authorization configuration to match the app and data center; OAuth cannot be activated until those values are configured in the Tuya console.

## PDF receipt and invoice scanning

The scanner accepts PDFs up to 3 MiB. Domus first extracts an existing PDF text layer; if none is present, it rasterizes at bounded resolution and runs Tesseract OCR using Italian and English language data. At most the first 20 pages are processed, with a 20-second execution deadline and a 64 KiB extracted-text limit. The temporary PDF and page images are removed after processing; they are not persisted. The extracted text is parsed by the same receipt/invoice parser and must be reviewed before import.

The Docker runtime installs `poppler-utils`, `tesseract-ocr`, and Italian/English Tesseract language data. Non-Docker installations must provide `pdftotext`, `pdftoppm`, `tesseract`, and the `ita`/`eng` language packs on PATH.
