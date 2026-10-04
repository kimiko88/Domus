import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import FamilyView from './FamilyView.vue'
import { useAuthStore } from '../stores/auth'
import { useFamilyStore } from '../stores/family'
import { i18n } from '../locales'

describe('FamilyView.vue', () => {
  let pinia: any

  beforeEach(() => {
    pinia = createPinia()
    setActivePinia(pinia)
    vi.restoreAllMocks()
  })

  it('renders family members list and role badges', async () => {
    const authStore = useAuthStore()
    authStore.user = {
      id: 'admin-1',
      username: 'admin_creator',
      role: 'admin',
      created_at: '2026-10-04T00:00:00Z',
      updated_at: '2026-10-04T00:00:00Z'
    }

    const familyStore = useFamilyStore()
    familyStore.members = [
      {
        id: 'admin-1',
        username: 'admin_creator',
        email: 'admin@domus.local',
        role: 'admin',
        enabled: true,
        created_at: '2026-10-04T00:00:00Z'
      },
      {
        id: 'member-2',
        username: 'kid_operator',
        email: 'kid@domus.local',
        role: 'operator',
        enabled: true,
        created_at: '2026-10-04T01:00:00Z'
      }
    ]
    familyStore.invites = []

    vi.spyOn(familyStore, 'fetchMembers').mockResolvedValue()
    vi.spyOn(familyStore, 'fetchInvites').mockResolvedValue()

    const wrapper = mount(FamilyView, {
      global: {
        plugins: [pinia, i18n]
      }
    })

    expect(wrapper.find('.section-title').text()).toContain('Gestione Famiglia & Permessi')
    const memberCards = wrapper.findAll('.member-card')
    expect(memberCards).toHaveLength(2)
    expect(memberCards[0].text()).toContain('admin_creator')
    expect(memberCards[1].text()).toContain('kid_operator')
    expect(wrapper.find('.invite-btn').exists()).toBe(true)
  })

  it('hides admin actions and invite button for viewers', async () => {
    const authStore = useAuthStore()
    authStore.user = {
      id: 'viewer-1',
      username: 'guest_viewer',
      role: 'viewer',
      created_at: '2026-10-04T00:00:00Z',
      updated_at: '2026-10-04T00:00:00Z'
    }

    const familyStore = useFamilyStore()
    familyStore.members = [
      {
        id: 'viewer-1',
        username: 'guest_viewer',
        role: 'viewer',
        enabled: true,
        created_at: '2026-10-04T00:00:00Z'
      }
    ]

    vi.spyOn(familyStore, 'fetchMembers').mockResolvedValue()

    const wrapper = mount(FamilyView, {
      global: {
        plugins: [pinia, i18n]
      }
    })

    expect(wrapper.find('.invite-btn').exists()).toBe(false)
    expect(wrapper.find('.invites-section').exists()).toBe(false)
    expect(wrapper.find('.role-select').exists()).toBe(false)
  })

  it('opens invite modal and generates invite URL', async () => {
    const authStore = useAuthStore()
    authStore.user = {
      id: 'admin-1',
      username: 'admin',
      role: 'admin',
      created_at: '2026-10-04T00:00:00Z',
      updated_at: '2026-10-04T00:00:00Z'
    }

    const familyStore = useFamilyStore()
    familyStore.members = []
    familyStore.invites = []
    vi.spyOn(familyStore, 'fetchMembers').mockResolvedValue()
    vi.spyOn(familyStore, 'fetchInvites').mockResolvedValue()
    vi.spyOn(familyStore, 'createInvite').mockResolvedValue({
      id: 'inv-123',
      token: 'tok-secret-abc',
      role: 'operator',
      expires_at: '2026-10-06T00:00:00Z',
      invite_url: '/register?token=tok-secret-abc'
    })

    const wrapper = mount(FamilyView, {
      global: {
        plugins: [pinia, i18n]
      }
    })

    // Click invite button
    await wrapper.find('.invite-btn').trigger('click')
    expect(wrapper.find('.modal-card').exists()).toBe(true)

    // Trigger generate invite
    const genBtn = wrapper.findAll('button').find(b => b.text().includes('Genera Link'))
    expect(genBtn).toBeDefined()
    await genBtn!.trigger('click')

    expect(familyStore.createInvite).toHaveBeenCalledWith('operator', 48)
  })
})
