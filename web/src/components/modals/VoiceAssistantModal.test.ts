import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import VoiceAssistantModal from './VoiceAssistantModal.vue'
import { useAssistantStore } from '../../stores/assistant'

describe('VoiceAssistantModal.vue', () => {
  let assistantStore: ReturnType<typeof useAssistantStore>

  beforeEach(() => {
    setActivePinia(createPinia())
    assistantStore = useAssistantStore()
    vi.restoreAllMocks()
  })

  it('renders modal when isOpen is true', () => {
    const wrapper = mount(VoiceAssistantModal, {
      props: { isOpen: true }
    })

    expect(wrapper.find('.assistant-modal').exists()).toBe(true)
    expect(wrapper.text()).toContain('Domus AI Assistant')
    expect(wrapper.text()).toContain('Offline-First')
  })

  it('does not render modal when isOpen is false', () => {
    const wrapper = mount(VoiceAssistantModal, {
      props: { isOpen: false }
    })

    expect(wrapper.find('.assistant-modal').exists()).toBe(false)
  })

  it('sends prompt when form submitted', async () => {
    const sendSpy = vi.spyOn(assistantStore, 'sendPrompt').mockResolvedValue(null)

    const wrapper = mount(VoiceAssistantModal, {
      props: { isOpen: true }
    })

    const input = wrapper.find('.prompt-input')
    await input.setValue('Spegni le luci')
    await wrapper.find('.assistant-footer').trigger('submit')

    expect(sendSpy).toHaveBeenCalledWith('Spegni le luci')
  })
})
