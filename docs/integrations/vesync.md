# VeSync Cloud integration

Domus includes an administrator-managed VeSync cloud connector for compatible Levoit / VeSync devices. This is an optional integration: Domus core automations and the local Xiaomi MiIO adapter do not depend on VeSync.

## Before you start: what must be registered?

**You do not need a VeSync developer account, developer portal project, App ID, App Secret, API key, OAuth redirect URL, QR code, pairing code, or token copied from a website.**

You need:

1. A normal VeSync consumer account (an account created with **Sign Up** in the official VeSync app or account site; a guest session is not sufficient for third-party-connected services).
2. Your Levoit / VeSync device already added to that account and shown as online in the VeSync app.
3. The email address and password used for that VeSync account.
4. The country associated with the account and its IANA time zone (for example `Europe/Rome`).

Official app/account entry points:

- [VeSync account portal](https://account.vesync.com/)
- [VeSync app on Apple App Store](https://apps.apple.com/app/vesync/id1132150426)
- [VeSync app on Google Play](https://play.google.com/store/apps/details?id=com.etekcity.vesync)

For product-specific first-time pairing, follow the instructions in the device's own manual and in the VeSync app. The usual flow is **Sign Up / Log In → + / Add Device → select the product → follow the in-app Wi-Fi setup**. Keep the phone and device close during Bluetooth-assisted onboarding, and grant the permissions requested by the app. VeSync's product manuals explicitly distinguish a personal account from a guest session.

## Configure VeSync in Domus

1. Sign in to Domus as an administrator.
2. Open **Settings → Integrations → Vendors → VeSync Cloud**.
3. Enter the same **email** and **password** used in the VeSync app.
4. Select the account country. For an Italian account use `IT`; for example, `DE`, `FR`, `ES`, `GB`, `US`, `CA`, and `JP` are also available in the current form.
5. Leave the time zone as `Europe/Rome` for Italy, or enter the correct IANA time-zone identifier for the home (for example `Europe/Berlin`).
6. Select **Connect account**.
7. Domus performs the VeSync two-step sign-in, then immediately requests the account's device list. The encrypted session is saved only after both operations succeed.
8. The supported devices appear below the form. Use **Refresh devices** to fetch the latest cloud state.

You do **not** have to copy an access token or an authorization code out of VeSync. VeSync issues those values during the server-side login flow and Domus stores the resulting session.

### Which country / region should I choose?

Choose the country of the VeSync account, not the country of the Raspberry Pi or the physical device IP address. Domus maps `US`, `CA`, `MX`, and `JP` to the US API region; other country codes use the EU region in the current client. The cloud's returned supported region is checked during login.

For a home in Italy, the normal values are:

| Field | Value |
| --- | --- |
| Country | `IT` |
| Region selected by Domus | `EU` |
| Time zone | `Europe/Rome` |

## What Domus stores and sends

- The account password is used only during the login exchange. Domus does not persist the plaintext password.
- The VeSync account ID and session token are stored in the existing encrypted integration configuration using Domus' AES-256-GCM secret storage.
- Status and device-list APIs do not return the account token.
- Disconnect clears the stored session.
- The provider cloud receives authentication and device/control requests. This is **not local-LAN control** and requires Internet access even when the device is connected to the same Wi-Fi network as Domus.
- VeSync's device API used by this integration is an unofficial / reverse-engineered interface, not a public stable developer API. A VeSync service-side change can break login or commands.

## Current MFA / two-factor limitation

The current Domus VeSync login flow implements the ordinary password → authorization-code → session-token exchange. It does **not yet implement an additional MFA challenge** (email OTP, authenticator OTP, or backup code).

If VeSync responds that the account requires a second factor, Domus will report an authentication failure. **Do not disable MFA or weaken account security to work around this limitation.** Use the VeSync app normally and wait until Domus adds a supported MFA challenge flow before connecting an account that enforces that step. The [open-source VeSync client project issue on MFA](https://github.com/webdjoe/pyvesync/issues/367) documents a separate challenge flow, but that does not mean this Domus connector currently supports it.

## Device support and controls

The first release has explicit capability profiles for selected Levoit purifier models and regional variants (including selected Core, Vital and Everest Air models). Domus exposes write controls only when the returned `deviceType` or `configModule` matches one of its known profiles.

For a recognized purifier, the UI can provide:

- **Refresh status** — request the current purifier state from VeSync.
- **Power on / off** — send the purifier power switch command.
- **Mode** — choose a mode that is declared for that exact model (for example sleep, auto, pet or turbo where supported).
- **Fan level** — select an integer level from that model's capability profile.

Other VeSync products (outlets, bulbs, switches, humidifiers, fans, air fryers and thermostats) may appear in the device list, but are read-only in this initial connector. A product being visible in the VeSync mobile app does not mean Domus has implemented its command profile.

If your purifier is not recognized, do not guess another model's command values. Provide the model/device type and firmware information when requesting support.

## Troubleshooting

| Symptom | What to check |
| --- | --- |
| Login rejected | Verify the email/password in the VeSync app first; check the selected country and Internet connectivity. |
| MFA / verification challenge | Not supported by this Domus version. Do not disable MFA as a workaround. |
| Login works but no devices appear | Confirm that the device is online in the VeSync app and that this is the same account that owns/shares the device. |
| Device appears but no power/mode controls | The device model is not in the current purifier capability allowlist; it remains read-only. |
| Commands fail while the app works | Check Internet access from the Domus host, reconnect the account if the session expired, and check whether the model/firmware is supported. |
| Device is offline in the VeSync app | Complete Wi-Fi setup again in the official app before refreshing Domus. |

For server-side diagnostics, inspect Domus logs without enabling request-body or credential logging:

```bash
sudo journalctl -u domus -n 200 --no-pager
```

Do not paste your VeSync password, session token, raw cloud response, or account identifiers into an issue, chat, screenshot, or public log.

## Administrator API

All routes require an authenticated Domus administrator and live under `/api/v1/integrations/vesync`:

- `GET /status` — configured/connected state, cloud region and account country; no secrets.
- `PUT /config` — sign in and save a session. JSON fields: `email`, `password`, `country_code`, `time_zone`.
- `GET /devices` — refresh the cloud device list.
- `POST /devices/{id}/commands` — accepts only `refresh`, `set_power`, `set_mode`, and `set_fan_level`.
- `POST /disconnect` — clear the stored session.

Example configuration request (use HTTPS and an administrator session):

```json
{
  "email": "account@example.com",
  "password": "entered-in-the-domus-form",
  "country_code": "IT",
  "time_zone": "Europe/Rome"
}
```

The password is a request input only; do not store it in scripts, shell history, reverse-proxy logs or frontend build-time environment variables.

## Further reading

- [VeSync account portal](https://account.vesync.com/)
- [VeSync / pyvesync authentication documentation](https://webdjoe.github.io/pyvesync/latest/authentication/)
- [Levoit product manuals and setup instructions](https://levoit.com/pages/product-manuals)
