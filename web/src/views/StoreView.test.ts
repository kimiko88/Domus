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
})
