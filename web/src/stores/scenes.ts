import { defineStore } from 'pinia'
import { ref } from 'vue'
import { apiClient } from '../api/client'
import type { Scene, SceneListResponse } from '../api/types'

export const useScenesStore = defineStore('scenes', () => {
  const scenes = ref<Scene[]>([])
  const loading = ref<boolean>(false)
  const applyingId = ref<string | null>(null)
  const error = ref<string | null>(null)

  async function fetchScenes(): Promise<Scene[]> {
    loading.value = true
    error.value = null
    try {
      const data = await apiClient.get<SceneListResponse>('/api/v1/scenes')
      scenes.value = data.scenes || []
      return scenes.value
    } catch (err: any) {
      error.value = err?.message || 'Failed to fetch scenes'
      throw err
    } finally {
      loading.value = false
    }
  }

  async function getScene(id: string): Promise<Scene> {
    loading.value = true
    error.value = null
    try {
      const data = await apiClient.get<Scene>(`/api/v1/scenes/${id}`)
      const index = scenes.value.findIndex((s) => s.id === id)
      if (index !== -1) {
        scenes.value[index] = data
      } else {
        scenes.value.push(data)
      }
      return data
    } catch (err: any) {
      error.value = err?.message || 'Failed to fetch scene'
      throw err
    } finally {
      loading.value = false
    }
  }

  async function createScene(data: Partial<Scene>): Promise<Scene> {
    loading.value = true
    error.value = null
    try {
      const created = await apiClient.post<Scene>('/api/v1/scenes', data)
      scenes.value.push(created)
      return created
    } catch (err: any) {
      error.value = err?.message || 'Failed to create scene'
      throw err
    } finally {
      loading.value = false
    }
  }

  async function updateScene(id: string, data: Partial<Scene>): Promise<Scene> {
    loading.value = true
    error.value = null
    try {
      const updated = await apiClient.put<Scene>(`/api/v1/scenes/${id}`, data)
      const index = scenes.value.findIndex((s) => s.id === id)
      if (index !== -1) {
        scenes.value[index] = updated
      }
      return updated
    } catch (err: any) {
      error.value = err?.message || 'Failed to update scene'
      throw err
    } finally {
      loading.value = false
    }
  }

  async function deleteScene(id: string): Promise<void> {
    loading.value = true
    error.value = null
    try {
      await apiClient.delete(`/api/v1/scenes/${id}`)
      scenes.value = scenes.value.filter((s) => s.id !== id)
    } catch (err: any) {
      error.value = err?.message || 'Failed to delete scene'
      throw err
    } finally {
      loading.value = false
    }
  }

  async function applyScene(id: string): Promise<void> {
    applyingId.value = id
    error.value = null
    try {
      await apiClient.post(`/api/v1/scenes/${id}/apply`)
    } catch (err: any) {
      error.value = err?.message || 'Failed to apply scene'
      throw err
    } finally {
      applyingId.value = null
    }
  }

  return {
    scenes,
    loading,
    applyingId,
    error,
    fetchScenes,
    getScene,
    createScene,
    updateScene,
    deleteScene,
    applyScene
  }
})
