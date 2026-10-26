import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import StoreView from './StoreView.vue'
import { apiClient } from '../api/client'

describe('StoreView.vue', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.spyOn(apiClient, 'get').mockImplementation(async (url: string) => {
      if (url.includes('/api/v1/store/packages')) {
        return [
          {
            id: 'pkg_aqara_wasm',
            package_id: 'community.aqara.cube',
            name: 'Aqara Magic Cube Controller',
            type: 'wasm_plugin',
            version: '1.2.0',
            author: 'DomusCommunity',
            description: 'Plugin WASM per interpretare rotazioni',
            installed: true
          },
          {
            id: 'pkg_bp_circadian',
            package_id: 'blueprints.lighting.circadian',
            name: 'Illuminazione Circadiana Dinamica',
            type: 'blueprint',
            version: '2.0.1',
            author: 'SmartHomeItaly',
            description: 'Regola temperatura colore',
            installed: false
          }
        ] as any
      }
      return null as any
    })
    vi.spyOn(apiClient, 'post').mockResolvedValue({ success: true } as any)
  })

  it('renders store title and filter pills', async () => {
    const wrapper = mount(StoreView)
    await flushPromises()

    expect(wrapper.text()).toContain('Domus Store & Marketplace Estensioni')
    expect(wrapper.text()).toContain('WASM Plugins')
    expect(wrapper.text()).toContain('Blueprints')
  })

  it('filters packages and triggers 1-click install', async () => {
    const wrapper = mount(StoreView)
    await flushPromises()

    expect(wrapper.text()).toContain('Aqara Magic Cube Controller')
    expect(wrapper.text()).toContain('Illuminazione Circadiana Dinamica')

    const installBtns = wrapper.findAll('.install-btn')
    expect(installBtns.length).toBeGreaterThan(0)

    // Click install on uninstalled package
    await installBtns[1].trigger('click')
    expect(apiClient.post).toHaveBeenCalledWith(
      '/api/v1/store/packages/pkg_bp_circadian/install',
      {}
    )
  })

  it('filters packages by search query', async () => {
    const wrapper = mount(StoreView)
    await flushPromises()

    const searchInput = wrapper.find('input#store-search')
    expect(searchInput.exists()).toBe(true)

    await searchInput.setValue('Aqara')
    await flushPromises()

    expect(wrapper.text()).toContain('Aqara Magic Cube Controller')
    expect(wrapper.text()).not.toContain('Illuminazione Circadiana Dinamica')
  })

  it('opens and closes details modal on card click', async () => {
    const wrapper = mount(StoreView)
    await flushPromises()

    const cards = wrapper.findAll('.package-card')
    expect(cards.length).toBe(2)

    await cards[0].trigger('click')
    await flushPromises()

    const modal = wrapper.find('.modal-card')
    expect(modal.exists()).toBe(true)
    expect(modal.text()).toContain('Aqara Magic Cube Controller')
    expect(modal.text()).toContain('community.aqara.cube')

    const closeBtn = wrapper.find('.btn-close')
    await closeBtn.trigger('click')
    await flushPromises()

    expect(wrapper.find('.modal-card').exists()).toBe(false)
  })

  it('handles community plugin proposal modal submission', async () => {
    const wrapper = mount(StoreView)
    await flushPromises()

    const proposeBtn = wrapper.find('.btn-propose-community')
    expect(proposeBtn.exists()).toBe(true)
    await proposeBtn.trigger('click')

    expect(wrapper.find('.modal-card.modal-lg').exists()).toBe(true)
    expect(wrapper.text()).toContain('Proponi un Plugin Open Source')

    // Fill form
    await wrapper.find('input#pkg-name').setValue('LoRa Emergency Beacon')
    await wrapper.find('input#pkg-id').setValue('domus.community.lora_beacon')
    await wrapper.find('input#pkg-repo').setValue('https://github.com/community/lora-beacon')

    // Mock successful submission response
    vi.spyOn(apiClient, 'post').mockResolvedValueOnce({
      id: 'pkg_comm_lora_beacon',
      package_id: 'domus.community.lora_beacon',
      name: 'LoRa Emergency Beacon',
      type: 'wasm_plugin',
      tier: 'community',
      review_status: 'pending_review',
      version: '1.0.0',
      author: 'Community',
      description: 'LoRa beacon for emergencies',
      repository_url: 'https://github.com/community/lora-beacon',
      license: 'GPL-3.0',
      installed: false
    } as any)

    await wrapper.find('form').trigger('submit.prevent')
    await flushPromises()

    expect(apiClient.post).toHaveBeenCalledWith(
      '/api/v1/store/community/submit',
      expect.objectContaining({
        name: 'LoRa Emergency Beacon',
        package_id: 'domus.community.lora_beacon',
        repository_url: 'https://github.com/community/lora-beacon'
      })
    )
  })
})
