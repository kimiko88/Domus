# Xiaomi Home and Imou Life integrations

Domus provides administrator-only setup and management under **Settings → Integrations → Xiaomi Home and Imou Life**. Project credentials, OAuth tokens, local device tokens and temporary callback capabilities are stored in the existing AES-256-GCM encrypted integration configuration. They are never returned by device-list APIs.

## Xiaomi Home

### Local miIO pairing (no developer registration required)

This is the **local** Xiaomi path. It is independent of Xiaomi Open Platform OAuth.

**You do not need** a Xiaomi developer account, an App ID/App Secret, an OAuth scope, a public HTTPS callback, or a copied cloud access token to pair a local miIO device. You do need the device's own **32-character hexadecimal miIO LAN token**, its private LAN IP, and permission to administer that device.

#### Step 1 — prepare the device and network

1. Add the device to the official Xiaomi Home / Mi Home app and complete its Wi-Fi setup.
2. Make sure the device is online in the app and connected to the same LAN/VLAN as the Domus Raspberry Pi. Guest Wi-Fi, client isolation, separate VLANs or inter-VLAN firewall rules can prevent discovery and control.
3. Find the device's IPv4 address in the router's DHCP client list. A DHCP reservation is recommended so the IP does not change.
4. Confirm that the product uses the miIO LAN protocol. BLE Mesh/Zigbee sub-devices, products controlled only through a hub, and cloud-only models are not covered by this adapter.

#### Step 2 — obtain the local token

The token is a device-specific secret, not the Xiaomi account password and not the OAuth access token used by Xiaomi Open Platform.

The recommended community-documented route is to run python-miio on **your own trusted computer**, not to submit the account password to Domus or to a token-extraction website:

```bash
# Linux / macOS. A separate virtual environment avoids changing system Python.
python3 -m venv ~/.venvs/domus-miio
source ~/.venvs/domus-miio/bin/activate
python -m pip install --upgrade pip
python -m pip install python-miio micloud

# The command prompts interactively for Xiaomi account details and region.
# Do not put the password in the command line.
miiocli cloud list
```

At the prompts:

- enter the Xiaomi account email/username;
- enter the Xiaomi account password only in the interactive password prompt;
- choose the Mi Home cloud locale. Try all first; otherwise choose the locale matching the Mi Home account region (de, us, sg, cn, etc.).

The command lists the cloud account's devices and, where the account/API permits it, their model, LAN IP and local token. Match the row to the device by **name, model and IP**. The token Domus expects is exactly 32 hexadecimal characters (0-9, a-f).

Security precautions:

- Run this on a computer you control. miiocli cloud list uses a community-maintained client to sign in to Xiaomi Cloud; it is not an official Xiaomi token-export feature.
- Never paste the Xiaomi password into Domus, a shell command, a web form, issue tracker, screenshot, or chat.
- Treat the extracted token like a password. Do not commit it, save it in a shared notes file, or upload it to an online decoder or token-extraction website.
- Keep the terminal private because the command prints tokens to its output. Close the terminal after copying the one relevant token and remove any captured terminal logs.
- If the cloud list does not expose a usable token for the device/region, stop here. Legacy guides involving old APKs, downgraded mobile apps, rooted phones or private app databases are unreliable and can weaken account/device security. This Domus guide does not recommend installing old unofficial APKs.

The python-miio documentation describes miiocli cloud list, its optional micloud dependency and the locale prompt: [Getting started / Obtaining tokens](https://python-miio.readthedocs.io/en/latest/discovery.html). Its legacy token-extraction page explicitly warns that those methods are historical and may not work with current app versions: [Legacy token extraction](https://python-miio.readthedocs.io/en/stable/legacy_token_extraction.html).

#### Step 3 — discover the device from Domus

1. Open Domus as an administrator.
2. Go to **Settings → Integrations → Vendors → Xiaomi Home · local control**.
3. Select **Discover devices on LAN**.
4. Wait for the scan to finish. A discovered result shows the LAN IP and the miIO device ID. The discovery handshake does **not** return or recover the token and may not identify the model.
5. Select **Use this IP**. If discovery does not find the device, enter the private IPv4 address manually.

The scanner probes active private IPv4 subnets attached to Ethernet/Wi-Fi interfaces. It excludes loopback, link-local, public, VPN and common virtual interfaces; networks wider than /24 are narrowed to the /24 containing the host interface address, and a scan is capped at 512 targets. It does not scan the Internet or arbitrary hostnames.

#### Step 4 — pair and validate

1. Enter a display name.
2. Enter the exact 32-character local token from Step 2 in **Local miIO token**.
3. Select **Pair device**.
4. Domus sends authenticated miIO.info to the device. The token is saved only if the device accepts it.
5. The token is stored in the existing AES-256-GCM encrypted integration configuration. Device-list and status APIs never return it.

Do not enter the Xiaomi account password in the token field. Do not paste the Xiaomi Open Platform App Secret or an OAuth access token there.

#### Step 5 — control and understand model limits

The current local UI provides connection probing and write controls only for recognized zhimi.airpurifier.* purifier models:

- Power on / off.
- Supported purifier modes (auto, silent, favorite) where accepted by the device.

Other devices may pair and pass miIO.info, but their model-specific write controls remain disabled until a validated command profile is implemented. The API allows a bounded read_properties request; it does not expose arbitrary miIO method names. Pairing a device therefore does not imply full control support.

For local communication, Domus and the device must remain mutually reachable on the LAN. The miIO transport uses UDP port 54321. Do not forward this port from the Internet to the Raspberry Pi.

#### Troubleshooting

| Symptom | What to check |
| --- | --- |
| LAN discovery returns no devices | Confirm same LAN/VLAN, disable guest/client isolation, check DHCP IP and local firewall rules; verify the product is an miIO IP device. |
| Device is discovered but pairing fails | Check the IP and the 32-hex token; confirm the token belongs to this exact device and has not changed after a reset/re-pair. |
| miIO.info times out | Check device power, Wi-Fi status, VLAN/firewall and UDP 54321 reachability. |
| Device pairs but no power/mode controls appear | The model is not in the currently supported zhimi.airpurifier.* command profile. |
| Token begins with all zeroes or all f characters | Treat it as a placeholder/invalid token and do not save it. |

#### Local administrator API

All of these routes require an authenticated Domus administrator:

- POST /api/v1/integrations/xiaomi-home/discovery — scan eligible private IPv4 LAN subnets; returns IP/device ID only.
- POST /api/v1/integrations/xiaomi-home/devices — pair after validating the supplied local token with miIO.info. JSON fields: ip, name, token.
- GET /api/v1/integrations/xiaomi-home/status — status and sanitized device list.
- POST /api/v1/integrations/xiaomi-home/devices/{id}/probe — authenticated reachability/model check.
- POST /api/v1/integrations/xiaomi-home/devices/{id}/commands — constrained read_properties, set_power and set_mode operations.
- DELETE /api/v1/integrations/xiaomi-home/devices/{id} — remove a paired device and its stored token.

### Xiaomi Open Platform OAuth and MIoT Cloud (developer registration required)

This is the **cloud** Xiaomi path. It is separate from local miIO pairing above. If you only want local IP/token control, skip this entire section: a Xiaomi Open Platform application is not needed for local miIO.

Unlike local miIO pairing, this cloud integration requires a Xiaomi developer application and a public HTTPS callback.

#### Step 1 — register as a Xiaomi developer and create an application

1. Open the [Xiaomi Developer Platform](https://dev.mi.com/).
2. Sign up with the Xiaomi developer account and complete the requested developer/company or individual information.
3. Submit the developer registration and wait for Xiaomi's review/approval. Xiaomi's developer guide describes this registration and review as a prerequisite to using Xiaomi Account services.
4. After approval, open the developer platform and create an application. Select an application type suitable for a server-backed web application / service.
5. Open the application's service configuration and activate **Xiaomi Account Services** (Account Service). An App ID alone is not sufficient if Account Services have not been activated.
6. Save the generated **App ID** and **App Secret**. These are application credentials, not the Xiaomi account password. Store the App Secret in a password manager until it is entered in Domus.

Official starting points:
- [Xiaomi Developer Platform — beginner's guide](https://dev.mi.com/docs/passport/en/user-guide/)
- [Xiaomi OAuth 2.0 overview](https://dev.mi.com/docs/passport/oauth2/)

#### Step 2 — request the correct Xiaomi scopes

Request the following scopes for the application and ensure Xiaomi has approved them:

| Scope | Purpose in this integration |
| --- | --- |
| `6000` | Mi Home services — required to access the user's Mi Home resources. |
| `1` | Xiaomi user profile — used for the profile/identity information shown by Domus. |

The official [Xiaomi scope list](https://dev.mi.com/docs/passport/en/scopes/) defines scope 6000 as use of Mi Home services and scope 1 as access to the user's profile. Do not request unrelated scopes.

#### Step 3 — provide a public HTTPS callback address

Xiaomi's authorization server must be able to redirect the user's browser to Domus. A Tailscale-only URL such as a `*.ts.net` address is private to your tailnet and is not reachable by Xiaomi's public OAuth service.

1. Choose a stable public HTTPS hostname and configure TLS.
2. Ensure the public ingress exposes only the required callback path(s), not the Domus administration UI or the full API.
3. In Domus, open the Xiaomi cloud section and enter:
   - **App ID** — from the Xiaomi Developer Platform.
   - **App Secret** — from the same application.
   - **Region** — choose the Xiaomi/Mi Home account region (`eu`, `us`, or `sg` in the current Domus form).
   - **Public Base URL** — the externally reachable HTTPS origin, for example `https://domus.example.net` (no callback path appended).
4. Save the configuration. Domus returns the exact OAuth callback URL and event callback URL.
5. Copy the exact OAuth callback URL displayed by Domus into the application's Redirect URI / OAuth callback configuration on the Xiaomi Developer Platform. Match the scheme, hostname, path and any secret path segment exactly. Do not add a trailing slash or change letter case.
6. If the ingress supports path-based routing, forward only the exact OAuth callback and (if used) event callback paths to Domus. Redact those callback URLs from Nginx/CDN/load-balancer access logs because the path contains a temporary capability secret.

The OAuth callback must be publicly reachable over HTTPS. The UI's normal login session can remain private over Tailscale; only the provider callback path needs public reachability. Use the separate callback-only proxy approach described in the deployment/networking documentation rather than publishing the whole Domus site.

#### Step 4 — authorize the Xiaomi account

1. Return to Domus and select **Authorize account**.
2. Domus opens Xiaomi's OAuth consent page. Sign in with the Xiaomi account that owns the Mi Home devices and review the requested scopes.
3. Approve access. Xiaomi redirects the browser to the previously registered callback with a short-lived authorization code and the OAuth state.
4. Domus verifies the state and exchanges the code server-side using the App ID, App Secret and registered redirect URI.
5. The authorization code is single-use and expires after five minutes. Do not try to copy it into a form or reuse it; Domus handles the exchange automatically.
6. Return to the private Domus tab. The integration status should show the account as connected.

Xiaomi's [Authorization Code Grant documentation](https://dev.mi.com/docs/passport/en/authorization-code/) specifies that the redirect URI must match the one registered for the application, the code can be used only once, and it is valid for five minutes.

#### Step 5 — import cloud devices and entities

1. Select **Refresh devices** to load the devices visible to the authorized Xiaomi account/region.
2. Select **Sync entities** to import devices with a supported MIoT-Spec-V2 device URN.
3. Review the created entities and assign them to the appropriate Domus areas.
4. Use the normal Domus device/entity controls and automations. Writes are checked against the MIoT specification's writable access mode.

The cloud adapter can list homes and devices, fetch the public MIoT-Spec-V2 instance document, create entities from readable/writable properties, read properties, write only properties declared writable, invoke actions declared by the device, and subscribe to property/event/device-status notifications.

Only devices exposed by the authorized Xiaomi account and region and publishing a compatible MIoT specification can be synchronized automatically. Unsupported products are skipped rather than assigned guessed entities.

#### Credentials and API routes

The App Secret is entered only in the administrator UI and is stored encrypted. Xiaomi account passwords are entered only on Xiaomi's own consent page; Domus does not ask for or store the Xiaomi account password. Access/refresh tokens, MAC key metadata, OAuth state and callback capabilities are encrypted at rest.

Administrator API:

- `PUT /api/v1/integrations/xiaomi-home/cloud/config` with `client_id`, `client_secret`, `region` (`eu`, `us`, or `sg`) and `public_base_url`.
- `GET /api/v1/integrations/xiaomi-home/cloud/status`
- `GET /api/v1/integrations/xiaomi-home/cloud/authorize`
- `GET /api/v1/integrations/xiaomi-home/cloud/callback/{secret}` — public OAuth callback. Domus also verifies a random OAuth `state` value.
- `GET /api/v1/integrations/xiaomi-home/cloud/devices`
- `GET /api/v1/integrations/xiaomi-home/cloud/homes`
- `POST /api/v1/integrations/xiaomi-home/cloud/sync`
- `GET /api/v1/integrations/xiaomi-home/cloud/devices/{did}/spec`
- `GET /api/v1/integrations/xiaomi-home/cloud/devices/{did}/properties`
- `PUT /api/v1/integrations/xiaomi-home/cloud/devices/{did}/properties`
- `PUT /api/v1/integrations/xiaomi-home/cloud/devices/{did}/actions`
- `POST /api/v1/integrations/xiaomi-home/cloud/events/subscribe`
- `POST /api/v1/integrations/xiaomi-home/cloud/disconnect`

The cloud API base host is selected from a fixed region allowlist; arbitrary API hosts cannot be entered. Access tokens are refreshed when their known expiry approaches.

#### Troubleshooting

| Symptom | What to check |
| --- | --- |
| OAuth says client/application is invalid | Confirm developer registration was approved, Account Services is activated, and the App ID/App Secret belong to the same application. |
| Scope / permission denied | Confirm Xiaomi approved both required scopes (6000 and 1) for this application. |
| Redirect URI mismatch | Re-copy the exact callback URL returned by Domus; compare HTTPS scheme, hostname, full path and case. |
| Callback times out | Check public DNS/TLS and the callback-only reverse-proxy route. A Tailscale-only hostname is not publicly reachable. |
| OAuth state error | Start authorization again from Domus; do not manually replay an old callback URL or code. |
| Account connects but device list is empty | Check Mi Home account, device sharing/ownership and the selected Xiaomi region. Some devices are not exposed through the cloud API. |
| Device appears but does not sync | It may not publish a MIoT-Spec-V2 URN supported by the importer. |

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
