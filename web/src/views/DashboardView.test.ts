import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import DashboardView from './DashboardView.vue'
import { useEntitiesStore } from '../stores/entities'
import { useAreasStore } from '../stores/areas'
import { apiClient } from '../api/client'
import { i18n } from '../locales'
import type { Entity } from '../api/types'

describe('DashboardView.vue', () => {
  let entitiesStore: ReturnType<typeof useEntitiesStore>
  let areasStore: ReturnType<typeof useAreasStore>

  beforeEach(() => {
    setActivePinia(createPinia())
    entitiesStore = useEntitiesStore()
    areasStore = useAreasStore()
    vi.spyOn(entitiesStore, 'fetchEntities').mockResolvedValue([])
    vi.spyOn(areasStore, 'fetchAreas').mockResolvedValue([])
    vi.spyOn(apiClient, 'get').mockResolvedValue([])
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

  it('renders marketplace summary and showcase on dashboard', async () => {
    vi.spyOn(apiClient, 'get').mockImplementation(async (url: string) => {
      if (url.includes('/api/v1/store/packages')) {
        return [
          {
            id: 'pkg_bess_test',
            package_id: 'domus.bess.arbitrage',
            name: 'BESS Arbitrage Optimizer',
            type: 'wasm_plugin',
            version: '1.0.0',
            author: 'Domus Team',
            description: 'Batteria intelligente e picchi di carico',
            installed: true
          },
          {
            id: 'pkg_solar_test',
            package_id: 'domus.solar.forecast',
            name: 'Solar PV Forecaster',
            type: 'blueprint',
            version: '1.2.0',
            author: 'EcoEnergy',
            description: 'Previsioni solari basate su meteo locale',
            installed: false
          }
        ] as any
      }
      return [] as any
    })

    const wrapper = mount(DashboardView, {
      global: { plugins: [i18n] }
    })
    await wrapper.vm.$nextTick()
    // Give promises time to resolve
    await new Promise((r) => setTimeout(r, 10))

    expect(wrapper.text()).toContain('Marketplace & Estensioni')
    expect(wrapper.text()).toContain('BESS Arbitrage Optimizer')
    expect(wrapper.text()).toContain('Solar PV Forecaster')
    expect(wrapper.text()).toContain('Estensioni Installate')
  })

  it('navigates to marketplace tab and supports 1-click install', async () => {
    vi.spyOn(apiClient, 'get').mockImplementation(async (url: string) => {
      if (url.includes('/api/v1/store/packages')) {
        return [
          {
            id: 'pkg_wasi_modbus',
            package_id: 'domus.wasi.modbus',
            name: 'Modbus TCP Reader',
            type: 'wasm_plugin',
            version: '1.1.0',
            author: 'IndustrialIoT',
            description: 'Lettura contatori Modbus',
            installed: false
          }
        ] as any
      }
      return [] as any
    })
    const postSpy = vi.spyOn(apiClient, 'post').mockResolvedValue({ success: true } as any)

    const wrapper = mount(DashboardView, {
      global: { plugins: [i18n] }
    })
    await wrapper.vm.$nextTick()
    await new Promise((r) => setTimeout(r, 10))

    // Click Marketplace filter pill
    const mpPill = wrapper.findAll('.filter-pill').find((p) => p.text().includes('Marketplace'))
    expect(mpPill).toBeDefined()
    await mpPill!.trigger('click')

    expect(wrapper.text()).toContain('Modbus TCP Reader')
    expect(wrapper.find('#dash-marketplace-search').exists()).toBe(true)

    // Click 1-click install button
    const installBtn = wrapper.find('.btn-install-toggle')
    expect(installBtn.exists()).toBe(true)
    await installBtn.trigger('click')

    expect(postSpy).toHaveBeenCalledWith('/api/v1/store/packages/pkg_wasi_modbus/install', {})
  })
})

