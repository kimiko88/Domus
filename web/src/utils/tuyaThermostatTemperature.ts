export interface TuyaThermostatSetpoint {
  encoded: number
  degrees: number
}

// Device value (left) -> physical thermostat temperature (right).
export const TUYA_THERMOSTAT_SETPOINTS: readonly TuyaThermostatSetpoint[] = [
  { encoded: 25.8, degrees: 1 },
  { encoded: 25.9, degrees: 1.5 },
  { encoded: 26, degrees: 2 },
  { encoded: 27, degrees: 7 },
  { encoded: 28, degrees: 12 },
  { encoded: 29, degrees: 17 },
  { encoded: 29.8, degrees: 21 },
  { encoded: 30, degrees: 22 }
] as const

export function encodeTuyaThermostatSetpoint(degrees: number): number | undefined {
  if (!Number.isFinite(degrees)) return undefined
  return TUYA_THERMOSTAT_SETPOINTS.find(item => Math.abs(item.degrees - degrees) < 0.001)?.encoded
}

export function decodeTuyaThermostatSetpoint(encoded: number): number | undefined {
  if (!Number.isFinite(encoded)) return undefined
  return TUYA_THERMOSTAT_SETPOINTS.find(item => Math.abs(item.encoded - encoded) < 0.051)?.degrees
}
