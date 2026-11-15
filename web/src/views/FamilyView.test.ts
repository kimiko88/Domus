import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import FamilyView from './FamilyView.vue'
import { useAuthStore } from '../stores/auth'
import { useFamilyStore } from '../stores/family'
import { i18n } from '../locales'
import { apiClient } from '../api/client'

describe('FamilyView.vue', () => {
  let pinia: any

  beforeEach(() => {
    pinia = createPinia()
    setActivePinia(pinia)
    vi.restoreAllMocks()
    vi.spyOn(apiClient, 'get').mockImplementation(async () => [] as any)
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
    expect(memberCards[1].find('.permissions-btn').exists()).toBe(true)
    await memberCards[1].find('.permissions-btn').trigger('click')
    expect(wrapper.find('.permission-modal-card').exists()).toBe(true)
    expect(wrapper.find('.permission-section').exists()).toBe(true)
    expect(wrapper.find('.invite-btn').exists()).toBe(true)
  })

  it('renders family permission controls in Russian', async () => {
    const previousLocale = i18n.global.locale.value
    i18n.global.locale.value = 'ru'
    try {
      const authStore = useAuthStore()
      authStore.user = {
        id: 'admin-1',
        username: 'admin',
        role: 'admin',
        created_at: '2026-10-04T00:00:00Z',
        updated_at: '2026-10-04T00:00:00Z'
      }

      const familyStore = useFamilyStore()
      familyStore.members = [
        { id: 'admin-1', username: 'admin', role: 'admin', enabled: true, created_at: '2026-10-04T00:00:00Z' },
        { id: 'member-2', username: 'member', role: 'operator', enabled: true, created_at: '2026-10-04T01:00:00Z' }
      ]
      vi.spyOn(familyStore, 'fetchMembers').mockResolvedValue()
      vi.spyOn(familyStore, 'fetchInvites').mockResolvedValue()
      vi.spyOn(familyStore, 'fetchGuests').mockResolvedValue()
      const updatePermissions = vi.spyOn(familyStore, 'updateMemberPermissions').mockResolvedValue()

      expect(i18n.global.t('family.title')).toBe('Управление семьёй и разрешениями')
      const wrapper = mount(FamilyView, {
        global: { plugins: [pinia, i18n] }
      })
      await flushPromises()

      expect(wrapper.find('.section-title').exists()).toBe(true)
      expect(wrapper.find('.section-title').text()).toContain('Управление семьёй и разрешениями')
      const permissionsButton = wrapper.find('.permissions-btn')
      expect(permissionsButton.exists()).toBe(true)
      expect(permissionsButton.attributes('title')).toBe('Управление разрешениями')
      await permissionsButton.trigger('click')
      await flushPromises()
      const permissionModal = wrapper.find('.permission-modal-card')
      expect(permissionModal.exists()).toBe(true)
      expect(permissionModal.text()).toContain('Доступные помещения')
      expect(permissionModal.text()).toContain('Разрешённые действия')
      expect(permissionModal.text()).toContain('Управление автоматизациями')
      expect(permissionModal.text()).toContain('Управление системными настройками')

      const settingsToggle = permissionModal.findAll('label.permission-check')
        .find(label => label.text().includes('Управление системными настройками'))
      expect(settingsToggle).toBeDefined()
      await settingsToggle!.find('input').setValue(true)
      await flushPromises()
      const saveButton = permissionModal.find('.modal-footer').findAll('button')
        .find(button => button.text().includes('Сохранить разрешения'))
      expect(saveButton).toBeDefined()
      await saveButton!.trigger('click')
      await flushPromises()
      expect(updatePermissions).toHaveBeenCalledWith(
        'member-2',
        expect.objectContaining({ can_manage_settings: true })
      )
    } finally {
      i18n.global.locale.value = previousLocale
    }
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
  it('allows administrators to invite a child account with restricted permissions', async () => {
    const authStore = useAuthStore()
    authStore.user = {
      id: 'admin-1',
      username: 'admin',
      role: 'admin',
      created_at: '2026-10-04T00:00:00Z',
      updated_at: '2026-10-04T00:00:00Z'
    }

    const familyStore = useFamilyStore()
    vi.spyOn(familyStore, 'fetchMembers').mockResolvedValue()
    vi.spyOn(familyStore, 'fetchInvites').mockResolvedValue()
    const createInvite = vi.spyOn(familyStore, 'createInvite').mockResolvedValue({
      id: 'inv-kid',
      token: 'kid-token',
      role: 'kid',
      expires_at: '2026-10-06T00:00:00Z',
      invite_url: '/register?token=kid-token'
    })

    const wrapper = mount(FamilyView, {
      global: { plugins: [pinia, i18n] }
    })

    await wrapper.find('.invite-btn').trigger('click')
    await wrapper.find('input[value="kid"]').setValue()
    const genBtn = wrapper.findAll('button').find(button => button.text().includes('Genera Link'))
    expect(genBtn).toBeDefined()
    await genBtn!.trigger('click')

    expect(createInvite).toHaveBeenCalledWith('kid', 48)
  })
})
