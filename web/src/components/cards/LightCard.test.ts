import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import LightCard from './LightCard.vue'
import { useEntitiesStore } from '../../stores/entities'
import type { Entity } from '../../api/types'

describe('LightCard.vue', () => {
  let entitiesStore: ReturnType<typeof useEntitiesStore>

  beforeEach(() => {
    setActivePinia(createPinia())
    entitiesStore = useEntitiesStore()
  })

  it('renders light name, state off, and toggle button', () => {
    const entity: Entity = {
      id: 'light.desk',
      domain: 'light',
      name: 'Desk Lamp',
      state: 'off',
      attributes: {},
      last_changed: '2026-10-04T00:00:00Z',
      last_updated: '2026-10-04T00:00:00Z'
    }

    const wrapper = mount(LightCard, {
      props: { entity }
    })

    expect(wrapper.text()).toContain('Desk Lamp')
    expect(wrapper.text()).toContain('Off')
    expect(wrapper.classes()).not.toContain('is-on')
  })

  it('renders state on and brightness percentage', () => {
    const entity: Entity = {
      id: 'light.ceiling',
      domain: 'light',
      name: 'Ceiling Light',
      state: 'on',
      attributes: { brightness: 128 },
      last_changed: '2026-10-04T00:00:00Z',
      last_updated: '2026-10-04T00:00:00Z'
    }

    const wrapper = mount(LightCard, {
      props: { entity }
    })

    expect(wrapper.text()).toContain('On')
    // 128 / 255 ≈ 50%
    expect(wrapper.text()).toContain('50%')
    expect(wrapper.find('.light-card').classes()).toContain('is-on')
  })

  it('calls callService when toggle button is clicked', async () => {
    const entity: Entity = {
      id: 'light.hallway',
      domain: 'light',
      name: 'Hallway Light',
      state: 'off',
      attributes: {},
      last_changed: '2026-10-04T00:00:00Z',
      last_updated: '2026-10-04T00:00:00Z'
    }

    const callServiceSpy = vi.spyOn(entitiesStore, 'callService').mockResolvedValue(undefined)

    const wrapper = mount(LightCard, {
      props: { entity }
    })

    const toggleBtn = wrapper.find('.toggle-btn')
    await toggleBtn.trigger('click')

    expect(callServiceSpy).toHaveBeenCalledWith('light', 'turn_on', 'light.hallway', undefined)
  })

  it('adjusts brightness when slider changes', async () => {
    const entity: Entity = {
      id: 'light.bedroom',
      domain: 'light',
      name: 'Bedroom Light',
      state: 'on',
      attributes: { brightness: 100 },
      last_changed: '2026-10-04T00:00:00Z',
      last_updated: '2026-10-04T00:00:00Z'
    }

    const callServiceSpy = vi.spyOn(entitiesStore, 'callService').mockResolvedValue(undefined)

    const wrapper = mount(LightCard, {
      props: { entity }
    })

    const slider = wrapper.find('input[type="range"]')
    expect(slider.exists()).toBe(true)

    await slider.setValue(80) // 80%
    await slider.trigger('change')

    // 80% of 255 = 204
    expect(callServiceSpy).toHaveBeenCalledWith('light', 'turn_on', 'light.bedroom', {
      brightness: 204
    })
  })
})
