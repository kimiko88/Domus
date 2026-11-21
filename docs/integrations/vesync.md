# VeSync Cloud integration

Domus provides a native VeSync cloud integration for compatible Levoit and other VeSync devices.

## Network and privacy model

- This integration communicates with VeSync cloud endpoints. It is not a local-LAN integration and requires Internet access.
- The API is an unofficial, reverse-engineered client implementation. VeSync may change its authentication flow, payloads, supported regions or device APIs without notice.
- The account password is used only during the two-step login exchange. Domus does not persist it.
- The resulting account ID and cloud token are stored inside Domus' existing encrypted integration configuration.
- The UI and status endpoints never return the account token.
- Disconnect clears the stored encrypted session.
- If VeSync expires or revokes the token, reconnect from the integrations page using the account email and password.

## Setup

1. Open **Integrations → Vendors → VeSync Cloud**.
2. Enter the VeSync account email and password.
3. Select the country associated with the account and the IANA time zone (for example `Europe/Rome`).
4. Domus authenticates using VeSync's authorization-code exchange and validates the resulting session by requesting the device list.
5. The session is encrypted and the sanitized device list is shown in Domus.

Italy and most European countries use the EU API region. The US, Canada, Mexico and Japan use the US API region. If the cloud reports a different region at login, Domus follows the returned supported region.

## Supported endpoints

All routes are administrator-only under `/api/v1/integrations/vesync`:

- `GET /status` returns whether a cloud session is configured; it never returns secrets.
- `PUT /config` authenticates and stores a new encrypted session. Request fields: `email`, `password`, `country_code`, `time_zone`.
- `GET /devices` retrieves a fresh cloud device list.
- `POST /devices/{id}/commands` accepts the allowlisted device operations below.
- `POST /disconnect` clears the stored session.

## Device support

The device list includes sanitized model, status and connection information. Control capabilities are exposed only for purifier models with a known command profile. The initial allowlist includes selected Core 200S/300S/400S/600S, Vital 100S/200S, Everest Air and compatible regional model identifiers.

Supported purifier commands:

- `refresh`: retrieve purifier status.
- `set_power`: boolean on/off.
- `set_mode`: a mode listed for that exact model; manual mode is intentionally controlled through fan level.
- `set_fan_level`: an integer level listed for that exact model.

Other VeSync device types are discovered and displayed but are read-only in this first implementation. Bypass V2 command methods are fixed in the client; arbitrary provider method names cannot be submitted by callers.

## Limitations

- VeSync is cloud-only; commands will not work during an Internet outage even when the device remains reachable on Wi-Fi.
- The provider API is not an official public API. Treat updates to the reverse-engineered protocol as potentially breaking.
- Model capability tables are conservative and should be expanded only after validating a model's device type and command behavior.
- Hardware validation with a real VeSync account and a real supported Levoit device is still required before treating this integration as production-verified.
