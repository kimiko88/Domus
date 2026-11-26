# BroadLink Local

Domus BroadLink Local communicates directly with supported RM remotes over the local network. It does not require a BroadLink account, cloud token, or external Python process.

## Supported models

- RM Mini / RM Mini 3 (IR)
- RM4 Mini / RM4C Mini and RM4 TV Mate (IR)
- RM Pro and RM4 Pro / RM4C Pro (IR + RF)

Support is detected from the BroadLink device type reported by UDP discovery. Smart plugs, sensors and other BroadLink product families are deliberately ignored.

## Setup

1. Add the remote to Wi-Fi in the official BroadLink app first.
2. Reserve a stable DHCP address for the remote when possible.
3. In the BroadLink app, disable **Lock device** for local control.
4. In Domus, open **Integrations → BroadLink Local**, scan the LAN (or scan a private IP), and pair the discovered RM remote.
5. To capture a button, select IR (or RF on Pro devices), start learning, press the physical remote button near the RM device, then save the captured command with a name.

Local discovery sends UDP broadcast packets to port 80. Control and learning use encrypted BroadLink UDP command packets to port 80. The AES session key returned by device authentication is encrypted at rest using Domus' configured envelope cipher.

## Plugin Manager e persistenza

BroadLink è registrato come tipo singleton `broadlink_local`. Nelle installazioni precedenti, Domus adotta automaticamente i telecomandi già accoppiati creando l'istanza persistente con l'ID storico `broadlink_local`. Un'installazione nuova non riceve un'istanza vuota automaticamente: l'amministratore può crearla dal Plugin Manager.

Il Plugin Manager governa avvio, arresto, riavvio e stato abilitato/disabilitato. Le route HTTP storiche risolvono l'adapter gestito a ogni richiesta; quando l'istanza è disabilitata o non è disponibile, le richieste restituiscono HTTP 503. Non viene usato un puntatore legacy come fallback.

Le tabelle `broadlink_devices` e `broadlink_commands` sono mantenute per compatibilità dei dati. Le chiavi AES di sessione continuano a essere cifrate con l'Envelope Cipher Domus; i codici IR/RF sono conservati nel database locale. BroadLink resta single-instance finché queste tabelle non saranno separate per integration ID.

## API

All management routes require an authenticated administrator:

- `GET /api/v1/integrations/broadlink-local/devices`
- `POST /api/v1/integrations/broadlink-local/discovery/scan` — optional body `{"ip":"192.168.1.50"}`
- `POST /api/v1/integrations/broadlink-local/devices` — `{"ip":"192.168.1.50","name":"TV soggiorno"}`
- `GET /api/v1/integrations/broadlink-local/devices/{deviceID}/commands`
- `POST /api/v1/integrations/broadlink-local/devices/{deviceID}/learn/start` — `{"medium":"ir"}` or `{"medium":"rf"}`
- `POST /api/v1/integrations/broadlink-local/devices/{deviceID}/learn/result` — `{"name":"TV Power","medium":"ir"}`
- `POST /api/v1/integrations/broadlink-local/devices/{deviceID}/commands/send` — `{"name":"TV Power","medium":"ir"}`
- `DELETE /api/v1/integrations/broadlink-local/devices/{deviceID}`

A paired remote is provisioned as a Domus device with a writable `remote.*` entity. Automations can issue `send_command` with `params.name` and optional `params.medium`.

## Notes

IR/RF transmission is one-way: an RM remote cannot confirm that the television, air conditioner, or other target appliance actually changed state. A successful UDP response confirms acceptance by the BroadLink remote, not the final state of the controlled appliance. RF capture is supported only for Pro models.
