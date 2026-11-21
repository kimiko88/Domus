import { describe, expect, it } from 'vitest'
import {
  decodeTuyaThermostatSetpoint,
  encodeTuyaThermostatSetpoint,
  TUYA_THERMOSTAT_SETPOINTS
} from './tuyaThermostatTemperature'

describe('Tuya thermostat setpoint conversion', () => {
  it('generates every device step from 25.8 to 30 in half-degree increments', () => {
    expect(TUYA_THERMOSTAT_SETPOINTS).toHaveLength(43)
    expect(TUYA_THERMOSTAT_SETPOINTS[0]).toEqual({ encoded: 25.8, degrees: 1 })
    expect(TUYA_THERMOSTAT_SETPOINTS[1]).toEqual({ encoded: 25.9, degrees: 1.5 })
    expect(TUYA_THERMOSTAT_SETPOINTS[2]).toEqual({ encoded: 26, degrees: 2 })
    expect(TUYA_THERMOSTAT_SETPOINTS[3]).toEqual({ encoded: 26.1, degrees: 2.5 })
    expect(TUYA_THERMOSTAT_SETPOINTS[12]).toEqual({ encoded: 27, degrees: 7 })
    expect(TUYA_THERMOSTAT_SETPOINTS[22]).toEqual({ encoded: 28, degrees: 12 })
    expect(TUYA_THERMOSTAT_SETPOINTS[32]).toEqual({ encoded: 29, degrees: 17 })
    expect(TUYA_THERMOSTAT_SETPOINTS[40]).toEqual({ encoded: 29.8, degrees: 21 })
    expect(TUYA_THERMOSTAT_SETPOINTS[42]).toEqual({ encoded: 30, degrees: 22 })
  })

  it('encodes each physical half-degree step into the device scale', () => {
    for (const item of TUYA_THERMOSTAT_SETPOINTS) {
      expect(encodeTuyaThermostatSetpoint(item.degrees)).toBe(item.encoded)
    }
    expect(encodeTuyaThermostatSetpoint(0.5)).toBeUndefined()
    expect(encodeTuyaThermostatSetpoint(1.2)).toBeUndefined()
    expect(encodeTuyaThermostatSetpoint(22.5)).toBeUndefined()
    expect(encodeTuyaThermostatSetpoint(Number.NaN)).toBeUndefined()
  })

  it('decodes device values including floating point serialization noise', () => {
    for (const item of TUYA_THERMOSTAT_SETPOINTS) {
      expect(decodeTuyaThermostatSetpoint(item.encoded)).toBe(item.degrees)
    }
    expect(decodeTuyaThermostatSetpoint(25.90000001)).toBe(1.5)
    expect(decodeTuyaThermostatSetpoint(24)).toBeUndefined()
    expect(decodeTuyaThermostatSetpoint(30.1)).toBeUndefined()
    expect(decodeTuyaThermostatSetpoint(Number.NaN)).toBeUndefined()
  })
})
