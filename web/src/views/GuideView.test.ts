import { describe, it, expect, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import GuideView from './GuideView.vue'
import { useTourStore } from '../stores/tour'
import { i18n } from '../locales'
import { createRouter, createMemoryHistory } from 'vue-router'

const router = createRouter({
  history: createMemoryHistory(),
  routes: [{ path: '/guide', component: GuideView }]
})

describe('GuideView.vue', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    localStorage.clear()
  })

  it('renders hero title and subtitle', () => {
    const wrapper = mount(GuideView, {
      global: {
        plugins: [createPinia(), i18n, router]
      }
    })

    expect(wrapper.find('.hero-title').exists()).toBe(true)
    expect(wrapper.find('.hero-subtitle').exists()).toBe(true)
    expect(wrapper.find('.btn-tour-hero').exists()).toBe(true)
  })

  it('triggers tour store when start tour button is clicked', async () => {
    const pinia = createPinia()
    setActivePinia(pinia)
    const tourStore = useTourStore()

    const wrapper = mount(GuideView, {
      global: {
        plugins: [pinia, i18n, router]
      }
    })

    expect(tourStore.isOpen).toBe(false)
    const tourBtn = wrapper.find('.btn-tour-hero')
    await tourBtn.trigger('click')
    expect(tourStore.isOpen).toBe(true)
    expect(tourStore.currentStep).toBe(0)
  })

  it('filters topics by search query', async () => {
    const wrapper = mount(GuideView, {
      global: {
        plugins: [createPinia(), i18n, router]
      }
    })

    const initialCards = wrapper.findAll('.topic-card').length
    expect(initialCards).toBeGreaterThan(0)

    const searchInput = wrapper.find('.search-input')
    await searchInput.setValue('nonexistent_random_xyz')
    expect(wrapper.findAll('.topic-card').length).toBe(0)
    expect(wrapper.find('.empty-state').exists()).toBe(true)
  })

  it('filters topics by category tab', async () => {
    const wrapper = mount(GuideView, {
      global: {
        plugins: [createPinia(), i18n, router]
      }
    })

    const tabs = wrapper.findAll('.tab-btn')
    const energyTab = tabs.find(t => t.text().includes('Energia') || t.text().includes('Energy'))
    if (energyTab) {
      await energyTab.trigger('click')
      const cards = wrapper.findAll('.topic-card')
      expect(cards.length).toBeGreaterThan(0)
    }
  })
})
