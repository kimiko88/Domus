import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import DashboardView from './DashboardView.vue'
import { useEntitiesStore } from '../stores/entities'
import { useAreasStore } from '../stores/areas'
import { i18n } from '../locales'
import type { Entity } from '../api/types'

describe('DashboardView.vue', () => {
  let entitiesStore: ReturnType<typeof useEntitiesStore>
  let areasStore: ReturnType<typeof useAreasStore>

  beforeEach(() => {
    setActivePinia(createPinia())
    entitiesStore = useEntitiesStore()
    areasStore = useAreasStore()
  })

  it('fetches entities and areas on mount', async () => {
    const fetchEntitiesSpy = vi.spyOn(entitiesStore, 'fetchEntities').mockResolvedValue([])
    const fetchAreasSpy = vi.spyOn(areasStore, 'fetchAreas').mockResolvedValue([])

    mount(DashboardView, {
      global: { plugins: [i18n] }
    })

    expect(fetchEntitiesSpy).toHaveBeenCalled()
    expect(fetchAreasSpy).toHaveBeenCalled()
  })

  it('renders summary badges and entity cards', async () => {
    const mockEntities: Entity[] = [
      {
        id: 'light.living_room',
        domain: 'light',
        name: 'Living Room Light',
        state: 'on',
        attributes: { brightness: 255 },
        last_changed: '2026-10-04T00:00:00Z',
        last_updated: '2026-10-04T00:00:00Z'
      },
      {
        id: 'switch.ac',
        domain: 'switch',
        name: 'Air Conditioner',
        state: 'off',
        attributes: {},
        last_changed: '2026-10-04T00:00:00Z',
        last_updated: '2026-10-04T00:00:00Z'
      }
    ]

    vi.spyOn(entitiesStore, 'fetchEntities').mockImplementation(async () => {
      mockEntities.forEach((e) => {
        entitiesStore.entities[e.id] = e
      })
      return mockEntities
    })

    const wrapper = mount(DashboardView, {
      global: { plugins: [i18n] }
    })
    await wrapper.vm.$nextTick()

    expect(wrapper.text()).toContain('Living Room Light')
    expect(wrapper.text()).toContain('Air Conditioner')
    expect(wrapper.text()).toContain('1 Attive') // 1 active light
  })

  it('filters entities when domain filter pill is clicked', async () => {
    const mockEntities: Entity[] = [
      {
        id: 'light.study',
        domain: 'light',
        name: 'Study Lamp',
        state: 'off',
        attributes: {},
        last_changed: '2026-10-04T00:00:00Z',
        last_updated: '2026-10-04T00:00:00Z'
      },
      {
        id: 'switch.tv',
        domain: 'switch',
        name: 'TV Plug',
        state: 'on',
        attributes: {},
        last_changed: '2026-10-04T00:00:00Z',
        last_updated: '2026-10-04T00:00:00Z'
      }
    ]

    mockEntities.forEach((e) => {
      entitiesStore.entities[e.id] = e
    })

    const wrapper = mount(DashboardView, {
      global: { plugins: [i18n] }
    })
    await wrapper.vm.$nextTick()

    // Find and click filter for switches
    const switchPill = wrapper.findAll('.filter-pill').find((p) => p.text().includes('Interruttori'))
    expect(switchPill).toBeDefined()
    await switchPill!.trigger('click')

    expect(wrapper.text()).toContain('TV Plug')
    expect(wrapper.text()).not.toContain('Study Lamp')
  })
})
