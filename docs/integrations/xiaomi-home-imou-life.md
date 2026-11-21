# Xiaomi Home and Imou Life integrations

## Xiaomi Home (initial local MIoT/miIO support)

Domus currently implements a **local-only first stage** for Xiaomi IP devices that expose the legacy miIO UDP protocol on port 54321.

- Pair from an administrator account using the device's LAN IP, a display name, and the device's 32-character / 16-byte local miIO token.
- Domus verifies the token by requesting `miIO.info` before storing the device.
- The token is stored only inside the encrypted integration configuration and is never returned by the API.
- Targets are restricted to private LAN IP addresses. Hostnames, public IP addresses, loopback, URLs and arbitrary ports are rejected.
- This stage exposes device identity and reachability only; it does not yet publish a complete set of entities or generic write commands.

The local protocol does not cover every Xiaomi Home product. BLE Mesh/Zigbee devices, hubs that expose only cloud APIs, and models without a usable local miIO token are not supported by this first stage. Do not paste a Xiaomi account password or extract tokens from another person's account. Use only a local token you are authorized to administer.

Administrator API:
- `GET /api/v1/integrations/xiaomi-home/status`
- `GET /api/v1/integrations/xiaomi-home/devices`
- `POST /api/v1/integrations/xiaomi-home/devices` with `{"ip":"192.168.1.20","name":"Purificatore","token":"<32 hex chars>"}`
- `GET /api/v1/integrations/xiaomi-home/devices/{id}`
- `POST /api/v1/integrations/xiaomi-home/devices/{id}/probe`
- `DELETE /api/v1/integrations/xiaomi-home/devices/{id}`

## Imou Life (official Open Platform OAuth)

The Imou integration uses the official server-side Open Platform API. Domus does not ask users for their Imou account password and does not embed project secrets in the browser.

Prerequisites:
1. Create an Imou Open Platform developer project and obtain its Access Key, Secret Key and data-centre identifier. The project/API permissions must be approved by Imou.
2. Obtain the Imou user-consent H5 authorization URL from the project configuration.
3. Provide Domus with a **publicly reachable HTTPS base URL**. A tailnet-only Tailscale URL is not reachable by Imou's cloud callback service. A narrowly routed public HTTPS reverse proxy or Tailscale Funnel may be used; do not expose the whole Domus administration interface without additional access controls.
4. Save the exact callback URL returned by Domus in the Imou project console.

Administrator API:
- `PUT /api/v1/integrations/imou-life/config` with `access_key`, `secret_key`, `data_center`, `authorization_url` and `public_base_url`.
- `GET /api/v1/integrations/imou-life/status`
- `GET /api/v1/integrations/imou-life/authorize`
- Complete the Imou consent flow. Imou redirects the one-time authorization code to the configured callback route, protected by a random 256-bit path capability. Domus redacts that path from its own request logs.
- `GET /api/v1/integrations/imou-life/devices?page=1&page_size=50`
- `GET /api/v1/integrations/imou-life/devices/{deviceID}/stream?channel=0&stream=0`
- `POST /api/v1/integrations/imou-life/disconnect`

Access and refresh tokens, project keys and callback capability are AES-256-GCM encrypted at rest using Domus' existing master key. The stream endpoint is administrator-only and returns a sensitive RTSP URL with no-cache headers. It is an initial integration surface, **not yet a video proxy**: avoid sharing the returned URL. A future release should proxy streams through an isolated media service and expose only short-lived viewer sessions.

## Current limitations and next work

- Xiaomi: MIoT specification discovery, supported-model capability mapping, safe read/write property lists, entity creation, polling, push/event support, and optional cloud OAuth.
- Imou: motion/person event webhooks, channel/recording search, camera entity creation, media proxy, and secure viewer sessions.
- Both: reconnect/backoff, per-account isolation, token/key rotation UX, diagnostics without secrets, and end-to-end tests.

Never send project secrets, refresh tokens, Xiaomi tokens or RTSP URLs to frontend logs, analytics or third-party services.
