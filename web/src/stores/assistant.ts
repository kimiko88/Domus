import { defineStore } from 'pinia'
import { ref } from 'vue'
import { apiClient } from '../api/client'
import type { AssistantActionExecuted, AssistantChatResponse, AssistantConfig } from '../api/types'

export interface ChatMessage {
  id: string
  sender: 'user' | 'assistant'
  text: string
  actions?: AssistantActionExecuted[]
  engine?: string
  timestamp: string
}

export const useAssistantStore = defineStore('assistant', () => {
  const messages = ref<ChatMessage[]>([
    {
      id: 'msg_welcome',
      sender: 'assistant',
      text: 'Ciao! Sono Domus Assistant, il tuo assistente locale offline. Come posso aiutarti oggi?',
      engine: 'local',
      timestamp: new Date().toISOString()
    }
  ])

  const isProcessing = ref(false)
  const error = ref<string | null>(null)
  const config = ref<AssistantConfig>({
    ollama_url: 'http://localhost:11434',
    model: 'llama3.2:3b',
    enabled: true,
    system_prompt: 'Sei Domus Assistant, l\'assistente vocale locale e privato della casa domotica.'
  })

  async function fetchConfig(): Promise<AssistantConfig> {
    try {
      const cfg = await apiClient.get<AssistantConfig>('/api/v1/assistant/config')
      config.value = cfg
      return cfg
    } catch {
      return config.value
    }
  }

  async function updateConfig(newCfg: Partial<AssistantConfig>): Promise<AssistantConfig> {
    try {
      const updated = await apiClient.put<AssistantConfig>('/api/v1/assistant/config', newCfg)
      config.value = updated
      return updated
    } catch (err: any) {
      error.value = err?.message || 'Errore salvataggio config assistente'
      throw err
    }
  }

  async function sendPrompt(prompt: string): Promise<AssistantChatResponse | null> {
    const trimmed = prompt.trim()
    if (!trimmed) return null

    const userMsgId = 'msg_' + Date.now() + '_u'
    messages.value.push({
      id: userMsgId,
      sender: 'user',
      text: trimmed,
      timestamp: new Date().toISOString()
    })

    isProcessing.value = true
    error.value = null

    try {
      const resp = await apiClient.post<AssistantChatResponse>('/api/v1/assistant/chat', { prompt: trimmed })
      
      const assistantMsgId = 'msg_' + Date.now() + '_a'
      messages.value.push({
        id: assistantMsgId,
        sender: 'assistant',
        text: resp.response_text,
        actions: resp.actions_executed,
        engine: resp.engine_used,
        timestamp: new Date().toISOString()
      })

      return resp
    } catch (err: any) {
      const fallbackMsg = err?.message || 'Errore elaborazione comando assistente'
      error.value = fallbackMsg
      messages.value.push({
        id: 'msg_err_' + Date.now(),
        sender: 'assistant',
        text: `⚠️ Si è verificato un errore: ${fallbackMsg}`,
        timestamp: new Date().toISOString()
      })
      return null
    } finally {
      isProcessing.value = false
    }
  }

  function clearHistory() {
    messages.value = [
      {
        id: 'msg_welcome',
        sender: 'assistant',
        text: 'Cronologia azzerata. Come posso aiutarti?',
        engine: 'local',
        timestamp: new Date().toISOString()
      }
    ]
  }

  return {
    messages,
    isProcessing,
    error,
    config,
    fetchConfig,
    updateConfig,
    sendPrompt,
    clearHistory
  }
})
