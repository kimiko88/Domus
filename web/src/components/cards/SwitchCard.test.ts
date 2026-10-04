import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import SwitchCard from './SwitchCard.vue'
import { useEntitiesStore } from '../../stores/entities'
import type { Entity } from '../../api/types'

describe('SwitchCard.vue', () => {
  let entitiesStore: ReturnType<typeof useEntitiesStore>

  beforeEach(() => {
    setActivePinia(createPinia())
    entitiesStore = useEntitiesStore()
  })

  it('renders switch name and off state', () => {
    const entity: Entity = {
      id: 'switch.coffee_maker',
      domain: 'switch',
      name: 'Coffee Maker',
      state: 'off',
      attributes: {},
      last_changed: '2026-10-04T00:00:00Z',
      last_updated: '2026-10-04T00:00:00Z'
    }

    const wrapper = mount(SwitchCard, {
      props: { entity }
    })

    expect(wrapper.text()).toContain('Coffee Maker')
    expect(wrapper.text()).toContain('Off')
  })

  it('renders power usage if available in attributes', () => {
    const entity: Entity = {
      id: 'switch.washing_machine',
      domain: 'switch',
      name: 'Washing Machine',
      state: 'on',
      attributes: { power: 1250 },
      last_changed: '2026-10-04T00:00:00Z',
      last_updated: '2026-10-04T00:00:00Z'
    }

    const wrapper = mount(SwitchCard, {
      props: { entity }
    })

    expect(wrapper.text()).toContain('On')
    expect(wrapper.text()).toContain('1250 W')
  })

  it('toggles switch state on button click', async () => {
    const entity: Entity = {
      id: 'switch.fan',
      domain: 'switch',
      name: 'Air Fan',
      state: 'off',
      attributes: {},
      last_changed: '2026-10-04T00:00:00Z',
      last_updated: '2026-10-04T00:00:00Z'
    }

    const callServiceSpy = vi.spyOn(entitiesStore, 'callService').mockResolvedValue(undefined)

    const wrapper = mount(SwitchCard, {
      props: { entity }
    })

    const toggleBtn = wrapper.find('.toggle-btn')
    await toggleBtn.trigger('click')

    expect(callServiceSpy).toHaveBeenCalledWith('switch', 'turn_on', 'switch.fan', undefined)
  })
})
