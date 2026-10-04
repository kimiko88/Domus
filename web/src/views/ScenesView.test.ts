import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import ScenesView from './ScenesView.vue'
import { useScenesStore } from '../stores/scenes'
import type { Scene } from '../api/types'

describe('ScenesView.vue', () => {
  let scenesStore: ReturnType<typeof useScenesStore>

  beforeEach(() => {
    setActivePinia(createPinia())
    scenesStore = useScenesStore()
  })

  it('fetches and renders list of scenes', async () => {
    const mockScenes: Scene[] = [
      {
        id: 'scene_night',
        name: 'Tutto Spento',
        actions: [{ entity_key: 'light.all', service: 'turn_off' }],
        created_at: '',
        updated_at: ''
      },
      {
        id: 'scene_work',
        name: 'Concentrazione',
        actions: [{ entity_key: 'light.desk', service: 'turn_on' }],
        created_at: '',
        updated_at: ''
      }
    ]

    vi.spyOn(scenesStore, 'fetchScenes').mockResolvedValue(mockScenes)
    scenesStore.scenes = mockScenes

    const wrapper = mount(ScenesView)
    await wrapper.vm.$nextTick()

    expect(wrapper.text()).toContain('Tutto Spento')
    expect(wrapper.text()).toContain('Concentrazione')
    expect(wrapper.find('.btn-new-scene').exists()).toBe(true)
  })

  it('opens editor modal when new scene button is clicked', async () => {
    vi.spyOn(scenesStore, 'fetchScenes').mockResolvedValue([])

    const wrapper = mount(ScenesView)
    await wrapper.vm.$nextTick()

    await wrapper.find('.btn-new-scene').trigger('click')

    expect(wrapper.findComponent({ name: 'SceneEditorModal' }).props('isOpen')).toBe(true)
  })

  it('calls deleteScene when delete event is fired', async () => {
    const mockScenes: Scene[] = [
      {
        id: 'scene_del',
        name: 'Delete Scene',
        actions: [],
        created_at: '',
        updated_at: ''
      }
    ]
    scenesStore.scenes = mockScenes
    vi.spyOn(scenesStore, 'fetchScenes').mockResolvedValue(mockScenes)
    const deleteSpy = vi.spyOn(scenesStore, 'deleteScene').mockResolvedValue(undefined)

    // Mock confirm
    vi.spyOn(window, 'confirm').mockReturnValue(true)

    const wrapper = mount(ScenesView)
    await wrapper.vm.$nextTick()

    const sceneCard = wrapper.findComponent({ name: 'SceneCard' })
    sceneCard.vm.$emit('delete', mockScenes[0])

    expect(deleteSpy).toHaveBeenCalledWith('scene_del')
  })
})
