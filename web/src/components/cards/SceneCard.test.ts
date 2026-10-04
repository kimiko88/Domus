import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import SceneCard from './SceneCard.vue'
import { useScenesStore } from '../../stores/scenes'
import type { Scene } from '../../api/types'

describe('SceneCard.vue', () => {
  let scenesStore: ReturnType<typeof useScenesStore>

  beforeEach(() => {
    setActivePinia(createPinia())
    scenesStore = useScenesStore()
  })

  it('renders scene name and actions count', () => {
    const scene: Scene = {
      id: 'scene_night',
      name: 'Buonanotte',
      icon: 'moon',
      actions: [
        { entity_key: 'light.all', service: 'turn_off' },
        { entity_key: 'switch.tv', service: 'turn_off' }
      ],
      created_at: '2026-10-04T00:00:00Z',
      updated_at: '2026-10-04T00:00:00Z'
    }

    const wrapper = mount(SceneCard, {
      props: { scene }
    })

    expect(wrapper.text()).toContain('Buonanotte')
    expect(wrapper.text()).toContain('2 Azioni')
  })

  it('calls applyScene when activate button is clicked', async () => {
    const scene: Scene = {
      id: 'scene_morning',
      name: 'Buongiorno',
      actions: [{ entity_key: 'light.kitchen', service: 'turn_on' }],
      created_at: '',
      updated_at: ''
    }

    const applySpy = vi.spyOn(scenesStore, 'applyScene').mockResolvedValue(undefined)

    const wrapper = mount(SceneCard, {
      props: { scene }
    })

    const activateBtn = wrapper.find('.btn-apply')
    expect(activateBtn.exists()).toBe(true)

    await activateBtn.trigger('click')

    expect(applySpy).toHaveBeenCalledWith('scene_morning')
  })

  it('emits edit and delete events', async () => {
    const scene: Scene = {
      id: 'scene_movie',
      name: 'Cinema',
      actions: [],
      created_at: '',
      updated_at: ''
    }

    const wrapper = mount(SceneCard, {
      props: { scene }
    })

    const editBtn = wrapper.find('.btn-edit')
    await editBtn.trigger('click')
    expect(wrapper.emitted('edit')?.[0]).toEqual([scene])

    const deleteBtn = wrapper.find('.btn-delete')
    await deleteBtn.trigger('click')
    expect(wrapper.emitted('delete')?.[0]).toEqual([scene])
  })
})
