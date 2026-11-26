import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import EnergyDashboardView from './EnergyDashboardView.vue'
import { apiClient } from '../api/client'
import { i18n } from '../locales'

describe('EnergyDashboardView.vue', () => {
  beforeEach(() => {
    vi.restoreAllMocks()
    setActivePinia(createPinia())
    vi.spyOn(apiClient, 'get').mockImplementation(async (url: string) => {
      if (url.includes('/api/v1/energy/flow')) {
        return {
          solar_watts: 5000,
          grid_watts: -1200,
          battery_watts: -2000,
          battery_soc: 85,
          house_consumption_watts: 1800,
          solar_to_house_watts: 1800,
          solar_to_battery_watts: 2000,
          solar_to_grid_watts: 1200,
          battery_to_house_watts: 0,
          grid_to_house_watts: 0,
          grid_to_battery_watts: 0
        } as any
      }
      if (url.includes('/api/v1/energy/tariffs')) {
        return {
          current_slot: 'F1',
          current_price: 0.28,
          currency: 'EUR',
          tariffs: {
            f1_price: 0.28,
            f2_price: 0.24,
            f3_price: 0.19,
            gas_smc_price: 0.95,
            water_m3_price: 1.80
          }
        } as any
      }
      if (url.includes('/api/v1/energy/wallbox')) {
        return {
          charge_point_id: 'domus_wallbox_1',
          telemetry: {
            charge_point_id: 'domus_wallbox_1',
            status: 'Charging',
            active_power_w: 3680,
            offered_amps: 16,
            delivered_amps: 16,
            vehicle_soc_pct: 72,
            energy_wh: 12500,
            last_seen_at: new Date().toISOString()
          },
          modulation: {
            charge_point_id: 'domus_wallbox_1',
            mode: 'solar_optimized',
            target_amps: 16,
            expected_power_w: 3680,
            solar_surplus_w: 2500,
            home_grid_power_w: 800,
            reason: 'Solar optimized charging with 6A base + 10A solar surplus',
            timestamp: new Date().toISOString()
          }
        } as any
      }
      return null as any
    })
  })

  it('renders energy dashboard header and summary cards', async () => {
    const wrapper = mount(EnergyDashboardView)
    await flushPromises()

    expect(wrapper.text()).toContain('Monitoraggio Energetico & Flusso Solare')
    expect(wrapper.text()).toContain('5000 W')
    expect(wrapper.text()).toContain('1800 W')
  })

  it('renders dynamic Italian tariff cards (F1, F2, F3)', async () => {
    const wrapper = mount(EnergyDashboardView)
    await flushPromises()

    expect(wrapper.text()).toContain(i18n.global.t('energy.tariffF1'))
    expect(wrapper.text()).toContain(i18n.global.t('energy.tariffF2'))
    expect(wrapper.text()).toContain(i18n.global.t('energy.tariffF3'))
    expect(wrapper.text()).toContain('0.28 €/kWh')
  })

  it('renders water and gas utility pulse meter cards', async () => {
    const wrapper = mount(EnergyDashboardView)
    await flushPromises()

    expect(wrapper.text()).toContain('Contatore Acqua')
    expect(wrapper.text()).toContain('Contatore Gas')
    expect(wrapper.text()).toContain('m³')
    expect(wrapper.text()).toContain('Smc')
  })

  it('renders OCPP Wallbox telemetry and mode selector', async () => {
    const wrapper = mount(EnergyDashboardView)
    await flushPromises()

    expect(wrapper.text()).toContain(i18n.global.t('energy.wallboxTitle'))
    expect(wrapper.text()).toContain('Charging')
    expect(wrapper.text()).toContain('3680')
    expect(wrapper.text()).toContain('72')
    expect(wrapper.text()).toContain(i18n.global.t('energy.solarOnly'))
    expect(wrapper.text()).toContain(i18n.global.t('energy.solarOptimized'))
    expect(wrapper.text()).toContain(i18n.global.t('energy.fastCharge'))
    expect(wrapper.text()).toContain(i18n.global.t('energy.scheduledNight'))
    expect(wrapper.text()).toContain('Solar optimized charging with 6A base + 10A solar surplus')
  })

  it('triggers wallbox mode change and modulation recalculate', async () => {
    const postSpy = vi.spyOn(apiClient, 'post').mockResolvedValue({ status: 'updated' } as any)
    const wrapper = mount(EnergyDashboardView)
    await flushPromises()

    // Find and click the solar_only mode button
    const modeButtons = wrapper.findAll('.wb-mode-card')
    expect(modeButtons.length).toBe(4)
    await modeButtons[0].trigger('click')
    await flushPromises()

    expect(postSpy).toHaveBeenCalledWith('/api/v1/energy/wallbox/mode', { mode: 'solar_only' })

    // Find and click the recalculate modulation button
    const recalcBtn = wrapper.find('.wb-recalc-btn')
    expect(recalcBtn.exists()).toBe(true)
    await recalcBtn.trigger('click')
    await flushPromises()

    expect(postSpy).toHaveBeenCalledWith('/api/v1/energy/wallbox/modulate', expect.objectContaining({
      charge_point_id: 'domus_wallbox_1'
    }))
  })

  const unconfiguredLoad = {
    entity_key: 'switch.water_heater',
    name: 'Water heater',
    required_hours: 2,
    max_price_kwh: -1,
    active_now: false,
    auto_schedule: true,
    scheduled_hours: [2, 3],
    estimated_cost_eur: 0,
    savings_percent: 0
  }

  function mockDynamicTariffs(loads: Array<Record<string, any>>) {
    let calls = 0
    vi.spyOn(apiClient, 'get').mockImplementation(async (url: string) => {
      if (url.endsWith('/api/v1/energy/dynamic-tariffs')) {
        calls += 1
        return {
          schedule: {
            date: '2026-11-26',
            market: 'PUN',
            source: 'synthetic_estimate',
            is_estimate: true,
            hourly_prices: [{
              hour: 2,
              price_per_kwh: 0.1,
              slot: 'F3',
              is_cheapest: true,
              is_expensive: false,
              is_negative: false
            }],
            average_price: 0.3,
            min_price: 0.1,
            max_price: 0.5,
            cheapest_hours: [2]
          },
          flexible_loads: loads.map(load => ({
            ...load,
            scheduled_hours: [...(load.scheduled_hours || [])]
          }))
        } as any
      }
      return null as any
    })
    return () => calls
  }

  it('shows the estimate warning and avoids inventing costs before power is configured', async () => {
    mockDynamicTariffs([unconfiguredLoad])
    const wrapper = mount(EnergyDashboardView)
    await flushPromises()

    expect(wrapper.find('.tariff-estimate-warning').exists()).toBe(true)
    expect(wrapper.find('.tariff-estimate-warning').text()).toBe(i18n.global.t('energy.tariffEstimateWarning'))
    expect(wrapper.text()).toContain(i18n.global.t('energy.noPriceLimit'))
    expect(wrapper.text()).toContain(i18n.global.t('energy.loadPowerRequired'))
    expect(wrapper.find('.savings-tag').exists()).toBe(false)
    expect((wrapper.find('input.load-power-input').element as HTMLInputElement).value).toBe('')
  })

  it('persists configured load power and renders the returned cost and savings', async () => {
    mockDynamicTariffs([unconfiguredLoad])
    const updatedLoad = {
      ...unconfiguredLoad,
      power_kw: 1.5,
      estimated_cost_eur: 0.45,
      savings_percent: 20
    }
    const putSpy = vi.spyOn(apiClient, 'put').mockResolvedValue({
      status: 'updated',
      flexible_loads: [updatedLoad]
    } as any)

    const wrapper = mount(EnergyDashboardView)
    await flushPromises()
    const powerInput = wrapper.find('input.load-power-input')
    await powerInput.setValue('1.5')
    await powerInput.trigger('change')
    await flushPromises()

    expect(putSpy.mock.calls.filter(([url]) => url === '/api/v1/energy/flexible-loads')).toHaveLength(1)
    expect(putSpy).toHaveBeenCalledWith(
      '/api/v1/energy/flexible-loads',
      expect.objectContaining({
        entity_key: 'switch.water_heater',
        power_kw: 1.5
      })
    )
    expect(wrapper.find('.cost-stat').text()).toContain('0.45 €')
    expect(wrapper.find('.savings-tag').text()).toContain('-20%')
  })

  it('restores the server power value when saving a load power fails', async () => {
    const getCalls = mockDynamicTariffs([unconfiguredLoad])
    const putSpy = vi.spyOn(apiClient, 'put').mockRejectedValue(new Error('temporary network error'))
    const wrapper = mount(EnergyDashboardView)
    await flushPromises()

    const powerInput = wrapper.find('input.load-power-input')
    await powerInput.setValue('2.4')
    await powerInput.trigger('change')
    await flushPromises()

    expect(putSpy.mock.calls.filter(([url]) => url === '/api/v1/energy/flexible-loads')).toHaveLength(1)
    expect(getCalls()).toBe(2)
    expect((wrapper.find('input.load-power-input').element as HTMLInputElement).value).toBe('')
    expect(wrapper.text()).toContain(i18n.global.t('energy.loadPowerRequired'))
  })

})

