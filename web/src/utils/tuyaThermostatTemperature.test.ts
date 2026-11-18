import { describe, expect, it } from 'vitest'
import {
  decodeTuyaThermostatSetpoint,
  encodeTuyaThermostatSetpoint,
  TUYA_THERMOSTAT_SETPOINTS
} from './tuyaThermostatTemperature'

describe('Tuya thermostat setpoint conversion', () => {
  it('keeps all configured device values mapped to their thermostat degrees', () => {
    expect(TUYA_THERMOSTAT_SETPOINTS).toEqual([
      { encoded: 25.8, degrees: 1 },
      { encoded: 25.9, degrees: 1.5 },
      { encoded: 26, degrees: 2 },
      { encoded: 27, degrees: 7 },
      { encoded: 28, degrees: 12 },
      { encoded: 29, degrees: 17 },
      { encoded: 29.8, degrees: 21 },
      { encoded: 30, degrees: 22 }
    ])
  })

  it('encodes physical thermostat degrees into the value sent to Tuya', () => {
    expect(encodeTuyaThermostatSetpoint(1)).toBe(25.8)
    expect(encodeTuyaThermostatSetpoint(1.5)).toBe(25.9)
    expect(encodeTuyaThermostatSetpoint(2)).toBe(26)
    expect(encodeTuyaThermostatSetpoint(7)).toBe(27)
    expect(encodeTuyaThermostatSetpoint(12)).toBe(28)
    expect(encodeTuyaThermostatSetpoint(17)).toBe(29)
    expect(encodeTuyaThermostatSetpoint(21)).toBe(29.8)
    expect(encodeTuyaThermostatSetpoint(22)).toBe(30)
    expect(encodeTuyaThermostatSetpoint(20)).toBeUndefined()
  })

  it('decodes reported Tuya values back into thermostat degrees', () => {
    expect(decodeTuyaThermostatSetpoint(25.8)).toBe(1)
    expect(decodeTuyaThermostatSetpoint(25.9)).toBe(1.5)
    expect(decodeTuyaThermostatSetpoint(26)).toBe(2)
    expect(decodeTuyaThermostatSetpoint(27)).toBe(7)
    expect(decodeTuyaThermostatSetpoint(28)).toBe(12)
    expect(decodeTuyaThermostatSetpoint(29)).toBe(17)
    expect(decodeTuyaThermostatSetpoint(29.8)).toBe(21)
    expect(decodeTuyaThermostatSetpoint(30)).toBe(22)
    expect(decodeTuyaThermostatSetpoint(24)).toBeUndefined()
  })
})
