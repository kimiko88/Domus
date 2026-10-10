import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import HelpersView from './HelpersView.vue'
import { apiClient } from '../api/client'

describe('HelpersView.vue', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.spyOn(apiClient, 'get').mockImplementation(async (url: string) => {
      if (url.includes('/api/v1/helpers')) {
        return [
          {
            id: 'h_bool_1',
            entity_key: 'input_boolean.guest_mode',
            name: 'Modalità Ospiti',
            domain: 'input_boolean',
            state: 'off',
            config_json: '{}'
          },
          {
            id: 'h_num_1',
            entity_key: 'input_number.target_climate',
            name: 'Soglia Riscaldamento',
            domain: 'input_number',
            state: '20.50',
            config_json: '{"min": 15, "max": 25, "step": 0.5, "unit": "°C"}'
          },
          {
            id: 'h_timer_1',
            entity_key: 'timer.irrigation',
            name: 'Timer Irrigazione',
            domain: 'timer',
            state: 'idle',
            config_json: '{"duration_seconds": 900, "remaining_seconds": 900, "status": "idle"}'
          }
        ] as any
      }
      return null as any
    })
    vi.spyOn(apiClient, 'post').mockResolvedValue({ success: true } as any)
  })

  it('renders helper entities list grouped by domain', async () => {
    const wrapper = mount(HelpersView)
    await flushPromises()

    expect(wrapper.text()).toContain('Entità Virtuali & Helper')
    expect(wrapper.text()).toContain('Modalità Ospiti')
    expect(wrapper.text()).toContain('Soglia Riscaldamento')
    expect(wrapper.text()).toContain('Timer Irrigazione')
  })

  it('toggles boolean helper state', async () => {
    const wrapper = mount(HelpersView)
    await flushPromises()

    const toggleBtn = wrapper.find('.toggle-pill')
    expect(toggleBtn.exists()).toBe(true)

    await toggleBtn.trigger('click')
    expect(apiClient.post).toHaveBeenCalledWith(
      '/api/v1/helpers/h_bool_1/action',
      { action: 'turn_on' }
    )
  })

  it('opens create helper modal on add button click', async () => {
    const wrapper = mount(HelpersView)
    await flushPromises()

    const addBtn = wrapper.find('.create-btn')
    expect(addBtn.exists()).toBe(true)

    await addBtn.trigger('click')
    await wrapper.vm.$nextTick()

    expect(wrapper.find('.modal-backdrop').exists()).toBe(true)
    expect(wrapper.text()).toContain('Crea Nuovo Helper')
  })
})
