import { defineStore } from 'pinia'
import { ref } from 'vue'
import { apiClient } from '../api/client'
import type { Automation, AutomationListResponse, AutomationRun, AutomationRunsResponse } from '../api/types'

export const useAutomationsStore = defineStore('automations', () => {
  const automations = ref<Automation[]>([])
  const runs = ref<Record<string, AutomationRun[]>>({})
  const loading = ref<boolean>(false)
  const error = ref<string | null>(null)

  async function fetchAutomations(enabled?: boolean): Promise<Automation[]> {
    loading.value = true
    error.value = null
    try {
      const params: Record<string, any> = {}
      if (enabled !== undefined) {
        params.enabled = enabled
      }
      const data = await apiClient.get<AutomationListResponse>('/api/v1/automations', params)
      automations.value = data.automations || []
      return automations.value
    } catch (err: any) {
      error.value = err?.message || 'Failed to fetch automations'
      throw err
    } finally {
      loading.value = false
    }
  }

  async function getAutomation(id: string): Promise<Automation> {
    loading.value = true
    error.value = null
    try {
      const data = await apiClient.get<Automation>(`/api/v1/automations/${id}`)
      const index = automations.value.findIndex((a) => a.id === id)
      if (index !== -1) {
        automations.value[index] = data
      } else {
        automations.value.push(data)
      }
      return data
    } catch (err: any) {
      error.value = err?.message || 'Failed to fetch automation'
      throw err
    } finally {
      loading.value = false
    }
  }

  async function createAutomation(data: Partial<Automation>): Promise<Automation> {
    loading.value = true
    error.value = null
    try {
      const created = await apiClient.post<Automation>('/api/v1/automations', data)
      automations.value.push(created)
      return created
    } catch (err: any) {
      error.value = err?.message || 'Failed to create automation'
      throw err
    } finally {
      loading.value = false
    }
  }

  async function updateAutomation(id: string, data: Partial<Automation>): Promise<Automation> {
    loading.value = true
    error.value = null
    try {
      const updated = await apiClient.put<Automation>(`/api/v1/automations/${id}`, data)
      const index = automations.value.findIndex((a) => a.id === id)
      if (index !== -1) {
        automations.value[index] = updated
      }
      return updated
    } catch (err: any) {
      error.value = err?.message || 'Failed to update automation'
      throw err
    } finally {
      loading.value = false
    }
  }

  async function toggleAutomation(id: string): Promise<boolean> {
    const existing = automations.value.find((a) => a.id === id)
    if (!existing) return false
    const newEnabled = !existing.enabled
    await updateAutomation(id, { enabled: newEnabled })
    return newEnabled
  }

  async function deleteAutomation(id: string): Promise<void> {
    loading.value = true
    error.value = null
    try {
      await apiClient.delete(`/api/v1/automations/${id}`)
      automations.value = automations.value.filter((a) => a.id !== id)
      delete runs.value[id]
    } catch (err: any) {
      error.value = err?.message || 'Failed to delete automation'
      throw err
    } finally {
      loading.value = false
    }
  }

  async function triggerAutomation(id: string): Promise<void> {
    error.value = null
    try {
      await apiClient.post(`/api/v1/automations/${id}/trigger`)
    } catch (err: any) {
      error.value = err?.message || 'Failed to trigger automation'
      throw err
    }
  }

  async function fetchRuns(id: string, limit = 20): Promise<AutomationRun[]> {
    try {
      const data = await apiClient.get<AutomationRunsResponse>(`/api/v1/automations/${id}/runs`, { limit })
      runs.value[id] = data.runs || []
      return runs.value[id]
    } catch (err: any) {
      error.value = err?.message || 'Failed to fetch automation runs'
      throw err
    }
  }

  return {
    automations,
    runs,
    loading,
    error,
    fetchAutomations,
    getAutomation,
    createAutomation,
    updateAutomation,
    toggleAutomation,
    deleteAutomation,
    triggerAutomation,
    fetchRuns
  }
})
