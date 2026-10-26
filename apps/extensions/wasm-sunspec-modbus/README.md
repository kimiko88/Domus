# SunSpec Modbus RTU / RS485 Inverter & Heat Pump Driver (WASI)

Driver certificato e sandboxed per Domus per la comunicazione diretta con inverter fotovoltaici, contatori bidirezionali e pompe di calore su standard **SunSpec Alliance** via RS485 o Modbus TCP.

## Specifiche Tecniche
- **ID Pacchetto**: `domus.wasi.sunspec-modbus`
- **Livello**: Verified (Ufficiale Domus)
- **Runtime**: WebAssembly (WASI) isolato in memoria lineare protetta (zero container Docker)
- **Licenza**: Apache-2.0 (Open Source)
- **Capabilities**: `wasi:serial-port`, `energy:read`, `energy:control`, `events:write`
- **Modelli Supportati**: SunSpec Model 1 (Common), Model 101/103 (Inverter), Model 201/203 (Meter).
