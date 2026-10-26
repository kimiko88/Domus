import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import IntercomCallModal from './IntercomCallModal.vue'
import { apiClient } from '../../api/client'

describe('IntercomCallModal.vue', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  it('renders caller name and video stream container when open', () => {
    const wrapper = mount(IntercomCallModal, {
      props: {
        isOpen: true,
        callerName: 'Corriere Amazon',
        cameraId: 'cam_front_door'
      }
    })

    expect(wrapper.find('.intercom-modal').exists()).toBe(true)
    expect(wrapper.text()).toContain('Corriere Amazon')
    expect(wrapper.text()).toContain('LIVE WebRTC')
    expect(wrapper.text()).toContain('Apri Cancello')
  })

  it('toggles microphone mute state on click', async () => {
    const wrapper = mount(IntercomCallModal, {
      props: {
        isOpen: true
      }
    })

    const muteBtn = wrapper.find('.btn-mute-toggle')
    expect(muteBtn.text()).toContain('Microfono ON')

    await muteBtn.trigger('click')
    expect(muteBtn.text()).toContain('Muto')
  })

  it('triggers unlock gate API and emits event when clicking unlock', async () => {
    const postSpy = vi.spyOn(apiClient, 'post').mockResolvedValue({ status: 'unlocked' })

    const wrapper = mount(IntercomCallModal, {
      props: {
        isOpen: true,
        cameraId: 'cam_front_door'
      }
    })

    const unlockBtn = wrapper.find('.btn-intercom-unlock')
    await unlockBtn.trigger('click')

    expect(postSpy).toHaveBeenCalledWith('/api/v1/intercom/cam_front_door/unlock', {})
    expect(wrapper.emitted('unlocked')).toBeTruthy()
  })
})
