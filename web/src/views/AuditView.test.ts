import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import AuditView from './AuditView.vue'
import { useAuditStore } from '../stores/audit'
import { i18n } from '../locales'

describe('AuditView.vue', () => {
  let pinia: any

  beforeEach(() => {
    pinia = createPinia()
    setActivePinia(pinia)
    vi.clearAllMocks()
  })

  it('renders loading state initially', async () => {
    const auditStore = useAuditStore()
    auditStore.loading = true

    const wrapper = mount(AuditView, {
      global: {
        plugins: [pinia, i18n]
      }
    })

    expect(wrapper.find('.loading-state').exists()).toBe(true)
  })

  it('renders empty state when no audit entries exist', async () => {
    const auditStore = useAuditStore()
    auditStore.entries = []
    auditStore.loading = false

    const wrapper = mount(AuditView, {
      global: {
        plugins: [pinia, i18n]
      }
    })

    expect(wrapper.find('.empty-state').exists()).toBe(true)
  })

  it('renders audit entries correctly and toggles payload visibility', async () => {
    const auditStore = useAuditStore()
    auditStore.loading = false
    auditStore.entries = [
      {
        id: 101,
        actor_type: 'local_user',
        actor_id: 'usr_admin',
        action: 'auth.login',
        target_type: 'user',
        target_id: 'usr_admin',
        payload: { ip: '192.168.1.50' },
        created_at: '2026-10-04T12:00:00Z'
      }
    ]

    const wrapper = mount(AuditView, {
      global: {
        plugins: [pinia, i18n]
      }
    })

    expect(wrapper.text()).toContain('auth.login')
    expect(wrapper.text()).toContain('usr_admin')

    // Find payload toggle button
    const toggleBtn = wrapper.find('.btn-toggle-payload')
    expect(toggleBtn.exists()).toBe(true)

    // Initially payload pre is not visible
    expect(wrapper.find('.payload-viewer').exists()).toBe(false)

    // Click to show payload
    await toggleBtn.trigger('click')
    expect(wrapper.find('.payload-viewer').exists()).toBe(true)
    expect(wrapper.find('.payload-viewer').text()).toContain('192.168.1.50')
  })
})
