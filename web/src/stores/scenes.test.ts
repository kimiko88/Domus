import { describe, it, expect, beforeEach, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useScenesStore } from './scenes'
import { apiClient } from '../api/client'
import type { Scene, SceneListResponse } from '../api/types'

describe('useScenesStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('fetches scenes and populates list', async () => {
    const mockResponse: SceneListResponse = {
      scenes: [
        {
          id: 'scene_movie',
          name: 'Cinema Night',
          icon: 'film',
          actions: [
            { entity_key: 'light.living_room', service: 'turn_on', params: { brightness: 50 } },
            { entity_key: 'switch.tv', service: 'turn_on' }
          ],
          created_at: '2026-10-04T00:00:00Z',
          updated_at: '2026-10-04T00:00:00Z'
        }
      ],
      total: 1
    }

    vi.spyOn(apiClient, 'get').mockResolvedValue(mockResponse)

    const store = useScenesStore()
    const result = await store.fetchScenes()

    expect(result).toHaveLength(1)
    expect(store.scenes).toHaveLength(1)
    expect(store.scenes[0].name).toBe('Cinema Night')
  })

  it('creates a new scene and adds it to store', async () => {
    const newScene: Partial<Scene> = {
      name: 'Relax Evening',
      icon: 'sunset',
      actions: [{ entity_key: 'light.lamp', service: 'turn_on', params: { brightness: 100 } }]
    }

    const created: Scene = {
      ...(newScene as any),
      id: 'scene_relax',
      created_at: '2026-10-04T00:00:00Z',
      updated_at: '2026-10-04T00:00:00Z'
    }

    vi.spyOn(apiClient, 'post').mockResolvedValue(created)

    const store = useScenesStore()
    const result = await store.createScene(newScene)

    expect(result.id).toBe('scene_relax')
    expect(store.scenes).toContainEqual(created)
  })

  it('updates an existing scene', async () => {
    const store = useScenesStore()
    store.scenes = [
      {
        id: 'scene_1',
        name: 'Old Scene',
        actions: [{ entity_key: 'light.1', service: 'turn_off' }],
        created_at: '',
        updated_at: ''
      }
    ]

    const updated = { ...store.scenes[0], name: 'New Scene Name' }
    vi.spyOn(apiClient, 'put').mockResolvedValue(updated)

    await store.updateScene('scene_1', { name: 'New Scene Name' })

    expect(store.scenes[0].name).toBe('New Scene Name')
  })

  it('deletes a scene from store', async () => {
    const store = useScenesStore()
    store.scenes = [
      {
        id: 'scene_del',
        name: 'Delete Me',
        actions: [],
        created_at: '',
        updated_at: ''
      }
    ]

    vi.spyOn(apiClient, 'delete').mockResolvedValue({})

    await store.deleteScene('scene_del')

    expect(store.scenes).toHaveLength(0)
  })

  it('applies a scene', async () => {
    const postSpy = vi.spyOn(apiClient, 'post').mockResolvedValue({ status: 'applied' })

    const store = useScenesStore()
    await store.applyScene('scene_1')

    expect(postSpy).toHaveBeenCalledWith('/api/v1/scenes/scene_1/apply')
  })
})
