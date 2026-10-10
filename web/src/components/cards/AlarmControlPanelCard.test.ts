import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import AlarmControlPanelCard from './AlarmControlPanelCard.vue'
import { useAlarmStore } from '../../stores/alarm'
import { i18n } from '../../locales'

describe('AlarmControlPanelCard.vue', () => {
  let alarmStore: ReturnType<typeof useAlarmStore>

  beforeEach(() => {
    setActivePinia(createPinia())
    alarmStore = useAlarmStore()
    vi.restoreAllMocks()
  })

  it('renders alarm header, state badge, and keypad', async () => {
    const wrapper = mount(AlarmControlPanelCard, {
      global: { plugins: [i18n] }
    })
    await flushPromises()

    expect(wrapper.text()).toContain('Sistema Antifurto Domus')
    expect(wrapper.text()).toContain('Disarmato')
    expect(wrapper.find('.keypad-grid').exists()).toBe(true)
    expect(wrapper.findAll('.key-btn').length).toBe(12) // 0-9, clear, backspace
  })

  it('appends digits when keypad buttons are clicked', async () => {
    const wrapper = mount(AlarmControlPanelCard, {
      global: { plugins: [i18n] }
    })
    await flushPromises()

    const digit1 = wrapper.findAll('.key-btn').find(b => b.text() === '1')
    const digit2 = wrapper.findAll('.key-btn').find(b => b.text() === '2')

    await digit1?.trigger('click')
    await digit2?.trigger('click')

    const filledDots = wrapper.findAll('.pin-dot.filled')
    expect(filledDots.length).toBe(2)
  })

  it('arms alarm when mode button clicked', async () => {
    const armSpy = vi.spyOn(alarmStore, 'arm').mockResolvedValue({
      id: 'alarm_main',
      name: 'Sistema Antifurto Domus',
      state: 'armed_home',
      updated_at: ''
    } as any)

    const wrapper = mount(AlarmControlPanelCard, {
      global: { plugins: [i18n] }
    })
    await flushPromises()

    const homeBtn = wrapper.find('.btn-home')
    await homeBtn.trigger('click')

    expect(armSpy).toHaveBeenCalledWith('armed_home', '')
  })
})
