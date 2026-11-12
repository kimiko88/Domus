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
- `POST /api/v1/integrations/tuya-cloud/devices/{deviceID}/commands` — send up to 32 device datapoint commands.

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

## PDF receipt and invoice scanning

The scanner accepts PDFs up to 3 MiB. Domus first extracts an existing PDF text layer; if none is present, it rasterizes at bounded resolution and runs Tesseract OCR using Italian and English language data. At most the first 20 pages are processed, with a 20-second execution deadline and a 64 KiB extracted-text limit. The temporary PDF and page images are removed after processing; they are not persisted. The extracted text is parsed by the same receipt/invoice parser and must be reviewed before import.

The Docker runtime installs `poppler-utils`, `tesseract-ocr`, and Italian/English Tesseract language data. Non-Docker installations must provide `pdftotext`, `pdftoppm`, `tesseract`, and the `ita`/`eng` language packs on PATH.
