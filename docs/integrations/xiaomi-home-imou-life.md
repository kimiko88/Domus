# Xiaomi Home and Imou Life integrations

Domus provides administrator-only setup and management under **Settings → Integrations → Xiaomi Home and Imou Life**. Project credentials, OAuth tokens, local device tokens and temporary callback capabilities are stored in the existing AES-256-GCM encrypted integration configuration. They are never returned by device-list APIs.

## Xiaomi Home

### Local miIO pairing

The local adapter supports IP devices that expose the legacy Xiaomi miIO protocol over UDP port 54321.

- Enter the device's private LAN IP address, a display name, and the device's 32-character hexadecimal local miIO token.
- Domus probes the device with `miIO.info` before storing it.
- The token is stored encrypted and is never returned by the API.
- Local targets must be private IP literals. Hostnames, public IPs, loopback, URLs and arbitrary ports are rejected.

This is not universal Xiaomi Home support. BLE Mesh/Zigbee products, hubs that expose only cloud APIs, and models without a compatible local miIO token are not covered. Do not enter an account password or use a token you are not authorized to administer.

### Xiaomi Open Platform OAuth and MIoT Cloud

The cloud adapter connects a Xiaomi account using Xiaomi Open Platform OAuth. It does not need a token extracted from the local app or a Xiaomi account password.

Prerequisites:

1. Register a Xiaomi Open Platform application and obtain its App ID and App Secret. The application needs the **Mi Home service scope (6000)** and the **profile scope (1)**. Xiaomi must approve the requested service access for the application.
2. Select the region that matches the Xiaomi account and its Mi Home devices.
3. Configure a publicly reachable HTTPS base URL for OAuth and event callbacks. A tailnet-only Tailscale address is not reachable from Xiaomi's cloud callback service.
4. Register the exact OAuth callback URL returned by Domus. Configure any reverse proxy/CDN to redact callback paths in its access logs.

After OAuth, Domus can list homes and devices, fetch the public MIoT-Spec-V2 instance document, create entities from readable/writable properties, read properties, write only properties declared writable, invoke actions declared by the device, and subscribe to property/event/device-status notifications.

The MIoT entity generator preserves service/property identifiers, access modes, value format, unit, value range and value list in entity attributes. Device commands issued through the ordinary Domus entity endpoint are checked against the MIoT specification before they are sent to Xiaomi.

Only cloud devices with a valid MIoT-Spec-V2 device URN can be synchronized automatically. Products that are not exposed by the account's Xiaomi region or do not publish a compatible specification are skipped rather than assigned guessed entities.

Administrator API:

- `PUT /api/v1/integrations/xiaomi-home/cloud/config` with `client_id`, `client_secret`, `region` (`eu`, `us`, or `sg`) and `public_base_url`.
- `GET /api/v1/integrations/xiaomi-home/cloud/status`
- `GET /api/v1/integrations/xiaomi-home/cloud/authorize`
- `GET /api/v1/integrations/xiaomi-home/cloud/callback/{secret}` is the public OAuth callback. Domus also verifies a random OAuth `state` value to prevent callback forgery.
- `GET /api/v1/integrations/xiaomi-home/cloud/devices`
- `GET /api/v1/integrations/xiaomi-home/cloud/homes`
- `POST /api/v1/integrations/xiaomi-home/cloud/sync` imports supported devices and MIoT entities.
- `GET /api/v1/integrations/xiaomi-home/cloud/devices/{did}/spec`
- `GET /api/v1/integrations/xiaomi-home/cloud/devices/{did}/properties`
- `PUT /api/v1/integrations/xiaomi-home/cloud/devices/{did}/properties`
- `PUT /api/v1/integrations/xiaomi-home/cloud/devices/{did}/actions`
- `POST /api/v1/integrations/xiaomi-home/cloud/events/subscribe`
- `POST /api/v1/integrations/xiaomi-home/cloud/disconnect`

The cloud API base host is selected from a fixed region allowlist; arbitrary API hosts cannot be entered. Access and refresh tokens, MAC key metadata, OAuth state and callback capabilities are encrypted at rest. Access tokens are refreshed when their known expiry approaches.

## Imou Life

### Open Platform OAuth

Imou uses the official server-side Open Platform OAuth flow. Domus does not ask for an Imou account password and never puts the project Secret Key in the frontend.

Prerequisites:

1. Create an Imou Open Platform project and obtain its Access Key, Secret Key and data-centre identifier.
2. Obtain the user-consent H5 authorization URL from the project configuration.
3. Configure a publicly reachable HTTPS base URL for callbacks. A tailnet-only Tailscale URL cannot receive callbacks from Imou's cloud service.
4. Register the exact OAuth callback URL returned by Domus. Configure the reverse proxy, CDN and load balancer in front of Domus to redact both OAuth and event callback capability paths from their access logs and analytics.

Domus exchanges the OAuth code for the end-user `Ot_` token and refresh token. It keeps these separate from the project administrator `At_` token, which is obtained only when configuring event push or playback. Project keys, tokens and callback capabilities are AES-256-GCM encrypted at rest.

### Cameras, capabilities, PTZ and events

After authorization, Domus lists the devices shared with or bound to that Imou user. It then reads each device's capability list. Property reads/writes and service invocations are accepted only when the device advertises the corresponding capability; property access modes and service input types/ranges are checked before sending a command.

PTZ controls appear only when the camera advertises `ptz.aim.relative`. Domus sends the documented `horizontal`, `vertical`, `zoom` and `duration` inputs. Other PTZ services and presets are not assumed to exist on every model.

The event callback can be enabled for the supported Imou callback categories. Domus maps alarm, capability/person/motion detections and device online/offline changes to its event bus so they can be used by notifications and automations. Callback paths are protected by independent random 256-bit capabilities and redacted from Domus application logs. The event deduplication cache is currently in memory; a restart clears its history.

### Live preview and recording playback

The browser never receives an RTSP URL or camera stream credentials.

- Live preview obtains a temporary stream URL from Imou and proxies it through the Domus backend.
- Recording search uses the camera's advertised `record.file.list` capability.
- Recording playback requests the documented Imou temporary video-download URL, decrypts the AES-256-GCM response on the server, and streams the media through the same proxy.
- FFmpeg must be installed on the Domus host (for Raspberry Pi OS/Debian, install the distribution's `ffmpeg` package).
- The proxy limits concurrent previews to two and the stream session to 30 minutes. FFmpeg is launched as a separate process with a restricted protocol allowlist, bounded output resolution/frame rate and no provider URL in its command-line arguments or logs.
- The RTSP host must be one of the approved Imou media domains and resolve to public IP addresses. Private, loopback, link-local and carrier-grade NAT destinations are rejected.

The Imou video-download API requires a project administrator token and playback permission for the device. A valid user OAuth grant does not guarantee that every device/account has that additional playback permission; provider permission errors are returned without exposing the stream URL.

Administrator API:

- `PUT /api/v1/integrations/imou-life/config` with `access_key`, `secret_key`, `data_center`, `authorization_url` and `public_base_url`.
- `GET /api/v1/integrations/imou-life/status`
- `GET /api/v1/integrations/imou-life/authorize`
- `GET /api/v1/integrations/imou-life/callback/{secret}` completes the user OAuth grant.
- `GET /api/v1/integrations/imou-life/devices?page=1&page_size=50`
- `GET /api/v1/integrations/imou-life/devices/{deviceID}/capabilities?channel=0`
- `GET /api/v1/integrations/imou-life/devices/{deviceID}/preview?channel=0&stream=0`
- `GET /api/v1/integrations/imou-life/devices/{deviceID}/recordings?begin={RFC3339}&end={RFC3339}&channel=0`
- `GET /api/v1/integrations/imou-life/devices/{deviceID}/recording-preview?begin={RFC3339}&end={RFC3339}&channel=0&business_type=localRecord`
- `GET /api/v1/integrations/imou-life/devices/{deviceID}/messages?begin={RFC3339}&end={RFC3339}`
- `POST /api/v1/integrations/imou-life/devices/{deviceID}/services/{serviceID}` invokes a service only if the capability is advertised.
- `POST /api/v1/integrations/imou-life/events/config` and `GET /api/v1/integrations/imou-life/events/status`
- `POST /api/v1/integrations/imou-life/disconnect`

All management endpoints are administrator-only. The OAuth callback and event receiver are the only public Imou routes; put only those routes behind the external callback ingress. Do not expose the whole Domus administration interface to the public Internet.

## Current limitations and next work

- The current configuration supports one active cloud account per provider. Multi-account profiles, account selector UI and per-account household isolation are still needed.
- Xiaomi local miIO is still a limited adapter; it does not cover all Xiaomi device classes or local BLE Mesh/Zigbee devices.
- Xiaomi MIoT cloud synchronization currently imports readable/writable properties and uses the normal Domus command path. Full spec event argument decoding, per-model action forms, property polling/backoff and a richer diagnostics view remain future work.
- Imou event deduplication is in-memory rather than persistent across restarts; delivery retry/backoff and per-account device isolation need further work.
- Imou recording playback depends on the provider's project-level playback permissions and model capabilities.
- Both providers still need explicit key rotation UI, multi-account support and richer per-device diagnostics.

Never send project secrets, refresh tokens, Xiaomi tokens, decrypted video URLs or RTSP credentials to frontend logs, analytics or third-party services.
