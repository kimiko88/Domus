import { describe, expect, it } from 'vitest'
import { parseAdvancedAutomationAction } from './automationActionValidation'

describe('parseAdvancedAutomationAction', () => {
  it('preserves choose branches, nested sequences and the default fallback', () => {
    const source = {
      type: 'choose',
      choose: [
        {
          conditions: [{ condition: 'state', entity_key: 'binary_sensor.motion', state: 'on' }],
          sequence: [{ type: 'service', service: 'turn_on', target_entity_key: 'light.hall' }]
        },
        {
          conditions: [{ condition: 'time', after: '22:00:00', before: '06:00:00' }],
          sequence: [{ type: 'delay', delay_ms: 5000 }]
        }
      ],
      default: [{ type: 'event', event_type: 'automation.no_match', event_payload: { source: 'flow-editor' } }]
    }

    expect(parseAdvancedAutomationAction(JSON.stringify(source), 'choose')).toEqual(source)
  })

  it('accepts if/else with separate action sequences', () => {
    const source = {
      type: 'if',
      if: [{ condition: 'numeric_state', entity_key: 'sensor.room', above: 25 }],
      then: [{ type: 'service', service: 'turn_on', target_entity_key: 'switch.fan' }],
      else: [{ type: 'service', service: 'turn_off', target_entity_key: 'switch.fan' }]
    }

    expect(parseAdvancedAutomationAction(JSON.stringify(source), 'if')).toEqual(source)
  })

  it('accepts bounded repeat and parallel actions', () => {
    const repeat = {
      type: 'repeat',
      repeat: {
        count: 3,
        sequence: [{ type: 'delay', delay_ms: 1000 }]
      }
    }
    const parallel = {
      type: 'parallel',
      parallel: [
        { type: 'service', service: 'turn_on', target_entity_key: 'light.kitchen' },
        { type: 'climate_profile', target_entity_key: 'climate.hall', profile: 'eco', duration_minutes: 30 }
      ]
    }

    expect(parseAdvancedAutomationAction(JSON.stringify(repeat), 'repeat')).toEqual(repeat)
    expect(parseAdvancedAutomationAction(JSON.stringify(parallel), 'parallel')).toEqual(parallel)
  })

  it('rejects invalid JSON and a mismatched node type', () => {
    expect(() => parseAdvancedAutomationAction('{invalid', 'choose')).toThrow('Configurazione JSON non valida')
    expect(() => parseAdvancedAutomationAction(JSON.stringify({ type: 'if' }), 'choose')).toThrow('Il campo type')
  })

  it('rejects conditions with no entity or threshold before saving', () => {
    const source = {
      type: 'if',
      if: [{ condition: 'numeric_state', entity_key: 'sensor.room' }],
      then: [{ type: 'service', service: 'turn_on', target_entity_key: 'switch.fan' }]
    }
    expect(() => parseAdvancedAutomationAction(JSON.stringify(source), 'if')).toThrow('above o below')
  })

  it('rejects runtime action types not implemented by the engine', () => {
    const source = {
      type: 'choose',
      choose: [{
        conditions: [{ condition: 'state', entity_key: 'binary_sensor.motion', state: 'on' }],
        sequence: [{ type: 'shell_command', command: 'unsafe' }]
      }]
    }
    expect(() => parseAdvancedAutomationAction(JSON.stringify(source), 'choose')).toThrow('tipo di azione non supportato')
  })

  it('rejects unbounded repeat loops and empty parallel groups', () => {
    const unboundedRepeat = {
      type: 'repeat',
      repeat: { while: [{ condition: 'state', entity_key: 'binary_sensor.motion', state: 'on' }], sequence: [] }
    }
    const emptyParallel = { type: 'parallel', parallel: [] }
    expect(() => parseAdvancedAutomationAction(JSON.stringify(unboundedRepeat), 'repeat')).toThrow('sequence non può essere vuota')
    expect(() => parseAdvancedAutomationAction(JSON.stringify(emptyParallel), 'parallel')).toThrow('almeno 1 elemento')
  })

  it('requires integer milliseconds, duration minutes and repeat counts', () => {
    const fractionalDelay = { type: 'delay', delay_ms: 1.5 }
    const fractionalDuration = {
      type: 'climate_profile',
      target_entity_key: 'climate.hall',
      profile: 'eco',
      duration_minutes: 1.5
    }
    const fractionalRepeat = {
      type: 'repeat',
      repeat: { count: 1.5, sequence: [{ type: 'delay', delay_ms: 1000 }] }
    }

    expect(() => parseAdvancedAutomationAction(JSON.stringify(fractionalDelay), 'delay')).toThrow('numero intero')
    expect(() => parseAdvancedAutomationAction(JSON.stringify(fractionalDuration), 'climate_profile')).toThrow('numero intero')
    expect(() => parseAdvancedAutomationAction(JSON.stringify(fractionalRepeat), 'repeat')).toThrow('numero intero')
  })
})
