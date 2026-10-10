import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import CamerasView from './CamerasView.vue'
import { apiClient } from '../api/client'

describe('CamerasView.vue', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.spyOn(apiClient, 'get').mockImplementation(async (url: string) => {
      if (url.includes('/api/v1/cameras/events')) {
        return [
          {
            id: 'det_1',
            camera_id: 'cam_front',
            label: 'person',
            score: 0.95,
            detected_at: 'Adesso'
          }
        ] as any
      }
      if (url.includes('/api/v1/cameras')) {
        return [
          {
            id: 'cam_front',
            name: 'Ingresso Principale',
            stream_url: 'rtsp://192.168.1.50/live',
            stream_type: 'webrtc',
            detection_enabled: true
          }
        ] as any
      }
      return null as any
    })
  })

  it('renders cameras view title and feeds', async () => {
    const wrapper = mount(CamerasView)
    await flushPromises()

    expect(wrapper.text()).toContain('Videosorveglianza & Visione AI')
    expect(wrapper.text()).toContain('Ingresso Principale')
    expect(wrapper.text()).toContain('LIVE (WebRTC)')
  })

  it('renders Frigate AI detection events timeline', async () => {
    const wrapper = mount(CamerasView)
    await flushPromises()

    expect(wrapper.text()).toContain('Timeline Eventi Riconoscimento Locale')
    expect(wrapper.text()).toContain('PERSON')
    expect(wrapper.text()).toContain('95%')
  })
})
