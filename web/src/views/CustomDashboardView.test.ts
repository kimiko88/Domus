import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import CustomDashboardView from './CustomDashboardView.vue'
import { apiClient } from '../api/client'

describe('CustomDashboardView.vue', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.spyOn(apiClient, 'get').mockImplementation(async (url: string) => {
      if (url.includes('/api/v1/dashboards')) {
        return [
          {
            id: 'tablet_living_room',
            title: 'Tablet Salotto',
            icon: 'tablet',
            is_default: true,
            kiosk_config_json: JSON.stringify({
              enabled: false,
              screensaverTimeoutSec: 60,
              wakeOnPresence: true,
              showClock: true,
              showWeather: true
            }),
            layout_json: '[]'
          }
        ] as any
      }
      return null as any
    })
  })

  it('renders dashboard title and profile selector', async () => {
    const wrapper = mount(CustomDashboardView)
    await wrapper.vm.$nextTick()

    expect(wrapper.text()).toContain('Dashboard Personalizzata')
    expect(wrapper.find('.profile-select').exists()).toBe(true)
    expect(wrapper.find('.kiosk-toggle-btn').exists()).toBe(true)
  })

  it('renders circular dial thermostat and adjusts temperature', async () => {
    const wrapper = mount(CustomDashboardView)
    await wrapper.vm.$nextTick()

    const thermostatCard = wrapper.find('.thermostat-card')
    expect(thermostatCard.exists()).toBe(true)
    expect(thermostatCard.text()).toContain('Clima & Termostato Salotto')

    // Find + 0.5 button and click
    const plusBtn = wrapper.find('.temp-btn.primary')
    expect(plusBtn.exists()).toBe(true)
    await plusBtn.trigger('click')

    expect(wrapper.text()).toContain('20.5°C')
  })

  it('toggles kiosk mode and full screen view', async () => {
    // Mock document.documentElement.requestFullscreen
    document.documentElement.requestFullscreen = vi.fn().mockResolvedValue(undefined)
    document.exitFullscreen = vi.fn().mockResolvedValue(undefined)

    const wrapper = mount(CustomDashboardView)
    await wrapper.vm.$nextTick()

    const kioskBtn = wrapper.find('.kiosk-toggle-btn')
    await kioskBtn.trigger('click')
    await wrapper.vm.$nextTick()

    expect(wrapper.classes()).toContain('kiosk-active')
  })
})
