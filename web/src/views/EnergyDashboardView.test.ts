import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import EnergyDashboardView from './EnergyDashboardView.vue'
import { apiClient } from '../api/client'

describe('EnergyDashboardView.vue', () => {
  beforeEach(() => {
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

    expect(wrapper.text()).toContain('F1 (Picco: Lun-Ven 08-19)')
    expect(wrapper.text()).toContain('F2 (Intermedia')
    expect(wrapper.text()).toContain('F3 (Fuori Picco')
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

    expect(wrapper.text()).toContain('Ricarica Solare Intelligente & Wallbox OCPP')
    expect(wrapper.text()).toContain('Charging')
    expect(wrapper.text()).toContain('3680')
    expect(wrapper.text()).toContain('72')
    expect(wrapper.text()).toContain('Solo Solare (100% FV)')
    expect(wrapper.text()).toContain('Solare Ottimizzato (Ibrido)')
    expect(wrapper.text()).toContain('Fast Charge (Massima Potenza)')
    expect(wrapper.text()).toContain('Notturna Economica')
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
})

