import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import MediaPlayerView from './MediaPlayerView.vue'
import { apiClient } from '../api/client'

describe('MediaPlayerView.vue', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.spyOn(apiClient, 'get').mockImplementation(async (url: string) => {
      if (url.includes('/api/v1/media/players')) {
        return [
          {
            id: 'media_living',
            name: 'Google Nest Hub Salotto',
            platform: 'cast',
            state: 'playing',
            volume: 0.70,
            muted: false,
            current_title: 'Radio Monte Carlo',
            current_artist: 'Live Streaming Web'
          }
        ] as any
      }
      return null as any
    })
    vi.spyOn(apiClient, 'post').mockResolvedValue({ success: true } as any)
  })

  it('renders media players and playback details', async () => {
    const wrapper = mount(MediaPlayerView)
    await flushPromises()

    expect(wrapper.text()).toContain('Audio Multi-room')
    expect(wrapper.text()).toContain('Google Nest Hub Salotto')
    expect(wrapper.text()).toContain('Radio Monte Carlo')
    expect(wrapper.text()).toContain('70%')
  })

  it('opens voice announcement modal on broadcast button click', async () => {
    const wrapper = mount(MediaPlayerView)
    await flushPromises()

    const broadcastBtn = wrapper.find('.broadcast-trigger-btn')
    expect(broadcastBtn.exists()).toBe(true)

    await broadcastBtn.trigger('click')
    await wrapper.vm.$nextTick()

    expect(wrapper.find('.modal-backdrop').exists()).toBe(true)
    expect(wrapper.text()).toContain('Diffusione Vocale Sincronizzata')
  })

  it('opens add player modal on add button click', async () => {
    const wrapper = mount(MediaPlayerView)
    await flushPromises()

    const addBtn = wrapper.find('.add-player-btn')
    expect(addBtn.exists()).toBe(true)

    await addBtn.trigger('click')
    await wrapper.vm.$nextTick()

    expect(wrapper.find('.modal-backdrop').exists()).toBe(true)
    expect(wrapper.text()).toContain('Registra Media Player nel Database')
  })
})
