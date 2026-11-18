# Browser Web Push setup (VAPID)

Domus sends browser push notifications with the Web Push protocol. VAPID credentials are **not** TLS certificates and do not replace the HTTPS certificate already used by the web server. The VAPID private key stays on the Domus server; only the public key is returned to the authenticated web client when it creates a browser subscription.

## 1. Generate the server key pair

Build the dedicated generator from the repository:

```bash
make build-vapid
```

On the Domus server, run it as root and use a real contact address controlled by the administrator:

```bash
sudo ./bin/domus-vapid -subject mailto:admin@example.com
```

The command writes the three variables to the existing systemd environment file:

```text
/etc/domus/domus.env
```

It preserves unrelated environment entries, refuses to overwrite an existing VAPID configuration, creates the file atomically and sets permissions to `0600`. The private key is never printed to the terminal. The systemd unit already loads this file through its `EnvironmentFile` directive.

To use a URL as the VAPID contact instead:

```bash
sudo ./bin/domus-vapid -subject https://domus.example.com
```

The variables written are:

```ini
DOMUS_VAPID_PUBLIC_KEY=...
DOMUS_VAPID_PRIVATE_KEY=...
DOMUS_VAPID_SUBJECT=mailto:admin@example.com
```

Do not paste the private key into tickets or chat, do not commit `domus.env`, and keep an encrypted backup of the file. If you lose the private key or rotate the pair, existing browser subscriptions must be recreated using the new public key.

## 2. Restart Domus

```bash
sudo systemctl restart domus
sudo systemctl status domus --no-pager
```

Domus logs whether browser Web Push is enabled, but never logs the private key. The server must have outbound HTTPS access (TCP 443) to the browser push services. Incoming connections to the push provider are not required.

For a non-systemd/manual deployment, export the same three `DOMUS_VAPID_*` variables in the environment of the Domus process before starting it.

## 3. Enable notifications on each browser/device

Sign in to Domus, open **Settings → Browser push notifications**, and press **Enable push notifications**. The browser permission prompt is intentionally opened only after that explicit button press. Each browser profile/device gets its own subscription. Use **Send test notification** to verify delivery.

On iPhone/iPad, first open Domus in Safari, use Share → Add to Home Screen, add it, and then launch Domus from the new Home Screen icon. Allow notifications when prompted. Web Push on iOS is provided to installed Home Screen web apps, not an ordinary Safari tab.

## 4. User-controlled reminders

The settings panel stores preferences per Domus user:

- Household tasks: a Web Push notification is sent when the task scheduler marks an assigned or family task overdue (the scheduler checks every minute).
- Medicines: the reminder lead time is per user (default 7 days). The effective expiry date is used, including the configured period after opening (PAO). Low-stock notifications use each medicine's own minimum-stock threshold.
- Pantry/food: the reminder lead time is per user (default 3 days). Low-stock notifications use each food item's minimum-stock threshold.
- Security, device, automation, network, camera, intercom and finance categories can be enabled or disabled independently.

Medicine and food expiry/stock checks run hourly. A logical reminder is deduplicated per user and item/date; low-stock reminders become eligible again after the quantity rises above the configured minimum and later drops below it. Set an expiry lead time to 0 to disable that type of expiry reminder. Browser permission and Domus preferences both need to allow delivery.

## 5. HTTPS and troubleshooting

Browser Push requires a secure context (HTTPS, or localhost during development), a working service worker and browser permission. Confirm that the web UI is served from the same trusted HTTPS origin configured for Domus. If the Settings panel reports that VAPID is not configured, check that all three environment variables are present in `/etc/domus/domus.env` and restart the service. If a test notification fails, check outbound DNS/TCP 443 connectivity and the Domus service log; expired subscriptions (HTTP 404/410 from the provider) are removed automatically.
