import { defineStore } from 'pinia'
import { ref } from 'vue'
import { apiClient } from '../api/client'
import type { TemplateEntity } from '../api/types'

export const useTemplatesStore = defineStore('templates', () => {
  const templates = ref<TemplateEntity[]>([])
  const loading = ref<boolean>(false)
  const error = ref<string | null>(null)

  async function fetchTemplates(): Promise<TemplateEntity[]> {
    loading.value = true
    error.value = null
    try {
      const data = await apiClient.get<{ templates: TemplateEntity[]; total: number }>('/api/v1/template-entities')
      templates.value = data.templates || []
      return templates.value
    } catch (err: any) {
      error.value = err?.message || 'Failed to fetch template entities'
      throw err
    } finally {
      loading.value = false
    }
  }

  async function createTemplate(template: Partial<TemplateEntity>): Promise<TemplateEntity> {
    loading.value = true
    error.value = null
    try {
      const created = await apiClient.post<TemplateEntity>('/api/v1/template-entities', template)
      templates.value.push(created)
      return created
    } catch (err: any) {
      error.value = err?.message || 'Failed to create template entity'
      throw err
    } finally {
      loading.value = false
    }
  }

  async function deleteTemplate(id: string): Promise<void> {
    loading.value = true
    error.value = null
    try {
      await apiClient.delete(`/api/v1/template-entities/${id}`)
      templates.value = templates.value.filter((t) => t.id !== id)
    } catch (err: any) {
      error.value = err?.message || 'Failed to delete template entity'
      throw err
    } finally {
      loading.value = false
    }
  }

  async function previewTemplate(template: string): Promise<string> {
    try {
      const data = await apiClient.post<{ result: string }>('/api/v1/template-entities/preview', { template })
      return data.result
    } catch (err: any) {
      throw new Error(err?.message || 'Preview evaluation failed')
    }
  }

  return {
    templates,
    loading,
    error,
    fetchTemplates,
    createTemplate,
    deleteTemplate,
    previewTemplate,
  }
})
