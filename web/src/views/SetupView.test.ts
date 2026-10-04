import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import SetupView from './SetupView.vue'
import { useAuthStore } from '../stores/auth'

describe('SetupView.vue', () => {
  let authStore: ReturnType<typeof useAuthStore>

  beforeEach(() => {
    setActivePinia(createPinia())
    authStore = useAuthStore()
  })

  it('renders initial admin setup form', () => {
    const wrapper = mount(SetupView)
    expect(wrapper.text()).toContain('Configurazione Iniziale')
    expect(wrapper.find('input[id="admin-username"]').exists()).toBe(true)
    expect(wrapper.find('input[id="admin-password"]').exists()).toBe(true)
    expect(wrapper.find('input[id="admin-password-confirm"]').exists()).toBe(true)
  })

  it('shows error when passwords do not match', async () => {
    const wrapper = mount(SetupView)
    await wrapper.find('input[id="admin-username"]').setValue('admin')
    await wrapper.find('input[id="admin-password"]').setValue('password123')
    await wrapper.find('input[id="admin-password-confirm"]').setValue('mismatch')
    await wrapper.find('form').trigger('submit.prevent')

    expect(wrapper.text()).toContain('Le password non coincidono')
  })

  it('calls authStore.registerFirstAdmin when form is valid', async () => {
    const registerSpy = vi.spyOn(authStore, 'registerFirstAdmin').mockResolvedValue(true)

    const wrapper = mount(SetupView)
    await wrapper.find('input[id="admin-username"]').setValue('superadmin')
    await wrapper.find('input[id="admin-password"]').setValue('secure_pass_123')
    await wrapper.find('input[id="admin-password-confirm"]').setValue('secure_pass_123')
    await wrapper.find('form').trigger('submit.prevent')

    expect(registerSpy).toHaveBeenCalledWith('superadmin', 'secure_pass_123')
  })
})
