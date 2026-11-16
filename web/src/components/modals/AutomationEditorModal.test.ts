import { describe, it, expect, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import AutomationEditorModal from './AutomationEditorModal.vue'
import { apiClient } from '../../api/client'

describe('AutomationEditorModal.vue', () => {
  it('renders modal with initial values for editing', () => {
    const existing = {
      id: 'auto_1',
      name: 'Night Mode',
      description: 'Turn off all',
      enabled: true,
      execution_mode: 'single' as const,
      cooldown_seconds: 5,
      triggers: [{ platform: 'time', at: '23:00:00' }],
      conditions: [],
      actions: [{ type: 'service', service: 'turn_off', target_entity_key: 'light.all' }],
      created_at: '',
      updated_at: ''
    }

    const wrapper = mount(AutomationEditorModal, {
      props: {
        isOpen: true,
        automation: existing
      }
    })

    expect((wrapper.find('input[name="auto-name"]').element as HTMLInputElement).value).toBe('Night Mode')
    expect((wrapper.find('textarea[name="dsl-json"]').element as HTMLTextAreaElement).value).toContain('23:00:00')
  })

  it('detects and displays JSON syntax errors', async () => {
    const wrapper = mount(AutomationEditorModal, {
      props: {
        isOpen: true
      }
    })

    await wrapper.find('input[name="auto-name"]').setValue('Test')
    await wrapper.find('textarea[name="dsl-json"]').setValue('{ invalid json here')
    await wrapper.find('form').trigger('submit.prevent')

    expect(wrapper.text()).toContain('Sintassi JSON non valida')
    expect(wrapper.emitted('save')).toBeUndefined()
  })

  it('validates triggers and actions invariants', async () => {
    const wrapper = mount(AutomationEditorModal, {
      props: {
        isOpen: true
      }
    })

    await wrapper.find('input[name="auto-name"]').setValue('Test')
    // Valid JSON but missing triggers or actions
    await wrapper.find('textarea[name="dsl-json"]').setValue(JSON.stringify({
      triggers: [],
      conditions: [],
      actions: []
    }))
    await wrapper.find('form').trigger('submit.prevent')

    expect(wrapper.text()).toContain('Almeno un trigger è richiesto')
    expect(wrapper.emitted('save')).toBeUndefined()
  })

  it('emits save with parsed valid data', async () => {
    const wrapper = mount(AutomationEditorModal, {
      props: {
        isOpen: true
      }
    })

    await wrapper.find('input[name="auto-name"]').setValue('Morning Sunlight')
    await wrapper.find('textarea[name="dsl-json"]').setValue(JSON.stringify({
      triggers: [{ platform: 'time', at: '07:00:00' }],
      conditions: [],
      actions: [{ type: 'service', service: 'turn_on', target_entity_key: 'light.bedroom' }]
    }))

    await wrapper.find('form').trigger('submit.prevent')

    expect(wrapper.emitted('save')).toHaveLength(1)
    const saved = wrapper.emitted('save')?.[0]?.[0] as any
    expect(saved.name).toBe('Morning Sunlight')
    expect(saved.triggers).toHaveLength(1)
    expect(saved.actions).toHaveLength(1)
  })
})

it('generates a daily thermostat rule from a temperature sensor', async () => {
  const entities = [
    {
      id: 'sensor-uuid',
      entity_key: 'sensor.room_temperature',
      domain: 'sensor',
      name: 'Temperatura soggiorno',
      state: '18.5',
      attributes: { device_class: 'temperature', unit_of_measurement: '°C' },
      last_changed: '',
      last_updated: ''
    },
    {
      id: 'climate-uuid',
      entity_key: 'climate.living_room',
      domain: 'climate',
      name: 'Termostato soggiorno',
      state: 'auto',
      attributes: { current_temperature: 18.5 },
      last_changed: '',
      last_updated: ''
    }
  ]
  const getSpy = vi.spyOn(apiClient, 'get').mockResolvedValue(entities as any)
  const wrapper = mount(AutomationEditorModal, { props: { isOpen: true } })

  await wrapper.find('[data-testid="climate-program-toggle"]').trigger('click')
  await flushPromises()
  expect(wrapper.find('select[name="climate-sensor"]').exists()).toBe(true)

  await wrapper.find('input[name="climate-time"]').setValue('06:45')
  await wrapper.find('#climate-threshold').setValue('18')
  await wrapper.find('#climate-cold-setpoint').setValue('21.5')
  await wrapper.find('#climate-warm-setpoint').setValue('17.5')
  await wrapper.find('[data-testid="generate-climate-program"]').trigger('click')

  const dsl = JSON.parse((wrapper.find('textarea[name="dsl-json"]').element as HTMLTextAreaElement).value)
  expect(dsl.triggers).toEqual([{ id: 'climate_slot_1', platform: 'time', at: '06:45:00' }])
  expect(dsl.conditions[0]).toMatchObject({ condition: 'numeric_state', entity_key: 'sensor.room_temperature' })
  expect(dsl.actions[0].choose[0].conditions).toEqual([
    { condition: 'trigger', trigger_id: 'climate_slot_1' }
  ])
  const temperatureChoice = dsl.actions[0].choose[0].sequence[0]
  expect(temperatureChoice.choose[0].conditions[0]).toMatchObject({ below: 18 })
  expect(temperatureChoice.choose[0].sequence[0]).toMatchObject({
    type: 'service',
    service: 'set_temperature',
    target_entity_key: 'climate.living_room',
    params: { temperature: 21.5 }
  })
  expect(temperatureChoice.default[0].params.temperature).toBe(17.5)

  getSpy.mockRestore()
  wrapper.unmount()
})


it('supports multiple daily thermostat schedule slots using the same live temperature sensor', async () => {
  const entities = [
    {
      id: 'sensor-uuid',
      entity_key: 'sensor.room_temperature',
      domain: 'sensor',
      name: 'Temperatura soggiorno',
      state: '18.5',
      attributes: { device_class: 'temperature', unit_of_measurement: '°C' },
      last_changed: '',
      last_updated: ''
    },
    {
      id: 'climate-uuid',
      entity_key: 'climate.living_room',
      domain: 'climate',
      name: 'Termostato soggiorno',
      state: 'auto',
      attributes: { current_temperature: 18.5 },
      last_changed: '',
      last_updated: ''
    }
  ]
  const getSpy = vi.spyOn(apiClient, 'get').mockResolvedValue(entities as any)
  const wrapper = mount(AutomationEditorModal, { props: { isOpen: true } })

  await wrapper.find('[data-testid="climate-program-toggle"]').trigger('click')
  await flushPromises()
  await wrapper.find('[data-testid="add-climate-slot"]').trigger('click')

  const timeInputs = wrapper.findAll('input[name="climate-time"]')
  expect(timeInputs).toHaveLength(2)
  await timeInputs[1].setValue('18:30')
  await wrapper.find('#climate-threshold-slot-2').setValue('17')
  await wrapper.find('#climate-cold-setpoint-slot-2').setValue('20')
  await wrapper.find('#climate-warm-setpoint-slot-2').setValue('16')

  await wrapper.find('[data-testid="generate-climate-program"]').trigger('click')

  const dsl = JSON.parse((wrapper.find('textarea[name="dsl-json"]').element as HTMLTextAreaElement).value)
  expect(dsl.triggers).toEqual([
    { id: 'climate_slot_1', platform: 'time', at: '07:00:00' },
    { id: 'climate_slot_2', platform: 'time', at: '18:30:00' }
  ])
  expect(dsl.actions[0].choose).toHaveLength(2)
  expect(dsl.actions[0].choose[1].conditions).toEqual([
    { condition: 'trigger', trigger_id: 'climate_slot_2' }
  ])
  const eveningChoice = dsl.actions[0].choose[1].sequence[0]
  expect(eveningChoice.choose[0].conditions[0]).toMatchObject({ below: 17 })
  expect(eveningChoice.choose[0].sequence[0].params.temperature).toBe(20)
  expect(eveningChoice.default[0].params.temperature).toBe(16)

  getSpy.mockRestore()
  wrapper.unmount()
})

it('rejects duplicate times in a thermostat schedule', async () => {
  const entities = [
    {
      id: 'sensor-uuid',
      entity_key: 'sensor.room_temperature',
      domain: 'sensor',
      name: 'Temperatura soggiorno',
      state: '18.5',
      attributes: { device_class: 'temperature', unit_of_measurement: '°C' },
      last_changed: '',
      last_updated: ''
    },
    {
      id: 'climate-uuid',
      entity_key: 'climate.living_room',
      domain: 'climate',
      name: 'Termostato soggiorno',
      state: 'auto',
      attributes: { current_temperature: 18.5 },
      last_changed: '',
      last_updated: ''
    }
  ]
  const getSpy = vi.spyOn(apiClient, 'get').mockResolvedValue(entities as any)
  const wrapper = mount(AutomationEditorModal, { props: { isOpen: true } })

  await wrapper.find('[data-testid="climate-program-toggle"]').trigger('click')
  await flushPromises()
  await wrapper.find('[data-testid="add-climate-slot"]').trigger('click')
  await wrapper.findAll('input[name="climate-time"]')[1].setValue('07:00')
  await wrapper.find('[data-testid="generate-climate-program"]').trigger('click')

  expect(wrapper.text()).toContain('è duplicato')
  getSpy.mockRestore()
  wrapper.unmount()
})
