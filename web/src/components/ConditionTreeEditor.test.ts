import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import ConditionTreeEditor from './ConditionTreeEditor.vue'
import type { AutomationCondition } from '../api/types'

describe('ConditionTreeEditor', () => {
  it('edits a nested leaf and propagates the updated tree to its root', async () => {
    const source: AutomationCondition = {
      condition: 'and',
      conditions: [
        { condition: 'state', entity_key: 'binary_sensor.motion', state: 'on' },
        { condition: 'time', after: '20:00:00' }
      ]
    }
    const wrapper = mount(ConditionTreeEditor, {
      props: { modelValue: source, entities: [] }
    })

    await wrapper.get('[data-test="condition-state"]').setValue('off')

    const emitted = wrapper.emitted('update:modelValue')
    expect(emitted).toBeTruthy()
    expect(emitted?.at(-1)?.[0]).toMatchObject({
      condition: 'and',
      conditions: [
        { condition: 'state', entity_key: 'binary_sensor.motion', state: 'off' },
        { condition: 'time', after: '20:00:00' }
      ]
    })
  })

  it('adds and removes children from AND/OR groups', async () => {
    const source: AutomationCondition = {
      condition: 'or',
      conditions: [{ condition: 'state', entity_key: 'binary_sensor.motion', state: 'on' }]
    }
    const wrapper = mount(ConditionTreeEditor, {
      props: { modelValue: source, entities: [] }
    })

    await wrapper.get('[data-test="add-condition"]').trigger('click')
    let latest = wrapper.emitted('update:modelValue')?.at(-1)?.[0] as AutomationCondition
    expect(latest.conditions).toHaveLength(2)
    expect(latest.conditions?.[1]).toMatchObject({ condition: 'state', state: 'on' })

    await wrapper.setProps({ modelValue: latest })
    await wrapper.get('[data-test="remove-condition-0"]').trigger('click')
    latest = wrapper.emitted('update:modelValue')?.at(-1)?.[0] as AutomationCondition
    expect(latest.conditions).toHaveLength(1)
  })

  it('wraps a leaf when changed to a logical operator and preserves NOT cardinality', async () => {
    const leaf: AutomationCondition = {
      condition: 'state',
      entity_key: 'binary_sensor.motion',
      state: 'on'
    }
    const wrapper = mount(ConditionTreeEditor, {
      props: { modelValue: leaf, entities: [] }
    })

    await wrapper.get('[data-test="condition-type"]').setValue('not')
    const wrapped = wrapper.emitted('update:modelValue')?.at(-1)?.[0] as AutomationCondition
    expect(wrapped).toEqual({ condition: 'not', conditions: [leaf] })

    await wrapper.setProps({ modelValue: wrapped })
    expect(wrapper.get('[data-test="add-condition"]').attributes('disabled')).toBeDefined()
    expect(wrapper.find('[data-test="remove-condition-0"]').exists()).toBe(false)
  })

  it('edits numeric thresholds, time windows and trigger identifiers', async () => {
    const numeric = mount(ConditionTreeEditor, {
      props: {
        modelValue: { condition: 'numeric_state', entity_key: 'sensor.room', above: 24 },
        entities: []
      }
    })
    await numeric.get('[data-test="condition-below"]').setValue('18.5')
    expect(numeric.emitted('update:modelValue')?.at(-1)?.[0]).toMatchObject({
      condition: 'numeric_state',
      entity_key: 'sensor.room',
      above: 24,
      below: 18.5
    })

    const time = mount(ConditionTreeEditor, {
      props: { modelValue: { condition: 'time', after: '20:00:00' }, entities: [] }
    })
    await time.get('[data-test="condition-before"]').setValue('06:00:00')
    expect(time.emitted('update:modelValue')?.at(-1)?.[0]).toMatchObject({
      condition: 'time',
      after: '20:00:00',
      before: '06:00:00'
    })

    const trigger = mount(ConditionTreeEditor, {
      props: { modelValue: { condition: 'trigger', trigger_id: 'motion_detected' }, entities: [] }
    })
    await trigger.get('[data-test="condition-trigger-id"]').setValue('door_open')
    expect(trigger.emitted('update:modelValue')?.at(-1)?.[0]).toMatchObject({
      condition: 'trigger',
      trigger_id: 'door_open'
    })
  })

  it('keeps unsupported condition structures read-only and intact', () => {
    const source = {
      condition: 'custom_future_condition',
      vendor_field: { keep: true }
    } as unknown as AutomationCondition
    const wrapper = mount(ConditionTreeEditor, {
      props: { modelValue: source, entities: [] }
    })

    expect(wrapper.text()).toContain('non è modificabile')
    expect(wrapper.text()).toContain('custom_future_condition')
    expect(wrapper.text()).toContain('vendor_field')
    expect(wrapper.emitted('update:modelValue')).toBeUndefined()
  })
  it('preserves all children when converting a multi-condition group to NOT', async () => {
    const conditions: AutomationCondition[] = [
      { condition: 'state', entity_key: 'binary_sensor.motion', state: 'on' },
      { condition: 'time', after: '20:00:00' }
    ]
    const wrapper = mount(ConditionTreeEditor, {
      props: { modelValue: { condition: 'and', conditions }, entities: [] }
    })

    await wrapper.get('[data-test="condition-type"]').setValue('not')
    expect(wrapper.emitted('update:modelValue')?.at(-1)?.[0]).toEqual({
      condition: 'not',
      conditions: [{ condition: 'and', conditions }]
    })
  })

})
