import { describe, it, expect, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import AppTourModal from './AppTourModal.vue'
import { useTourStore } from '../../stores/tour'
import { i18n } from '../../locales'

describe('AppTourModal.vue', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    localStorage.clear()
  })

  it('does not render when tour is closed', () => {
    const store = useTourStore()
    store.isOpen = false

    const wrapper = mount(AppTourModal, {
      global: {
        plugins: [i18n]
      }
    })

    expect(wrapper.find('.tour-backdrop').exists()).toBe(false)
  })

  it('renders modal content when tour is open', () => {
    const store = useTourStore()
    store.startTour()

    const wrapper = mount(AppTourModal, {
      global: {
        plugins: [i18n]
      }
    })

    expect(wrapper.find('.tour-backdrop').exists()).toBe(true)
    expect(wrapper.find('.step-title').exists()).toBe(true)
    expect(wrapper.find('.btn-skip').exists()).toBe(true)
  })

  it('navigates next and prev steps on button click', async () => {
    const store = useTourStore()
    store.startTour()

    const wrapper = mount(AppTourModal, {
      global: {
        plugins: [i18n]
      }
    })

    expect(store.currentStep).toBe(0)
    const nextBtn = wrapper.find('.btn-next')
    await nextBtn.trigger('click')
    expect(store.currentStep).toBe(1)

    const prevBtn = wrapper.find('.btn-prev')
    await prevBtn.trigger('click')
    expect(store.currentStep).toBe(0)
  })

  it('skips the tour when skip button is clicked', async () => {
    const store = useTourStore()
    store.startTour()

    const wrapper = mount(AppTourModal, {
      global: {
        plugins: [i18n]
      }
    })

    const skipBtn = wrapper.find('.btn-skip')
    await skipBtn.trigger('click')
    expect(store.isOpen).toBe(false)
    expect(store.isCompleted).toBe(true)
  })
})
