export interface TuyaThermostatSetpoint {
  encoded: number
  degrees: number
}

const DEVICE_MIN = 25.8
const DEVICE_MAX = 30
const DEVICE_STEP = 0.1
const THERMOSTAT_MIN = 1
const THERMOSTAT_STEP = 0.5
const STEP_COUNT = Math.round((DEVICE_MAX - DEVICE_MIN) / DEVICE_STEP)

// The thermostat uses a non-standard scale:
// 25.8 -> 1 °C, then each +0.1 device step -> +0.5 °C.
// Keep the whole supported scale in the selector, not only the sample points.
export const TUYA_THERMOSTAT_SETPOINTS: readonly TuyaThermostatSetpoint[] = Array.from(
  { length: STEP_COUNT + 1 },
  (_, index) => ({
    encoded: Number((DEVICE_MIN + index * DEVICE_STEP).toFixed(1)),
    degrees: Number((THERMOSTAT_MIN + index * THERMOSTAT_STEP).toFixed(1))
  })
)

export function encodeTuyaThermostatSetpoint(degrees: number): number | undefined {
  if (!Number.isFinite(degrees)) return undefined

  const step = Math.round((degrees - THERMOSTAT_MIN) / THERMOSTAT_STEP)
  const normalizedDegrees = THERMOSTAT_MIN + step * THERMOSTAT_STEP
  if (step < 0 || step > STEP_COUNT || Math.abs(normalizedDegrees - degrees) > 0.001) {
    return undefined
  }

  return Number((DEVICE_MIN + step * DEVICE_STEP).toFixed(1))
}

export function decodeTuyaThermostatSetpoint(encoded: number): number | undefined {
  if (!Number.isFinite(encoded) || encoded < DEVICE_MIN - 0.02 || encoded > DEVICE_MAX + 0.02) {
    return undefined
  }

  const step = Math.round((encoded - DEVICE_MIN) / DEVICE_STEP)
  const normalizedEncoded = DEVICE_MIN + step * DEVICE_STEP
  if (step < 0 || step > STEP_COUNT || Math.abs(normalizedEncoded - encoded) > 0.02) {
    return undefined
  }

  return Number((THERMOSTAT_MIN + step * THERMOSTAT_STEP).toFixed(1))
}
